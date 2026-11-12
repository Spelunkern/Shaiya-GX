package proto

// Pools is 0x0521 (Game.exe 0x595090; Rapture buildCurrentHitpoints).
type Pools struct{ HP, MP, SP uint32 }

func DecodePools(p []byte) (m Pools, err error) {
	r := NewReader(p)
	m.HP, m.MP, m.SP = r.U32(), r.U32(), r.U32()
	return m, r.Err()
}

// MobHealth is 0x0305/0x0610 (0x58f2d0; Rapture buildMobTargetHP).
type MobHealth struct {
	ID uint32
	HP int32
}

func DecodeMobHealth(p []byte) (m MobHealth, err error) {
	r := NewReader(p)
	m.ID, m.HP = r.U32(), r.I32()
	return m, r.Err()
}

// AttackResult is 0x0502/0503/0605, result + source + target + damage.
// Rapture buildCharacterMobAttack/buildMobAttackCharacter; 0x58f6d0/0x590ca0.
type AttackResult struct {
	Result         uint8
	Source, Target uint32
	HP, SP, MP     uint16
}

func DecodeAttackResult(p []byte) (m AttackResult, err error) {
	r := NewReader(p)
	m.Result = r.U8()
	m.Source, m.Target = r.U32(), r.U32()
	m.HP, m.SP, m.MP = r.U16(), r.U16(), r.U16()
	return m, r.Err()
}

// Resurrection is 0x0551 (0x592090; Rapture buildCharacterResurrect).
type Resurrection struct {
	ID      uint32
	Kind    uint8
	Exp     uint32
	X, Y, Z float32
}

func DecodeResurrection(p []byte) (m Resurrection, err error) {
	r := NewReader(p)
	m.ID, m.Kind, m.Exp = r.U32(), r.U8(), r.U32()
	m.X, m.Y, m.Z = r.F32(), r.F32(), r.F32()
	return m, r.Err()
}

// TargetMobPacket is the ID-only writer 0x59b710 (0x0304).
func TargetMobPacket(id uint32) []byte { return idPacket(TargetMobGetState, id) }

type ReviveMode uint8

const (
	ReviveBind     ReviveMode = 2
	ReviveParty    ReviveMode = 3
	ReviveRuneBind ReviveMode = 4
	ReviveRuneHere ReviveMode = 5
)

// RevivePacket requests the bind point (writer 0x599270, 0x0553 mode 2).
func RevivePacket() []byte { return ReviveChoicePacket(ReviveBind) }

func ReviveChoicePacket(mode ReviveMode) []byte {
	w := NewWriter(ResurrectRequest)
	w.U8(uint8(mode))
	return w.Bytes()
}

// StopAttackPacket: writer 0x598430, opcode only.
func StopAttackPacket() []byte { return NewWriter(AutoAttackStop).Bytes() }
