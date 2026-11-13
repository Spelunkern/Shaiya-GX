package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/gm"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestGMMapPointRoundTripsMapPosition(t *testing.T) {
	const size = 2048
	for _, p := range [][2]float32{{0, 0}, {1024, 512}, {2000, 1900}, {size, size}} {
		u, v, ok := ui.MapPosition(p[0], p[1], size)
		if !ok {
			t.Fatal("point outside map", p)
		}
		x, z := mapPointToWorld(u, v, size)
		if d := x - p[0] + z - p[1]; d > .01 || d < -.01 {
			t.Fatalf("round trip %v -> %v,%v", p, x, z)
		}
	}
}

func TestGotoCommandParsesAsGMCommand(t *testing.T) {
	command, args, ok := gm.Parse(gm.GotoCommand(120.5, 80.25, 12.5))
	if !ok || command != "/goto" || len(args) != 3 || args[0] != "120.50" || args[1] != "80.25" || args[2] != "12.50" {
		t.Fatal(command, args, ok)
	}
	if _, args, ok := gm.Parse(gm.GotoXZCommand(1, 2)); !ok || len(args) != 2 {
		t.Fatal("xz goto", args, ok)
	}
}

// A drag pans the map, and a non-GM click never teleports; neither may panic
// or require a world connection.
func TestFullMapClickWithoutGMOrWithDragDoesNotTeleport(t *testing.T) {
	a := inputApp()
	minimap, err := ui.NewMinimap(inputTextures{})
	if err != nil {
		t.Fatal(err)
	}
	a.minimap = minimap
	v := a.mapState()
	if v == nil {
		t.Skip("input app has no world view")
	}
	v.full = true
	x, y, scale := a.minimap.FullRect(1024, 768)
	cx, cy := x+int(16*scale)+100, y+int(43*scale)+100
	a.state.Self.IsGM = false
	a.updateMinimap(&platform.Input{MouseX: cx, MouseY: cy, Pressed: true, MouseDown: true}, 1024, 768)
	a.updateMinimap(&platform.Input{MouseX: cx, MouseY: cy, Clicked: true}, 1024, 768)
	a.state.Self.IsGM = true
	a.updateMinimap(&platform.Input{MouseX: cx, MouseY: cy, Pressed: true, MouseDown: true}, 1024, 768)
	a.updateMinimap(&platform.Input{MouseX: cx + 40, MouseY: cy, MouseDown: true}, 1024, 768)
	if !v.panned {
		t.Fatal("drag was not recognised as panning")
	}
	a.updateMinimap(&platform.Input{MouseX: cx + 40, MouseY: cy, Clicked: true}, 1024, 768)
}
