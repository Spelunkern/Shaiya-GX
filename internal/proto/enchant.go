package proto

import "fmt"

type EnchantResult struct {
	Result, LapisiaBag, LapisiaSlot, LapisiaCount uint8
	MaterialBag, MaterialSlot, MaterialCount      uint8
	ItemBag, ItemSlot                             uint8
	Money                                         uint32
	Craft                                         string
}

func DecodeEnchantResult(p []byte) (EnchantResult, error) {
	r := NewReader(p)
	v := EnchantResult{Result: r.U8(), LapisiaBag: r.U8(), LapisiaSlot: r.U8(), LapisiaCount: r.U8(),
		MaterialBag: r.U8(), MaterialSlot: r.U8(), MaterialCount: r.U8(), ItemBag: r.U8(), ItemSlot: r.U8(), Money: r.U32(), Craft: r.Fixed(21)}
	if err := r.Err(); err != nil {
		return v, err
	}
	if v.Result > 1 || v.LapisiaBag > 5 || v.MaterialBag > 5 || v.ItemBag > 5 || v.LapisiaSlot >= 24 || v.MaterialSlot >= 24 || v.ItemSlot >= 24 {
		return v, fmt.Errorf("invalid enchant result")
	}
	l, m, i := [2]uint8{v.LapisiaBag, v.LapisiaSlot}, [2]uint8{v.MaterialBag, v.MaterialSlot}, [2]uint8{v.ItemBag, v.ItemSlot}
	if l == m || l == i || m == i {
		return v, fmt.Errorf("overlapping enchant positions")
	}
	return v, nil
}

func EnchantPacket(lapisiaBag, lapisiaSlot, materialBag, materialSlot, materialCount, itemBag, itemSlot uint8) []byte {
	w := NewWriter(EnchantAdd)
	w.Raw([]byte{lapisiaBag, lapisiaSlot, materialBag, materialSlot, materialCount, itemBag, itemSlot})
	return w.Bytes()
}
