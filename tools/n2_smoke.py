#!/usr/bin/env python3
"""N2 against isolated SQLite login/world processes and a disposable DB copy.

Requires Rapture's build/ binaries and the original client assets. No Docker,
SQL credentials, personal runtime DB or existing game service is used.
"""
import os,secrets,shutil,socket,sqlite3,subprocess,tempfile,time
from pathlib import Path
client=Path(__file__).resolve().parents[1]
server=Path(os.environ.get('RAPTURE_DIR',Path.home()/'Desktop/Rapture')).expanduser().resolve()
children=[]
with tempfile.TemporaryDirectory(prefix='rapture-sqlite-n2-') as temp:
 temp=Path(temp);db=temp/'rapture.sqlite';shutil.copyfile(server/'data/rapture.sqlite',db)
 env=dict(os.environ,RAPTURE_DB_PATH=str(db),RAPTURE_DATA_DIR=str(server/'data'),RAPTURE_LOGIN_LISTEN='127.0.0.1:40800',RAPTURE_WORLD_LISTEN='127.0.0.1:40810',RAPTURE_PUBLIC_IP='127.0.0.1',SHAIYA_TEST_LOGIN_ADDR='127.0.0.1:40800',SHAIYA_TEST_WORLD_PORT='40810',SHAIYA_TEST_DISPOSABLE='1',SHAIYA_TEST_RENDER='1',SHAIYA_TEST_NATIVE_CONTROLS='1',SHAIYA_TEST_ACTIONS='1',SHAIYA_TEST_ARTIFACTS=str(client/'work'),SHAIYA_DATA=os.environ.get('SHAIYA_DATA',str(Path.home()/'Desktop/Go project')),GOTOOLCHAIN='local')
 env['PATH']=str(Path.home()/'.local/opt/go-1.27.3/bin')+':'+env['PATH']
 for port in (40800,40810):
  with socket.socket() as check:
   if check.connect_ex(('127.0.0.1',port))==0:raise RuntimeError('test port is already in use: '+str(port))
 users=[]
 with sqlite3.connect(db) as c:
  for _ in range(2):
   name='gop'+secrets.token_hex(5);password=secrets.token_hex(6);uid=c.execute('SELECT MAX(UserUID)+1 FROM PS_UserData_Users_Master').fetchone()[0]
   c.execute("INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,?,0,0,0,0,0,'N',0)",(uid,name,password));users.append((name,password,uid))
 try:
  for name,port in [('rapture_login',40800),('rapture_world',40810)]:
   log=open(client/('work/sqlite-'+name+'.log'),'w')
   process=subprocess.Popen(['nice',str(server/'build'/name)],cwd=temp,env=env,stdout=log,stderr=subprocess.STDOUT);children.append(process)
   deadline=time.monotonic()+30
   while True:
    with socket.socket() as s:
     if s.connect_ex(('127.0.0.1',port))==0:break
    if process.poll() is not None or time.monotonic()>deadline:raise RuntimeError('server startup failed: '+name)
    time.sleep(.1)
  for name,password,uid in users:
   account_env=dict(env,SHAIYA_TEST_USER=name,SHAIYA_PASSWORD=password)
   for test,extra in [('TestFactionAgainstRapture',{'SHAIYA_TEST_FACTION':'1'}),('TestCreationAgainstRapture',{'SHAIYA_TEST_CREATE':'Go'+secrets.token_hex(5)})]:
    subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^'+test+'$','-count=1','-v'],cwd=client,env=dict(account_env,**extra),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   for n,(_,_,uid) in enumerate(users):c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=?,PosY=78,PosZ=1762 WHERE UserUID=?',(578+n*2,uid))
  with sqlite3.connect(db) as c:
   uid=users[0][2]
   char_id=c.execute('SELECT CharID FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(uid,)).fetchone()[0]
   c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=1 AND Slot=23',(char_id,))
   item_uid=c.execute('SELECT COALESCE(MAX(ItemUID),0)+1 FROM PS_GameData_CharItems').fetchone()[0]
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,25004,?,25,4,1,23,0,0,0,0,0,0,0,'',5,datetime('now'),'N2',0)",(char_id,item_uid))
  env.update(SHAIYA_TEST_USER=users[0][0],SHAIYA_PASSWORD=users[0][1],SHAIYA_TEST_PARTNER=users[1][0],SHAIYA_TEST_PARTNER_PASSWORD=users[1][1])
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestNativeControlsAgainstRapture$','-count=1','-v'],cwd=client,env=env,check=True,timeout=120)
 finally:
  for process in reversed(children):
   process.terminate()
   try:process.wait(timeout=10)
   except subprocess.TimeoutExpired:process.kill();process.wait()
 print('N2 SQLite: isolated servers stopped, temporary database and accounts removed')
