package proto

// StatPoints is the 0x0208 wire order: STR, DEX, REC, INT, WIS, LUC.
// Writer 0x598340, reader 0x58eda0; Rapture handleSetStatPoint echoes deltas.
type StatPoints [6]uint16

func (d StatPoints) Total() uint32 {
	var total uint32
	for _, n := range d {
		total += uint32(n)
	}
	return total
}

func StatPointsPacket(d StatPoints) []byte {
	w := NewWriter(SetStatPoint)
	for _, n := range d {
		w.U16(n)
	}
	return w.Bytes()
}

func DecodeStatPoints(p []byte) (d StatPoints, err error) {
	r := NewReader(p)
	for i := range d {
		d[i] = r.U16()
	}
	return d, r.Err()
}

// AdditionalStats is 0x0526: twelve dwords, session vtable slot 0xfc.
// Attribute order differs from 0x0208 (Rapture buildAdditionalStats).
type AdditionalStats struct {
	Str, Rec, Int, Wis, Dex, Luc             uint32
	MinAttack, MaxAttack, MinMagic, MaxMagic uint32
	Defense, Resistance                      uint32
}

func DecodeAdditionalStats(p []byte) (s AdditionalStats, err error) {
	r := NewReader(p)
	s.Str, s.Rec, s.Int, s.Wis, s.Dex, s.Luc = r.U32(), r.U32(), r.U32(), r.U32(), r.U32(), r.U32()
	s.MinAttack, s.MaxAttack, s.MinMagic, s.MaxMagic, s.Defense, s.Resistance = r.U32(), r.U32(), r.U32(), r.U32(), r.U32(), r.U32()
	return s, r.Err()
}

// StatReset is 0x0214 (0x595680; Rapture buildStatsReset). Its six
// absolute attributes use the roster order, unlike allocation deltas.
type StatReset struct {
	Result                       uint8
	Points                       uint16
	Str, Rec, Int, Wis, Dex, Luc uint16
}

func DecodeStatReset(p []byte) (s StatReset, err error) {
	r := NewReader(p)
	s.Result = r.U8()
	if s.Result != 1 {
		return s, r.Err()
	}
	s.Points, s.Str, s.Rec, s.Int, s.Wis, s.Dex, s.Luc = r.U16(), r.U16(), r.U16(), r.U16(), r.U16(), r.U16(), r.U16()
	return s, r.Err()
}

// MaxEnergy is 0x050b (0x58fcd0; Rapture buildMaxEnergy): 0 HP, 1 SP, 2 MP.
type MaxEnergy struct {
	ID    uint32
	Kind  uint8
	Value uint32
}

func DecodeMaxEnergy(p []byte) (s MaxEnergy, err error) {
	r := NewReader(p)
	s.ID, s.Kind, s.Value = r.U32(), r.U8(), r.U32()
	return s, r.Err()
}
