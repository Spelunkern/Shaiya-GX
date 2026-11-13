#!/usr/bin/env python3
"""Exercise a disposable guild against owned Rapture processes and SQL fixtures.

The Rapture SQL container must already be ready. Existing game services are
never stopped; the script only terminates processes it started itself.
"""
import importlib.util
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import tempfile
import time

raise SystemExit("Legacy SQL Server probe. Use tools/n2_smoke.py for GX Server SQLite integration.")

root = Path(__file__).resolve().parents[1]
rapture = Path(os.environ.get("GX_SERVER_DIR", os.environ.get("RAPTURE_DIR", root / "server")))
spec = importlib.util.spec_from_file_location("rapture_sql", rapture / "tools/sqlserver.py")
sqltool = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sqltool)
creation_mode = "--creation" in sys.argv
bank_mode = "--bank" in sys.argv
duel_mode = "--duel" in sys.argv
notice_mode = "--notice" in sys.argv
names = [("gu" + secrets.token_hex(4), secrets.token_hex(6), "G" + secrets.token_hex(5)) for _ in range(7 if creation_mode else 3)]
guild_name = "T" + secrets.token_hex(7)
opponent_guild_name = "T" + secrets.token_hex(7)
uids = []
guild_id = None
opponent_guild_id = None
children = []
cfg = tempfile.TemporaryDirectory(prefix="shaiya-guild-")
env = dict(os.environ, XDG_CONFIG_HOME=cfg.name, SHAIYA_TEST_DISPOSABLE="1",
           RAPTURE_LOGIN_LISTEN="127.0.0.1:30800", RAPTURE_WORLD_LISTEN="127.0.0.1:30810",
           RAPTURE_PUBLIC_IP="127.0.0.1", RAPTURE_DB_HOST="127.0.0.1", RAPTURE_DB_PORT="14346",
           SHAIYA_DATA=str(Path.home() / "Desktop/Go project"), GOTOOLCHAIN="local",
           GOPATH=str(Path.home() / ".local/share/go"))
env["PATH"] = str(Path.home() / ".local/opt/go-1.27.3/bin") + ":" + env["PATH"]


def query(statement, database="PS_GameData"):
    try:
        return sqltool.sql(statement, database)
    except subprocess.CalledProcessError as exc:
        message = exc.stderr or "SQL failed"
        for _, password, _ in names:
            message = message.replace(password, "[masked]")
        raise RuntimeError(message) from None


def listening(port):
    with socket.socket() as sock:
        sock.settimeout(.2)
        return sock.connect_ex(("127.0.0.1", port)) == 0


def start_server(name, port):
    with (root / "work" / f"guild-test-{name}.log").open("w") as log:
        proc = subprocess.Popen(["nice", str(rapture / "build" / name)], cwd=rapture, env=env, stdout=log, stderr=subprocess.STDOUT)
    children.append(proc)
    deadline = time.monotonic() + 60
    while not listening(port):
        if proc.poll() is not None or time.monotonic() >= deadline:
            raise RuntimeError(f"{name} did not start")
        time.sleep(.25)
    return proc


def stop_server(proc):
    if proc.poll() is None:
        proc.terminate()
        proc.wait(timeout=10)


def run_test(test, test_env):
    subprocess.run(["nice", "go", "test", "-tags", "integration", "./internal/app", "-run", test, "-count=1", "-v"],
                   cwd=root, env=test_env, check=True, timeout=90)


