package proto

import "testing"

func TestCharacterSpeedPacket(t *testing.T) {
	// 0x591a30 reads u32,u8,u8; 255 is a native blocked index.
	p := []byte{0x1c, 0x05, 0x44, 0x33, 0x22, 0x11, 7, 255}
	m, err := DecodeCharacterSpeed(p)
	if err != nil || m.ID != 0x11223344 || m.Attack != 7 || m.Move != 255 {
		t.Fatal(m, err)
	}
	for n := 0; n < len(p); n++ {
		if _, err := DecodeCharacterSpeed(p[:n]); err == nil {
			t.Fatalf("accepted speed packet truncated at %d", n)
		}
	}
}
