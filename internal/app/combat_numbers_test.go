package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestCombatNumbersDetachExpireAndStayBounded(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob, X: 4, Z: 8}
	a.showCombatNumber(2, 0, false, false, true)
	if len(a.combatNumbers) != 0 {
		t.Fatal("zero healing invented a number")
	}
	a.showCombatNumber(2, 0, true, true, false)
	if len(a.combatNumbers) != 1 || !a.combatNumbers[0].Missed || a.combatNumbers[0].Critical {
		t.Fatal(a.combatNumbers)
	}
	a.state.Entities[2].X = 20
	if a.combatNumbers[0].Point[0] != 4 {
		t.Fatal("confirmed number followed the moving target")
	}
	for range 150 {
		a.showCombatNumber(2, 12, false, false, false)
	}
	if len(a.combatNumbers) != 128 {
		t.Fatal("unbounded queue", len(a.combatNumbers))
	}
	a.clock = 1
	a.expireCombatNumbers()
	if len(a.combatNumbers) != 0 {
		t.Fatal("last expired number remained visible")
	}
}

func TestCombatNumberBillboardSizeUsesDepthAndCameraPitch(t *testing.T) {
	projection := render.PerspectiveFovLH(math.Pi/4, 4.0/3, .1, 100)
	view := render.LookAtLH(render.V3{0, 2, -5}, render.V3{0, 0, 0}, render.V3{0, 1, 0})
	right := render.V3{view[0], view[4], view[8]}
	_, _, width, height, ok := combatGlyphRect(render.V3{0, 1.2, 0}, .2, .2*1.1875, view, projection, 800, 600)
	if !ok || math.Abs(float64(height/width-1.1875)) > .0001 {
		t.Fatal("pitched quad lost billboard aspect", width, height, ok, right)
	}
	_, _, farWidth, _, ok := combatGlyphRect(render.V3{0, 1.2, 15}, .2, .2*1.1875, view, projection, 800, 600)
	if !ok || farWidth >= width {
		t.Fatal("numbers used fixed screen sizes")
	}
	if _, _, _, _, ok := combatGlyphRect(render.V3{0, 1.2, -20}, .2, .2, view, projection, 800, 600); ok {
		t.Fatal("behind-camera number drawn")
	}
}

func TestRequestedTargetFacingAndSkillApproachBeforeAcknowledgment(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.state.Target = 1, 2
	a.wv.facing = render.V3{0, 0, 1}
	a.wv.visualFacing = render.V3{0, 0, -1}
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob, X: 10}
	if !a.faceRequestedTarget(2) || a.wv.facing != (render.V3{1, 0, 0}) || a.wv.visualFacing.Len() != 0 {
		t.Fatal("target facing waited for ACK", a.wv.facing, a.wv.visualFacing)
	}
	a.state.Skills[3] = proto.LearnedSkill{Number: 3, ID: 7, Level: 1}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 7, Level: 1}: {AttackRange: 5}}
	packet := proto.UseSkillPacket(3, 2, false)
	if !a.approachSkillPacket(packet) || a.wv.approachTarget != 2 || !a.wv.travel || len(a.wv.approachSkill) == 0 {
		t.Fatal("targeted spell did not approach", a.wv)
	}
	if a.wv.destination[0] <= 0 || a.wv.destination[0] >= 10 {
		t.Fatal("spell destination did not stop inside range", a.wv.destination)
	}
	a.state.Entities[2].X = 1
	if a.approachSkillPacket(packet) || a.wv.travel || a.wv.approachTarget != 0 || len(a.wv.approachSkill) != 0 {
		t.Fatal("in-range spell retained the old pending approach")
	}
}
