//go:build integration

package scene

import (
	"bytes"
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestCharacterShadowOnOriginalSMODGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "object shadow", Width: 256, Height: 256, Hidden: true, Debug: true})
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
	assets := NewAssets(arc)
	model, err := assets.SMOD("Entity/Building/guild_house01.SMOD")
	if err != nil {
		t.Fatal(err)
	}
	w := &World{Objects: []Instance{{Model: model, World: render.Identity()}}}
	c, err := BuildCharacter(assets, Appearance{Helmet: -1}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	c.Update(.1)
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	k, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer k.Delete()
	target, err := render.NewShadowMap()
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	s.Proj, s.Ambient = render.PerspectiveFovLH(math.Pi/4, 1, .1, 200), 1
	tested := 0
	for _, part := range model.Parts {
		for i := 0; i+2 < len(part.shadowIndices); i += 3 {
			p0, p1, p2 := part.shadowVertices[part.shadowIndices[i]].Pos, part.shadowVertices[part.shadowIndices[i+1]].Pos, part.shadowVertices[part.shadowIndices[i+2]].Pos
			normal := p1.Sub(p0).Cross(p2.Sub(p0))
			if normal.Len() < 5 || math.Abs(float64(normal.Norm()[1])) < .95 {
				continue
			}
			pos := p0.Add(p1).Add(p2).Scale(1.0 / 3)
			c.World = render.Basis(pos, render.V3{0, 0, 1}, render.V3{0, 1, 0})
			eye := pos.Add(render.V3{0, 7, -10})
			s.View = render.LookAtLH(eye, pos.Add(render.V3{0, 1, 0}), render.V3{0, 1, 0})
			draw := func(shadow bool) []byte {
				gl.Viewport(0, 0, 256, 256)
				gl.DepthMask(true)
				gl.ClearColor(.3, .3, .3, 1)
				gl.ClearDepth(1)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				s.Begin()
				w.Draw(s)
				c.Draw(k, s)
				if shadow {
					c.DrawShadow(target, k, s, w)
				}
				pixels := make([]byte, 256*256*4)
				gl.ReadPixels(0, 0, 256, 256, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
				return pixels
			}
			before, after := draw(false), draw(true)
			radii := make([]float32, len(model.Parts))
			for j := range model.Parts {
				radii[j] = model.Parts[j].shadowRadius
				model.Parts[j].shadowRadius = float32(math.Inf(1))
			}
			unculled := draw(true)
			for j := range model.Parts {
				model.Parts[j].shadowRadius = radii[j]
			}
			if !bytes.Equal(after, unculled) {
				t.Fatal("part bounds changed visible shadow pixels")
			}
			darkened := 0
			for j := 0; j < len(before); j += 4 {
				if int(before[j])-int(after[j]) > 8 {
					darkened++
				}
			}
			tested++
			if darkened > 10 {
				if code := gl.GetError(); code != gl.NO_ERROR {
					t.Fatalf("GL error %#x", code)
				}
				t.Logf("original building shadow: %d darker pixels after %d candidates", darkened, tested)
				return
			}
			if tested >= 40 {
				t.Fatalf("no visible SMOD shadow in %d candidates", tested)
			}
		}
	}
	t.Fatalf("no visible SMOD shadow in %d candidates", tested)
}
