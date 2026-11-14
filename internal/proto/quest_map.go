package proto

import (
	"fmt"
	"math"
)

// GXQuestNPCs supplies map metadata, not visible/interactive world entities.
const GXQuestNPCs Opcode = 0x7F04
const QuestNPCBegin uint8 = 1
const QuestNPCEnd uint8 = 2
const QuestNPCChunkLimit = (65535 - 9) / 11

type QuestNPC struct {
	Type   uint8
	TypeID uint16
	X, Z   float32
}

type QuestNPCChunk struct {
	MapID uint16
	Flags uint8
	NPCs  []QuestNPC
}

func DecodeQuestNPCs(p []byte) (QuestNPCChunk, error) {
	r := NewReader(p)
	c := QuestNPCChunk{MapID: r.U16(), Flags: r.U8()}
	n := int(r.U16())
	if r.Err() != nil || n > QuestNPCChunkLimit || c.Flags & ^uint8(3) != 0 || r.Remaining() != n*11 {
		return c, fmt.Errorf("invalid quest NPC catalog")
	}
	c.NPCs = make([]QuestNPC, n)
	for i := range c.NPCs {
		q := &c.NPCs[i]
		q.Type, q.TypeID, q.X, q.Z = r.U8(), r.U16(), r.F32(), r.F32()
		if q.Type < 1 || q.Type > 13 || q.TypeID == 0 || math.IsNaN(float64(q.X)) || math.IsNaN(float64(q.Z)) || math.IsInf(float64(q.X), 0) || math.IsInf(float64(q.Z), 0) {
			return c, fmt.Errorf("invalid quest NPC position")
		}
	}
	return c, r.Err()
}
