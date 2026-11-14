package game

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func questNPCChunk(mapID uint16, flags uint8, rows ...proto.QuestNPC) []byte {
	w := proto.NewWriter(proto.GXQuestNPCs)
	w.U16(mapID)
	w.U8(flags)
	w.U16(uint16(len(rows)))
	for _, n := range rows {
		w.U8(n.Type)
		w.U16(n.TypeID)
		w.F32(n.X)
		w.F32(n.Z)
	}
	return w.Bytes()
}

func TestQuestNPCCatalogAtomicAndMetadataOnly(t *testing.T) {
	s := New()
	s.MapID = 1
	s.SelfID = 42
	row := proto.QuestNPC{Type: 8, TypeID: 1, X: 900, Z: 800}
	apply := func(p []byte) {
		t.Helper()
		if _, err := s.Apply(p); err != nil {
			t.Fatal(err)
		}
	}
	apply(questNPCChunk(1, proto.QuestNPCBegin, row))
	if s.MapQuestNPCsReady || len(s.MapQuestNPCs) != 0 || s.QuestMapRevision != 0 {
		t.Fatal("partial catalog published")
	}
	apply(questNPCChunk(1, proto.QuestNPCEnd, proto.QuestNPC{Type: 8, TypeID: 2, X: 1000, Z: 1000}))
	if !s.MapQuestNPCsReady || len(s.MapQuestNPCs) != 2 || len(s.Entities) != 0 || s.QuestMapRevision != 1 {
		t.Fatal("metadata created entities or failed atomic publication")
	}
	apply(questNPCChunk(2, 3, row))
	if s.QuestMapRevision != 1 || len(s.MapQuestNPCs) != 2 {
		t.Fatal("other-map catalog contaminated current map")
	}
	if _, err := s.Apply(questNPCChunk(1, 3, proto.QuestNPC{Type: 8, TypeID: 1, X: float32(math.NaN())})); err == nil {
		t.Fatal("accepted nonfinite coordinates")
	}
	if s.QuestMapRevision != 1 || len(s.MapQuestNPCs) != 2 {
		t.Fatal("malformed catalog mutated state")
	}
	move := proto.NewWriter(proto.CharacterMove)
	move.U32(42)
	move.U8(0)
	move.U16(0)
	move.F32(200)
	move.F32(0)
	move.F32(200)
	apply(move.Bytes())
	if s.QuestMapRevision != 1 {
		t.Fatal("walking invalidated quest eligibility")
	}
	teleport := proto.NewWriter(proto.CharacterMapTeleport)
	teleport.U32(42)
	teleport.U16(2)
	teleport.F32(10)
	teleport.F32(0)
	teleport.F32(10)
	apply(teleport.Bytes())
	if s.MapQuestNPCsReady || len(s.MapQuestNPCs) != 0 || s.QuestMapRevision != 2 {
		t.Fatal("map transfer retained catalog")
	}
}
