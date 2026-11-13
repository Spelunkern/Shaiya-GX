//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"testing"
)

func TestGMNPCCommandsAgainstIsolatedGX(t *testing.T) {
	st, _, send, wait := testworld.Enter(t)
	wait(func() bool { return st.Self.IsGM && st.GMTurbo })
	send(proto.ChatPacket(proto.ChannelNormal, false, "/turbo"))
	wait(func() bool { return !st.GMTurbo }) // Also drains the initial map snapshot.
	for _, tc := range []struct {
		command string
		kind    uint8
	}{{"/nmake 1 1", 1}, {"/nmake 2 1", 2}} {
		before := make(map[uint32]bool, len(st.Entities))
		for id := range st.Entities {
			before[id] = true
		}
		history := len(st.Chat)
		send(proto.ChatPacket(proto.ChannelNormal, false, tc.command))
		wait(func() bool { return len(st.Chat) > history })
		count := 0
		for id, e := range st.Entities {
			if before[id] || e.Kind != game.KindNpc {
				continue
			}
			if e.NpcType != tc.kind || e.TypeID != 1 {
				t.Fatalf("wrong summoned NPC: %+v", e)
			}
			dx, dz := e.X-st.Self.X, e.Z-st.Self.Z
			if dx*dx+dz*dz < 4 || e.Y != st.Self.Y {
				t.Fatal("NPC not placed around GM at server height")
			}
			count++
		}
		if count != 1 {
			t.Fatalf("%s spawned %d NPCs; response=%s", tc.command, count, st.Chat[len(st.Chat)-1].Text)
		}
	}
	history := len(st.Chat)
	entities := len(st.Entities)
	send(proto.ChatPacket(proto.ChannelNormal, false, "/nmake 1001"))
	wait(func() bool { return len(st.Chat) > history })
	if len(st.Entities) != entities {
		t.Fatal("obsolete combined ID created NPC")
	}
	t.Log("GM default turbo, toggle, merchant 1/1 and gatekeeper 2/1 confirmed over live GX protocol")
}
