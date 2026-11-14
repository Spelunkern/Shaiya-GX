//go:build integration

package scene

import (
	"bytes"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"math"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestCreatureDepthAfterOtherPassesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" || os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("needs assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "creature terrain depth", Width: 320, Height: 240, Hidden: true, Debug: true})
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
	if _, err = assets.MonsterDefinition(0); err != nil {
		t.Fatal(err)
	}
	ids := []uint16{2009}
	for i, d := range assets.monsters {
		name := strings.ToLower(d.Name)
		if strings.Contains(name, "windtail") || strings.Contains(name, "wood spirit") {
			ids = append(ids, uint16(i))
			t.Logf("native monster %d %s", i, d.Name)
		}
	}
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
	tex := render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{20, 40, 200, 255}}})
	defer tex.Delete()
	pixels := func() []byte {
		p := make([]byte, 320*240*4)
		gl.ReadPixels(0, 0, 320, 240, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
		return p
	}
	for _, id := range ids {
		actor, e := assets.BuildMonster(id)
		if e != nil {
			t.Fatal(e)
		}
		actor.World = render.Identity()
		actor.SetAction(6)
		actor.Update(.1)
		lo, hi, valid := actor.WorldBounds()
		if !valid {
			t.Fatalf("missing bounds %d", id)
		}
		mid := lo.Add(hi).Scale(.5)
		distance := max(float32(3), hi.Sub(lo).Len()*2)
		s.SetCameraView(mid.Add(render.V3{0, 0, -distance}), mid, render.V3{0, 1, 0})
		s.Proj = render.PerspectiveFovLH(math.Pi/4, 320.0/240, .1, distance*4)
		z := mid[2] - distance*.5
		vertices := []render.Vertex{{Pos: render.V3{mid[0] - distance*2, mid[1] - distance*2, z}}, {Pos: render.V3{mid[0] + distance*2, mid[1] - distance*2, z}}, {Pos: render.V3{mid[0] + distance*2, mid[1] + distance*2, z}}, {Pos: render.V3{mid[0] - distance*2, mid[1] + distance*2, z}}}
		terrain := render.NewMesh(vertices, []uint32{0, 1, 2, 0, 2, 3})
		for _, reverse := range []bool{false, true} {
			s.SetReverseDepth(reverse)
			s.Ambient = 1
			gl.Viewport(0, 0, 320, 240)
			s.Begin()
			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			empty := pixels()
			actor.Draw(k, s)
			if bytes.Equal(empty, pixels()) {
				t.Fatalf("fixture monster %d is not visible", id)
			}
			for frame := 0; frame < 6; frame++ {
				s.Begin()
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				s.DrawModel(terrain, tex, render.Identity(), -1, false)
				before := pixels()
				// State left by UI previews and transparent/world-effect passes must
				// never change the next creature's depth convention or first material.
				gl.Disable(gl.DEPTH_TEST)
				gl.DepthFunc(gl.ALWAYS)
				gl.DepthMask(false)
				gl.Enable(gl.BLEND)
				actor.Update(.033)
				actor.Draw(k, s)
				if !bytes.Equal(before, pixels()) {
					t.Fatalf("monster %d leaks through terrain reverse=%t frame=%d", id, reverse, frame)
				}
			}
		}
		terrain.Delete()
	}
}
