package platform

import "testing"

func TestDisplayModesAreWindowedUniqueAndSorted(t *testing.T) {
	modes := NormalizeDisplayModes([]DisplayMode{{}, {Width: 1920, Height: 1080}, {Width: 800, Height: 600}, {Width: 800, Height: 600, Kind: DisplayWindowed}, {Kind: DisplayFullscreen}, {Width: 0, Height: 600}, {Width: 99999, Height: 768}})
	if len(modes) != 2 || modes[0] != (DisplayMode{Width: 800, Height: 600}) || modes[1] != (DisplayMode{Width: 1920, Height: 1080}) {
		t.Fatal(modes)
	}
	for _, mode := range modes {
		if mode.Label() == "Window" || mode.Label() == "Fullscreen" || mode.Kind != DisplayLegacy {
			t.Fatal("non-resolution row", mode)
		}
	}
}

func TestCommonSizesRetainCurrentAndNativeResolution(t *testing.T) {
	modes := addWindowModes([]DisplayMode{{Width: 1920, Height: 1080}}, 1900, 1000, 1013, 731)
	has := func(w, h int) bool {
		for _, m := range modes {
			if m.Width == w && m.Height == h {
				return true
			}
		}
		return false
	}
	for _, p := range [][2]int{{1920, 1080}, {1600, 900}, {1366, 768}, {1013, 731}} {
		if !has(p[0], p[1]) {
			t.Fatal("missing resolution", p)
		}
	}
	for _, m := range modes {
		if m.Width <= 0 || m.Height <= 0 {
			t.Fatal("special mode exposed", m)
		}
	}
	small := addWindowModes(nil, 1920, 1080, 640, 360)
	found := false
	for _, m := range small {
		found = found || m.Width == 640 && m.Height == 360
	}
	if !found {
		t.Fatal("current custom window missing")
	}
}
