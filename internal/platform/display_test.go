package platform

import "testing"

func TestDisplayModesAreValidUniqueAndSorted(t *testing.T) {
	win := DisplayMode{Width: 1600, Height: 900, Windowed: true}
	modes := NormalizeDisplayModes([]DisplayMode{{Width: 1920, Height: 1080}, {Width: 800, Height: 600}, win, {Width: 800, Height: 600}, {Width: 0, Height: 600}, {Width: 99999, Height: 768}})
	if len(modes) != 4 || modes[0] != (DisplayMode{}) || modes[1] != win || modes[2] != (DisplayMode{Width: 800, Height: 600}) || modes[3] != (DisplayMode{Width: 1920, Height: 1080}) {
		t.Fatal(modes)
	}
	if modes[0].Label() != "Window" || modes[1].Label() != "1600x900 Win" || modes[3].Label() != "1920x1080" {
		t.Fatal("resolution labels")
	}
}
