//go:build integration

package scene

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestRealDispelOpaqueSceneGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 128, Height: 128, Hidden: true})
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
	assets := NewAssets(arc)
	raw, err := arc.ReadFile("Effect/DA_target_dispel.EFT")
	if err != nil {
		t.Fatal(err)
	}
	lib, err := effect.ParseEFT(raw)
	if err != nil {
		t.Fatal(err)
	}
	destinationAlpha := false
	for _, e := range lib.Effects {
		p := e.Parameters()
		destinationAlpha = destinationAlpha || p.SourceBlend == 6 || p.SourceBlend == 7 || p.DestinationBlend == 6 || p.DestinationBlend == 7
	}
	if !destinationAlpha {
		t.Fatal("fixture does not use destination alpha")
	}
	// The third variant references emitter 10, the destination-alpha layer.
	x, err := LoadEffect(assets, "Effect/DA_target_dispel.EFT", 2)
	if err != nil {
		t.Fatal(err)
	}
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View = render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0})
	s.Proj = render.PerspectiveFovLH(1, 1, .1, 100)
	target, err := render.NewWaterTarget(128, 128)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	changedRGB, changedAlpha := 0, 0
	for frame := 0; frame < 60; frame++ {
		x.Update(1.0 / 60)

		draw := func(protected bool) []byte {
			var pixels []byte
			target.Capture(render.V3{.35, .55, .75}, func() {
				if !protected {
					gl.ColorMask(true, true, true, true)
					// Earlier opaque world materials can carry unused low alpha.
					gl.ClearColor(.35, .55, .75, .25)
					gl.Clear(gl.COLOR_BUFFER_BIT)
				} // reproduce old framebuffer behaviour
				s.Begin()
				x.Draw(s)
				s.End()
				pixels = make([]byte, 128*128*4)
				gl.ReadPixels(0, 0, 128, 128, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
				if frame == 20 {
					if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
						name := "dispel-opaque.png"
						if !protected {
							name = "dispel-before.png"
						}
						if err := render.SavePNG(filepath.Join(dir, name), 128, 128); err != nil {
							t.Fatal(err)
						}
					}
				}
			})
			return pixels
		}
		before, after := draw(false), draw(true)
		for i := 0; i < len(after); i += 4 {
			if after[i+3] != 255 {
				t.Fatal("Dispel modified scene alpha", frame, after[i:i+4])
			}
			if before[i+3] != 255 {
				changedAlpha++
			}
			if before[i] != after[i] || before[i+1] != after[i+1] || before[i+2] != after[i+2] {
				changedRGB++
			}
		}
	}
	if changedRGB == 0 || changedAlpha == 0 {
		t.Fatalf("real effect did not reproduce alpha-dependent artifact: rgb=%d alpha=%d", changedRGB, changedAlpha)
	}
	t.Logf("60 real Dispel poses: %d RGB samples corrected; %d alpha samples previously non-opaque", changedRGB, changedAlpha)
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
