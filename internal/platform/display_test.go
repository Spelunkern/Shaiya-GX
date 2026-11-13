package platform

import "testing"

func TestDisplayModesAreValidUniqueAndSorted(t *testing.T) {
	modes := NormalizeDisplayModes([]DisplayMode{{1920, 1080}, {800, 600}, {800, 600}, {0, 600}, {99999, 768}})
	if len(modes) != 3 || modes[0] != (DisplayMode{}) || modes[1] != (DisplayMode{800, 600}) || modes[2] != (DisplayMode{1920, 1080}) {
		t.Fatal(modes)
	}
	if modes[0].Label() != "Window" || modes[2].Label() != "1920x1080" {
		t.Fatal("resolution labels")
	}
}
