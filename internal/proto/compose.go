package proto

import "fmt"

type ComposeResult struct {
	Result, Bag, Slot uint8
	Craft             string
}

func DecodeComposeResult(p []byte) (ComposeResult, error) {
	r := NewReader(p)
	v := ComposeResult{Result: r.U8(), Bag: r.U8(), Slot: r.U8(), Craft: r.Fixed(21)}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.Bag > 5 || v.Slot >= 24 {
		return v, fmt.Errorf("invalid recreation result")
	}
	return v, nil
}
func ComposePacket(runeBag, runeSlot, itemBag, itemSlot uint8) []byte {
	w := NewWriter(ItemCompose)
	w.Raw([]byte{runeBag, runeSlot, itemBag, itemSlot})
	return w.Bytes()
}
