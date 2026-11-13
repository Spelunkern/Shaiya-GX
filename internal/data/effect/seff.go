package effect

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
)

// SEFF is the original sequential-particle catalogue, distinct from EFT.
// Layout follows Parsec SeffEffect and Game.exe 0x530c70/0x530790.
type SEFF struct {
	Version   int32
	Timestamp [6]uint16
	Records   map[int32][]SEFFEmitter
}

type SEFFEmitter struct {
	Rate, Blend, Shape, LifetimeMS            uint32
	Speed, Alpha                              float32
	Texture                                   string
	Color                                     [3]uint8
	Origin                                    [3]float32
	Extent, HalfSize                          float32
	Local                                     bool
	OrbitSpeed, SizeRandomness, VerticalSpeed float32
	Reserved                                  int32
}

func ParseSEFF(b []byte) (*SEFF, error) {
	if len(b) < 20 || len(b) > 16<<20 {
		return nil, fmt.Errorf("SEFF: invalid length %d", len(b))
	}
	r := &reader{b: b}
	s := &SEFF{Version: r.i32(), Records: map[int32][]SEFFEmitter{}}
	if s.Version < 5 || s.Version > 7 {
		return nil, fmt.Errorf("SEFF: unsupported version %d", s.Version)
	}
	for i := range s.Timestamp {
		s.Timestamp[i] = binary.LittleEndian.Uint16(b[r.off:])
		r.off += 2
	}
	byteAt := func() uint8 {
		if r.err != nil {
			return 0
		}
		if r.off >= len(b) {
			r.err = fmt.Errorf("SEFF: truncated at %d", r.off)
			return 0
		}
		v := b[r.off]
		r.off++
		return v
	}
	n := r.count(8)
	for range n {
		id := r.i32()
		if _, ok := s.Records[id]; id < 0 || ok {
			return nil, fmt.Errorf("SEFF: invalid/repeated record %d", id)
		}
		layers := make([]SEFFEmitter, r.count(64))
		for i := range layers {
			e := &layers[i]
			e.Rate, e.Speed, e.Blend, e.Shape, e.LifetimeMS, e.Alpha = r.word(), r.f32(), r.word(), r.word(), r.word(), r.f32()
			length := r.count(2)
			if length > 256 {
				return nil, fmt.Errorf("SEFF: texture name too long")
			}
			units := make([]uint16, length)
			for j := range units {
				lo, hi := byteAt(), byteAt()
				units[j] = uint16(lo) | uint16(hi)<<8
			}
			e.Texture = strings.TrimRight(string(utf16.Decode(units)), "\x00")
			e.Color = [3]uint8{byteAt(), byteAt(), byteAt()}
			e.Origin, e.Extent, e.HalfSize = r.vec(), r.f32(), r.f32()
			e.Local = byteAt() != 0
			e.OrbitSpeed, e.SizeRandomness, e.VerticalSpeed = r.f32(), r.f32(), r.f32()
			e.Reserved = -999
			if s.Version > 5 {
				e.Reserved = r.i32()
			}
		}
		if r.err != nil {
			return nil, fmt.Errorf("SEFF: %w", r.err)
		}
		s.Records[id] = layers
	}
	if r.err != nil {
		return nil, fmt.Errorf("SEFF: %w", r.err)
	}
	if r.off != len(b) {
		return nil, fmt.Errorf("SEFF: %d trailing bytes", len(b)-r.off)
	}
	return s, nil
}

// WeaponFamily normalizes the original item type without dropping family 9
// (reverse dagger), which would shift every subsequent recipe (0x4cba90).
func WeaponFamily(typ uint8) uint8 {
	if typ >= 1 && typ <= 15 {
		return typ
	}
	if typ >= 45 && typ <= 65 {
		return [...]uint8{1, 2, 3, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 10, 11, 12, 12, 13, 13, 14, 15}[typ-45]
	}
	return 0
}

// 0x530790 ignores TypeID, rejects +0..4 and clamps +11..49 to +10.
// The weapon constructor 0x429b10 clears the armor encoding (>=50).
func WeaponEnchantRecord(typ, level uint8) (int32, bool) {
	family := WeaponFamily(typ)
	if family == 0 || level < 5 || level >= 50 {
		return 0, false
	}
	return int32(6*(family-1) + min(level, 10) - 4), true
}
