#!/usr/bin/env python3
"""Initialize/query the portable development database or create a local account."""
import argparse,getpass,os,shutil,sqlite3
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]

def main():
 p=argparse.ArgumentParser(description=__doc__)
 p.add_argument('--path',type=Path,default=Path(os.environ.get('RAPTURE_DB_PATH',ROOT/'runtime/rapture.sqlite')))
 sub=p.add_subparsers(dest='action',required=True)
 sub.add_parser('init')
 q=sub.add_parser('query');q.add_argument('--file',type=Path,required=True)
 a=sub.add_parser('account');a.add_argument('name');a.add_argument('--admin',action='store_true')
 args=p.parse_args();path=args.path.expanduser().resolve()
 path.parent.mkdir(parents=True,exist_ok=True)
 if not path.exists():
  lock=Path(str(path)+'.init');lock.mkdir()
  try:
   temp=path.with_name('.'+path.name+'.next')
   shutil.copyfile(ROOT/'data/rapture.sqlite',temp)
   temp.replace(path)
  finally:lock.rmdir()
 with sqlite3.connect(path) as c:
  c.execute('PRAGMA busy_timeout=10000')
  if c.execute('PRAGMA user_version').fetchone()[0]!=1:raise RuntimeError('unsupported schema')
  if args.action=='init':print('SQLite ready:',path)
  elif args.action=='query':
   # Queries use explicit PS_GameData_/PS_UserData_/PS_GameDefs_ table prefixes.
   statement=''
   statements=[]
   for char in args.file.read_text():
    statement+=char
    if char==';' and sqlite3.complete_statement(statement):
     statements.append(statement);statement=''
   if statement.strip():statements.append(statement)
   for statement in statements:
    result=c.execute(statement)
    if result.description:
     for row in result:print('\t'.join('NULL' if x is None else str(x) for x in row))

  else:
   password=os.environ.get('RAPTURE_ACCOUNT_PASSWORD') or getpass.getpass('Password: ')
   if not 1<=len(args.name.encode("utf-8"))<=18 or not 1<=len(password.encode("utf-8"))<=16:raise ValueError('account/password exceeds UTF-8 byte limits (18/16)')
   c.execute('BEGIN IMMEDIATE')
   if c.execute('SELECT 1 FROM PS_UserData_Users_Master WHERE UserID=?',(args.name,)).fetchone():raise ValueError('account already exists')
   uid=c.execute('SELECT COALESCE(MAX(UserUID),0)+1 FROM PS_UserData_Users_Master').fetchone()[0]
   c.execute("INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,?,?,?,0,0,0,'N',0)",(uid,args.name,password,int(args.admin),255 if args.admin else 0))
   print('Account created:',args.name,'UID',uid)

if __name__=='__main__':main()
