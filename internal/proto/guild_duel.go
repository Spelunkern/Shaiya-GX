package proto

const (
	GuildDuelRequest   Opcode = 0x2501
	GuildDuelAnswer    Opcode = 0x2502
	GuildDuelCountdown Opcode = 0x2503
	GuildDuelStart     Opcode = 0x2504
	GuildDuelResult    Opcode = 0x2505
	GuildDuelLeave     Opcode = 0x2506
)

type GuildDuelCenter struct {
	X, Z          float32
	OpponentGuild uint32
}

func DecodeGuildDuelCenter(p []byte) (GuildDuelCenter, error) {
	r := NewReader(p)
	v := GuildDuelCenter{X: r.F32(), Z: r.F32(), OpponentGuild: r.U32()}
	return v, r.Err()
}
func GuildDuelRequestPacket(id uint32) []byte { return idPacket(GuildDuelRequest, id) }
func GuildDuelAnswerPacket(accept bool) []byte {
	w := NewWriter(GuildDuelAnswer)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
