package app

import (
	"bytes"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestActionsUseAssignedNumberAndItemPosition(t *testing.T) {
	a := &App{state: game.New(), skillDefs: map[sdata.SkillKey]sdata.Skill{{ID: 30, Level: 1}: {Target: 2}}}
	a.state.SelfID, a.state.Target = 42, 99
	a.state.Entities[99] = &game.Entity{ID: 99, Kind: game.KindMob}
	a.state.Inventory[[2]uint8{3, 8}] = proto.InventoryItem{Bag: 3, Slot: 8, Type: 25, TypeID: 4, Count: 1}
	a.state.Skills[7] = proto.LearnedSkill{ID: 30, Level: 1, Number: 7}
	a.state.QuickSlots = []proto.QuickSlot{{Slot: 9, Bag: 100, Number: 7}, {Slot: 2, Bag: 3, Number: 8}}
	if got := a.actionPacket(9); !bytes.Equal(got, []byte{0x11, 0x05, 7, 42, 0, 0, 0}) {
		t.Fatalf("self skill=%x", got)
	}
	a.skillDefs[sdata.SkillKey{ID: 30, Level: 1}] = sdata.Skill{Target: 1}
	if got := a.actionPacket(9); !bytes.Equal(got, []byte{0x17, 0x05, 7, 99, 0, 0, 0}) {
		t.Fatalf("target skill=%x", got)
	}
	a.state.Party.Members = []proto.PartyMember{{ID: 99}}
	if got := a.actionPacket(9); !bytes.Equal(got, []byte{0x11, 0x05, 7, 99, 0, 0, 0}) {
		t.Fatalf("party user skill=%x", got)
	}
	if got := a.actionPacket(2); !bytes.Equal(got, []byte{0x0a, 0x05, 3, 8}) {
		t.Fatalf("item=%x", got)
	}
	if a.actionPacket(0) != nil {
		t.Fatal("empty assigned slot fell back to learned skill")
	}
	a.inventory.pending = true
	if a.actionPacket(2) != nil || a.actionPacket(9) == nil {
		t.Fatal("inventory pending should block item use while allowing skills")
	}
	a.inventory.pending = false
	delete(a.state.Inventory, [2]uint8{3, 8})
	if a.actionPacket(2) != nil {
		t.Fatal("empty assigned position sent an item request")
	}
	a.state.Self.Dead = true
	if a.actionPacket(9) != nil || a.actionPacket(2) != nil {
		t.Fatal("dead character sent an action")
	}
}

func TestSkillBookCategoriesAndConfirmedLearningLimits(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Class: 0, Race: 0, Mode: 1, Level: 4}
	a.state.Details.SkillPoint = 3
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{
		{ID: 1, Level: 1}: {Show: 1, RequiredLevel: 1, Points: 1, Classes: [6]uint8{1}},
		{ID: 2, Level: 1}: {Show: 3, RequiredLevel: 4, Points: 2, Effect: 1, Classes: [6]uint8{1}},
		{ID: 3, Level: 3}: {Show: 3, RequiredLevel: 1, Points: 1, Effect: 1, Classes: [6]uint8{1}},
	}
	a.state.Skills[7] = proto.LearnedSkill{Number: 7, ID: 3, Level: 3}
	if keys := a.visibleSkills(); len(keys) != 1 || keys[0].ID != 1 {
		t.Fatal("passive category", keys)
	}
	a.skills.category = 2
	keys := a.visibleSkills()
	if len(keys) != 2 || keys[0].ID != 2 || keys[1].ID != 3 {
		t.Fatal("combat category lost learned skill", keys)
	}
	if !a.canLearn(keys[0]) || a.canLearn(keys[1]) {
		t.Fatal("learn/max learned eligibility")
	}
	a.state.Details.SkillPoint = 1
	if a.canLearn(keys[0]) {
		t.Fatal("not enough points")
	}
	a.state.Details.SkillPoint = 3
	a.state.Self.Dead = true
	if a.canLearn(keys[0]) {
		t.Fatal("dead character could learn")
	}
	a.skills.assignMode = true
	keys = a.visibleSkills()
	if len(keys) != 1 || keys[0].ID != 3 {
		t.Fatal("assignment did not use confirmed learned skills", keys)
	}
}

func TestEquipmentActionKeepsSwapPosition(t *testing.T) {
	a := &App{state: game.New()}
	a.state.QuickSlots = []proto.QuickSlot{{Bar: 1, Slot: 3, Bag: 2, Number: 7}}
	key := [2]uint8{2, 7}
	equip := [2]uint8{0, 5}
	a.state.Inventory[key] = proto.InventoryItem{Bag: 2, Slot: 7, Type: 1, TypeID: 2, Count: 1}
	a.state.Inventory[equip] = proto.InventoryItem{Bag: 0, Slot: 5, Type: 1, TypeID: 3, Count: 1}
	for i := 0; i < 4; i++ {
		if got := a.actionPacket(13); !bytes.Equal(got, proto.ItemMovePacket(2, 7, 0, 5)) {
			t.Fatalf("equipment shortcut did not swap: %x", got)
		}
		// Server-confirmed positions remain the bar's identity; the item changes.
		a.state.Inventory[key], a.state.Inventory[equip] = a.state.Inventory[equip], a.state.Inventory[key]
	}
	a.inventory.pending = true
	if a.actionPacket(13) != nil {
		t.Fatal("duplicate equip request while awaiting server")
	}
}
