package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
)

func TestMinimapAndBothEffectStripsDragWithPointerOwnership(t *testing.T) {
	for _, kind := range []string{"minimap", "buff", "debuff"} {
		a := inputApp()
		a.minimap, _ = ui.NewMinimap(inputTextures{})
		var p *ui.WindowPosition
		var x, y int
		if kind == "minimap" {
			p = &a.minimap.Position
			x, y = a.minimap.Origin(1024, 768)
			x += 40
			y += 170
		} else {
			effect := uint8(4)
			group := 0
			if kind == "debuff" {
				effect = 3
				group = 1
			}
			a.buffs.rows = []buffBadge{{}}
			a.buffs.rows[0].def.Effect = effect
			p = &a.buffLayout.Position[group]
			x, y = a.buffSlot(1024, 768, 0)
			x += 4
			y += 10
		}
		controls := a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
		if a.windowDrag.position != p || controls.Pressed || controls.Down != "W" {
			t.Fatal(kind, "did not own press")
		}
		controls = a.worldControls(&platform.Input{MouseX: x - 120, MouseY: y + 80, Clicked: true}, 1024, 768)
		if controls.Clicked || !p.Moved || a.windowDrag.position != nil {
			t.Fatal(kind, "lost drag/release")
		}
	}
}

func TestActionApproachSelectsNearbyLootAndCancelsOnManualInput(t *testing.T) {
	a := inputApp()
	a.state.GroundItems = map[uint32]proto.GroundItem{7: {ID: 7, X: 15, Z: 0}, 8: {ID: 8, X: 25, Z: 0}}
	if !a.pickupNearest() || a.wv.approachTarget != 7 || !a.wv.travel {
		t.Fatal("pickup did not approach closest loot")
	}
	a.updateActionApproach(&platform.Input{Down: "W"})
	if a.wv.approachTarget != 0 || a.wv.travel {
		t.Fatal("manual movement did not cancel approach")
	}
	a.state.Entities[9] = &game.Entity{ID: 9, Kind: game.KindMob, X: 20}
	a.state.Target = 9
	if !a.autoAttackNearest() || a.wv.approachTarget != 9 || !a.wv.travel {
		t.Fatal("attack did not approach target")
	}
	delete(a.state.Entities, 9)
	a.updateActionApproach(&platform.Input{})
	if a.wv.approachTarget != 0 || a.wv.travel {
		t.Fatal("removed target left travel active")
	}
}
