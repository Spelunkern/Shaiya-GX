//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestOriginalGameMenuPagesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true, Debug: true})
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
	if err = a.Preview("world-options"); err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	for page, name := range []string{"menu", "graphics", "sound", "user", "extra-hotkeys", "hotkeys", "help"} {
		a.gameMenu.Open(ui.MenuPage(page))
		a.Draw(1024, 768)
		if err = render.SavePNG(filepath.Join(dir, "native-options-"+name+".png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
		if e := gl.GetError(); e != gl.NO_ERROR {
			t.Fatalf("%s: GL error %#x", name, e)
		}
	}
	a.menuOpen = false
	a.chatOpen = true
	a.chatInput.Text = "Chat with original channel selector"
	a.chatUI.ModeOpen = true
	a.chatUI.Channel = 3
	a.Draw(1024, 768)
	if err = render.SavePNG(filepath.Join(dir, "native-chat-selector.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
}
