package proto

import "fmt"

type ItemRepairResult struct {
	Bag, Slot uint8
	Money     uint32
}

func DecodeItemRepair(p []byte) (ItemRepairResult, error) {
	r := NewReader(p)
	v := ItemRepairResult{Bag: r.U8(), Slot: r.U8(), Money: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if !(v.Bag == 255 && v.Slot == 255) && (v.Bag > 5 || v.Bag == 0 && v.Slot >= 13 || v.Bag != 0 && v.Slot >= 24) {
		return v, fmt.Errorf("invalid repair position")
	}
	return v, nil
}

func ItemRepairPacket(bag, slot uint8) []byte {
	w := NewWriter(ItemRepair)
	w.Raw([]byte{bag, slot})
	return w.Bytes()
}
