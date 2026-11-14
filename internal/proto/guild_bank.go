package proto

import "fmt"

const GuildBankBag, GuildBankSlots = 255, 240

type GuildBankResultInfo struct{ Result, Arg uint8 }
type GuildBankEvent struct {
	Item  InventoryItem
	Actor uint32
}

func readGuildBankItem(r *Reader) InventoryItem {
	m := InventoryItem{Bag: GuildBankBag, Slot: r.U8(), Type: r.U8(), TypeID: r.U8(), Quality: r.U16()}
	for i := range m.Gems {
		m.Gems[i] = r.U8()
	}
	m.Count, m.Craft = r.Count(), r.Fixed(21)
	return m
}

func DecodeGuildBankList(p []byte) ([]InventoryItem, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/33 {
		return nil, ErrShort
	}
	items := make([]InventoryItem, n)
	var seen [GuildBankSlots]bool
	for i := range items {
		items[i] = readGuildBankItem(r)
		if int(items[i].Slot) >= GuildBankSlots || seen[items[i].Slot] {
			return nil, fmt.Errorf("guild bank: invalid or duplicate slot %d", items[i].Slot)
		}
		seen[items[i].Slot] = true
	}
	return items, r.Err()
}

func DecodeGuildBankEvent(p []byte) (GuildBankEvent, error) {
	r := NewReader(p)
	event := GuildBankEvent{Item: readGuildBankItem(r), Actor: r.U32()}
	if err := r.Err(); err != nil {
		return event, err
	}
	if int(event.Item.Slot) >= GuildBankSlots {
		return event, fmt.Errorf("guild bank: invalid slot %d", event.Item.Slot)
	}
	return event, nil
}

func DecodeGuildBankResult(p []byte) (GuildBankResultInfo, error) {
	r := NewReader(p)
	result := GuildBankResultInfo{Result: r.U8(), Arg: r.U8()}
	return result, r.Err()
}
