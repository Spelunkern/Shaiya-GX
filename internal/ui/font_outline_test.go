package ui

import "testing"

func TestOutlineUnionAlpha(t *testing.T) {
	if got := outlineUnionAlpha(0, 0, 0, 0); got != 0 {
		t.Fatalf("empty neighbors gave %d, want 0", got)
	}
	if got := outlineUnionAlpha(0, 255, 0, 0); got != 255 {
		t.Fatalf("opaque neighbor gave %d, want 255", got)
	}
	// Four half-alpha overlaps under SRC_ALPHA / ONE_MINUS_SRC_ALPHA produce
	// (1 - (1 - 128/255)^4) * 255 = 239.31 -> 239.
	if got := outlineUnionAlpha(128, 128, 128, 128); got != 239 {
		t.Fatalf("four 128 neighbors gave %d, want 239", got)
	}
	if got := outlineUnionAlpha(64, 0, 128, 0); got < 159 || got > 161 {
		t.Fatalf("mixed neighbors gave %d, want ~160", got)
	}
}
