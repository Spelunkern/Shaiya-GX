//go:build integration

// Package testworld shares the disposable Rapture connection for N2 tests.
package testworld

import (
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
)

// LivingMob confirms HP because entry snapshots can include corpses from an
// earlier probe. Selecting the closest snapshot alone does not prove liveness.
func LivingMob(t *testing.T, st *game.State, send func([]byte), wait func(func() bool)) *game.Entity {
	t.Helper()
	for {
		var mob *game.Entity
		wait(func() bool {
			best := float32(math.MaxFloat32)
			for _, e := range st.Entities {
				if e.Kind != game.KindMob || e.Dead || e.HealthKnown && e.HP <= 0 {
					continue
				}
				dx, dz := e.X-st.Self.X, e.Z-st.Self.Z
				if distance := dx*dx + dz*dz; distance < best {
					best, mob = distance, e
				}
			}
			return mob != nil
		})
		send(proto.TargetMobPacket(mob.ID))
		wait(func() bool { return mob.HealthKnown })
		if !mob.Dead && mob.HP > 0 {
			return mob
		}
	}
}

// This test mutates only the disposable account provisioned by n2_smoke.py.
func Connect(t *testing.T) (*game.State, *session.World, func([]byte), func(func() bool)) {
	return ConnectAs(t, os.Getenv("SHAIYA_TEST_USER"), os.Getenv("SHAIYA_PASSWORD"))
}

func ConnectAs(t *testing.T, user, password string) (*game.State, *session.World, func([]byte), func(func() bool)) {
	if os.Getenv("SHAIYA_TEST_DISPOSABLE") != "1" {
		t.Skip("requires disposable N2 account")
	}
	address := os.Getenv("SHAIYA_TEST_LOGIN_ADDR")
	if address == "" {
		address = "127.0.0.1:30800"
	}
	l, err := session.Dial(address, user, password)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Servers) == 0 {
		t.Fatal("no server")
	}
	ip, err := l.SelectServer(l.Servers[0].ID, proto.ClientVersion)
	if err != nil {
		t.Fatal(err)
	}
	port := 30810
	if value := os.Getenv("SHAIYA_TEST_WORLD_PORT"); value != "" {
		n, e := strconv.ParseUint(value, 10, 16)
		if e != nil {
			t.Fatal(e)
		}
		port = int(n)
	}
	w, err := l.DialWorld(ip, port)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	st := game.New()
	send := func(p []byte) {
		t.Helper()
		if err := w.Send(p); err != nil {
			t.Fatal(err)
		}
	}
	wait := func(cond func() bool) {
		t.Helper()
		deadline := time.NewTimer(35 * time.Second)
		tick := time.NewTicker(250 * time.Millisecond)
		defer tick.Stop()
		defer deadline.Stop()
		for !cond() {
			select {
			case p, ok := <-w.Packets:
				if !ok {
					t.Fatal("world disconnected")
				}
				if _, err := st.Apply(p); err != nil {
					t.Fatalf("packet 0x%04x: %v", proto.Op(p), err)
				}
				if proto.Op(p) == proto.CharacterSkillBar && st.Phase == game.PhaseLoading {
					send(proto.MapLoadedPacket())
				}
			case <-tick.C:
			case <-deadline.C:
				t.Fatal("timed out waiting for server state")
			}
		}
	}

	return st, w, send, wait
}

// Enter selects the disposable fixture's first character and loads its world.
func Enter(t *testing.T) (*game.State, *session.World, func([]byte), func(func() bool)) {
	return EnterAs(t, os.Getenv("SHAIYA_TEST_USER"), os.Getenv("SHAIYA_PASSWORD"))
}

func EnterAs(t *testing.T, user, password string) (*game.State, *session.World, func([]byte), func(func() bool)) {
	st, w, send, wait := ConnectAs(t, user, password)
	wait(func() bool { return st.Phase == game.PhaseCharSelect && st.Slots[0].ID != 0 })
	send(proto.SelectCharacterPacket(st.Slots[0].ID))
	wait(func() bool { return st.Phase == game.PhaseInWorld && st.InventoryReady })
	return st, w, send, wait
}
