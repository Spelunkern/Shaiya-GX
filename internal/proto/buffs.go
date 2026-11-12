package proto

// ActiveBuff is keyed by instance, not skill ID. Remaining is only present in
// the 0x010a snapshot; 0x050d supplies the seven-byte application identity.
// Readers: 0x591d70/0x58fd80. Rapture buildCharacterActiveBuffs/buildBuffAdd.
type ActiveBuff struct {
	Instance     uint32
	Skill        uint16
	Level        uint8
	Remaining    uint32
	HasRemaining bool
}

func DecodeActiveBuffs(p []byte) (out []ActiveBuff, err error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > r.Remaining()/11 {
		return nil, ErrShort
	}
	out = make([]ActiveBuff, n)
	for i := range out {
		out[i] = ActiveBuff{Instance: r.U32(), Skill: r.U16(), Level: r.U8(), Remaining: r.U32(), HasRemaining: true}
	}
	return out, r.Err()
}

func DecodeBuffAdd(p []byte) (b ActiveBuff, err error) {
	r := NewReader(p)
	b.Instance, b.Skill, b.Level = r.U32(), r.U16(), r.U8()
	return b, r.Err()
}

// CancelBuffPacket is 0x0511 number 0xff and the skill ID (not instance).
// It reuses writer 0x598c60; Rapture handleUseSkill/cancelBuff.
func CancelBuffPacket(skill uint16) []byte { return UseSkillPacket(0xff, uint32(skill), true) }
