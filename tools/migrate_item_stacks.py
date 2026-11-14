#!/usr/bin/env python3
"""Legacy CSV import preparation: cap supplies at 255 and export client limits.

Runtime uses Cash.SData/ProductList; use restore_native_cash.py to install.

Only definition counts change in SQL; existing player items stay untouched.
Run with stopped services. Never replaces a runtime with a seed database.
"""
import argparse
from contextlib import closing
import csv
import hashlib
import io
import json
import sqlite3
from pathlib import Path


def migrate(databases, catalog, client_data):
    limits = None
    # Validate every target before writing any database.
    for path in databases:
        if not path.is_file():
            raise ValueError(f'Database does not exist: {path}')
        with closing(sqlite3.connect(path.resolve().as_uri()+'?mode=ro', uri=True)) as db:
            current = {int(t)*1000+int(i): 255 if int(n or 0)>1 else 1
                       for t,i,n in db.execute('SELECT Type,TypeID,Count FROM PS_GameDefs_Items')}
        if limits is None:
            limits = current
        elif limits != current:
            raise ValueError('Item definitions differ between databases')
    for path in databases:
        with closing(sqlite3.connect(path)) as db, db:
            db.execute('UPDATE PS_GameDefs_Items SET Count=255 WHERE Count>1')
        manifest = path.with_suffix('.manifest.json')
        if manifest.exists():
            doc=json.loads(manifest.read_text())
            doc['sha256']=hashlib.sha256(path.read_bytes()).hexdigest()
            manifest.write_text(json.dumps(doc,indent=2)+'\n')
    output=io.StringIO(newline='')
    writer=csv.writer(output,lineterminator='\n')
    writer.writerow(['item_id','max_stack'])
    writer.writerows(sorted(limits.items()))
    data=output.getvalue()
    catalog.with_name('item-stack-limits.csv').write_text(data)
    (client_data/'item-stack-limits.csv').write_text(data)
    with catalog.open(newline='') as f:
        rows=list(csv.reader(f))
    changed=0
    for row in rows[1:]:
        for col in range(7,len(row),2):
            if limits.get(int(row[col] or 0))==255 and int(row[col+1] or 0)>0:
                changed += row[col+1]!='255'
                row[col+1]='255'
    with catalog.open('w',newline='') as f:
        csv.writer(f,lineterminator='\n').writerows(rows)
    (client_data/'cash.csv').write_bytes(catalog.read_bytes())
    print(f'{len(limits)} item definitions; {sum(n==255 for n in limits.values())} stackable; {changed} mall quantities updated')


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--database',type=Path,action='append',required=True)
    parser.add_argument('--catalog',type=Path,required=True)
    parser.add_argument('--client-data',type=Path,required=True)
    args=parser.parse_args()
    migrate(args.database,args.catalog,args.client_data)
