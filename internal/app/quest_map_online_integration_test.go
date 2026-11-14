//go:build integration

package app

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestMapQuestCatalogAgainstGX(t *testing.T) {
	raw := os.Getenv("SHAIYA_TEST_VISIBILITY_ACCOUNTS")
	if raw == "" {
		t.Skip("requires disposable map fixture")
	}
	var accounts [][2]string
	if err := json.Unmarshal([]byte(raw), &accounts); err != nil || len(accounts) == 0 {
		t.Fatal("invalid fixture")
	}
	st, _, send, wait := testworld.EnterAs(t, accounts[0][0], accounts[0][1])
	wait(func() bool { return st.MapQuestNPCsReady })
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := &App{state: st, arc: arc}
	a.loadNPCs()
	if err = a.loadQuests(); err != nil {
		t.Fatal(err)
	}
	far, farAvailable, visible := 0, 0, 0
	for _, e := range st.Entities {
		if e.Kind == game.KindNpc {
			visible++
		}
	}
	for _, n := range st.MapQuestNPCs {
		if math.Hypot(float64(n.X-st.Self.X), float64(n.Z-st.Self.Z)) > 200 {
			far++
			if _, ok := a.npcQuestMapKind(&game.Entity{Kind: game.KindNpc, NpcType: n.Type, TypeID: n.TypeID}); ok {
				farAvailable++
			}
		}
	}
	if far == 0 || len(st.MapQuestNPCs) <= visible {
		t.Fatalf("catalog still restricted to visible NPCs: catalog%d visible%d far%d", len(st.MapQuestNPCs), visible, far)
	}
	marks := a.questMapMarkers(10000, nil)
	if len(marks) < farAvailable {
		t.Fatal("far available quest markers missing")
	}
	revision, builds := st.QuestMapRevision, a.questMapCache.builds
	for i := 0; i < 5; i++ {
		send(proto.MovePacket(0, 0, st.Self.X+float32(i)*.05, st.Self.Y, st.Self.Z))
	}
	history := len(st.Chat)
	send(proto.ChatPacket(proto.ChannelNormal, false, "/attack invalid"))
	wait(func() bool { return len(st.Chat) > history })
	a.questMapMarkers(10000, nil)
	if st.QuestMapRevision != revision || a.questMapCache.builds != builds {
		t.Fatal("walking resent/rebuilt the quest catalog")
	}
	t.Logf("map%d catalog=%d visibleNPCs=%d farNPCs=%d farAvailable=%d markers=%d; movement preserved cache", st.MapID, len(st.MapQuestNPCs), visible, far, farAvailable, len(marks))
}
