package proto

import "fmt"

// GXForgeQuote is a read-only preview; execution still validates live inventory.
const GXForgeQuote Opcode = 0x7F03
const (
	ForgeQuoteLink uint8 = iota
	ForgeQuoteEnchant
	ForgeQuoteExtract
)

type ForgeQuoteRequest struct {
	Serial                        uint32
	Kind                          uint8
	Item, Lapis, Material, Hammer [2]uint8
	Count                         uint8
}
type ForgeQuote struct {
	Serial     uint32
	Kind       uint8
	Valid      bool
	Rate, Gold uint32 // rate in successes per million
	Lapis      uint8
	Material   uint32
}

func ForgeQuotePacket(q ForgeQuoteRequest) []byte {
	w := NewWriter(GXForgeQuote)
	w.U32(q.Serial)
	w.U8(q.Kind)
	for _, k := range [][2]uint8{q.Item, q.Lapis, q.Material, q.Hammer} {
		w.Raw(k[:])
	}
	w.U8(q.Count)
	return w.Bytes()
}
func DecodeForgeQuoteRequest(p []byte) (q ForgeQuoteRequest, err error) {
	r := NewReader(p)
	q.Serial = r.U32()
	q.Kind = r.U8()
	for _, k := range []*[2]uint8{&q.Item, &q.Lapis, &q.Material, &q.Hammer} {
		k[0], k[1] = r.U8(), r.U8()
	}
	q.Count = r.U8()
	if r.Err() != nil || r.Remaining() != 0 || q.Kind > ForgeQuoteExtract {
		return q, fmt.Errorf("invalid forge quote request")
	}
	return q, nil
}
func DecodeForgeQuote(p []byte) (q ForgeQuote, err error) {
	r := NewReader(p)
	q.Serial = r.U32()
	q.Kind = r.U8()
	q.Valid = r.Bool()
	q.Rate = r.U32()
	q.Gold = r.U32()
	q.Lapis = r.U8()
	q.Material = r.U32()
	if r.Err() != nil || r.Remaining() != 0 || q.Kind > ForgeQuoteExtract || q.Rate > 1000000 {
		return q, fmt.Errorf("invalid forge quote")
	}
	return q, nil
}
