package audio

import (
	"bytes"
	_ "embed"
	"testing"
)

// Generated 440 Hz sine at 22050 Hz, mono, 80 ms; no game recording.
//
//go:embed testdata/tone.ogg
var oggTone []byte

func TestOGGResamplesAndDuplicatesMono(t *testing.T) {
	pcm, err := DecodeOGG(oggTone)
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != SampleRate*4*80/1000 {
		t.Fatal("duration", len(pcm))
	}
	nonzero := false
	for i := 0; i < len(pcm); i += 4 {
		if !bytes.Equal(pcm[i:i+2], pcm[i+2:i+4]) {
			t.Fatal("mono channels differ")
		}
		nonzero = nonzero || pcm[i] != 0 || pcm[i+1] != 0
	}
	if !nonzero {
		t.Fatal("silent output")
	}
}
func TestOGGRejectsMalformedAndTruncated(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte("RIFFnot an ogg"), oggTone[:15], oggTone[:len(oggTone)/2]} {
		if _, err := DecodeOGG(raw); err == nil {
			t.Fatal("accepted invalid OGG")
		}
	}
}
