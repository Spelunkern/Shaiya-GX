//go:build integration

package scene

import (
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestCharacterShadowOnOriginalVANIGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "VANI shadow", Width: 256, Height: 256, Hidden: true, Debug: true})
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
	asset, err := loadVani(assets, "Entity/VAni/login_flatwater01a.VANI")
	if err != nil {
		t.Fatal(err)
	}
	asset.worlds = []render.M4{render.Identity()}
	w := &World{vanis: []*vaniAsset{asset}}
	w.elapsed = .5
	defer w.Close()
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
	for _, part := range asset.parts {
		for _, face := range part.faces {
			frame := 6 % asset.count
			p0, p1, p2 := render.V3(part.frames[frame][face[0]].Pos), render.V3(part.frames[frame][face[1]].Pos), render.V3(part.frames[frame][face[2]].Pos)
			n := p1.Sub(p0).Cross(p2.Sub(p0))
			if n.Len() < 5 || math.Abs(float64(n.Norm()[1])) < .95 {
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
			darkened := 0
			for i := 0; i < len(before); i += 4 {
				if int(before[i])-int(after[i]) > 8 {
					darkened++
				}
			}
			tested++
			if darkened > 10 {
				if code := gl.GetError(); code != gl.NO_ERROR {
					t.Fatalf("GL error %#x", code)
				}
				t.Logf("VANI shadow: %d darker pixels after %d candidates", darkened, tested)
				return
			}
			if tested >= 50 {
				t.Fatalf("no VANI shadow in %d candidates", tested)
			}
		}
	}
	t.Fatalf("no VANI shadow in %d candidates", tested)
}
