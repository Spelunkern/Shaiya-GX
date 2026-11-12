package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
)

func TestDeathChoiceFollowsRunePartyAndUltimate(t *testing.T) {
	for _, tc := range []struct {
		name           string
		state          deathView
		choice         int
		want           proto.ReviveMode
		sent, location bool
	}{
		{"normal bind", deathView{}, 1, proto.ReviveBind, true, false},
		{"party", deathView{party: true}, 2, proto.ReviveParty, true, false},
		{"rune first step", deathView{rune: true}, 1, 0, false, true},
		{"rune bind", deathView{rune: true, choosingLocation: true}, 1, proto.ReviveRuneBind, true, true},
		{"rune here", deathView{rune: true, choosingLocation: true}, 2, proto.ReviveRuneHere, true, true},
		{"rune party", deathView{rune: true, party: true}, 2, proto.ReviveParty, true, false},
		{"ultimate rune", deathView{rune: true, ultimate: true}, 1, proto.ReviveRuneBind, true, false},
		{"ultimate no party option", deathView{ultimate: true, party: true}, 2, 0, false, false},
		{"waiting", deathView{waiting: true}, 1, 0, false, false},
		{"pending", deathView{pending: true}, 1, 0, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, sent := deathChoice(&tc.state, tc.choice)
			if got != tc.want || sent != tc.sent || tc.state.choosingLocation != tc.location {
				t.Fatalf("mode %d sent %v state %+v", got, sent, tc.state)
			}
		})
	}
}
func TestDeathRuneUsesRealTypeAndConfirmedInventory(t *testing.T) {
	a := inputApp()
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 44, TypeID: 3}: {Special: 37}, {Type: 26, TypeID: 4}: {Special: 37}}
	a.state.Inventory[[2]uint8{0, 1}] = proto.InventoryItem{Type: 44, TypeID: 3, Count: 1}
	a.state.Inventory[[2]uint8{1, 2}] = proto.InventoryItem{Type: 26, TypeID: 4, Count: 1}
	if a.hasResurrectionRune() {
		t.Fatal("equipped or unrelated item counted as rune")
	}
	a.state.Inventory[[2]uint8{1, 3}] = proto.InventoryItem{Type: 44, TypeID: 3, Count: 1}
	if !a.hasResurrectionRune() {
		t.Fatal("valid rune missing")
	}
}

func TestDuelDefeatDoesNotOfferRuneOrUltimateDeletion(t *testing.T) {
	a := inputApp()
	a.deathUI, _ = ui.NewDeathDialog(inputTextures{})
	a.state.SelfID = 42
	a.state.Slots[0] = proto.CharacterSlot{ID: 42, Mode: 3}
	a.state.DeathLossFree = true
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 100, TypeID: 44}: {Special: 37}}
	a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Type: 100, TypeID: 44, Count: 1}
	a.initDeath()
	if a.wv.death.rune || a.wv.death.ultimate {
		t.Fatalf("duel defeat offered protection or ultimate deletion: %+v", a.wv.death)
	}
}
