package netx

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestCTRCarriesLittleEndianCounter(t *testing.T) {
	key := [16]byte{1}
	ctr := [16]byte{0xff, 0xff}
	a, b := NewCTR(key, ctr), NewCTR(key, ctr)
	msg := bytes.Repeat([]byte("shaiya"), 20)
	enc := append([]byte(nil), msg...)
	a.XOR(enc[:7])
	a.XOR(enc[7:]) // split calls keep the partial block
	b.XOR(enc)
	if !bytes.Equal(enc, msg) {
		t.Fatal("CTR is not symmetric across split calls")
	}
}

func TestXORTableMatchesChain(t *testing.T) {
	seed := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	tab := NewXORTable(seed)
	d0 := sha256.Sum256(seed[:])
	d1 := sha256.Sum256(d0[16:32])
	if !bytes.Equal(tab.buf[:32], d0[:]) || !bytes.Equal(tab.buf[32:64], d1[:]) {
		t.Fatal("table chain differs from 0x401520")
	}
	p := []byte{0, 0, 0}
	tab.XOR(p)
	if !bytes.Equal(p, tab.buf[3:6]) {
		t.Fatal("payload must use table[n:2n]")
	}
}

func TestFrameRoundTrip(t *testing.T) {
	f, err := AppendFrame(nil, []byte{0x01, 0xA1, 9})
	if err != nil || len(f) != 5 || f[0] != 5 {
		t.Fatalf("frame %v %v", f, err)
	}
	p, err := ReadFrame(bufio.NewReader(bytes.NewReader(f)))
	if err != nil || !bytes.Equal(p, []byte{0x01, 0xA1, 9}) {
		t.Fatalf("payload %v %v", p, err)
	}
}

func FuzzReadFrame(f *testing.F) {
	f.Add([]byte{5, 0, 1, 2, 3})
	f.Fuzz(func(t *testing.T, b []byte) { ReadFrame(bufio.NewReader(bytes.NewReader(b))) })
}
