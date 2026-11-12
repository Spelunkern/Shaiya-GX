package audio

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func wave(kind, channels, bits uint16, rate uint32, data []byte) []byte {
	b := make([]byte, 44+len(data)+(len(data)&1))
	copy(b, "RIFF")
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[4:], uint32(len(b)-8))
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], kind)
	binary.LittleEndian.PutUint16(b[22:], channels)
	binary.LittleEndian.PutUint32(b[24:], rate)
	binary.LittleEndian.PutUint16(b[32:], channels*bits/8)
	binary.LittleEndian.PutUint16(b[34:], bits)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(len(data)))
	copy(b[44:], data)
	return b
}
func TestWAVConversion(t *testing.T) {
	raw := wave(1, 1, 8, 22050, []byte{0, 128, 255})
	pcm, err := DecodeWAV(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != 24 {
		t.Fatal("resampled frame count")
	}
	first := int16(binary.LittleEndian.Uint16(pcm))
	middle := int16(binary.LittleEndian.Uint16(pcm[8:]))
	if first != -32768 || middle != 0 || !bytes.Equal(pcm[:2], pcm[2:4]) {
		t.Fatal("PCM sign, resampling, or mono duplication")
	}
	stereo := []byte{0x34, 0x12, 0x00, 0x80}
	got, err := DecodeWAV(wave(1, 2, 16, 44100, stereo))
	if err != nil || !bytes.Equal(got, stereo) {
		t.Fatal("native-rate stereo was altered")
	}
	for n := 0; n < len(raw); n++ {
		if _, err := DecodeWAV(raw[:n]); err == nil {
			t.Fatalf("accepted truncation at %d", n)
		}
	}
	if _, err := DecodeWAV(wave(2, 1, 16, 22050, []byte{0, 0})); err == nil {
		t.Fatal("compressed WAV interpreted as PCM")
	}
}
func FuzzWAV(f *testing.F) {
	f.Add(wave(1, 1, 16, 22050, []byte{0, 0}))
	f.Fuzz(func(t *testing.T, b []byte) { DecodeWAV(b) })
}

func TestWAVHighPrecisionFormats(t *testing.T) {
	for _, tc := range []struct {
		kind, bits uint16
		data       []byte
		want       int16
	}{
		{1, 24, []byte{0, 0, 0x80}, -32768},
		{1, 32, []byte{0, 0, 0, 0x80}, -32768},
		{3, 32, []byte{0, 0, 0x80, 0x3f}, 32767},
		{3, 32, []byte{0, 0, 0xc0, 0x7f}, 0},
	} {
		pcm, err := DecodeWAV(wave(tc.kind, 1, tc.bits, 44100, tc.data))
		if err != nil {
			t.Fatal(err)
		}
		if got := int16(binary.LittleEndian.Uint16(pcm)); got != tc.want {
			t.Fatalf("%d/%d: got %d want %d", tc.kind, tc.bits, got, tc.want)
		}
	}
}
