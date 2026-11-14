#!/usr/bin/env python3
"""Explicit offline content migration: 255 gems, 18 targeted runes and native mall.

Back up each named DB and both client tables. Only Items/ProductList change;
existing accounts, characters and inventory quantities are left untouched.
Input client tables must be native plaintext SData (the installed GX format).
"""
import argparse
import csv
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import sqlite3
import struct
from contextlib import closing

STATS = ('STR','DEX','REC','INT','WIS','LUC','HP','MP','SP')

def runes():
    return [dict(type_id=mode*9+n+1, req_vg=(mode+1)*100+n+1,
                 name=f'{"Random" if mode==0 else "Perfect"} {stat} Recreation Rune',
                 description=f'{"Rerolls" if mode==0 else "Maximizes"} only {stat}. Preserves other recreation stats, enchantment and lapis. Respects the equipment stat pool, maximum OJ count and roll limit. Used at the blacksmith; consumes one rune.')
            for mode in range(2) for n,stat in enumerate(STATS)]

class Native:
    def __init__(self, raw): self.raw,self.pos=raw,0
    def take(self,n):
        if n<0 or self.pos+n>len(self.raw): raise ValueError('Truncated native SData')
        p=self.raw[self.pos:self.pos+n];self.pos+=n;return p
    def word(self): return struct.unpack('<I',self.take(4))[0]
    def string(self): return self.take(self.word())
    def end(self):
        if self.pos!=len(self.raw): raise ValueError('Trailing native SData bytes')

def string(value):
    if isinstance(value,str):value=value.encode('utf-8')
    return struct.pack('<I',len(value))+value

def patch_items(raw):
    src=Native(raw); count=src.word()
    if not 1<=count<=256: raise ValueError('Expected plaintext Item.SData')
    groups=[];template=None;seen=set()
    for _ in range(count):
        records=[]
        for _ in range(src.word()):
            name,desc,body=src.string(),src.string(),bytearray(src.take(81))
            seen.add((body[0],body[1]))
            if body[0] in (30,95):body[80]=255
            if (body[0],body[1])==(100,171):template=body[:]
            # Idempotent migrations replace only our reserved definitions.
            if body[0]==101 and 1<=body[1]<=18:
                expected=next(r for r in runes() if r['type_id']==body[1])
                if body[35]!=62 or struct.unpack_from('<H',body,28)[0]!=expected['req_vg']:
                    raise ValueError('Reserved native rune ID already occupied')
                continue
            records.append((name,desc,body))
        groups.append(records)
    src.end()
    if template is None:raise ValueError('Missing original recreation rune')
    while len(groups)<101:groups.append([])
    for rune in runes():
        body=template[:];body[0],body[1],body[35],body[80]=101,rune['type_id'],62,255
        struct.pack_into('<H',body,28,rune['req_vg'])
        groups[100].append((rune['name'].encode(),rune['description'].encode(),body))
    out=bytearray(struct.pack('<I',len(groups)))
    for records in groups:
        out.extend(struct.pack('<I',len(records)))
        for name,desc,body in records:out.extend(string(name)+string(desc)+body)
    return bytes(out)

def read_cash(raw):
    src=Native(raw);out=[];count=src.word()
    if count>100000:raise ValueError('Expected plaintext Cash.SData')
    for _ in range(count):
        nums=struct.unpack('<4I',src.take(16))
        items=[list(struct.unpack('<IB',src.take(5))) for _ in range(24)]
        fields=[src.string() for _ in range(3)]
        out.append(dict(nums=list(nums),items=items,fields=fields))
    src.end();return out

def patch_cash(raw):
    products=read_cash(raw)
    for p in products:
        for item in p['items']:
            if item[0]//1000 in (30,95):item[1]=255
    index=max(p['nums'][0] for p in products)
    codes={p['fields'][1] for p in products}
    for r in runes():
        code=f'GX_Recreate_{r["type_id"]:02}'.encode()
        existing=next((p for p in products if p['fields'][1]==code),None)
        fields=[r['name'].encode(),code,r['description'].encode()]
        items=[[101000+r['type_id'],255]]+[[0,0] for _ in range(23)]
        if existing:existing.update(items=items,fields=fields)
        else:
            index+=1;products.append(dict(nums=[index,4,0,0],items=items,fields=fields))
    out=bytearray(struct.pack('<I',len(products)))
    for p in products:
        out.extend(struct.pack('<4I',*p['nums']))
        for item in p['items']:out.extend(struct.pack('<IB',*item))
        for field in p['fields']:out.extend(string(field))
    return bytes(out),products

