package render

import (
	"github.com/go-gl/gl/v4.5-core/gl"
	"testing"
)

func TestAuthoredEFTBlendCatalog(t *testing.T) {
	for _, tc := range []struct {
		s, d     int32
		src, dst uint32
	}{
		{4, 1, gl.SRC_ALPHA, gl.ONE}, {4, 5, gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA},
		{11, 8, gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA}, {12, 8, gl.ONE_MINUS_SRC_ALPHA, gl.SRC_ALPHA},
	} {
		s, d, ok := EFTBlend(tc.s, tc.d)
		if !ok || s != tc.src || d != tc.dst {
			t.Fatalf("%+v → %d/%d", tc, s, d)
		}
	}
	if _, _, ok := EFTBlend(13, 0); ok {
		t.Fatal("accepted invalid authored factor")
	}
}
