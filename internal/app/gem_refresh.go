package app

import (
	"sort"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// gemProtectionSlots snapshots all possible server-side Lucky Charm slots.
// Rapture groups item types 25, 44 and 100 as real type 26 and looks only in
// bags. Do not filter by local Special: client data may differ from the DB.
func gemProtectionSlots(items map[[2]uint8]proto.InventoryItem) [][2]uint8 {
	var slots [][2]uint8
	for key, item := range items {
		if key[0] >= 1 && key[0] <= 5 && key[1] < 24 && item.Count > 0 &&
			(item.Type == 25 || item.Type == 44 || item.Type == 100) {
			slots = append(slots, key)
		}
	}
	sort.Slice(slots, func(i, j int) bool {
		return slots[i][0] < slots[j][0] || slots[i][0] == slots[j][0] && slots[i][1] < slots[j][1]
	})
	return slots
}

// gemRefresh tracks distinct authoritative same-slot echoes. Snapshot slots
// before linking, since a consumed last charm disappears from the server.
type gemRefresh struct {
	pending map[[2]uint8]bool
}

func (r *gemRefresh) start(slots [][2]uint8) {
	r.pending = make(map[[2]uint8]bool, len(slots))
	for _, key := range slots {
		r.pending[key] = true
	}
}

func (r *gemRefresh) acknowledge(p []byte) bool {
	if proto.Op(p) != proto.ItemBagToBag || len(r.pending) == 0 {
		return false
	}
	items, _, _, err := proto.DecodeWarehouseMove(p)
	if err != nil {
		return false
	}
	a := [2]uint8{items[0].Bag, items[0].Slot}
	b := [2]uint8{items[1].Bag, items[1].Slot}
	if a != b || !r.pending[a] {
		return false
	}
	delete(r.pending, a)
	return true
}
