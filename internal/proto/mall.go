package proto

import "fmt"

type MallItem struct {
	Bag, Slot, Type, TypeID uint8
	Count                   uint16
}
type MallPurchase struct {
	Result          uint8
	HasDetails      bool
	Points          uint32
	Code            string
	Timestamp, Cost uint32
	Items           []MallItem
}
type MallGiftResult struct {
	Result          uint8
	HasDetails      bool
	Target          string
	Points          uint32
	Code            string
	Timestamp, Cost uint32
}
type MallStoredItem struct {
	Slot, Type, TypeID uint8
	Count              uint16
}
type MallClaim struct{ Slot, Bag, InventorySlot uint8 }

// EP5 readers 0x597160/0x596b90 read only the result byte on errors;
// Rapture may append details even in the failure case.
func DecodeMallPurchase(p []byte) (MallPurchase, error) {
	r := NewReader(p)
	result := MallPurchase{Result: r.U8()}
	if err := r.Err(); err != nil {
		return result, err
	}
	if result.Result != 0 && r.Remaining() == 0 {
		return result, nil
	}
	result.Points, result.Code, result.Timestamp, result.Cost = r.U32(), r.Fixed(21), r.U32(), r.U32()
	n := int(r.U8())
	if err := r.Err(); err != nil {
		return result, err
	}
	if n > r.Remaining()/5 {
		return result, ErrShort
	}
	result.Items = make([]MallItem, n)
	for i := range result.Items {
		result.Items[i] = MallItem{r.U8(), r.U8(), r.U8(), r.U8(), r.Count()}
	}
	result.HasDetails = true
	return result, r.Err()
}
func DecodeMallGiftResult(p []byte) (MallGiftResult, error) {
	r := NewReader(p)
	result := MallGiftResult{Result: r.U8()}
	if err := r.Err(); err != nil {
		return result, err
	}
	if result.Result != 0 && r.Remaining() == 0 {
		return result, nil
	}
	result.Target, result.Points, result.Code, result.Timestamp, result.Cost = r.Fixed(21), r.U32(), r.Fixed(21), r.U32(), r.U32()
	if err := r.Err(); err != nil {
		return result, err
	}
	result.HasDetails = true
	return result, nil
}
func DecodeMallStored(p []byte) ([]MallStoredItem, error) {
	r := NewReader(p)
	n := int(r.U8())
	if err := r.Err(); err != nil {
		return nil, err
	}
	if n > 240 || n > r.Remaining()/4 {
		return nil, ErrShort
	}
	items := make([]MallStoredItem, n)
	var seen [240]bool
	for i := range items {
		it := MallStoredItem{r.U8(), r.U8(), r.U8(), r.Count()}
		if it.Slot >= 240 || it.Type == 0 || it.Count == 0 || seen[it.Slot] {
			return nil, fmt.Errorf("mall: invalid stored item %d", it.Slot)
		}
		seen[it.Slot] = true
		items[i] = it
	}
	return items, r.Err()
}
func DecodeMallClaim(p []byte) (MallClaim, error) {
	r := NewReader(p)
	m := MallClaim{r.U8(), r.U8(), r.U8()}
	if err := r.Err(); err != nil {
		return m, err
	}
	if m.Slot >= 240 || m.Bag < 1 || m.Bag > 5 || m.InventorySlot >= 24 {
		return m, fmt.Errorf("mall: invalid claim location")
	}
	return m, nil
}

func MallPointsPacket() []byte { return NewWriter(CashPoint).Bytes() }
func MallBuyPacket(code string) []byte {
	w := NewWriter(PointBuy)
	w.Fixed(code, 21, true)
	return w.Bytes()
}
func MallGiftPacket(code, target string) []byte {
	w := NewWriter(PointGiftSend)
	w.Fixed(code, 21, true)
	w.Fixed(target, 21, true)
	return w.Bytes()
}
func MallClaimPacket(slot uint8) []byte {
	w := NewWriter(PointStoredItemGet)
	w.U8(slot)
	return w.Bytes()
}