try:
    if any(listening(port) for port in (30800, 30810)):
        raise RuntimeError("game services already running")
    for user, password, _ in names:
        uid = int(query(f"""BEGIN TRAN;
DECLARE @uid int=(SELECT ISNULL(MAX(UserUID),0)+1 FROM dbo.Users_Master WITH (TABLOCKX));
INSERT dbo.Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point)
VALUES(@uid,'{user}','{password}',0,0,0,0,0,'N',0);
COMMIT; SELECT @uid;""", "PS_UserData"))
        uids.append(uid)
    start_server("gx_login", 30800)
    world = start_server("gx_world", 30810)
    for user, password, character in names:
        account_env = dict(env, SHAIYA_TEST_USER=user, SHAIYA_PASSWORD=password)
        run_test("TestFactionAgainstRapture", dict(account_env, SHAIYA_TEST_FACTION="1"))
        run_test("TestCreationAgainstRapture", dict(account_env, SHAIYA_TEST_CREATE=character))
    # Map 1's 11/1 Guild-Master of Light is at this position in 1.svmap.
    if creation_mode:
        query(f"UPDATE dbo.Chars SET Map=1,PosX=999.836,PosY=12.76,PosZ=748.293 WHERE UserUID IN ({','.join(map(str,uids))});")
        query(f"UPDATE dbo.Chars SET Level=11,Money=10000 WHERE UserUID={uids[0]} AND CharName='{names[0][2]}';")
        test_env = dict(env, SHAIYA_TEST_USER=names[0][0], SHAIYA_PASSWORD=names[0][1], SHAIYA_TEST_GUILD=guild_name,
                        SHAIYA_TEST_GUILD_ACCOUNTS=json.dumps([[user,password] for user,password,_ in names]))
        run_test("TestGuildCreationAgainstRapture", test_env)
    else:
        query(f"UPDATE dbo.Chars SET Map=1,PosX=999.836,PosY=12.76,PosZ=748.293 WHERE UserUID={uids[2]} AND CharName='{names[2][2]}';")
    # Guilds are loaded once when the world process starts. Restart only ours
    # after inserting a guild for its disposable master.
        stop_server(world)
        master_id = int(query(f"SELECT CharID FROM dbo.Chars WHERE UserUID={uids[0]} AND CharName='{names[0][2]}';"))
        guild_id = int(query("SELECT ISNULL(MAX(GuildID),0)+1 FROM dbo.Guilds;"))
        query(f"""DECLARE @rc int;
EXEC @rc=dbo.usp_Create_Guild2_E @GuildID={guild_id},@GuildName='{guild_name}',
 @MasterUserID='{names[0][0]}',@MasterCharID={master_id},@MasterName='{names[0][2]}',@Country=0,@Remark='Disposable guild';
IF @rc<>0 THROW 50000,'Guild creation fixture failed',1;
EXEC @rc=dbo.usp_Create_GuildChar_E @GuildID={guild_id},@CharID={master_id},@GuildLevel=1;
IF @rc<>0 THROW 50000,'Guild member fixture failed',1;""")
        if bank_mode:
            partner_id = int(query(f"SELECT CharID FROM dbo.Chars WHERE UserUID={uids[1]} AND CharName='{names[1][2]}';"))
            item_uid = secrets.randbits(62)+1
            partner_item_uid = secrets.randbits(62)+1
            query(f"""DECLARE @rc int;
EXEC @rc=dbo.usp_Create_GuildChar_E @GuildID={guild_id},@CharID={partner_id},@GuildLevel=9;
IF @rc<>0 THROW 50000,'Guild observer fixture failed',1;
UPDATE dbo.GuildDetails SET Rank=1 WHERE GuildID={guild_id};
UPDATE dbo.Chars SET Map=1,PosX=491,PosY=78,PosZ=1785,Money=10000 WHERE UserUID IN ({uids[0]},{uids[1]});
IF EXISTS(SELECT 1 FROM dbo.CharItems WHERE CharID={master_id} AND Bag=1 AND Slot=23 AND Del=0) THROW 50000,'Bank fixture slot occupied',1;
IF EXISTS(SELECT 1 FROM dbo.CharItems WHERE CharID={partner_id} AND Bag=1 AND Slot=23 AND Del=0) THROW 50000,'Observer bank fixture slot occupied',1;
INSERT dbo.CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del)
VALUES({master_id},25004,{item_uid},25,4,1,23,0,0,0,0,0,0,0,'',5,GETDATE(),'N',0),
({partner_id},25004,{partner_item_uid},25,4,1,23,0,0,0,0,0,0,0,'',5,GETDATE(),'N',0);""")
        if notice_mode:
            partner_id = int(query(f"SELECT CharID FROM dbo.Chars WHERE UserUID={uids[1]} AND CharName='{names[1][2]}';"))
            query(f"""DECLARE @rc int;
EXEC @rc=dbo.usp_Create_GuildChar_E @GuildID={guild_id},@CharID={partner_id},@GuildLevel=9;
IF @rc<>0 THROW 50000,'Notice observer fixture failed',1;""")
        if duel_mode:
            partner_id = int(query(f"SELECT CharID FROM dbo.Chars WHERE UserUID={uids[1]} AND CharName='{names[1][2]}';"))
            member_id = int(query(f"SELECT CharID FROM dbo.Chars WHERE UserUID={uids[2]} AND CharName='{names[2][2]}';"))
            opponent_guild_id = guild_id + 1
            query(f"""DECLARE @rc int;
EXEC @rc=dbo.usp_Create_Guild2_E @GuildID={opponent_guild_id},@GuildName='{opponent_guild_name}',
 @MasterUserID='{names[1][0]}',@MasterCharID={partner_id},@MasterName='{names[1][2]}',@Country=0,@Remark='Disposable opponent';
IF @rc<>0 THROW 50000,'Opponent guild fixture failed',1;
EXEC @rc=dbo.usp_Create_GuildChar_E @GuildID={opponent_guild_id},@CharID={partner_id},@GuildLevel=1;
IF @rc<>0 THROW 50000,'Opponent master fixture failed',1;
EXEC @rc=dbo.usp_Create_GuildChar_E @GuildID={opponent_guild_id},@CharID={member_id},@GuildLevel=9;
IF @rc<>0 THROW 50000,'Opponent member fixture failed',1;
UPDATE dbo.Chars SET Map=1,PosX=578,PosY=78,PosZ=1762 WHERE UserUID IN ({uids[0]},{uids[1]},{uids[2]});
UPDATE dbo.Chars SET Str=150,Dex=150 WHERE UserUID={uids[0]} AND CharName='{names[0][2]}';""")
        start_server("gx_world", 30810)
        test_env = dict(env, SHAIYA_TEST_USER=names[0][0], SHAIYA_PASSWORD=names[0][1],
                        SHAIYA_TEST_PARTNER=names[1][0], SHAIYA_TEST_PARTNER_PASSWORD=names[1][1],
                        SHAIYA_TEST_GUILD=guild_name)
        if bank_mode:
            run_test("TestGuildBankAgainstRapture", test_env)
        elif notice_mode:
            run_test("TestGuildNoticeAndEtinAgainstRapture", test_env)
        elif duel_mode:
            duel_env = dict(test_env, SHAIYA_TEST_GUILD_THIRD=names[2][0], SHAIYA_TEST_GUILD_THIRD_PASSWORD=names[2][1])
            run_test("TestGuildDuelAgainstRapture", duel_env)
        else:
            run_test("TestGuildInviteAcceptAgainstRapture", test_env)
            applicant_env = dict(test_env, SHAIYA_TEST_PARTNER=names[2][0], SHAIYA_TEST_PARTNER_PASSWORD=names[2][1])
            run_test("TestGuildApplicationAgainstRapture", applicant_env)
            query(f"UPDATE dbo.Chars SET Map=1,PosX=999.836,PosY=12.76,PosZ=748.293 WHERE UserUID={uids[0]} AND CharName='{names[0][2]}';")
            run_test("TestGuildDisbandSmallAgainstRapture", applicant_env)
