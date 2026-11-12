package proto

import "fmt"

// GemAddResult follows the 16-byte Rapture 0x0801 response.
type GemAddResult struct {
	Result, GemBag, GemSlot, GemCount  uint8
	ItemBag, ItemSlot, Position, GemID uint8
	Money                              uint32
}

func DecodeGemAddResult(p []byte) (GemAddResult, error) {
	r := NewReader(p)
	v := GemAddResult{Result: r.U8(), GemBag: r.U8(), GemSlot: r.U8(), GemCount: r.U8(), ItemBag: r.U8(), ItemSlot: r.U8(), Position: r.U8(), GemID: r.U8()}
	r.U32()
	v.Money = r.U32()
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.GemBag > 5 || v.GemSlot >= 24 || v.ItemBag > 5 || v.ItemSlot >= 24 || (v.Result == 1 && (v.Position >= 6 || v.GemID == 0)) {
		return v, fmt.Errorf("invalid gem add result")
	}
	return v, nil
}

func GemAddPacket(gemBag, gemSlot, itemBag, itemSlot uint8) []byte {
	w := NewWriter(GemAdd)
	w.Raw([]byte{gemBag, gemSlot, itemBag, itemSlot})
	return w.Bytes()
}

// GemRemoveResult is Rapture's 0x0802 response. Each removed flag indexes
// the old socket; recovered items are independently delivered by 0x0205.
type GemRemoveResult struct {
	Result, ItemBag, ItemSlot uint8
	Removed, IDs, Bags, Slots [6]uint8
	Money                     uint32
}

func DecodeGemRemoveResult(p []byte) (GemRemoveResult, error) {
	r := NewReader(p)
	v := GemRemoveResult{Result: r.U8(), ItemBag: r.U8(), ItemSlot: r.U8()}
	for i := range v.Removed {
		v.Removed[i] = r.U8()
	}
	for i := range v.IDs {
		v.IDs[i] = r.U8()
	}
	for i := range v.Bags {
		v.Bags[i] = r.U8()
	}
	for i := range v.Slots {
		v.Slots[i] = r.U8()
	}
	v.Money = r.U32()
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.ItemBag > 5 || v.ItemSlot >= 24 {
		return v, fmt.Errorf("invalid gem remove result")
	}
	for i, removed := range v.Removed {
		if removed > 1 || (removed != 0 && (v.Result == 0 || v.IDs[i] == 0 || v.Bags[i] < 1 || v.Bags[i] > 5 || v.Slots[i] >= 24)) {
			return v, fmt.Errorf("invalid extracted gem")
		}
	}
	return v, nil
}

func GemRemovePacket(itemBag, itemSlot uint8) []byte {
	w := NewWriter(GemRemove)
	w.Raw([]byte{itemBag, itemSlot})
	return w.Bytes()
}

// GemAddHammerPacket includes the optional linking hammer positions.
func GemAddHammerPacket(gemBag, gemSlot, itemBag, itemSlot, hammerBag, hammerSlot uint8) []byte {
	w := NewWriter(GemAdd)
	w.Raw([]byte{gemBag, gemSlot, itemBag, itemSlot, hammerBag, hammerSlot})
	return w.Bytes()
}
