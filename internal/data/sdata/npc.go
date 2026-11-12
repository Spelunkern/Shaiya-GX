package sdata

import "fmt"

// NPC definitions are the first thirteen lists of EP5 NpcQuest.SData.
// Parsec NpcQuestBaseNpc/Merchant/GateKeeper; Rapture GetNpc uses type + row (one-based).
// Quest links and quest definitions after these lists are intentionally not parsed.
type NPCKey struct {
	Type uint8
	ID   uint16
}
type NPCGate struct {
	MapID   uint16
	X, Y, Z float32
	Name    string
	Cost    uint32
}

type NPC struct {
	Type                                    uint8
	TypeID                                  uint16
	Model, MoveDistance, MoveSpeed, Faction int32
	Name, Welcome                           string
	Gates                                   [3]NPCGate
	MerchantType                            uint8
	Items                                   []ItemKey
	StartQuests, EndQuests                  []uint16
}

func ParseNPCs(raw []byte) (map[NPCKey]NPC, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: b}
	return parseNPCs(c)
}

func parseNPCs(c *cursor) (map[NPCKey]NPC, error) {
	out := map[NPCKey]NPC{}
	for group := uint8(1); group <= 13 && c.err == nil; group++ {
		n := c.count(35)
		if n > 65535 {
			return nil, fmt.Errorf("npc: too many rows %d", n)
		}
		for i := 0; i < n && c.err == nil; i++ {
			def := NPC{Type: c.u8(), TypeID: c.u16()}
			if group == 1 {
				def.MerchantType = c.u8()
			}
			def.Model, def.MoveDistance, def.MoveSpeed, def.Faction = int32(c.u32()), int32(c.u32()), int32(c.u32()), int32(c.u32())
			def.Name, def.Welcome = c.str(), c.str()
			if group == 1 {
				for j, k := 0, c.count(2); j < k; j++ {
					def.Items = append(def.Items, ItemKey{c.u8(), c.u8()})
				}
			}
			if group == 2 {
				for j := range def.Gates {
					gate := &def.Gates[j]
					gate.MapID = c.u16()
					gate.X, gate.Y, gate.Z = c.f32(), c.f32(), c.f32()
					gate.Name, gate.Cost = c.str(), c.u32()
				}
			}
			def.StartQuests = questWords(c)
			def.EndQuests = questWords(c)
			out[NPCKey{group, uint16(i + 1)}] = def
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	return out, nil
}
