#!/usr/bin/env python3
"""Apply explicit forge item definitions to stopped, explicitly selected DBs.

Only PS_GameDefs_Items.ReqVg/Count are changed. Character data is untouched.
Backups are required and created using SQLite backup, never by replacing a
personal runtime with the distributed seed.
"""
import argparse
from contextlib import closing
import csv
import hashlib
import io
import json
from pathlib import Path
import sqlite3


def read_patches(path):
    with path.open(newline='', encoding='utf-8') as f:
        rows=list(csv.DictReader(f))
    return [(int(r['type']),int(r['type_id']),int(r['req_vg']),int(r['stack'])) for r in rows]


def apply(path, patches):
    if not path.is_file(): raise ValueError('missing database '+str(path))
    with closing(sqlite3.connect(path)) as db:
        with db:
            for typ,identity,vg,stack in patches:
                if not 0<=vg<=100 or stack not in (0,255): raise ValueError('invalid forge definition')
                row=db.execute('SELECT Special FROM PS_GameDefs_Items WHERE Type=? AND TypeID=?',(typ,identity)).fetchone()
                if row is None: raise ValueError('missing forge item')
                if row[0] not in (36,58) and typ!=95: raise ValueError('refusing unrelated item')
                db.execute('UPDATE PS_GameDefs_Items SET ReqVg=?,Count=CASE WHEN ?=0 THEN Count ELSE ? END WHERE Type=? AND TypeID=?',(vg,stack,stack,typ,identity))
            if db.execute('PRAGMA integrity_check').fetchone()[0]!='ok': raise ValueError('integrity failure')
        limits=list(db.execute('SELECT Type*1000+TypeID,CASE WHEN Count>1 THEN 255 ELSE 1 END FROM PS_GameDefs_Items ORDER BY Type,TypeID'))
    manifest=path.with_suffix('.manifest.json')
    if manifest.exists():
        doc=json.loads(manifest.read_text());doc['sha256']=hashlib.sha256(path.read_bytes()).hexdigest();manifest.write_text(json.dumps(doc,indent=2)+'\n')
    return limits


def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--database',action='append',type=Path,required=True)
    ap.add_argument('--backup-dir',type=Path,required=True)
    ap.add_argument('--patches',type=Path,default=Path(__file__).parent/'data/forge-item-patches.csv')
    ap.add_argument('--client-data',type=Path,required=True)
    args=ap.parse_args();args.backup_dir.mkdir(parents=True,exist_ok=True)
    patches=read_patches(args.patches)
    for n,path in enumerate(args.database):
        if not path.is_file():raise ValueError('missing database')
        backup=args.backup_dir/f'database-{n}-before.sqlite'
        if backup.exists():raise ValueError('backup already exists; choose a new backup directory')
        with closing(sqlite3.connect(path)) as source, closing(sqlite3.connect(backup)) as dest:source.backup(dest)
    limits=None
    for path in args.database:
        current=apply(path,patches)
        if limits is not None and current!=limits:raise ValueError('target item definitions differ')
        limits=current
    output=io.StringIO(newline='');w=csv.writer(output,lineterminator='\n');w.writerow(['item_id','max_stack']);w.writerows(limits)
    for target in (args.database[0].parent/'item-stack-limits.csv',args.client_data/'item-stack-limits.csv'):target.write_text(output.getvalue())
    print(f'Updated {len(patches)} forge definitions in {len(args.database)} databases; no character rows edited')

if __name__=='__main__':main()
