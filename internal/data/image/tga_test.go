package image

import "testing"

func TestTruncatedTGAImageIDReturnsError(t *testing.T) {
	b := make([]byte, 18)
	b[0], b[2], b[12], b[14], b[16] = 255, 2, 1, 1, 24
	if _, err := Decode(b); err == nil {
		t.Fatal("truncated image ID accepted")
	}
}
