//go:build integration

package app

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestInventoryModelInCompleteFrameGL(t *testing.T) {
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
	if err := render.Init(true); err != nil {
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
	if err := a.Preview("world-inventory"); err != nil {
		t.Fatal(err)
	}
	a.frameDT = 0
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	capture := func(name string) image.Image {
		t.Helper()
		a.Draw(1024, 768)
		path := filepath.Join(dir, name)
		if err := render.SavePNG(path, 1024, 768); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		im, err := png.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		return im
	}
	for range 10 {
		a.Draw(1024, 768)
	}
	visible := capture("inventory-frame-visible.png")
	if a.inventory.preview == nil {
		t.Fatal("inventory actor was not assembled")
	}
	a.inventory.preview = nil // Keep the cached appearance; suppress just the model.
	hidden := capture("inventory-frame-without-model.png")
	x, y := a.inventoryUI.Origin(1024, 768)
	different := 0
	for py := y + 43; py < y+250; py++ {
		for px := x + 54; px < x+190; px++ {
			r, g, b, _ := visible.At(px, py).RGBA()
			rr, gg, bb, _ := hidden.At(px, py).RGBA()
			if r != rr || g != gg || b != bb {
				different++
			}
		}
	}
	if different < 500 {
		t.Fatalf("inventory actor invisible in complete reverse-depth frame: %d changed pixels", different)
	}
	t.Logf("full Draw path: inventory model covers %d pixels", different)
}
