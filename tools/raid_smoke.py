#!/usr/bin/env python3
"""Verifica raid con siete cuentas descartables; Rapture debe estar iniciado."""
import importlib.util
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile

root = Path(__file__).resolve().parents[1]
rapture = Path(os.environ.get("RAPTURE_DIR", Path.home() / "Desktop/Rapture"))
spec = importlib.util.spec_from_file_location("rapture_sql", rapture / "tools/sqlserver.py")
sqltool = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sqltool)
accounts = []


def query(statement, database):
    try:
        return sqltool.sql(statement, database)
    except subprocess.CalledProcessError as error:
        message = error.stderr or "SQL failed"
        for account in accounts:
            message = message.replace(account["Password"], "[masked]")
        raise RuntimeError(message) from None


with tempfile.TemporaryDirectory(prefix="shaiya-raid-") as config:
    env = dict(os.environ, XDG_CONFIG_HOME=config, SHAIYA_TEST_DISPOSABLE="1",
               GOTOOLCHAIN="local", GOPATH=str(Path.home() / ".local/share/go"))
    env["PATH"] = str(Path.home() / ".local/opt/go-1.27.3/bin") + ":" + env["PATH"]
    try:
        for _ in range(7):
            account = {"User": "gor" + secrets.token_hex(5), "Password": secrets.token_hex(6),
                       "character": "Gr" + secrets.token_hex(5)}
            accounts.append(account)
            account["uid"] = int(query(f"""BEGIN TRAN;
DECLARE @uid int=(SELECT ISNULL(MAX(UserUID),0)+1 FROM dbo.Users_Master WITH (TABLOCKX));
INSERT dbo.Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point)
VALUES(@uid,'{account["User"]}','{account["Password"]}',0,0,0,0,0,'N',0);
COMMIT; SELECT @uid;""", "PS_UserData"))
            individual = dict(env, SHAIYA_TEST_USER=account["User"], SHAIYA_PASSWORD=account["Password"])
            subprocess.run(["go", "test", "-tags", "integration", "./internal/app", "-run", "^TestFactionAgainstRapture$", "-count=1"],
                           cwd=root, env=dict(individual, SHAIYA_TEST_FACTION="1"), check=True, timeout=45)
            subprocess.run(["go", "test", "-tags", "integration", "./internal/app", "-run", "^TestCreationAgainstRapture$", "-count=1"],
                           cwd=root, env=dict(individual, SHAIYA_TEST_CREATE=account["character"]), check=True, timeout=45)
        credentials = [{"User": a["User"], "Password": a["Password"]} for a in accounts]
        subprocess.run(["go", "test", "-tags", "integration", "./internal/app", "-run", "^TestRaidSevenAgainstRapture$", "-count=1", "-v"],
                       cwd=root, env=dict(env, SHAIYA_TEST_RAID_ACCOUNTS=json.dumps(credentials)), check=True, timeout=120)
    finally:
        for account in accounts:
            if "uid" not in account:
                continue
            uid, user = account["uid"], account["User"]
            query(f"""DECLARE @ids TABLE(id int);
INSERT @ids SELECT CharID FROM dbo.Chars WHERE UserUID={uid};
DELETE dbo.CharQuests WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo.CharItems WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo.CharSkills WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo.CharQuickSlots WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo._DeletedChars WHERE CharID IN(SELECT id FROM @ids);
DELETE dbo.Chars WHERE UserUID={uid};
DELETE dbo.UserMaxGrow WHERE UserUID={uid};
DELETE dbo.UserStoredItems WHERE UserUID={uid};""", "PS_GameData")
            query(f"DELETE dbo.Users_Master WHERE UserUID={uid} AND UserID='{user}';", "PS_UserData")
        print("Raid N2: cuentas y personajes descartables eliminados", flush=True)
