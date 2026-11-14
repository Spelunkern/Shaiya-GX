#!/usr/bin/env python3
"""Exercise continuous forge UI against isolated native login/world processes."""
import argparse
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import sqlite3
import subprocess
import tempfile
import time
from apply_forge_rules import apply, read_patches

ROOT=Path(__file__).resolve().parents[1]

def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--world',type=Path,default=ROOT/'server/build/gx_world')
    ap.add_argument('--item-data',type=Path,required=True)
    ap.add_argument('--data',type=Path,default=Path.home()/'Desktop/Shaiya GX/data')
    args=ap.parse_args()
    logs=ROOT/'work/forge-rules';logs.mkdir(parents=True,exist_ok=True)
    for port in (41800,41810):
        with socket.socket() as s:
            if s.connect_ex(('127.0.0.1',port))==0:raise RuntimeError('isolated test port occupied')
    children=[]
    with tempfile.TemporaryDirectory(prefix='gx-forge-n2-') as tmp:
        dbpath=Path(tmp)/'test.sqlite';shutil.copy2(ROOT/'server/data/rapture.sqlite',dbpath)
        apply(dbpath,read_patches(ROOT/'tools/data/forge-item-patches.csv'))
        user='fg'+secrets.token_hex(5);password=secrets.token_hex(6)
        with sqlite3.connect(dbpath) as db:
            db.row_factory=sqlite3.Row
            uid=db.execute('SELECT MAX(UserUID)+1 FROM PS_UserData_Users_Master').fetchone()[0]
            db.execute("INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,?,1,255,0,0,0,'N',0)",(uid,user,password))
            db.execute('INSERT INTO PS_GameData_UserMaxGrow(ServerID,UserUID,Country,MaxGrow,Del) VALUES(1,?,0,3,0)',(uid,))
            ch=dict(db.execute('SELECT * FROM PS_GameData_Chars WHERE Del=0 LIMIT 1').fetchone())
            cid=db.execute('SELECT MAX(CharID)+1 FROM PS_GameData_Chars').fetchone()[0]
            ch.update(CharID=cid,UserUID=uid,UserID=user,CharName='Forge'+secrets.token_hex(4),Slot=0,Del=0,Map=1,PosX=578,PosY=78,PosZ=1762,Level=1,Money=20000000,HP=5000,MP=1000,SP=1000)
            if 'RowID' in ch:ch['RowID']=db.execute('SELECT COALESCE(MAX(RowID),0)+1 FROM PS_GameData_Chars').fetchone()[0]
            db.execute('INSERT INTO PS_GameData_Chars('+','.join('"'+k+'"' for k in ch)+') VALUES('+','.join('?' for k in ch)+')',list(ch.values()))
            event=db.execute('SELECT * FROM PS_GameData_CharEvents LIMIT 1').fetchone()
            if event:
                e=dict(event);e['CharID']=cid
                if 'RowID' in e:e['RowID']=db.execute('SELECT COALESCE(MAX(RowID),0)+1 FROM PS_GameData_CharEvents').fetchone()[0]
                db.execute('INSERT INTO PS_GameData_CharEvents('+','.join('"'+k+'"' for k in e)+') VALUES('+','.join('?' for k in e)+')',list(e.values()))
            fixtures=[(1,2,1),(95,1,3),(25,43,3),(100,171,3),(30,1,2),(100,82,3),(100,140,2),(101,10,3),(101,1,3)]
            for slot,(typ,identity,count) in enumerate(fixtures):
                db.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,?,?,?,?,1,?,500,0,0,0,0,0,0,'00000000000000000000',?,datetime('now'),'N2',0)",(cid,typ*1000+identity,secrets.randbits(62),typ,identity,slot,count))
        env=dict(os.environ,RAPTURE_DB_PATH=str(dbpath),RAPTURE_DATA_DIR=str(ROOT/'server/data'),RAPTURE_LOGIN_LISTEN='127.0.0.1:41800',RAPTURE_WORLD_LISTEN='127.0.0.1:41810',RAPTURE_PUBLIC_IP='127.0.0.1',SHAIYA_TEST_LOGIN_ADDR='127.0.0.1:41800',SHAIYA_TEST_WORLD_PORT='41810',SHAIYA_TEST_DISPOSABLE='1',SHAIYA_TEST_FORGE_CONTINUOUS='1',SHAIYA_DATA=str(args.data),SHAIYA_TEST_ITEM_FILE=str(args.item_data.resolve()),SHAIYA_TEST_USER=user,SHAIYA_PASSWORD=password,GOTOOLCHAIN='local')
        env['PATH']=str(Path.home()/'.local/opt/go-1.27.3/bin')+':'+env['PATH']
        try:
            for executable,port in [(ROOT/'server/build/gx_login',41800),(args.world.resolve(),41810)]:
                log=(logs/(executable.name+'.log')).open('w')
                p=subprocess.Popen([str(executable)],cwd=tmp,env=env,stdout=log,stderr=subprocess.STDOUT);children.append((p,log))
                until=time.monotonic()+30
                while True:
                    with socket.socket() as s:
                        if s.connect_ex(('127.0.0.1',port))==0:break
                    if p.poll() is not None or time.monotonic()>until:raise RuntimeError('isolated startup failed')
                    time.sleep(.05)
            subprocess.run(['go','test','-tags','integration','./internal/app','-run','^TestForgeContinuousAgainstServer$','-count=1','-v'],cwd=ROOT,env=env,check=True,timeout=90)
            with sqlite3.connect(dbpath) as db:
                rows=db.execute('SELECT Type,TypeID,Count,Craftname FROM PS_GameData_CharItems WHERE CharID=? AND Del=0 ORDER BY Bag,Slot',(cid,)).fetchall()
                assert db.execute('SELECT Money FROM PS_GameData_Chars WHERE CharID=?',(cid,)).fetchone()[0]==14987715
                target=next(r for r in rows if r[0]==1 and r[1]==2)
                assert target[2]==1 and len(target[3])==20 and any(x!='0' for x in target[3][:18])
                for typ,identity in [(95,1),(25,43),(100,171)]:assert next(r[2] for r in rows if r[:2]==(typ,identity))==1
                (logs/'n2-verification.json').write_text(json.dumps({'gold':14987715,'equipment_retained':True,'stacks_retained':True,'craftname':target[3]},indent=2)+'\n')
        finally:
            for p,log in reversed(children):
                if p.poll() is None:
                    p.terminate()
                    try:p.wait(timeout=10)
                    except subprocess.TimeoutExpired:p.kill();p.wait()
                log.close()

if __name__=='__main__':main()
