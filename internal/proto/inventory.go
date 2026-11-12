package proto

import "fmt"

// InventoryItem: 0x0106 reader 0x58dd30, Rapture buildInventoryPackets.
type InventoryItem struct {
	Bag, Slot, Type, TypeID uint8
	Quality                 uint16
	Gems                    [6]uint8
	Count                   uint8
	Craft                   string
}

func DecodeInventory(p []byte) ([]InventoryItem, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/34 {
		return nil, ErrShort
	}
	items := make([]InventoryItem, n)
	for i := range items {
		m := &items[i]
		m.Bag, m.Slot, m.Type, m.TypeID = r.U8(), r.U8(), r.U8(), r.U8()
		m.Quality = r.U16()
		for j := range m.Gems {
			m.Gems[j] = r.U8()
		}
		m.Count = r.U8()
		m.Craft = r.Fixed(21)
		if m.Bag > 5 || m.Slot >= 24 {
			return nil, fmt.Errorf("inventory: invalid position %d/%d", m.Bag, m.Slot)
		}
	}
	return items, r.Err()
}

// Slot updates reorder count before quality (Rapture writeSlot).
func readItemSlot(r *Reader) InventoryItem {
	m := InventoryItem{Bag: r.U8(), Slot: r.U8(), Type: r.U8(), TypeID: r.U8(), Count: r.U8()}
	m.Quality = r.U16()
	for j := range m.Gems {
		m.Gems[j] = r.U8()
	}
	m.Craft = r.Fixed(21)
	return m
}
func DecodeItemMove(p []byte) ([2]InventoryItem, error) {
	r := NewReader(p)
	m := [2]InventoryItem{readItemSlot(r), readItemSlot(r)}
	return m, r.Err()
}
func DecodeItemAdd(p []byte) (InventoryItem, error) {
	r := NewReader(p)
	m := readItemSlot(r)
	return m, r.Err()
}

// ItemMovePacket is 0x0204: source bag/slot, destination bag/slot.
// Game.exe writer 0x598270; Rapture handleItemBagToBag.
func ItemMovePacket(srcBag, srcSlot, dstBag, dstSlot uint8) []byte {
	w := NewWriter(ItemBagToBag)
	w.Raw([]byte{srcBag, srcSlot, dstBag, dstSlot})
	return w.Bytes()
}

// GroundItem is 0x0401 (Game.exe 0x58f320; Rapture buildMapAddItem).
type GroundItem struct {
	ID                       uint32
	Unk, Type, TypeID, Count uint8
	X, Y, Z                  float32
}

func DecodeGroundItem(p []byte) (m GroundItem, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Unk, m.Type, m.TypeID, m.Count = r.U8(), r.U8(), r.U8(), r.U8()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	return m, r.Err()
}

// PickupPacket is 0x0205, ID-only (writer 0x5982c0; Rapture handlePickupItem).
func PickupPacket(id uint32) []byte { return idPacket(AddItem, id) }

// UseItemPacket: writer 0x598b80 (bag/slot), four bytes.
func UseItemPacket(bag, slot uint8) []byte {
	w := NewWriter(UseItem)
	w.U8(bag)
	w.U8(slot)
	return w.Bytes()
}

type ItemUseResult struct {
	ID                             uint32
	Bag, Slot, Type, TypeID, Count uint8
}

func DecodeItemUse(p []byte) (m ItemUseResult, err error) {
	r := NewReader(p)
	m.ID = r.U32()
	m.Bag, m.Slot, m.Type, m.TypeID, m.Count = r.U8(), r.U8(), r.U8(), r.U8(), r.U8()
	return m, r.Err()
}
