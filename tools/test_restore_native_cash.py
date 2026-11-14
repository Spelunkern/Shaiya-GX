import shutil
import sqlite3
import struct
import tempfile
import unittest
from pathlib import Path

from restore_native_cash import catalog, encode, migrate

ROOT=Path(__file__).resolve().parents[1]

class NativeCashMigrationTests(unittest.TestCase):
    def test_preserves_products_and_splits_owned_quantities(self):
        rows=[]
        for code,item in [('POTION','25001'),('RUNE','100067')]:
            p=dict(index='1',category='1',icon='0',price='0',name=code,code=code,description='Test')
            for i in range(1,25):p[f'item_id_{i}']='0';p[f'quantity_{i}']='0'
            p['item_id_1']=item;p['quantity_1']='255'
            rows.append(p)
        self.assertEqual(len(rows),2)
        self.assertTrue(all(int(p[f'quantity_{i}'])<=255 for p in rows for i in range(1,25)))
        raw=encode(rows)
        self.assertEqual(struct.unpack_from('<I',raw)[0],2)
        with tempfile.TemporaryDirectory() as tmp:
            target=Path(tmp)/'test.sqlite'
            shutil.copy2(ROOT/'server/data/rapture.sqlite',target)
            with sqlite3.connect(target) as db:
                db.row_factory=sqlite3.Row
                item=dict(db.execute('SELECT * FROM PS_GameData_CharItems LIMIT 1').fetchone())
                item.update(CharID=999001,ItemUID=999001,ItemID=100067,Type=100,TypeID=67,Bag=1,Slot=0,Count=999,Del=0)
                db.execute('INSERT INTO PS_GameData_CharItems('+','.join(item)+') VALUES('+','.join('?' for _ in item)+')',list(item.values()))
            changes,limits=migrate(target,rows)
            self.assertEqual(len(changes),1)
            with sqlite3.connect(target) as db:
                counts=[r[0] for r in db.execute('SELECT Count FROM PS_GameData_CharItems WHERE CharID=999001')]
                self.assertEqual(sorted(counts),[234,255,255,255])
                self.assertEqual(sum(counts),999)
                self.assertEqual(db.execute('SELECT COUNT(*) FROM PS_GameDefs_ProductList').fetchone()[0],2)
                for p in rows:
                    expected=[p['name'],int(p['price'])]+[int(p[k]) for i in range(1,25) for k in (f'item_id_{i}',f'quantity_{i}')]
                    cols=['ProductName','BuyCost']+[c for i in range(1,25) for c in (f'ItemID{i}',f'ItemCount{i}')]
                    actual=db.execute('SELECT '+','.join(cols)+' FROM PS_GameDefs_ProductList WHERE ProductCode=?',(p['code'],)).fetchone()
                    self.assertEqual(tuple(expected),actual)
                migrate(target,rows)
                self.assertEqual(db.execute('SELECT SUM(Count) FROM PS_GameData_CharItems WHERE CharID=999001').fetchone()[0],999)
    def test_native_layout_contains_all_24_slots(self):
        p=dict(index='1',category='1',icon='0',price='0',name='Native',code='TEST',description='Text')
        for i in range(1,25):p[f'item_id_{i}']='25001';p[f'quantity_{i}']='255'
        raw=encode([p])
        for i in range(24):self.assertEqual(struct.unpack_from('<IB',raw,20+i*5),(25001,255))
