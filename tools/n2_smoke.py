#!/usr/bin/env python3
"""N2 against isolated SQLite login/world processes and a disposable DB copy.

Requires Rapture's build/ binaries and the original client assets. No Docker,
SQL credentials, personal runtime DB or existing game service is used.
"""
import os,secrets,shutil,socket,sqlite3,subprocess,tempfile,time
from pathlib import Path
client=Path(__file__).resolve().parents[1]
server=Path(os.environ.get('GX_SERVER_DIR',os.environ.get('RAPTURE_DIR',client/'server'))).expanduser().resolve()
children=[]
with tempfile.TemporaryDirectory(prefix='rapture-sqlite-n2-') as temp:
 temp=Path(temp);db=temp/'rapture.sqlite';shutil.copyfile(server/'data/rapture.sqlite',db)
 env=dict(os.environ,RAPTURE_DB_PATH=str(db),RAPTURE_DATA_DIR=str(server/'data'),RAPTURE_LOGIN_LISTEN='127.0.0.1:40800',RAPTURE_WORLD_LISTEN='127.0.0.1:40810',RAPTURE_PUBLIC_IP='127.0.0.1',SHAIYA_TEST_LOGIN_ADDR='127.0.0.1:40800',SHAIYA_TEST_WORLD_PORT='40810',SHAIYA_TEST_DISPOSABLE='1',SHAIYA_TEST_RENDER='1',SHAIYA_TEST_NATIVE_CONTROLS='1',SHAIYA_TEST_ACTIONS='1',SHAIYA_TEST_ARTIFACTS=str(client/'work'),SHAIYA_DATA=os.environ.get('SHAIYA_DATA',str(Path.home()/'Desktop/Shaiya GX/data')),GOTOOLCHAIN='local')
 env['PATH']=str(Path.home()/'.local/opt/go-1.27.3/bin')+':'+env['PATH']
 for port in (40800,40810):
  with socket.socket() as check:
   if check.connect_ex(('127.0.0.1',port))==0:raise RuntimeError('test port is already in use: '+str(port))
 users=[]
 with sqlite3.connect(db) as c:
  for _ in range(2):
   name='測'+secrets.token_hex(5);password='密'+secrets.token_hex(5);uid=c.execute('SELECT MAX(UserUID)+1 FROM PS_UserData_Users_Master').fetchone()[0]
   c.execute("INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,?,0,0,0,0,0,'N',0)",(uid,name,password));users.append((name,password,uid))
 try:
  for name,port in [('gx_login',40800),('gx_world',40810)]:
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
   for test,extra in [('TestFactionAgainstRapture',{'SHAIYA_TEST_FACTION':'1'}),('TestCreationAgainstRapture',{'SHAIYA_TEST_CREATE':'試🧙'+secrets.token_hex(4)})]:
    subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^'+test+'$','-count=1','-v'],cwd=client,env=dict(account_env,**extra),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   for n,(_,_,uid) in enumerate(users):
    c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=?,PosY=78,PosZ=1762,Level=40,SkillPoint=100 WHERE UserUID=?',(578+n*2,uid))
   c.execute('UPDATE PS_GameData_Chars SET Family=0,Job=5 WHERE UserUID=?',(users[1][2],))
   c.execute('UPDATE PS_GameData_Chars SET HP=500 WHERE UserUID=?',(users[0][2],))
  with sqlite3.connect(db) as c:
   uid=users[0][2]
   char_id=c.execute('SELECT CharID FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(uid,)).fetchone()[0]
   c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=1 AND Slot=23',(char_id,))
   item_uid=c.execute('SELECT COALESCE(MAX(ItemUID),0)+1 FROM PS_GameData_CharItems').fetchone()[0]
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,25004,?,25,4,1,23,0,0,0,0,0,0,0,'',5,datetime('now'),'N2',0)",(char_id,item_uid))
   c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=1 AND Slot=22',(char_id,))
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,24001,?,24,1,1,22,0,0,0,0,0,0,0,'',1,datetime('now'),'N2',0)",(char_id,item_uid+1))
   # Enchant fixtures belong only to this disposable database.
   c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=0 AND Slot=5',(char_id,))
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,1001,?,1,1,0,5,1200,0,0,0,0,0,0,'00000000000000000005',1,datetime('now'),'N2',0)",(char_id,item_uid+2))
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,1001,?,1,1,1,21,1200,0,0,0,0,0,0,'00000000000000000000',1,datetime('now'),'N2',0)",(char_id,item_uid+3))
   c.execute('UPDATE PS_GameData_Chars SET Money=1000000000 WHERE CharID=?',(char_id,))
   weapon=c.execute('SELECT Type,TypeID FROM PS_GameData_CharItems WHERE CharID=? AND Bag=0 AND Slot=5',(char_id,)).fetchone()
   material=c.execute('SELECT Reqluc FROM PS_GameDefs_Items WHERE Type=? AND TypeID=?',weapon).fetchone()[0]
   for offset,(slot,item_id,count) in enumerate(((18,material,100),(19,95002,50)),4):
    c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=1 AND Slot=?',(char_id,slot))
    c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,?,?,?,?,1,?,0,0,0,0,0,0,0,'',?,datetime('now'),'N2',0)",(char_id,item_id,item_uid+offset,item_id//1000,item_id%1000,slot,count))
  env.update(SHAIYA_TEST_USER=users[0][0],SHAIYA_PASSWORD=users[0][1],SHAIYA_TEST_PARTNER=users[1][0],SHAIYA_TEST_PARTNER_PASSWORD=users[1][1])
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestNativeControlsAgainstRapture$','-count=1','-v'],cwd=client,env=env,check=True,timeout=120)
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Family=0,Job=5,MP=1000,SP=500,SkillPoint=100 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestFrozenIceAreaAgainstRapture$','-count=1','-v'],cwd=client,env=env,check=True,timeout=60)
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET MP=1000,SP=500,SkillPoint=100 WHERE UserUID=?',(users[1][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestFrozenIceAreaAgainstRapture$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_USER=users[1][0],SHAIYA_PASSWORD=users[1][1],SHAIYA_TEST_FROZEN_VICTIMS='1'),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   chars=[c.execute('SELECT CharID,CharName FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(u[2],)).fetchone() for u in users]
   guild_id=c.execute('SELECT COALESCE(MAX(GuildID),0)+1 FROM PS_GameData_Guilds').fetchone()[0]
   c.execute("INSERT INTO PS_GameData_Guilds(GuildID,GuildName,MasterUserID,MasterCharID,MasterName,Country,TotalCount,GuildPoint,CreateDate) VALUES(?,'N2NativeGuild',?,?,?,0,2,0,datetime('now'))",(guild_id,users[0][0],chars[0][0],chars[0][1]))
   c.execute("INSERT INTO PS_GameData_GuildDetails(GuildID,Etin,Remark) VALUES(?,1234,'Original notice')",(guild_id,))
   for n,(char_id,_) in enumerate(chars):c.execute("INSERT INTO PS_GameData_GuildChars(GuildID,CharID,GuildLevel,JoinDate) VALUES(?,?,?,datetime('now'))",(guild_id,char_id,1 if n==0 else 9))
   c.execute('INSERT INTO PS_GameData_CharQuests(CharID,QuestID,Delay,Count1,Count2,Count3,Success,Finish,Del) VALUES(?,1,0,0,0,0,0,0,0)',(chars[0][0],))
  # Refresh only the isolated world's startup guild cache, after its test
  # sessions have disconnected. Personal services and their DB are untouched.
  children[-1].terminate();children[-1].wait(timeout=10)
  log=open(client/'work/sqlite-gx_world.log','a')
  process=subprocess.Popen(['nice',str(server/'build/gx_world')],cwd=temp,env=env,stdout=log,stderr=subprocess.STDOUT);children.append(process)
  deadline=time.monotonic()+30
  while True:
   with socket.socket() as s:
    if s.connect_ex(('127.0.0.1',40810))==0:break
   if process.poll() is not None or time.monotonic()>deadline:raise RuntimeError('isolated guild world startup failed')
   time.sleep(.1)
  for test in ('TestGuildNoticeAndEtinAgainstRapture','TestNativeGuildQuestButtonsAgainstRapture'):
   subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^'+test+'$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_GUILD='1'),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=604,PosY=78,PosZ=1720,Money=50000 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestGatekeeperAgainstRapture$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_GATEKEEPER='1'),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=943,PosY=13,PosZ=777,Money=50000 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestGatekeeperCrossMapAgainstRapture$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_GATEKEEPER_PAID='1'),check=True,timeout=90)
  # A real duel uses only these temporary fixtures and verifies live 0x020E counters.
  with sqlite3.connect(db) as c:
   for n,(_,_,uid) in enumerate(users):
    c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=?,PosY=78,PosZ=1762,Family=0,Job=0,Level=40,Str=2000,Dex=2000,HP=?,MP=1000,SP=500,Money=50000 WHERE UserUID=?',(578+n*2,500 if n==0 else 1,uid))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestDuelVictoryWagersAgainstRapture$','-count=1','-v'],cwd=client,env=env,check=True,timeout=90)
 finally:
  for process in reversed(children):
   process.terminate()
   try:process.wait(timeout=10)
   except subprocess.TimeoutExpired:process.kill();process.wait()
 print('N2 SQLite: isolated servers stopped, temporary database and accounts removed')
