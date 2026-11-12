//go:build integration

package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestAutomaticQuestEndLoadsDefinitions(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("requires original assets")
	}
	arc, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	fixture := &App{arc: arc, state: game.New()}
	if err := fixture.loadQuests(); err != nil {
		t.Fatal(err)
	}
	first := fixture.questDefs[0]
	if !strings.Contains(first.Descriptions[0], "Hey there, newcomer") || !strings.Contains(first.Completion[0], "Welcome back") {
		t.Fatal("initial and completion messages do not match original quest 1")
	}
	var once, repeat, timed uint16
	for i, q := range fixture.questDefs {
		if q.TickReceive == 0 && once == 0 {
			once = uint16(i + 1)
		}
		if q.TickReceive != 0 && repeat == 0 {
			repeat = uint16(i + 1)
		}
		if q.MaxMinutes != 0 && q.TickReceive == 0 && timed == 0 {
			timed = uint16(i + 1)
		}
	}
	t.Logf("timed nonrepeatable candidate: %d", timed)
	if once == 0 || repeat == 0 {
		t.Fatal("original data lacks repeatable and nonrepeatable fixtures")
	}
	nameChecked := false
	for _, def := range fixture.questDefs {
		npc := fixture.npcDefs[sdata.NPCKey{Type: def.EndNPCType, ID: def.EndNPCID}]
		if def.EndNPCType != 0 && npc.Name != "" {
			if text := fixture.questObjective(def, proto.ActiveQuest{}); !strings.Contains(text, npc.Name) {
				t.Fatalf("hand-in NPC %q missing from %q", npc.Name, text)
			}
			nameChecked = true
			break
		}
	}
	if !nameChecked {
		t.Fatal("no quest with a named hand-in NPC")
	}
	for _, tc := range []struct {
		id       uint16
		finished bool
	}{{once, true}, {repeat, false}} {
		a := &App{arc: arc, state: game.New()}
		a.state.Quests = map[uint16]proto.ActiveQuest{tc.id: {ID: tc.id}}
		a.state.FinishedQuests = map[uint16]uint8{tc.id: 1}
		p := proto.NewWriter(proto.QuestEnd)
		p.U32(0)
		p.U16(tc.id)
		p.U8(1)
		p.U8(0)
		p.U32(0)
		p.U32(0)
		p.Raw([]byte{0, 0, 0, 0})
		if _, err := a.state.Apply(p.Bytes()); err != nil {
			t.Fatal(err)
		}
		a.confirmedQuest(p.Bytes())
		_, finished := a.state.FinishedQuests[tc.id]
		if !a.questDataLoaded || finished != tc.finished || len(a.state.Quests) != 0 {
			t.Fatalf("quest %d: definitions=%v finished=%v active=%v", tc.id, a.questDataLoaded, finished, a.state.Quests)
		}
	}
}

// The N2 fixture inserts an expired timed quest only for its disposable
// character. No quest UI is opened before the server sends the expiry.
func TestAutomaticQuestExpiryAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_QUEST_EXPIRY") != "1" {
		t.Skip("requires expired disposable quest fixture")
	}
	st, w, send, wait := testworld.Connect(t)
	wait(func() bool { return st.Phase == game.PhaseCharSelect && st.Slots[0].ID != 0 })
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state, a.world, a.arc = st, w, arc
	send(proto.SelectCharacterPacket(st.Slots[0].ID))
	deadline := time.Now().Add(20 * time.Second)
	for st.LastQuestEnd.ID != 41 {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("automatic quest expiry did not arrive")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !a.questDataLoaded || st.LastQuestEnd.Success != 0 {
		t.Fatal("expiry did not load definitions or report failure", st.LastQuestEnd)
	}
	if success, ok := st.FinishedQuests[41]; !ok || success != 0 {
		t.Fatal("automatic failure was not recorded", st.FinishedQuests[41], ok)
	}
	if _, active := st.Quests[41]; active {
		t.Fatal("expired quest remains active")
	}
	if err := w.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	next, _, _, waitNext := testworld.Enter(t)
	waitNext(func() bool { _, ok := next.FinishedQuests[41]; return ok })
	if next.FinishedQuests[41] != 0 {
		t.Fatal("Rapture did not persist the failed quest")
	}
}
