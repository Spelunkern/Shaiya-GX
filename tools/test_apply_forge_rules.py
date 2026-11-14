import hashlib
from pathlib import Path
import shutil
import sqlite3
import tempfile
import unittest
from apply_forge_rules import apply,read_patches

ROOT=Path(__file__).resolve().parents[1]
class ForgeDataTests(unittest.TestCase):
    def test_only_target_definition_fields_change(self):
        with tempfile.TemporaryDirectory() as tmp:
            target=Path(tmp)/'test.sqlite';shutil.copy2(ROOT/'server/data/rapture.sqlite',target)
            patches=read_patches(ROOT/'tools/data/forge-item-patches.csv')
            with sqlite3.connect(target) as db:
                chars=db.execute('SELECT * FROM PS_GameData_CharItems').fetchall()
                users=db.execute('SELECT * FROM PS_UserData_Users_Master').fetchall()
                old=db.execute('SELECT Type,TypeID,ItemName,ReqIg,ReqLuc,ReqWis,Server,Slot FROM PS_GameDefs_Items ORDER BY ItemID').fetchall()
            limits=apply(target,patches)
            with sqlite3.connect(target) as db:
                self.assertEqual(chars,db.execute('SELECT * FROM PS_GameData_CharItems').fetchall())
                self.assertEqual(users,db.execute('SELECT * FROM PS_UserData_Users_Master').fetchall())
                self.assertEqual(old,db.execute('SELECT Type,TypeID,ItemName,ReqIg,ReqLuc,ReqWis,Server,Slot FROM PS_GameDefs_Items ORDER BY ItemID').fetchall())
                for typ,identity,vg,stack in patches:
                    actual=db.execute('SELECT ReqVg,Count FROM PS_GameDefs_Items WHERE Type=? AND TypeID=?',(typ,identity)).fetchone()
                    self.assertEqual(actual[0],vg)
                    if stack:self.assertEqual(actual[1],255)
                self.assertEqual(dict(limits)[95001],255)
                before=db.execute('SELECT ReqVg,Count FROM PS_GameDefs_Items WHERE Type=100 AND TypeID=49').fetchone()
            with self.assertRaises(ValueError):apply(target,[(100,49,99,0),(255,255,10,0)])
            with sqlite3.connect(target) as db:self.assertEqual(before,db.execute('SELECT ReqVg,Count FROM PS_GameDefs_Items WHERE Type=100 AND TypeID=49').fetchone())
