package proto

import "fmt"

type GuildRemarkResult struct {
	Result uint8
	Remark string
}

func GuildRemarkPacket(remark string) []byte {
	w := NewWriter(GuildRemark)
	w.Fixed(remark, 65, true)
	return w.Bytes()
}
func DecodeGuildRemark(p []byte) (GuildRemarkResult, error) {
	r := NewReader(p)
	v := GuildRemarkResult{Result: r.U8(), Remark: r.Fixed(65)}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("guild remark: trailing bytes")
	}
	return v, nil
}
func GuildEtinPacket() []byte { return NewWriter(GuildEtin).Bytes() }
