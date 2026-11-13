package ui

import "testing"

func TestGoldNativeFormattingAndColourBoundaries(t *testing.T) {
	for _, row := range []struct {
		value uint32
		text  string
		rgb   [3]uint8
	}{
		{0, "0", [3]uint8{255, 255, 255}},
		{999, "999", [3]uint8{255, 255, 255}},
		{1_000, "1,000", [3]uint8{128, 255, 255}},
		{9_999, "9,999", [3]uint8{128, 255, 255}},
		{10_000, "10,000", [3]uint8{0, 128, 255}},
		{99_999, "99,999", [3]uint8{0, 128, 255}},
		{100_000, "100,000", [3]uint8{0, 255, 64}},
		{999_999, "999,999", [3]uint8{0, 255, 64}},
		{1_000_000, "1,000,000", [3]uint8{255, 255, 0}},
		{9_999_999, "9,999,999", [3]uint8{255, 255, 0}},
		{10_000_000, "10,000,000", [3]uint8{255, 128, 0}},
		{99_999_999, "99,999,999", [3]uint8{255, 128, 0}},
		{100_000_000, "100,000,000", [3]uint8{255, 0, 255}},
		{999_999_999, "999,999,999", [3]uint8{255, 0, 255}},
		{1_000_000_000, "1,000,000,000", [3]uint8{128, 0, 255}},
		{4_294_967_295, "4,294,967,295", [3]uint8{128, 0, 255}},
	} {
		if got := GoldText(row.value); got != row.text {
			t.Fatalf("%d: text %q want %q", row.value, got, row.text)
		}
		if got := GoldRGB(row.value); got != row.rgb {
			t.Fatalf("%d: RGB %v want %v", row.value, got, row.rgb)
		}
	}
}
