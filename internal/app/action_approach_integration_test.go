//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func verifyNativeApproach(t *testing.T, a *App, pump func(func() bool)) {
	t.Helper()
	v, st := a.wv, a.state
	start := v.pos
	send := func(p []byte) {
		t.Helper()
		if err := a.world.Send(p); err != nil {
			t.Fatal(err)
		}
	}
	// Drop one unit from the isolated runner's disposable stack.
	key := [2]uint8{1, 23}
	if st.Inventory[key].Count != 5 {
		t.Fatal("missing disposable pickup stack")
	}
	drop := proto.NewWriter(proto.RemoveItem)
	drop.U8(1)
	drop.U8(23)
	drop.U8(1)
	send(drop.Bytes())
	var loot uint32
	pump(func() bool {
		for id, item := range st.GroundItems {
			if item.Type == 25 && item.TypeID == 4 {
				loot = id
				return true
			}
		}
		return false
	})
	item := st.GroundItems[loot]
	v.place(render.V3{item.X + 15, item.Y, item.Z})
	send(v.movePacket(0))
	if !a.pickupNearest() || !v.travel || v.approachTarget != loot {
		t.Fatal("pickup action did not start approach")
	}
	for i := 0; i < 600 && v.approachTarget != 0; i++ {
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	if v.approachTarget != 0 {
		t.Fatal("loot approach never reached pickup distance")
	}
	pump(func() bool { _, present := st.GroundItems[loot]; return !present && st.Inventory[key].Count == 5 })
	// Pick a real visible living mob and begin fifteen units away.
	// First monster area in original map-one svmap: (789..809,1663..1680).
	v.place(render.V3{799.4, 39, 1671.7})
	send(v.movePacket(0))
	var enemy *game.Entity
	pump(func() bool {
		for _, e := range st.Entities {
			if e.Kind == game.KindMob && !e.Dead && (!e.HealthKnown || e.HP > 0) {
				enemy = e
				return true
			}
		}
		return false
	})
	v.place(render.V3{enemy.X + 15, enemy.Y, enemy.Z})
	send(v.movePacket(0))
	st.Target = 0
	if !a.autoAttackNearest() || v.approachTarget == 0 {
		t.Fatal("auto attack did not acquire and approach a nearby enemy")
	}
	target := st.Target
	for i := 0; i < 600 && v.approachTarget != 0; i++ {
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	if v.approachTarget != 0 {
		t.Fatal("attack approach did not finish")
	}
	pump(func() bool { return st.LastAttack.Source == st.SelfID && st.LastAttack.Target == target })
	send(proto.StopAttackPacket())
	v.place(start)
	send(v.movePacket(0))
	a.updateWorld(&platform.Input{}, 1.0/60)
	t.Log("action approach: nearest loot reached and picked up after ACK; nearest enemy reached and attacked after server confirmation")
}
