package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
)

func TestAnorEquipmentIgnoresCharacterMode(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	id, err := db.CreateCharacter(db.Account{UserUID: 1, UserID: "admin1"}, db.NewCharacter{Slot: 4, Name: "EquipModeProbe", HP: 100})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.Pool(db.GameData)
	t.Cleanup(func() {
		pool.Exec("DELETE FROM PS_GameData_CharItems WHERE CharID=?", id)
		pool.Exec("DELETE FROM PS_GameData_Chars WHERE CharID=?", id)
	})
	ch := db.Character{ID: id, Level: 60, Race: 0, Class: 5, Mode: 2}
	s, _ := gmWireSession(t, ch, false)
	item := db.Item{Bag: 1, Slot: 0, Type: 16, TypeID: 216, Count: 1}
	if err := db.AddCharacterItem(id, &item); err != nil {
		t.Fatal(err)
	}
	s.items = []db.Item{item}
	def := defs.Item(16, 216)
	if def == nil || def.Name != "Goddess Anor Helmet" {
		t.Fatal("actual inventory fixture missing")
	}
	for mode := uint8(0); mode <= 3; mode++ {
		s.char.Mode = mode
		if reason := s.equipmentMoveFailure(1, 0, 0, 0); reason != "" {
			t.Fatalf("mode %d blocked equipment: %s", mode, reason)
		}
	}
	s.char.Mode = 2
	s.handleItemBagToBag(proto.ItemMovePacket(1, 0, 0, 0))
	stored, err := db.CharacterItems(id)
	if err != nil || len(stored) != 1 || stored[0].Bag != 0 || stored[0].Slot != 0 || stored[0].UID != item.UID {
		t.Fatal("equip did not persist", stored, err)
	}
	s.char.Level = 1
	if canEquip(s.char, def, 0, nil) {
		t.Fatal("level restriction disappeared")
	}
	s.char.Level = 60
	s.char.Class = 0
	if canEquip(s.char, def, 0, nil) {
		t.Fatal("class restriction disappeared")
	}
}
