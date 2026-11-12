package proto

// DetectInfo is 0x0558 (reader 0x58ffd0); it carries no target ID.
type DetectInfo struct {
	DefenseElement uint32
	Level          uint16
	Mode           uint8
}

func DecodeDetectInfo(p []byte) (v DetectInfo, err error) {
	r := NewReader(p)
	v.DefenseElement, v.Level, v.Mode = r.U32(), r.U16(), r.U8()
	return v, r.Err()
}
