package effect

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"unicode/utf16"
)

func seffFixture(version int32) []byte {
	var b bytes.Buffer
	write := func(v any) {
		if err := binary.Write(&b, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	write(version)
	write([6]uint16{2026, 11, 12, 1, 2, 3})
	write(int32(1))
	write(int32(49))
	write(int32(1))
	write(uint32(20))
	write(float32(.03))
	write(uint32(0))
	write(uint32(3))
	write(uint32(500))
	write(float32(.5))
	name := utf16.Encode([]rune("glow雪.dds"))
	write(int32(len(name)))
	write(name)
	write([3]byte{80, 120, 255})
	write([3]float32{0, .7, 0})
	write(float32(5))
	write(float32(.1))
	write(uint8(1))
	write(float32(1))
	write(float32(.01))
	write(float32(2))
	if version > 5 {
		write(int32(123))
	}
	return b.Bytes()
}

func TestSEFFVersionsAndTruncation(t *testing.T) {
	for _, version := range []int32{5, 6, 7} {
		data := seffFixture(version)
		parsed, err := ParseSEFF(data)
		if err != nil {
			t.Fatal(err)
		}
		e := parsed.Records[49][0]
		if e.Texture != "glow雪.dds" || e.Rate != 20 || !e.Local || e.Origin[1] != .7 || e.LifetimeMS != 500 || e.Color != [3]byte{80, 120, 255} {
			t.Fatal("SEFF fields shifted", e)
		}
		if version == 5 && e.Reserved != -999 || version > 5 && e.Reserved != 123 {
			t.Fatal("version-specific tail", e)
		}
		for end := 0; end < len(data); end++ {
			if _, err := ParseSEFF(data[:end]); err == nil {
				t.Fatalf("accepted truncated version%d at%d", version, end)
			}
		}
		if _, err := ParseSEFF(append(data, 0)); err == nil {
			t.Fatal("accepted trailing data")
		}
		binary.LittleEndian.PutUint32(data[32:36], math.Float32bits(float32(math.NaN())))
		if _, err := ParseSEFF(data); err == nil {
			t.Fatal("accepted nonfinite particle speed")
		}
	}
}

func TestNativeWeaponEnchantFamiliesAndTiers(t *testing.T) {
	for family := uint8(1); family <= 15; family++ {
		for level := uint8(5); level <= 10; level++ {
			id, ok := WeaponEnchantRecord(family, level)
			if !ok || id != int32((int(family)-1)*6+int(level)-4) {
				t.Fatal(family, level, id, ok)
			}
		}
	}
	for _, tc := range []struct {
		typ, level uint8
		record     int32
		ok         bool
	}{
		{1, 4, 0, false}, {1, 5, 1, true}, {15, 10, 90, true}, {9, 5, 49, true}, {10, 5, 55, true},
		{57, 5, 49, true}, {58, 5, 55, true}, {59, 5, 61, true}, {62, 5, 73, true}, {65, 49, 90, true},
		{1, 11, 6, true}, {1, 49, 6, true}, {1, 50, 0, false}, {1, 255, 0, false}, {19, 10, 0, false}, {34, 10, 0, false}, {0, 10, 0, false}, {66, 10, 0, false},
	} {
		id, ok := WeaponEnchantRecord(tc.typ, tc.level)
		if id != tc.record || ok != tc.ok {
			t.Fatal(tc, id, ok)
		}
	}
}

func FuzzSEFF(f *testing.F) {
	f.Add(seffFixture(5))
	f.Fuzz(func(t *testing.T, b []byte) { ParseSEFF(b) })
}
