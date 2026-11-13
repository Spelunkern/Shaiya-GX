package main

import "testing"

func TestKnownWaveReferenceDoesNotRewriteBinaryCoincidences(t *testing.T) {
	names := map[string]bool{"bg_wind0005.wav": true}
	for _, tc := range []struct {
		raw  string
		want bool
	}{
		{"bg_wind0005.wav\x00", true},
		{"Sound/BG_WIND0005.WAV\x00", true},
		{"random.wav\x00", false},
		{"bg_wind0005.wav_more", false},
	} {
		at := waveRef.FindStringIndex(tc.raw)
		if got := knownWaveReference([]byte(tc.raw), at[0], at[1], names); got != tc.want {
			t.Fatalf("%q: %v, want %v", tc.raw, got, tc.want)
		}
	}
}
