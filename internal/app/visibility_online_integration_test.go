//go:build integration

package app

import (
	"encoding/json"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"os"
	"testing"
)

// Four disposable accounts: ordinary Light/Dark and GM Light/Dark.
func TestVisibilityAgainstIsolatedGX(t *testing.T) {
	var credentials [][2]string
	if raw := os.Getenv("SHAIYA_TEST_VISIBILITY_ACCOUNTS"); raw == "" {
		t.Skip("requires disposable visibility fixtures")
	} else if err := json.Unmarshal([]byte(raw), &credentials); err != nil || len(credentials) != 4 {
		t.Fatal("invalid visibility fixtures")
	}
	type client struct {
		state *game.State
		send  func([]byte)
		wait  func(func() bool)
	}
	var clients []client
	for _, a := range credentials {
		st, _, send, wait := testworld.EnterAs(t, a[0], a[1])
		clients = append(clients, client{st, send, wait})
	}
	barrier := func(c client) {
		n := len(c.state.Chat)
		c.send(proto.ChatPacket(proto.ChannelNormal, false, "/attack invalid"))
		c.wait(func() bool { return len(c.state.Chat) > n })
	}
	gm, observer := clients[2], clients[3]
	gm.wait(func() bool { return gm.state.Self.Concealed })
	observer.wait(func() bool { e := observer.state.Entities[gm.state.SelfID]; return e != nil && e.Concealed })
	for _, c := range clients[:2] {
		barrier(c)
		if c.state.Entities[gm.state.SelfID] != nil || c.state.Entities[observer.state.SelfID] != nil {
			t.Fatal("default hidden GM leaked to normal faction")
		}
	}
	command := func(text string) {
		n := len(gm.state.Chat)
		gm.send(proto.ChatPacket(proto.ChannelNormal, false, text))
		gm.wait(func() bool { return len(gm.state.Chat) > n })
	}
	command("/char off")
	for _, c := range clients[:2] {
		c.wait(func() bool { return c.state.Entities[gm.state.SelfID] != nil })
	}
	if gm.state.Self.Concealed {
		t.Fatal("char off kept ghost")
	}
	command("/attack off")
	if gm.state.Self.Concealed {
		t.Fatal("attack off changed visibility")
	}
	command("/char on")
	for _, c := range clients[:2] {
		c.state.Target = gm.state.SelfID
		c.wait(func() bool { return c.state.Entities[gm.state.SelfID] == nil })
		if c.state.Target != 0 {
			t.Fatal("hidden target remained selected")
		}
	}
	observer.wait(func() bool { e := observer.state.Entities[gm.state.SelfID]; return e != nil && e.Concealed })
	command("/attack on")
	if !gm.state.Self.Concealed {
		t.Fatal("attack on changed concealment")
	}
	if os.Getenv("SHAIYA_TEST_GM_SELF_BUFF") == "1" {
		gm.send(proto.UseSkillPacket(200, gm.state.SelfID, true))
		gm.wait(func() bool {
			return gm.state.LastSkill.ID == 30 && gm.state.LastSkill.Source == gm.state.SelfID && len(gm.state.Buffs) > 0
		})
		if !gm.state.Self.Concealed {
			t.Fatal("self buff revealed GM")
		}
		command("/attack on")
		if len(gm.state.Buffs) == 0 || !gm.state.Self.Concealed {
			t.Fatal("protection reset removed self buff or invisibility")
		}
		for _, c := range clients[:2] {
			barrier(c)
			if c.state.Entities[gm.state.SelfID] != nil {
				t.Fatal("self buff leaked hidden GM")
			}
		}
		t.Log("protected hidden GM self buff confirmed with native sound/effect result and preserved after /attack on")
	}
	t.Log("encrypted GX: both normal factions exclude hidden GMs; GMs see ghosts; stationary on/off refresh and target clear verified")
}
