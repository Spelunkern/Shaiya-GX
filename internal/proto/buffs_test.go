package proto

import (
	"bytes"
	"testing"
)

func TestBuffSnapshotAndApplicationHaveDifferentLengths(t *testing.T) {
	p := []byte{0x0a, 1, 1, 0x78, 0x56, 0x34, 0x12, 31, 0, 1, 0x2c, 1, 0, 0}
	rows, err := DecodeActiveBuffs(p)
	if err != nil || len(rows) != 1 || rows[0].Instance != 0x12345678 || rows[0].Skill != 31 || rows[0].Remaining != 300 || !rows[0].HasRemaining {
		t.Fatal(rows, err)
	}
	for i := range p {
		if _, err := DecodeActiveBuffs(p[:i]); err == nil {
			t.Fatal("truncated snapshot", i)
		}
	}
	add := []byte{0x0d, 5, 0x78, 0x56, 0x34, 0x12, 31, 0, 1}
	if b, err := DecodeBuffAdd(add); err != nil || b.Instance != rows[0].Instance || b.Skill != 31 || b.HasRemaining {
		t.Fatal(b, err)
	}
	for i := range add {
		if _, err := DecodeBuffAdd(add[:i]); err == nil {
			t.Fatal("truncated application", i)
		}
	}
	if got := CancelBuffPacket(31); !bytes.Equal(got, []byte{0x11, 5, 0xff, 31, 0, 0, 0}) {
		t.Fatalf("cancel must send skill ID, not instance: %x", got)
	}
}
