package proto

const (
	DuelRequest   Opcode = 0x2401
	DuelAnswer    Opcode = 0x2402
	DuelCountdown Opcode = 0x2403
	DuelStart     Opcode = 0x2404
	DuelResult    Opcode = 0x2405
	DuelEnd       Opcode = 0x2406
	DuelWindow    Opcode = 0x2407
	DuelWindowEnd Opcode = 0x2408
	DuelReady     Opcode = 0x2409
	DuelAdd       Opcode = 0x240a
	DuelRemove    Opcode = 0x240b
	DuelMoney     Opcode = 0x240c
	DuelAddOther  Opcode = 0x240d
)

type DuelChallenge struct{ Requester, Target uint32 }
type DuelResponse struct {
	Result uint8
	Player uint32
}

func DecodeDuelChallenge(p []byte) (d DuelChallenge, err error) {
	r := NewReader(p)
	d = DuelChallenge{r.U32(), r.U32()}
	return d, r.Err()
}
func DecodeDuelResponse(p []byte) (d DuelResponse, err error) {
	r := NewReader(p)
	d = DuelResponse{r.U8(), r.U32()}
	return d, r.Err()
}
func DecodeDuelCenter(p []byte) (x, z float32, err error) {
	r := NewReader(p)
	x, z = r.F32(), r.F32()
	return x, z, r.Err()
}
func DecodeDuelEmpty(p []byte) error     { return NewReader(p).Err() }
func DuelRequestPacket(id uint32) []byte { return idPacket(DuelRequest, id) }
func DuelAnswerPacket(accept bool) []byte {
	w := NewWriter(DuelAnswer)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
func DuelGiveUpPacket() []byte       { return NewWriter(DuelEnd).Bytes() }
func DuelReadyPacket(v uint8) []byte { w := NewWriter(DuelReady); w.U8(v); return w.Bytes() }
func DuelAddPacket(bag, slot, count, dest uint8) []byte {
	w := NewWriter(DuelAdd)
	w.Raw([]byte{bag, slot, count, dest})
	return w.Bytes()
}
func DuelRemovePacket(slot uint8) []byte  { w := NewWriter(DuelRemove); w.U8(slot); return w.Bytes() }
func DuelMoneyPacket(money uint32) []byte { w := NewWriter(DuelMoney); w.U32(money); return w.Bytes() }
