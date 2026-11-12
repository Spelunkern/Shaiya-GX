package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestServiceNPCSwitchesBetweenQuestAndService(t *testing.T) {
	for _, kind := range []uint8{1, 2, 3, 6} {
		t.Run(map[uint8]string{1: "merchant", 2: "gatekeeper", 3: "blacksmith", 6: "banker"}[kind], func(t *testing.T) {
			a := inputApp()
			a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
			a.bankUI, _ = ui.NewWarehouse(inputTextures{})
			a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
			a.gateUI, _ = ui.NewGatekeeper(inputTextures{})
			a.questDataLoaded = true
			a.questDefs = []sdata.Quest{{Name: "Quest"}}
			e := &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: kind, TypeID: 1}
			a.state.Entities[e.ID] = e
			a.shop.npcID = e.ID
			a.npcDefs[sdata.NPCKey{Type: kind, ID: 1}] = sdata.NPC{StartQuests: []uint16{1}}
			click := func(x, y int) {
				a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
				a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
			}
			x, y := a.serviceQuestOrigin(1024, 768, e)
			click(x+5, y+5)
			if !a.shop.questMode || a.npcQuest.npc != 42 {
				t.Fatal("quest dialogue did not open")
			}
			x, y = ui.NPCQuestOrigin(1024, 768)
			click(x+150, y+340)
			if a.shop.questMode || a.shop.npcID != 42 {
				t.Fatal("service did not reopen")
			}
		})
	}
}
