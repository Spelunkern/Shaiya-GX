//go:build integration

package app

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestHUDPortraitGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context and original assets; no server needed")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "portrait probe", Width: 96, Height: 96, Hidden: true, Debug: true})
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
	a.state, a.wv = game.New(), &worldView{}
	a.state.SelfID = 1
	mask := a.assets.Texture("interface/main_stats_circle2.tga")
	path := filepath.Join(t.TempDir(), "portrait.png")
	for race, jobs := range [4][]uint8{{0, 1, 5}, {2, 3, 4}, {2, 4, 5}, {0, 1, 3}} {
		for _, job := range jobs {
			for gender := range uint8(2) {
				name := fmt.Sprintf("race%d/job%d/gender%d", race, job, gender)
				a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: uint8(race), Class: job, Gender: gender, Height: 2, Face: 1, Hair: 1}
				gl.Viewport(0, 0, 96, 96)
				gl.Scissor(2, 3, 90, 89)
				gl.Disable(gl.SCISSOR_TEST)
				gl.ClearColor(0, 0, 1, 1)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				a.sprites.Begin(96, 96)
				portrait := a.hudPortrait(96, 96)
				if portrait == nil || portrait.W != 384 || portrait.H != 384 {
					t.Fatal(name, "missing snapshot")
				}
				if a.hudPortrait(96, 96) != portrait {
					t.Fatal(name, "snapshot regenerated without an appearance change")
				}
				var viewport, scissor [4]int32
				var framebuffer int32
				gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
				gl.GetIntegerv(gl.SCISSOR_BOX, &scissor[0])
				gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &framebuffer)
				if viewport != [4]int32{0, 0, 96, 96} || scissor != [4]int32{2, 3, 90, 89} || framebuffer != 0 || gl.IsEnabled(gl.SCISSOR_TEST) {
					t.Fatal(name, "snapshot changed enclosing framebuffer/viewport/scissor")
				}
				a.sprites.DrawMasked(portrait, mask, 16, 16)
				a.sprites.End()
				if err := render.SavePNG(path, 96, 96); err != nil {
					t.Fatal(err)
				}
				f, err := os.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				im, err := png.Decode(f)
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
				colored := 0
				for y := 24; y < 72; y++ {
					for x := 24; x < 72; x++ {
						r, g, _, _ := im.At(x, y).RGBA()
						if r > 1000 || g > 1000 {
							colored++
						}
					}
				}
				r, g, b, _ := im.At(16, 16).RGBA()
				if r != 0 || g != 0 || b != 65535 || colored < 100 {
					t.Fatal(name, "mask leaked or portrait empty", r, g, b, colored)
				}
				if err := gl.GetError(); err != gl.NO_ERROR {
					t.Fatalf("%s: GL error %#x", name, err)
				}
				// A cached snapshot must composite identically after a 3D pass.
				// The old image-nonempty check alone missed final-image defects.
				before, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				a.scene3d.Begin()
				a.scene3d.End()
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				a.sprites.Begin(96, 96)
				a.sprites.DrawMasked(a.hudPortrait(96, 96), mask, 16, 16)
				a.sprites.End()
				if err := render.SavePNG(path, 96, 96); err != nil {
					t.Fatal(err)
				}
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatal(name, "cached portrait changed in final UI composition", err)
				}
				if report := os.Getenv("SHAIYA_PORTRAIT_REPORT"); report != "" && race == 0 && job == 0 && gender == 0 {
					data, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(report, data, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				t.Log(name, "portrait pixels:", colored)
			}
		}
	}
	last := a.wv.portrait.ID
	a.releaseWorld()
	if gl.IsTexture(last) {
		t.Fatal("portrait texture survived world release")
	}
}
