package proto

import "fmt"

const WarehouseBag, WarehouseSlots = 100, 240

// WarehouseList: Game.exe reader 0x591030, account gold then counted records.
// Stored records omit Bag and place Count after Gems, unlike 0x0204 updates.
type WarehouseList struct {
	Gold  uint32
	Items []InventoryItem
}

func DecodeWarehouse(p []byte) (list WarehouseList, err error) {
	r := NewReader(p)
	list.Gold = r.U32()
	n := int(r.U8())
	if n > WarehouseSlots {
		return list, fmt.Errorf("warehouse: too many items %d", n)
	}
	if n > r.Remaining()/33 {
		return list, ErrShort
	}
	list.Items = make([]InventoryItem, n)
	seen := [WarehouseSlots]bool{}
	for i := range list.Items {
		m := &list.Items[i]
		m.Bag = WarehouseBag
		m.Slot, m.Type, m.TypeID = r.U8(), r.U8(), r.U8()
		m.Quality = r.U16()
		for j := range m.Gems {
			m.Gems[j] = r.U8()
		}
		m.Count, m.Craft = r.Count(), r.Fixed(21)
		if int(m.Slot) >= WarehouseSlots || seen[m.Slot] {
			return list, fmt.Errorf("warehouse: invalid/duplicate slot %d", m.Slot)
		}
		seen[m.Slot] = true
	}
	return list, r.Err()
}

// Withdraw confirmations append the resulting character money after both slots.
// Keep DecodeItemMove for existing callers that only need the two records.
func DecodeWarehouseMove(p []byte) (items [2]InventoryItem, money uint32, hasMoney bool, err error) {
	r := NewReader(p)
	items = [2]InventoryItem{readItemSlot(r), readItemSlot(r)}
	if err := r.Err(); err != nil {
		return items, 0, false, err
	}
	for _, item := range items {
		if item.Bag == GuildBankBag && int(item.Slot) >= GuildBankSlots {
			return items, 0, false, fmt.Errorf("guild bank: invalid slot %d", item.Slot)
		}
	}
	if (items[0].Bag == WarehouseBag && items[1].Bag != WarehouseBag) || (items[0].Bag == GuildBankBag && items[1].Bag != GuildBankBag) {
		money, hasMoney = r.U32(), true
	}
	return items, money, hasMoney, r.Err()
}
