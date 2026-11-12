package sdata

import "fmt"

type QuestItem struct{ Type, ID, Count uint8 }
type QuestResult struct {
	NeedMobID                               uint16
	NeedMobCount, NeedItemID, NeedItemCount uint8
	NeedTime                                uint32
	NeedHG                                  uint16
	NeedVG                                  int16
	NeedOG                                  uint8
	Experience, Money                       uint32
	ItemType, ItemID                        uint8
	NextQuest                               uint16
}
type Quest struct {
	ID                                                       uint16
	Name, Summary                                            string
	MinLevel, MaxLevel                                       uint16
	Faction, Mode                                            uint8
	Male, Female                                             bool
	Jobs                                                     [6]uint8
	HG                                                       uint16
	VG                                                       int16
	CG, OG, IG                                               uint8
	PrevQuest                                                uint16
	RequireParty                                             bool
	PartyJobs                                                [6]uint8
	MinMinutes, MaxMinutes, TickStart, TickKeep, TickReceive uint32
	StartType, StartNPCType                                  uint8
	StartNPCID                                               uint16
	StartItemType, StartItemID                               uint8
	StartItems                                               [3]QuestItem
	EndType, EndNPCType                                      uint8
	EndNPCID                                                 uint16
	FarmItems                                                [3]QuestItem
	PVPKills                                                 uint8
	MobIDs                                                   [2]uint16
	MobCounts                                                [2]uint8
	ResultType, UserSelect                                   uint8
	Results                                                  [3]QuestResult
	Descriptions                                             [4]string
	Completion                                               [3]string
}

func questWords(c *cursor) []uint16 {
	n := c.count(2)
	if n == 0 {
		return nil
	}
	v := make([]uint16, n)
	for i := range v {
		v[i] = c.u16()
	}
	return v
}
func questItem(c *cursor) QuestItem { return QuestItem{c.u8(), c.u8(), c.u8()} }

// ParseQuests uses the one-based row ID used by Rapture and NPC links;
// the embedded ID is kept for diagnostics. It also returns the NPC links.
func ParseQuests(raw []byte) (map[NPCKey]NPC, []Quest, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, nil, err
	}
	c := &cursor{b: b}
	npcs, err := parseNPCs(c)
	if err != nil {
		return nil, nil, err
	}
	for i := 0; i < 256*256 && c.err == nil; i++ {
		for j := 0; j < 2; j++ {
			n := c.count(2)
			c.take(n * 2)
		}
	}
	n := c.count(130)
	if n > 65535 {
		return nil, nil, fmt.Errorf("quest: too many rows %d", n)
	}
	quests := make([]Quest, 0, n)
	for i := 0; i < n && c.err == nil; i++ {
		q := Quest{ID: c.u16(), Name: c.str(), Summary: c.str()}
		q.MinLevel, q.MaxLevel = c.u16(), c.u16()
		q.Faction, q.Mode = c.u8(), c.u8()
		q.Male, q.Female = c.u8() != 0, c.u8() != 0
		for j := range q.Jobs {
			q.Jobs[j] = c.u8()
		}
		q.HG, q.VG = c.u16(), int16(c.u16())
		q.CG, q.OG, q.IG = c.u8(), c.u8(), c.u8()
		q.PrevQuest = c.u16()
		q.RequireParty = c.u8() != 0
		for j := range q.PartyJobs {
			q.PartyJobs[j] = c.u8()
		}
		q.MinMinutes, q.MaxMinutes = c.u32(), c.u32()
		q.TickStart, q.TickKeep, q.TickReceive = c.u32(), c.u32(), c.u32()
		q.StartType, q.StartNPCType, q.StartNPCID = c.u8(), c.u8(), c.u16()
		q.StartItemType, q.StartItemID = c.u8(), c.u8()
		for j := range q.StartItems {
			q.StartItems[j] = questItem(c)
		}
		q.EndType, q.EndNPCType, q.EndNPCID = c.u8(), c.u8(), c.u16()
		for j := range q.FarmItems {
			q.FarmItems[j] = questItem(c)
		}
		q.PVPKills = c.u8()
		for j := range q.MobIDs {
			q.MobIDs[j], q.MobCounts[j] = c.u16(), c.u8()
		}
		q.ResultType, q.UserSelect = c.u8(), c.u8()
		for j := range q.Results {
			r := &q.Results[j]
			r.NeedMobID, r.NeedMobCount = c.u16(), c.u8()
			r.NeedItemID, r.NeedItemCount = c.u8(), c.u8()
			r.NeedTime = c.u32()
			r.NeedHG, r.NeedVG, r.NeedOG = c.u16(), int16(c.u16()), c.u8()
			r.Experience, r.Money = c.u32(), c.u32()
			r.ItemType, r.ItemID, r.NextQuest = c.u8(), c.u8(), c.u16()
		}
		// The original EP5 records store three result messages first, followed
		// by the offer, journal, reminder and alternate text. Verified against
		// the actual English records (e.g. quests 1, 9 and 41).
		for j := range q.Completion {
			q.Completion[j] = c.str()
		}
		for j := range q.Descriptions {
			q.Descriptions[j] = c.str()
		}
		quests = append(quests, q)
	}
	if c.err != nil {
		return nil, nil, c.err
	}
	if c.off != len(b) {
		return nil, nil, fmt.Errorf("quest sdata: %d trailing bytes", len(b)-c.off)
	}
	return npcs, quests, nil
}
