//go:build integration

package app

import (
	"math"
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestMouseAttackAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_COMBAT") != "1" {
		t.Skip("requires disposable fixture near map-one mobs")
	}
	st, world, send, wait := testworld.Enter(t)
	mob := testworld.LivingMob(t, st, send, wait)
	// As in the combat probe, move only this disposable fixture into range.
	send(proto.MovePacket(0, 0, mob.X+1, mob.Y, mob.Z))
	a := inputApp()
	a.state, a.world = st, world
	a.wv.pos = render.V3{mob.X + 1, mob.Y, mob.Z}
	center := render.V3{mob.X, mob.Y + 1, mob.Z}
	a.scene3d = &render.Scene3D{
		View: render.LookAtLH(center.Add(render.V3{0, 0, -5}), center, render.V3{0, 1, 0}),
		Proj: render.PerspectiveFovLH(math.Pi/4, 1024.0/768, .1, 100),
	}
	click := func(now float64) {
		a.clock = now
		press := platform.Input{MouseX: 512, MouseY: 384, Pressed: true, MouseDown: true}
		a.worldControls(&press, 1024, 768)
		controls := a.worldControls(&platform.Input{MouseX: 512, MouseY: 384, Clicked: true}, 1024, 768)
		a.updateCombat(&controls, 1024, 768)
		if a.state == nil {
			t.Fatal("combat control disconnected", a.login.Message)
		}
	}
	click(1)
	if st.Target != mob.ID {
		t.Fatal("pointer did not select the visible mob", st.Target, mob.ID)
	}
	wait(func() bool { return st.Entities[mob.ID].HealthKnown })
	if st.LastAttack.Source == st.SelfID {
		t.Fatal("single click requested an attack")
	}
	click(1.2)
	wait(func() bool { return st.LastAttack.Source == st.SelfID && st.LastAttack.Target == mob.ID })
	if st.LastAttack.Result != 0 && st.LastAttack.Result != 1 && st.LastAttack.Result != 3 {
		t.Fatal("double-click attack rejected", st.LastAttack)
	}
	t.Logf("single click selected mob %d; second click confirmed result=%d damage=%d", mob.ID, st.LastAttack.Result, st.LastAttack.HP)
	send(proto.StopAttackPacket())
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
}
