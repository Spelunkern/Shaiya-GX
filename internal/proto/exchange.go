package proto

import "fmt"

const ExchangeSlots = 8

type ExchangeItem struct {
	Type, TypeID uint8
	Count        uint16
	Quality      uint16
	Gems         [6]uint8
	Craft        string
}
type ExchangeOwnItem struct {
	Bag, Slot, Destination uint8
	Count                  uint16
}
type ExchangeUpdate struct{ Kind, Value uint8 }
type ExchangeMoneyUpdate struct {
	Kind  uint8
	Money uint32
}

func DecodeExchangeID(p []byte) (uint32, error) { r := NewReader(p); id := r.U32(); return id, r.Err() }
func DecodeExchangeStart(p []byte) (id uint32, result uint8, err error) {
	r := NewReader(p)
	id = r.U32()
	result = r.U8()
	return id, result, r.Err()
}
func DecodeExchangeByte(p []byte) (uint8, error) { r := NewReader(p); v := r.U8(); return v, r.Err() }
func DecodeExchangeUpdate(p []byte) (u ExchangeUpdate, err error) {
	r := NewReader(p)
	u = ExchangeUpdate{r.U8(), r.U8()}
	return u, r.Err()
}
func DecodeExchangeMoney(p []byte) (m ExchangeMoneyUpdate, err error) {
	r := NewReader(p)
	m = ExchangeMoneyUpdate{r.U8(), r.U32()}
	return m, r.Err()
}
func DecodeExchangeOwnItem(p []byte) (item ExchangeOwnItem, err error) {
	r := NewReader(p)
	item = ExchangeOwnItem{Bag: r.U8(), Slot: r.U8(), Count: r.Count(), Destination: r.U8()}
	if err = r.Err(); err == nil && item.Destination >= ExchangeSlots {
		err = fmt.Errorf("exchange: invalid slot %d", item.Destination)
	}
	return
}
func DecodeExchangePartnerItem(p []byte) (slot uint8, item ExchangeItem, err error) {
	r := NewReader(p)
	slot = r.U8()
	item.Type, item.TypeID, item.Count = r.U8(), r.U8(), r.Count()
	item.Quality = r.U16()
	for i := range item.Gems {
		item.Gems[i] = r.U8()
	}
	item.Craft = r.Fixed(21)
	if err = r.Err(); err == nil && slot >= ExchangeSlots {
		err = fmt.Errorf("exchange: invalid slot %d", slot)
	}
	return
}
func ExchangeRequestPacket(id uint32) []byte {
	w := NewWriter(ExchangeRequest)
	w.U32(id)
	return w.Bytes()
}
func ExchangeAnswerPacket(accept bool) []byte {
	w := NewWriter(ExchangeAnswer)
	if accept {
		w.U8(0)
	} else {
		w.U8(1)
	}
	return w.Bytes()
}
func ExchangeReadyPacket(value uint8) []byte {
	w := NewWriter(ExchangeReady)
	w.U8(value)
	return w.Bytes()
}
func ExchangeAddPacket(bag, slot uint8, count uint16, dest uint8) []byte {
	w := NewWriter(ExchangeAdd)
	w.Raw([]byte{bag, slot})
	w.Count(count)
	w.U8(dest)
	return w.Bytes()
}
func ExchangeRemovePacket(slot uint8) []byte {
	w := NewWriter(ExchangeRemove)
	w.U8(slot)
	return w.Bytes()
}
func ExchangeMoneyPacket(money uint32) []byte {
	w := NewWriter(ExchangeAddMoney)
	w.U32(money)
	return w.Bytes()
}
func ExchangeConfirmPacket(confirm bool) []byte {
	w := NewWriter(ExchangeConfirm)
	if confirm {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
