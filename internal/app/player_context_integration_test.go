//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestOriginalPlayerContextGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "player menu", Width: 1024, Height: 734, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-players"); err != nil {
		t.Fatal(err)
	}
	a.state.Target = 2
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	draw := func() {
		gl.Viewport(0, 0, 1024, 734)
		a.frameDT = 0
		a.Draw(1024, 734)
	}
	pixels := func() []byte {
		out := make([]byte, 1024*734*4)
		gl.ReadPixels(0, 0, 1024, 734, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&out[0]))
		return out
	}
	draw()
	before := pixels()
	a.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 734)
	if !a.playerContext.open {
		t.Fatal("native player menu did not open")
	}
	a.lastMouseX, a.lastMouseY = 310, 35
	draw()
	after := pixels()
	changed := 0
	c := a.playerContext
	for y := c.y; y < c.y+c.height; y++ {
		for x := c.x; x < c.x+c.width; x++ {
			i := ((733-y)*1024 + x) * 4
			if before[i] != after[i] || before[i+1] != after[i+1] || before[i+2] != after[i+2] {
				changed++
			}
		}
	}
	if changed < 1000 {
		t.Fatal("original popup frame/text did not render", changed)
	}
	if err = render.SavePNG(filepath.Join(dir, "player-menu-original.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	a.playerContext.open = false
	entity := a.state.Entities[2]
	x, y, _, visible := projectLabel(a.visualEntityPosition(entity).Add(render.V3{0, 1, 0}), a.scene3d.View.Mul(a.scene3d.Proj), 1024, 734)
	if !visible {
		t.Fatal("player model is outside context test camera")
	}
	a.state.Target = 0
	a.worldControls(&platform.Input{MouseX: x, MouseY: y, RightPressed: true, RightDown: true}, 1024, 734)
	if !a.playerContext.open || a.playerContext.id != 2 {
		t.Fatal("right click on player model did not open its own context", x, y, a.playerContext)
	}
	a.state.Party.Members = []proto.PartyMember{{ID: a.state.SelfID, Name: "Self"}, {ID: 2, Name: "Ally"}}
	a.state.Party.Leader = a.state.SelfID
	a.playerContext.x, a.playerContext.y = 1020, 730
	draw()
	if a.playerContext.x+a.playerContext.width > 1024 || a.playerContext.y+a.playerContext.height > 734 {
		t.Fatal("popup escaped viewport after roster changed")
	}
	if err = render.SavePNG(filepath.Join(dir, "player-menu-party.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	a.state.Party.Raid = true
	draw()
	if err = render.SavePNG(filepath.Join(dir, "player-menu-raid.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("popup GL error %#x", code)
	}
	t.Logf("native TipBox popup and SysMsg labels rendered; %d changed pixels; party/raid and viewport resizing verified", changed)
}