finally:
    for child in reversed(children):
        stop_server(child)
    if creation_mode:
        found = query(f"SELECT GuildID FROM dbo.Guilds WHERE GuildName='{guild_name}' AND MasterUserID='{names[0][0]}';")
        if found:
            guild_id = int(found)
    for cleanup_guild_id, cleanup_guild_name in [(guild_id,guild_name),(opponent_guild_id,opponent_guild_name)]:
      if cleanup_guild_id is not None:
        # Match the random name as well as the ID before touching guild rows.
        query(f"""IF EXISTS(SELECT 1 FROM dbo.Guilds WHERE GuildID={cleanup_guild_id} AND GuildName='{cleanup_guild_name}')
BEGIN
 DELETE dbo.GuildStoredItems WHERE GuildID={cleanup_guild_id};
 DELETE dbo.GuildChars WHERE GuildID={cleanup_guild_id};
 DELETE dbo.GuildDetails WHERE GuildID={cleanup_guild_id};
 DELETE dbo.Guilds WHERE GuildID={cleanup_guild_id} AND GuildName='{cleanup_guild_name}';
END;""")
        query(f"DELETE dbo.CreatedGuilds WHERE GuildID={cleanup_guild_id} AND GuildName='{cleanup_guild_name}';", "PS_UserData")
    for uid, (user, _, _) in zip(uids, names):
        query(f"""DECLARE @ids TABLE(id int);
INSERT @ids SELECT CharID FROM dbo.Chars WHERE UserUID={uid};
DELETE dbo.CharQuests WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo.CharItems WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo.CharSkills WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo.CharQuickSlots WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo._DeletedChars WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo.Chars WHERE UserUID={uid};
DELETE dbo.UserMaxGrow WHERE UserUID={uid};
DELETE dbo.UserStoredItems WHERE UserUID={uid};""")
        query(f"DELETE dbo.Users_Master WHERE UserUID={uid} AND UserID='{user}';", "PS_UserData")
    cfg.cleanup()
    print("Guild smoke: disposable guild, accounts and characters removed", flush=True)
