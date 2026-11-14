package game

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Only eligibility inputs invalidate cached quest markers. Movement, health,
// animation, NPC view boundaries and ordinary rendering are intentionally absent.
func questMapStateChanged(op proto.Opcode) bool {
	switch op {
	case proto.CharacterList, proto.CharacterDetails, proto.LevelUpSelf, proto.CharacterMapTeleport,
		proto.QuestList, proto.QuestFinishedList, proto.QuestStart, proto.QuestCounter, proto.QuestChoose, proto.QuestEnd:
		return true
	}
	return false
}

func (s *State) applyQuestNPCs(p []byte) error {
	c, err := proto.DecodeQuestNPCs(p)
	if err != nil {
		return err
	}
	if c.MapID != s.MapID {
		return nil
	} // A delayed catalog from the preceding map.
	if c.Flags&proto.QuestNPCBegin != 0 {
		s.questNPCStaging = nil
		s.questNPCStagingActive = true
	}
	if !s.questNPCStagingActive || len(s.questNPCStaging)+len(c.NPCs) > 65535 {
		return fmt.Errorf("unexpected quest NPC catalog fragment")
	}
	s.questNPCStaging = append(s.questNPCStaging, c.NPCs...)
	if c.Flags&proto.QuestNPCEnd != 0 {
		s.MapQuestNPCs, s.MapQuestNPCsReady = s.questNPCStaging, true
		s.questNPCStaging, s.questNPCStagingActive = nil, false
		s.QuestMapRevision++
	}
	return nil
}

func questMapInventoryChanged(op proto.Opcode) bool {
	switch op {
	case proto.CharacterItems, proto.AddItem, proto.RemoveItem, proto.ItemBagToBag:
		return true
	}
	return false
}
