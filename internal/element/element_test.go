package element

import (
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestNativeCyclesTiersAndNeutral(t *testing.T) {
	for _, pair := range [][2]uint8{{1, 4}, {4, 3}, {3, 2}, {2, 1}} {
		a, d := pair[0], pair[1]
		if Multiplier(a, d) != 1.4 || Multiplier(a+4, d) != 1.6 || Multiplier(d, a) != .5 || Multiplier(a, d+4) != 1.3 || Multiplier(a+4, d+4) != 1.4 {
			t.Fatal("native cycle/tier changed", pair)
		}
	}
	if Multiplier(0, 1) != .8 || Multiplier(0, 5) != .7 || Multiplier(1, 0) != 1.2 || Multiplier(5, 0) != 1.3 || Multiplier(1, 6) != .4 || Multiplier(5, 6) != .5 {
		t.Fatal("native neutral or disadvantage changed")
	}
	for id := uint8(1); id <= 8; id++ {
		if Family(id) != (id-1)%4+1 || Tier(id) != (id-1)/4+1 || Multiplier(id, id) != 1 {
			t.Fatal(id)
		}
	}
	if Normalize(255) != 0 || MobElement(5) != 0 || Multiplier(255, 1) != 1 {
		t.Fatal("invalid indices")
	}
}

func TestIntrinsicAndFirstSocketElementPriority(t *testing.T) {
	for _, tc := range []struct {
		base uint8
		gems []uint8
		want uint8
	}{
		{0, []uint8{0, 5, 5}, 5}, {1, []uint8{0, 1}, 1}, {1, []uint8{2, 1}, 1},
		{0, []uint8{1, 2, 3}, 1}, {1, []uint8{5}, 1}, {255, []uint8{255, 4}, 4},
	} {
		if got := Merge(tc.base, tc.gems); got != tc.want {
			t.Fatalf("%+v got%d", tc, got)
		}
	}
}

// Read immutable local reference only. Neither the executable nor its table
// bytes are bundled as a fixture. This independently checks all 81 float bits.
func TestOriginalEP5ExecutableMatrix(t *testing.T) {
	filename := os.Getenv("SHAIYA_ORIGINAL_SERVER")
	if filename == "" {
		t.Skip("requires immutable original EP5 ps_game.exe")
	}
	raw, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547" {
		t.Fatal("unrecognized native server", got)
	}
	exe, err := pe.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer exe.Close()
	hdr, ok := exe.OptionalHeader.(*pe.OptionalHeader32)
	if !ok {
		t.Fatal("expected native x86 PE")
	}
	rva := uint32(0x542260) - hdr.ImageBase
	for _, section := range exe.Sections {
		if rva < section.VirtualAddress || rva+81*4 > section.VirtualAddress+section.Size {
			continue
		}
		data, err := section.Data()
		if err != nil {
			t.Fatal(err)
		}
		start := int(rva - section.VirtualAddress)
		for a := uint8(0); a < 9; a++ {
			for d := uint8(0); d < 9; d++ {
				offset := start + (int(a)*9+int(d))*4
				native := binary.LittleEndian.Uint32(data[offset:])
				if math.Float32bits(Multiplier(a, d)) != native {
					t.Fatalf("native matrix mismatch %d->%d", a, d)
				}
			}
		}
		return
	}
	t.Fatal("native matrix section missing")
}

func TestElementSourceRespectsSocketHolesAndRepeatedValues(t *testing.T) {
	for _, tc := range []struct {
		base   uint8
		gems   []uint8
		value  uint8
		source int
	}{
		{0, []uint8{0, 0, 5, 0, 2, 5}, 5, 2}, {3, []uint8{0, 3, 4, 5, 0, 3}, 3, -1},
		{0, []uint8{0, 0, 0, 0, 0, 8}, 8, 5}, {0, []uint8{0, 0, 0, 0, 0, 0}, 0, -1},
		{0, []uint8{2, 2, 2, 2, 2, 2}, 2, 0},
	} {
		v, source := Resolve(tc.base, tc.gems)
		if v != tc.value || source != tc.source {
			t.Fatal(tc, v, source)
		}
	}
}
