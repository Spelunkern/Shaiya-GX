//go:build integration

package scene

import (
	"math"
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestCharacterShadowOnRealDungeonGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "dungeon shadow", Width: 256, Height: 256, Hidden: true, Debug: true})
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
	w, err := LoadDungeon(assets, "world/3.wld")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	c, err := BuildCharacter(assets, Appearance{Race: 0, Gender: 0, Job: 0, Helmet: -1}, "000_normal")
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
	s.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .1, 200)
	s.Ambient = 1
	tested := 0
	for _, tri := range w.dgShadow.triangles {
		normal := tri.vertices[1].Sub(tri.vertices[0]).Cross(tri.vertices[2].Sub(tri.vertices[0]))
		if normal.Len() < 30 || math.Abs(float64(normal.Norm()[1])) < .95 {
			continue
		}
		pos := tri.vertices[0].Add(tri.vertices[1]).Add(tri.vertices[2]).Scale(1.0 / 3)
		if tested == 0 {
			x0, z0, x1, z1 := pos[0]-3, pos[2]-3, pos[0]+3, pos[2]+3
			vertices, indices := w.dgShadow.query(x0, z0, x1, z1, pos[1])
			seen := make(map[int]bool)
			v, idx := w.dgShadow.queryInto(x0, z0, x1, z1, pos[1], vertices, indices, seen)
			freshV, freshI := w.dgShadow.query(x0, z0, x1, z1, pos[1])
			if !reflect.DeepEqual(v, freshV) || !reflect.DeepEqual(idx, freshI) {
				t.Fatal("reused dungeon receiver changed geometry")
			}
			fresh := testing.AllocsPerRun(100, func() { w.dgShadow.query(x0, z0, x1, z1, pos[1]) })
			reused := testing.AllocsPerRun(100, func() { w.dgShadow.queryInto(x0, z0, x1, z1, pos[1], v, idx, seen) })
			t.Logf("dungeon receiver allocations: fresh %.1f, reused %.1f", fresh, reused)
			if reused >= fresh {
				t.Fatal("dungeon receiver buffers were not reused")
			}
		}
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
			t.Logf("real character projected shadow onto dungeon after %d floor candidates; %d darker pixels", tested, darkened)
			return
		}
		if tested >= 40 {
			break
		}
	}
	t.Fatalf("no visible character shadow on %d real dungeon floor candidates", tested)
}
