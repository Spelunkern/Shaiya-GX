package proto

import "testing"

func TestWarehouseSnapshotAndWithdrawalMoneyTruncation(t *testing.T) {
	w := NewWriter(WarehouseItemList)
	w.U32(123)
	w.U8(1)
	w.U8(239)
	w.U8(25)
	w.U8(4)
	w.U16(100)
	for i := 0; i < 6; i++ {
		w.U8(uint8(i + 1))
	}
	w.U8(10)
	w.Fixed("crafted", 21, true)
	p := w.Bytes()
	list, err := DecodeWarehouse(p)
	if err != nil || list.Gold != 123 || len(list.Items) != 1 || list.Items[0].Bag != 100 || list.Items[0].Slot != 239 || list.Items[0].Quality != 100 || list.Items[0].Count != 10 || list.Items[0].Gems[5] != 6 || list.Items[0].Craft != "crafted" {
		t.Fatal(list, err)
	}
	for n := 0; n < len(p); n++ {
		if _, err := DecodeWarehouse(p[:n]); err == nil {
			t.Fatal("accepted truncated warehouse", n)
		}
	}
	update := NewWriter(ItemBagToBag)
	for _, position := range [][2]uint8{{100, 239}, {1, 23}} {
		update.Raw(position[:])
		update.U8(25)
		update.U8(4)
		update.U8(10)
		update.U16(100)
		update.Raw(make([]byte, 6))
		update.Fixed("crafted", 21, true)
	}
	update.U32(99750)
	items, money, has, err := DecodeWarehouseMove(update.Bytes())
	if err != nil || !has || money != 99750 || items[1].Count != 10 || items[0].Slot != 239 {
		t.Fatal(items, money, has, err)
	}
	for n := 0; n < len(update.Bytes()); n++ {
		if _, _, _, err := DecodeWarehouseMove(update.Bytes()[:n]); err == nil {
			t.Fatal("accepted truncated money update", n)
		}
	}
}
func FuzzWarehouse(f *testing.F) {
	f.Add([]byte{0x11, 7, 0, 0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, p []byte) { _, _ = DecodeWarehouse(p); _, _, _, _ = DecodeWarehouseMove(p) })
}
