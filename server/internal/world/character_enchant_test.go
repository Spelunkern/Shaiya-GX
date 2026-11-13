package world

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// Initial appearances and later equipment changes must expose the same raw
// craft level, including armor encoding; the client selects the visual tier.
func TestInitialShapeEnchantMatchesEquipmentUpdate(t *testing.T) {
	for _, craft := range []string{"", "invalid", "00000000000000000000", "00000000000000000005", "00000000000000000010", "00000000000000000020", "00000000000000000051"} {
		for slot := uint8(0); slot < 8; slot++ {
			t.Run(fmt.Sprintf("%s/slot%d", craft, slot), func(t *testing.T) {
				ch := &db.Character{ID: 7, Name: "Enchant"}
				items := []db.Item{{Bag: 0, Slot: slot, Type: 1, TypeID: 2, Count: 1, CraftName: craft}}
				shape := buildCharacterShapeGuild(ch, 0, items, 0, "Guild")
				update := buildCharacterEquipment(ch, items, slot)
				if len(shape) != 99 || len(update) != 18 {
					t.Fatalf("wire lengths changed: %d/%d", len(shape), len(update))
				}
				start := 23 + int(slot)*3
				if !bytes.Equal(shape[start:start+3], update[9:12]) {
					t.Fatalf("initial %v differs from update %v", shape[start:start+3], update[9:12])
				}
				for other := 0; other < 8; other++ {
					if other != int(slot) && !bytes.Equal(shape[23+other*3:26+other*3], []byte{0, 0, 0}) {
						t.Fatal("empty slot gained equipment")
					}
				}
			})
		}
	}
}
