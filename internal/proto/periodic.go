package proto

// PeriodicEffect is 0x050F or 0x0607: target, skill, level and three
// resource amounts. Rapture's debuff and heal paths order the last two
// differently, so they are kept in wire order rather than guessed.
type PeriodicEffect struct {
	Target                   uint32
	Skill                    uint16
	Level                    uint8
	HP, Resource1, Resource2 uint16
}

func DecodePeriodicEffect(p []byte) (v PeriodicEffect, err error) {
	r := NewReader(p)
	v.Target, v.Skill, v.Level = r.U32(), r.U16(), r.U8()
	v.HP, v.Resource1, v.Resource2 = r.U16(), r.U16(), r.U16()
	return v, r.Err()
}
