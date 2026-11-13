package model

import (
	"encoding/binary"
	"testing"
)

func TestCloakPCIndicesPinsAndTruncation(t *testing.T) {
	// One named body, one 2x2 cloth and one named shoulder; zero is a valid
	// model/texture/bone index. No original game bytes are embedded here.
	raw := make([]byte, 4+132+4+524+4+132)
	put := func(off int, v uint32) { binary.LittleEndian.PutUint32(raw[off:], v) }
	put(0, 1)
	copy(raw[8:], "body.3DC")
	put(136, 1)
	put(140+16, 1)
	put(140+20, 1)
	put(140+44, 1)
	put(664, 1)
	copy(raw[672:], "shoulder.3DC")
	pc, err := ParseCloakPC(raw)
	if err != nil || len(pc.Records) != 1 || len(pc.Records[0].Pins) != 1 || pc.Bodies[0] != "body.3DC" || pc.Shoulders[0] != "shoulder.3DC" {
		t.Fatal(pc, err)
	}
	for n := 0; n < len(raw); n++ {
		if _, err = ParseCloakPC(raw[:n]); err == nil {
			t.Fatalf("accepted truncated PC (%d)", n)
		}
	}
	put(140+44+4, 4)
	if _, err = ParseCloakPC(raw); err == nil {
		t.Fatal("accepted a pin outside its grid")
	}
	put(140+44+4, 0)
	put(140, 1)
	if _, err = ParseCloakPC(raw); err == nil {
		t.Fatal("accepted a missing cloth body")
	}
}

func TestClothMeshAndTextureValidation(t *testing.T) {
	raw := make([]byte, 8+40+4+6)
	binary.LittleEndian.PutUint32(raw[4:], 1)
	binary.LittleEndian.PutUint32(raw[48:], 1)
	if _, err := ParseCloth3DC(raw); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < len(raw); n++ {
		if _, err := ParseCloth3DC(raw[:n]); err == nil {
			t.Fatalf("accepted truncated mesh (%d)", n)
		}
	}
	binary.LittleEndian.PutUint16(raw[52:], 1)
	if _, err := ParseCloth3DC(raw); err == nil {
		t.Fatal("accepted triangle outside vertex array")
	}
	ctl := make([]byte, 260)
	binary.LittleEndian.PutUint32(ctl, 1)
	copy(ctl[4:], "test.dds")
	if names, err := ParseClothTextures(ctl); err != nil || len(names) != 1 || names[0] != "test.dds" {
		t.Fatal(names, err)
	}
	if _, err := ParseClothTextures(ctl[:259]); err == nil {
		t.Fatal("accepted truncated texture table")
	}
}
