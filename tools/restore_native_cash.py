#!/usr/bin/env python3
"""One-time migration from the former CSV catalog to Cash.SData/ProductList.

Stop local services first. Only explicitly selected DBs are edited; existing
character quantities above 255 are split without loss, with new item UIDs.
"""
import argparse
import csv
import hashlib
import io
import json
import secrets
import sqlite3
import struct
from pathlib import Path


def catalog(source):
    with source.open(newline='', encoding='utf-8-sig') as f:
        rows = list(csv.DictReader(f))
    codes = set()
    for p in rows:
        if not p['code'] or p['code'] in codes:
            raise ValueError('empty or duplicate product code')
        codes.add(p['code'])
        for i in range(1, 25):
            key = f'quantity_{i}'
            p[key] = str(min(int(p[key] or 0), 255))
    return rows


def encode(rows):
    out = bytearray(struct.pack('<I', len(rows)))
    for p in rows:
        out.extend(struct.pack('<4I', *(int(p[k]) for k in ('index','category','icon','price'))))
        for i in range(1,25):
            out.extend(struct.pack('<IB', int(p[f'item_id_{i}'] or 0), int(p[f'quantity_{i}'] or 0)))
        for key in ('name','code','description'):
            value=p[key].encode('utf-8')
            out.extend(struct.pack('<I', len(value)))
            out.extend(value)
    return bytes(out)


def migrate(path, rows):
    with sqlite3.connect(path) as db:
        db.row_factory=sqlite3.Row
        db.execute('BEGIN IMMEDIATE')
        # Inventory splitting is planned before any persistent changes; any
        # unsupported oversized storage row aborts the entire transaction.
        for table in ('PS_GameData_UserStoredItems','PS_GameData_GuildStoredItems','PS_GameData_MarketItems'):
            if db.execute('SELECT COUNT(*) FROM '+table+' WHERE Count>255').fetchone()[0]:
                raise ValueError('oversized storage needs explicit slot migration: '+table)
        if db.execute('SELECT COUNT(*) FROM PS_GameData_UserStoredPointItems WHERE ItemCount>255').fetchone()[0]:
            raise ValueError('oversized gifts need explicit slot migration')
        changes=[]
        for row in db.execute('SELECT * FROM PS_GameData_CharItems WHERE Count>255 AND COALESCE(Del,0)=0').fetchall():
            if row['Bag']==0:
                raise ValueError('oversized equipped item')
            occupied={(r[0],r[1]) for r in db.execute('SELECT Bag,Slot FROM PS_GameData_CharItems WHERE CharID=? AND COALESCE(Del,0)=0',(row['CharID'],))}
            free=[(b,s) for b in range(1,6) for s in range(24) if (b,s) not in occupied]
            remaining=row['Count']-255
            if len(free)<(remaining+254)//255:
                raise ValueError('not enough inventory slots; no quantities discarded')
            db.execute('UPDATE PS_GameData_CharItems SET Count=255 WHERE CharID=? AND ItemUID=?',(row['CharID'],row['ItemUID']))
            added=0
            for bag,slot in free:
                if remaining==0: break
                new=dict(row)
                new.update(RowID=db.execute('SELECT COALESCE(MAX(RowID),0)+1 FROM PS_GameData_CharItems').fetchone()[0],ItemUID=secrets.randbits(62),Bag=bag,Slot=slot,Count=min(remaining,255))
                columns=list(new)
                db.execute('INSERT INTO PS_GameData_CharItems('+','.join('"'+c+'"' for c in columns)+') VALUES('+','.join('?' for _ in columns)+')',[new[c] for c in columns])
                remaining-=new['Count']; added+=1
            changes.append({'char':row['CharID'],'item_uid':row['ItemUID'],'total':row['Count'],'additional_stacks':added})
        for p in rows:
            for i in range(1,25):
                item=int(p[f'item_id_{i}'] or 0)
                if item and db.execute('SELECT 1 FROM PS_GameDefs_Items WHERE ItemID=?',(item,)).fetchone() is None:
                    raise ValueError('unknown catalog item '+str(item))
        columns=['RowID','ProductName','ProductCode','BuyCost']
        columns += [c for i in range(1,25) for c in (f'ItemID{i}',f'ItemCount{i}')]
        db.execute('DELETE FROM PS_GameDefs_ProductList')
        for n,p in enumerate(rows,1):
            values=[n,p['name'],p['code'],int(p['price'])]
            values += [int(p[k] or 0) for i in range(1,25) for k in (f'item_id_{i}',f'quantity_{i}')]
            db.execute('INSERT INTO PS_GameDefs_ProductList('+','.join(columns)+') VALUES('+','.join('?' for _ in columns)+')',values)
        db.execute('UPDATE PS_GameDefs_Items SET Count=255 WHERE Count>1')
        limits=[(int(r[0]),255 if int(r[1] or 0)>1 else 1) for r in db.execute('SELECT ItemID,Count FROM PS_GameDefs_Items ORDER BY ItemID')]
        if db.execute('PRAGMA integrity_check').fetchone()[0]!='ok': raise ValueError('integrity check failed')
    return changes,limits


def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--source',required=True,type=Path)
    ap.add_argument('--database',action='append',required=True,type=Path)
    ap.add_argument('--client-data',required=True,type=Path)
    ap.add_argument('--report',required=True,type=Path)
    args=ap.parse_args()
    rows=catalog(args.source)
    report={'products':len(rows),'databases':{}}
    limits=None
    for path in args.database:
        if not path.is_file(): raise ValueError('missing database')
        changes,limits=migrate(path,rows)
        report['databases'][str(path)]=changes
        manifest=path.with_suffix('.manifest.json')
        if manifest.exists():
            doc=json.loads(manifest.read_text())
            doc['sha256']=hashlib.sha256(path.read_bytes()).hexdigest()
            doc['tables']['PS_GameDefs_ProductList']=len(rows)
            manifest.write_text(json.dumps(doc,indent=2)+'\n')
    raw=encode(rows)
    target=args.client_data/'Cash.SData'
    temporary=target.with_suffix('.SData.new')
    temporary.write_bytes(raw); temporary.replace(target)
    out=io.StringIO(newline='')
    writer=csv.writer(out,lineterminator='\n');writer.writerow(['item_id','max_stack']);writer.writerows(limits)
    for target in [args.client_data/'item-stack-limits.csv',args.database[0].parent/'item-stack-limits.csv']:
        target.write_text(out.getvalue())
    report['cash_sha256']=hashlib.sha256(raw).hexdigest()
    args.report.write_text(json.dumps(report,indent=2)+'\n')
    print(json.dumps(report))

if __name__=='__main__': main()
