#!/usr/bin/env python3
"""Copy the current EP databases to one portable SQLite file (source read-only)."""
import argparse,base64,hashlib,json,re,sqlite3
from pathlib import Path
import sqlserver

def query_json(query,database):
    args=['exec','-i',sqlserver.CONTAINER,'sh','-c','export SQLCMDPASSWORD="$MSSQL_SA_PASSWORD"; exec "$@"','sqlcmd',sqlserver.SQLCMD,'-S','localhost','-U','sa','-C','-b','-r','1','-w','65535','-y','0','-d',database]
    result=sqlserver.docker(*args,input='SET NOCOUNT ON;\n'+query+'\nGO\n',capture_output=True)
    raw=''.join(result.stdout.splitlines()).strip()
    return json.loads(raw[raw.index('['):]) if '[' in raw else []

def quote(s):return '"'+s.replace('"','""')+'"'

def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--output',type=Path,required=True);a=p.parse_args()
    if a.output.exists():p.error('output already exists; choose a new path')
    a.output.parent.mkdir(parents=True,exist_ok=True)
    target=sqlite3.connect(a.output)
    report={};procedures={}
    try:
      for database in ['PS_UserData','PS_GameData','PS_GameDefs','PS_Billing']:
        schema=query_json("""SELECT t.name AS name,c.name AS column_name,ty.name AS type,c.is_nullable AS nullable,c.is_identity AS identity_column,dc.definition AS default_value,
ISNULL(ic.key_ordinal,0) AS pk FROM sys.tables t JOIN sys.columns c ON c.object_id=t.object_id JOIN sys.types ty ON ty.user_type_id=c.user_type_id
LEFT JOIN sys.default_constraints dc ON dc.object_id=c.default_object_id
LEFT JOIN sys.indexes ix ON ix.object_id=t.object_id AND ix.is_primary_key=1
LEFT JOIN sys.index_columns ic ON ic.object_id=t.object_id AND ic.index_id=ix.index_id AND ic.column_id=c.column_id
ORDER BY t.name,c.column_id FOR JSON PATH, INCLUDE_NULL_VALUES""",database)
        tables={}
        for c in schema:tables.setdefault(c['name'],[]).append(c)
        procedures[database]=query_json('SELECT o.name,m.definition FROM sys.sql_modules m JOIN sys.objects o ON o.object_id=m.object_id FOR JSON PATH',database)
        for table,columns in tables.items():
          name=database+'_'+table
          defs=[];pk=sorted([c for c in columns if c['pk']],key=lambda c:c['pk'])
          for c in columns:
            typ=c['type'].lower()
            kind='INTEGER' if typ in ('bit','tinyint','smallint','int','bigint') else 'REAL' if typ in ('real','float','money','smallmoney','decimal','numeric') else 'BLOB' if typ in ('binary','varbinary','image','timestamp') else 'TEXT COLLATE NOCASE'
            auto=c['identity_column'] and len(pk)==1 and c['pk']
            field=quote(c['column_name'])+' '+kind
            if auto:field+=' PRIMARY KEY AUTOINCREMENT'
            elif not c['nullable']:field+=' NOT NULL'
            if c['identity_column'] and not auto:field+=' DEFAULT 0'
            default=c['default_value']
            if default is not None:
              default=default.strip()
              while default.startswith('(') and default.endswith(')'):default=default[1:-1]
              if re.fullmatch(r'get(?:utc)?date\(\)',default,re.I):default='CURRENT_TIMESTAMP'
              if re.fullmatch(r"N?'(?:[^']|'')*'",default):default=re.sub(r'^N', '',default)
              if re.fullmatch(r"[-+\d.]+|'(?:[^']|'')*'|CURRENT_TIMESTAMP",default):field+=' DEFAULT '+default
              else:raise ValueError(f'unhandled default {name}.{c["column_name"]}: {default}')
            defs.append(field)
          if pk and not any(c['identity_column'] for c in pk):defs.append('PRIMARY KEY ('+','.join(quote(c['column_name']) for c in pk)+')')
          target.execute('CREATE TABLE '+quote(name)+' ('+','.join(defs)+')')
          # JSON contains exact 64-bit integers and escaped text; never parse tabular output.
          data=query_json('SELECT * FROM '+sqlserver.identifier(table)+' FOR JSON PATH, INCLUDE_NULL_VALUES',database)
          count=query_json('SELECT COUNT_BIG(*) AS total FROM '+sqlserver.identifier(table)+' FOR JSON PATH',database)
          if len(count)!=1 or count[0]['total']!=len(data):raise ValueError('source row count changed during export: '+name)
          names=[c['column_name'] for c in columns];values=[]
          for row in data:
            val=[]
            for c in columns:
              x=row.get(c['column_name'])
              if x is not None and c['type'].lower() in ('binary','varbinary','image','timestamp'):x=base64.b64decode(x)
              if x is not None and c['type'].lower() in ('datetime','datetime2','smalldatetime','date'):x=x.replace('T',' ')
              val.append(x)
            values.append(val)
          target.executemany('INSERT INTO '+quote(name)+' ('+','.join(map(quote,names))+') VALUES ('+','.join('?' for _ in names)+')',values)
          if any(c['identity_column'] and c['pk'] and len(pk)==1 for c in columns):
            identity=query_json("SELECT CAST(IDENT_CURRENT(N'"+table.replace("'","''")+"') AS bigint) AS value FOR JSON PATH",database)
            last=identity[0].get('value') if identity else None
            if last is not None:
              if target.execute('UPDATE sqlite_sequence SET seq=MAX(seq,?) WHERE name=?',(last,name)).rowcount==0:target.execute('INSERT INTO sqlite_sequence(name,seq) VALUES(?,?)',(name,last))
          indexes=query_json("SELECT ix.name,ix.is_unique,c.name AS column_name,ic.key_ordinal FROM sys.indexes ix JOIN sys.index_columns ic ON ic.object_id=ix.object_id AND ic.index_id=ix.index_id JOIN sys.columns c ON c.object_id=ic.object_id AND c.column_id=ic.column_id WHERE ix.object_id=OBJECT_ID(N'"+table.replace("'","''")+"') AND ix.is_primary_key=0 AND ix.type IN (1,2) AND ic.key_ordinal>0 ORDER BY ix.name,ic.key_ordinal FOR JSON PATH",database)
          groups={}
          for ix in indexes:groups.setdefault(ix['name'],[]).append(ix)
          for index,cols in groups.items():target.execute('CREATE '+('UNIQUE ' if cols[0]['is_unique'] else '')+'INDEX '+quote(database+'_'+index)+' ON '+quote(name)+' ('+','.join(quote(c['column_name']) for c in cols)+')')
          for c in columns:
            if c['identity_column'] and not (len(pk)==1 and c['pk']):
              col=quote(c['column_name']);tab=quote(name)
              target.execute('CREATE TRIGGER '+quote(name+'_identity')+' AFTER INSERT ON '+tab+' WHEN NEW.'+col+'=0 BEGIN UPDATE '+tab+' SET '+col+'=(SELECT COALESCE(MAX('+col+'),0)+1 FROM '+tab+' WHERE _rowid_<>NEW._rowid_) WHERE _rowid_=NEW._rowid_; END')
          report[name]=len(values)
          print(name,len(values),flush=True)
      target.execute('CREATE TABLE Rapture_LoginLog(ID INTEGER PRIMARY KEY AUTOINCREMENT,UserUID INTEGER NOT NULL,SessionID INTEGER NOT NULL,UserIP TEXT,LogType INTEGER NOT NULL,LogTime TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)')
      report['Rapture_LoginLog']=0
      target.execute('PRAGMA user_version=1')
      target.commit()
      if target.execute('PRAGMA integrity_check').fetchone()[0]!='ok':raise RuntimeError('integrity check failed')
      target.close()
      a.output.with_suffix('.manifest.json').write_text(json.dumps({'tables':report,'sha256':hashlib.sha256(a.output.read_bytes()).hexdigest()},indent=2)+'\n')
      # Original procedure bodies are local migration evidence, not runtime SQL.
      Path('.local/sqlserver-procedures.json').write_text(json.dumps(procedures,indent=2))
    except BaseException:
      target.close();a.output.unlink(missing_ok=True);raise

if __name__=='__main__':main()
