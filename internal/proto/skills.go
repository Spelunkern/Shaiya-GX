package proto

import "fmt"

// LearnedSkill: 0x0108, Rapture buildCharacterSkills (client 0x58e020).
type LearnedSkill struct {
	ID            uint16
	Level, Number uint8
	Delay         int32
}

func DecodeSkills(p []byte) ([]LearnedSkill, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/8 {
		return nil, ErrShort
	}
	out := make([]LearnedSkill, n)
	for i := range out {
		out[i] = LearnedSkill{r.U16(), r.U8(), r.U8(), r.I32()}
	}
	return out, r.Err()
}

type QuickSlot struct {
	Bar, Slot, Bag uint8
	Number         uint16
}

func DecodeQuickSlots(p []byte) ([]QuickSlot, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/5 {
		return nil, ErrShort
	}
	out := make([]QuickSlot, n)
	for i := range out {
		out[i] = QuickSlot{r.U8(), r.U8(), r.U8(), r.U16()}
	}
	return out, r.Err()
}

// SkillBarPacket: writer 0x599210, count + five bytes per assignment.
func SkillBarPacket(slots []QuickSlot) ([]byte, error) {
	if len(slots) > 128 {
		return nil, fmt.Errorf("skill bar: too many slots")
	}
	w := NewWriter(CharacterSkillBar)
	w.U8(uint8(len(slots)))
	for _, q := range slots {
		if q.Bar >= 5 || q.Slot >= 10 || !((q.Bag <= 5 && q.Number < 24) || (q.Bag == 100 && q.Number > 0)) {
			return nil, fmt.Errorf("skill bar: invalid slot %+v", q)
		}
		w.U8(q.Bar)
		w.U8(q.Slot)
		w.U8(q.Bag)
		w.U16(q.Number)
	}
	return w.Bytes(), nil
}

type LearnResult struct {
	Result, Number uint8
	ID             uint16
	Level          uint8
}

func DecodeLearnResult(p []byte) (m LearnResult, err error) {
	r := NewReader(p)
	m.Result, m.Number, m.ID, m.Level = r.U8(), r.U8(), r.U16(), r.U8()
	return m, r.Err()
}

// LearnSkillPacket: writer 0x5983c0, 5 bytes.
func LearnSkillPacket(id uint16, level uint8) []byte {
	w := NewWriter(LearnNewSkill)
	w.U16(id)
	w.U8(level)
	return w.Bytes()
}

// UseSkillPacket: writers 0x598c60/0x598cc0; number is learned slot, not SkillID.
func UseSkillPacket(number uint8, target uint32, userTarget bool) []byte {
	op := SkillUse
	if userTarget {
		op = SkillUseNoTarget
	}
	w := NewWriter(op)
	w.U8(number)
	w.U32(target)
	return w.Bytes()
}

type SkillResult struct {
	Result                       uint8
	Source, Target               uint32
	ID                           uint16
	Level                        uint8
	HP, SP, MP                   uint16
	HealedHP, HealedSP, HealedMP uint16
}

// SkillCast is 0x0510 (Game.exe 0x58fea0; Rapture beginSkillCast).
type SkillCast struct {
	Source, Target uint32
	ID             uint16
	Level          uint8
}

func DecodeSkillCast(p []byte) (m SkillCast, err error) {
	r := NewReader(p)
	m.Source, m.Target, m.ID, m.Level = r.U32(), r.U32(), r.U16(), r.U8()
	return m, r.Err()
}

func DecodeSkillResult(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	m.ID, m.Level = r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if Op(p) == SkillUserDrain || Op(p) == SkillMobDrain {
		m.HealedHP, m.HealedSP, m.HealedMP = r.U16(), r.U16(), r.U16()
	}
	if err := r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill result: %d trailing bytes", r.Remaining())
	}
	return m, nil
}

// 0x0514 and 0x051A carry extra HP/SP/MP damage with no result byte.
func DecodeSkillExtra(p []byte) (m SkillResult, err error) {
	r := NewReader(p)
	m.Source, m.Target, m.ID, m.Level = r.U32(), r.U32(), r.U16(), r.U8()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if err = r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("skill extra: %d trailing bytes", r.Remaining())
	}
	return m, nil
}

type ReflectedDamage struct {
	Source, Target uint32
	HP, SP, MP     uint16
}

// 0x0515 (user) and 0x051B (mob) share this layout.
func DecodeReflectedDamage(p []byte) (m ReflectedDamage, err error) {
	r := NewReader(p)
	m.Source, m.Target = r.U32(), r.U32()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	if err = r.Err(); err != nil {
		return m, err
	}
	if r.Remaining() != 0 {
		return m, fmt.Errorf("reflected damage: %d trailing bytes", r.Remaining())
	}
	return m, nil
}
