package proto

// Quest records and requests follow Rapture's 0x0901–0x0908 wire format.
type ActiveQuest struct {
	ID, MinutesLeft uint16
	Counters        [3]uint8
}
type FinishedQuest struct {
	ID      uint16
	Success uint8
}
type QuestEndResult struct {
	NPC               uint32
	ID                uint16
	Success, Variant  uint8
	Experience, Money uint32
	Item              [4]uint8
}

func DecodeQuestList(p []byte) ([]ActiveQuest, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/7 {
		return nil, ErrShort
	}
	out := make([]ActiveQuest, n)
	for i := range out {
		q := &out[i]
		q.ID = r.U16()
		q.MinutesLeft = r.U16()
		for j := range q.Counters {
			q.Counters[j] = r.U8()
		}
	}
	return out, r.Err()
}
func DecodeFinishedQuests(p []byte) ([]FinishedQuest, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/3 {
		return nil, ErrShort
	}
	out := make([]FinishedQuest, n)
	for i := range out {
		out[i] = FinishedQuest{r.U16(), r.U8()}
	}
	return out, r.Err()
}
func DecodeQuestStart(p []byte) (npc uint32, id uint16, err error) {
	r := NewReader(p)
	npc = r.U32()
	id = r.U16()
	return npc, id, r.Err()
}
func DecodeQuestCounter(p []byte) (id uint16, slot, value uint8, err error) {
	r := NewReader(p)
	id = r.U16()
	slot = r.U8()
	value = r.U8()
	return id, slot, value, r.Err()
}
func DecodeQuestEnd(p []byte) (q QuestEndResult, err error) {
	r := NewReader(p)
	q.NPC = r.U32()
	q.ID = r.U16()
	q.Success = r.U8()
	q.Variant = r.U8()
	q.Experience = r.U32()
	q.Money = r.U32()
	for i := range q.Item {
		q.Item[i] = r.U8()
	}
	return q, r.Err()
}
func DecodeQuestChoice(p []byte) (uint16, error) {
	r := NewReader(p)
	id := r.U16()
	return id, r.Err()
}
func QuestStartPacket(npc uint32, id uint16) []byte {
	w := NewWriter(QuestStart)
	w.U32(npc)
	w.U16(id)
	return w.Bytes()
}
func QuestEndPacket(npc uint32, id uint16) []byte {
	w := NewWriter(QuestEnd)
	w.U32(npc)
	w.U16(id)
	return w.Bytes()
}
func QuestChoosePacket(id uint16, choice uint8) []byte {
	w := NewWriter(QuestChoose)
	w.U16(id)
	w.U8(choice)
	return w.Bytes()
}
func QuestAbandonPacket(id uint16) []byte { w := NewWriter(QuestAbandon); w.U16(id); return w.Bytes() }
