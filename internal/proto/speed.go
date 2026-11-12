package proto

// CharacterSpeed is 0x051C (0x591a30; Rapture broadcastSpeed and
// buildCharacterDeathState). These are native indices, not units per second.
type CharacterSpeed struct {
	ID           uint32
	Attack, Move uint8
}

func DecodeCharacterSpeed(p []byte) (m CharacterSpeed, err error) {
	r := NewReader(p)
	m.ID, m.Attack, m.Move = r.U32(), r.U8(), r.U8()
	return m, r.Err()
}
