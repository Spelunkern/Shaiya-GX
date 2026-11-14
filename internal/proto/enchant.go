package proto

import "fmt"

type EnchantResult struct {
	Result, LapisiaBag, LapisiaSlot uint8
	LapisiaCount                    uint16
	MaterialBag, MaterialSlot       uint8
	MaterialCount                   uint16
	ItemBag, ItemSlot               uint8
	Money                           uint32
	Craft                           string
}

func DecodeEnchantResult(p []byte) (EnchantResult, error) {
	r := NewReader(p)
	v := EnchantResult{Result: r.U8(), LapisiaBag: r.U8(), LapisiaSlot: r.U8(), LapisiaCount: r.Count(),
		MaterialBag: r.U8(), MaterialSlot: r.U8(), MaterialCount: r.Count(), ItemBag: r.U8(), ItemSlot: r.U8(), Money: r.U32(), Craft: r.Fixed(21)}
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

func EnchantPacket(lapisiaBag, lapisiaSlot, materialBag, materialSlot uint8, materialCount uint16, itemBag, itemSlot uint8) []byte {
	w := NewWriter(EnchantAdd)
	w.Raw([]byte{lapisiaBag, lapisiaSlot, materialBag, materialSlot})
	w.Count(materialCount)
	w.Raw([]byte{itemBag, itemSlot})
	return w.Bytes()
}