def apply_database(path,products):
    with closing(sqlite3.connect(path)) as db:
        db.row_factory=sqlite3.Row
        with db:
            template=db.execute('SELECT * FROM PS_GameDefs_Items WHERE Type=100 AND TypeID=171').fetchone()
            if template is None:raise ValueError('Missing SQL recreation rune')
            db.execute('UPDATE PS_GameDefs_Items SET Count=255 WHERE Type IN (30,95)')
            for rune in runes():
                key=101000+rune['type_id']
                existing=db.execute('SELECT Special,ReqVg FROM PS_GameDefs_Items WHERE ItemID=?',(key,)).fetchone()
                if existing and tuple(existing)!=(62,rune['req_vg']):raise ValueError('Reserved custom rune ID already occupied')
                values=dict(template)
                values.update(ItemID=key,Type=101,TypeID=rune['type_id'],ItemName=rune['name'],ReqVg=rune['req_vg'],Count=255,Special=62)
                if 'RowID' in values:values['RowID']=db.execute('SELECT COALESCE(MAX(RowID),0)+1 FROM PS_GameDefs_Items').fetchone()[0]
                if existing:
                    db.execute('UPDATE PS_GameDefs_Items SET ItemName=?,Count=255 WHERE ItemID=?',(rune['name'],key))
                else:
                    columns=list(values)
                    db.execute('INSERT INTO PS_GameDefs_Items('+','.join('"'+c+'"' for c in columns)+') VALUES('+','.join('?' for _ in columns)+')',[values[c] for c in columns])
            for p in products:
                code=p['fields'][1].decode()
                found=db.execute('SELECT 1 FROM PS_GameDefs_ProductList WHERE ProductCode=?',(code,)).fetchone()
                values=dict(ProductName=p['fields'][0].decode(),ProductCode=code,BuyCost=p['nums'][3])
                for i,(identity,count) in enumerate(p['items'],1):
                    if identity and not db.execute('SELECT 1 FROM PS_GameDefs_Items WHERE ItemID=?',(identity,)).fetchone():raise ValueError('Unknown mall item '+str(identity))
                    values[f'ItemID{i}'],values[f'ItemCount{i}']=identity,count
                if found:
                    keys=[k for k in values if k!='ProductCode']
                    db.execute('UPDATE PS_GameDefs_ProductList SET '+','.join(k+'=?' for k in keys)+' WHERE ProductCode=?',[values[k] for k in keys]+[code])
                else:
                    values['RowID']=db.execute('SELECT COALESCE(MAX(RowID),0)+1 FROM PS_GameDefs_ProductList').fetchone()[0]
                    keys=list(values)
                    db.execute('INSERT INTO PS_GameDefs_ProductList('+','.join(keys)+') VALUES('+','.join('?' for _ in keys)+')',[values[k] for k in keys])
            if db.execute('PRAGMA integrity_check').fetchone()[0]!='ok':raise ValueError('Database integrity failure')
        limits=list(db.execute('SELECT Type*1000+TypeID,CASE WHEN Count>1 THEN 255 ELSE 1 END FROM PS_GameDefs_Items ORDER BY Type,TypeID'))
        manifest=path.with_suffix('.manifest.json')
        if manifest.exists():
            metadata=json.loads(manifest.read_text())
            for table in ('PS_GameDefs_Items','PS_GameDefs_ProductList'):
                metadata['tables'][table]=db.execute('SELECT COUNT(*) FROM '+table).fetchone()[0]
            metadata['sha256']=hashlib.sha256(path.read_bytes()).hexdigest()
            manifest.write_text(json.dumps(metadata,indent=2)+'\n')
    return [tuple(row) for row in limits]

def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--database',action='append',type=Path,required=True)
    ap.add_argument('--client-data',type=Path,required=True)
    ap.add_argument('--backup-dir',type=Path,required=True)
    args=ap.parse_args();args.backup_dir.mkdir(parents=True,exist_ok=False)
    item=args.client_data/'Item/Item.SData';cash=args.client_data/'Cash.SData'
    patched_items=patch_items(item.read_bytes());patched_cash,products=patch_cash(cash.read_bytes())
    shutil.copy2(item,args.backup_dir/'Item.SData');shutil.copy2(cash,args.backup_dir/'Cash.SData')
    for i,path in enumerate(args.database):
        if not path.is_file():raise ValueError('Missing explicit database')
        with closing(sqlite3.connect(path)) as source,closing(sqlite3.connect(args.backup_dir/f'database-{i}.sqlite')) as backup:source.backup(backup)
    # Verify every migration against disposable copies before touching any target.
    for i in range(len(args.database)):
        trial=args.backup_dir/f'trial-{i}.sqlite';shutil.copy2(args.backup_dir/f'database-{i}.sqlite',trial)
        apply_database(trial,products);trial.unlink()
    limits=None
    for path in args.database:
        current=apply_database(path,products)
        if limits is not None and current!=limits:raise ValueError('Stack definitions differ between targets')
        limits=current
    for target,content in ((item,patched_items),(cash,patched_cash)):
        staged=target.with_suffix('.gx-new');staged.write_bytes(content);os.replace(staged,target)
    output=io.StringIO();writer=csv.writer(output,lineterminator='\n');writer.writerow(['item_id','max_stack']);writer.writerows(limits)
    for target in (args.client_data/'item-stack-limits.csv',args.database[0].parent/'item-stack-limits.csv'):target.write_text(output.getvalue())
    print(json.dumps({'runes':runes(),'products':len(products),'databases':[str(p) for p in args.database]},indent=2))

if __name__=='__main__':main()
