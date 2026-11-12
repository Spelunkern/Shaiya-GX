package proto

import "fmt"

// BlessInit (0x0211) is faction:u8, count:u32, remaining milliseconds:u32;
// 0x020F updates the faction count without a timer.
type BlessingUpdate struct {
	Country            uint8
	Count, RemainingMS uint32
}

func DecodeBlessing(p []byte) (BlessingUpdate, error) {
	r := NewReader(p)
	b := BlessingUpdate{Country: r.U8(), Count: r.U32()}
	if Op(p) == BlessInit {
		b.RemainingMS = r.U32()
	}
	if err := r.Err(); err != nil {
		return b, err
	}
	if b.Country > 1 || r.Remaining() != 0 {
		return b, fmt.Errorf("invalid blessing country or trailing bytes")
	}
	return b, nil
}
