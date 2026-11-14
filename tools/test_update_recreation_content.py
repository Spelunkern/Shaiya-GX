import struct
import unittest
from update_recreation_content import Native, patch_items, patch_cash, read_cash, runes, string

class RecreationContentTests(unittest.TestCase):
    def test_native_items_preserve_fields_and_add_eighteen_runes(self):
        body=bytearray(range(81));body[0],body[1],body[35],body[80]=100,171,62,255
        gem=body[:];gem[0],gem[1],gem[80]=30,1,1
        raw=struct.pack('<II',1,2)+string('Rune')+string('Description')+body+string('Gem')+string('Gem description')+gem
        first=patch_items(raw)
        self.assertEqual(patch_items(first),first)
        r=Native(first);self.assertEqual(r.word(),101)
        records={}
        for _ in range(101):
            for _ in range(r.word()):
                name,desc,b=r.string(),r.string(),r.take(81);records[b[0],b[1]]=(name,desc,b)
        r.end()
        self.assertEqual(records[100,171][2],body)
        expected=gem[:];expected[80]=255
        self.assertEqual(records[30,1][2],expected)
        for rune in runes():
            b=records[101,rune['type_id']][2]
            self.assertEqual(struct.unpack_from('<H',b,28)[0],rune['req_vg'])
            self.assertEqual(b[35],62);self.assertEqual(b[80],255)
            self.assertEqual(records[101,rune['type_id']][0].decode(),rune['name'])

    def test_native_mall_updates_bundles_and_is_idempotent(self):
        raw=struct.pack('<I4I',1,1,1,0,0)
        raw+=struct.pack('<IBIB',30001,1,95001,1)+bytes(22*5)
        raw+=string('Supplies')+string('Supplies_1')+string('Description')
        result,products=patch_cash(raw)
        self.assertEqual(len(products),19)
        self.assertEqual(products[0]['items'][:2],[[30001,255],[95001,255]])
        self.assertEqual(patch_cash(result)[0],result)
        for p in read_cash(result)[1:]:
            self.assertEqual(p['items'][0][1],255)
            self.assertLessEqual(len(p['fields'][1]),20)

if __name__=='__main__':unittest.main()
