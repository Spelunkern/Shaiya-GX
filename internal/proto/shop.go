package proto

// ShopBuyPacket: npc global ID, catalog index, count (writer 0x598d20; Rapture handleShopBuy).
func ShopBuyPacket(npc uint32, index, count uint8) []byte {
	w := NewWriter(ShopBuy)
	w.U32(npc)
	w.U8(index)
	w.U8(count)
	return w.Bytes()
}

// ShopSellPacket: bag, slot, count (writer 0x598d70; Rapture handleShopSell).
func ShopSellPacket(bag, slot, count uint8) []byte {
	w := NewWriter(ShopSell)
	w.Raw([]byte{bag, slot, count})
	return w.Bytes()
}

type ShopResult struct {
	Result, Bag, Slot, Type, TypeID, Count uint8
	Money                                  uint32
}

func DecodeShopResult(p []byte) (m ShopResult, err error) {
	r := NewReader(p)
	m.Result, m.Bag, m.Slot, m.Type, m.TypeID, m.Count = r.U8(), r.U8(), r.U8(), r.U8(), r.U8(), r.U8()
	m.Money = r.U32()
	return m, r.Err()
}
