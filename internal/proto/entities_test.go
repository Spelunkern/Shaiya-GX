package proto

import "testing"

// Field order and conditional cloak follow Game.exe 0x58e360.
func shapeFixture(cloak, rapture bool) []byte {
	w := NewWriter(CharacterShape)
	w.U32(42)
	for _, v := range []uint8{0, 0, 1, 2, 3, 4, 2, 5, 1, 0, 2} {
		w.U8(v)
	}
	w.U32(123)
	for i := 0; i < 8; i++ {
		typ := uint8(0)
		if i == 7 && cloak {
			typ = 30
		}
		w.Raw([]byte{typ, uint8(i), 0})
	}
	w.Fixed("Player", 21, true)
	if cloak || rapture {
		w.Raw([]byte{1, 2, 3, 4, 5, 6})
	}
	w.Fixed("Guild", 25, true)
	return w.Bytes()
}

func TestUserShapeVariants(t *testing.T) {
	for _, variant := range []struct{ cloak, rapture bool }{{false, false}, {true, false}, {false, true}} {
		p := shapeFixture(variant.cloak, variant.rapture)
		m, err := DecodeUserShape(p)
		if err != nil || m.Name != "Player" || m.Guild != "Guild" || m.Race != 2 || m.Class != 5 || m.ID != 42 {
			t.Fatalf("bad shape: %+v %v", m, err)
		}
		for n := 0; n < len(p); n++ {
			// A six-byte-short Rapture extension has the native packet size;
			// its zero padding is indistinguishable from an empty guild.
			if variant.rapture && !variant.cloak && n == len(p)-6 {
				continue
			}
			if _, err := DecodeUserShape(p[:n]); err == nil {
				t.Fatalf("accepted truncated shape %d/%d", n, len(p))
			}
		}
	}
}

func TestEmptyCharacterSlot(t *testing.T) {
	w := NewWriter(CharacterList)
	w.U8(3)
	w.U32(0)
	m, err := DecodeCharacterSlot(w.Bytes())
	if err != nil || m.Slot != 3 || m.ID != 0 {
		t.Fatalf("empty slot: %+v %v", m, err)
	}
	if _, err := DecodeCharacterSlot(w.Bytes()[:6]); err == nil {
		t.Fatal("accepted truncated ID")
	}
}
