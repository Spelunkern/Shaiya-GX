//go:build integration

package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOriginalPortalWindowsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 734, Hidden: true, Debug: true})
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
	a.state = game.New()
	a.loadNPCs()
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	for _, test := range []struct {
		id   uint16
		name string
	}{{32, "portal-free"}, {2, "portal-paid"}, {3, "portal-three-destinations"}} {
		def := a.npcDefs[sdata.NPCKey{Type: 2, ID: test.id}]
		if a.assets.Texture(fmt.Sprintf("interface/NpcFace/%d.tga", def.Model)) == nil {
			t.Fatal("missing original portal portrait", def.Model)
		}
		gl.Viewport(0, 0, 1024, 734)
		gl.ClearColor(.08, .1, .12, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.gateUI.ResetWelcomeScroll()
		a.sprites.Begin(1024, 734)
		a.drawGatekeeper(1024, 734, &game.Entity{NpcType: 2, TypeID: test.id}, def)
		a.sprites.End()
		if err = render.SavePNG(filepath.Join(dir, test.name+".png"), 1024, 734); err != nil {
			t.Fatal(err)
		}
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("portal GL error %#x", code)
		}
	}
}
