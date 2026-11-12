//go:build integration

package scene

import (
	"bytes"
	"math"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestAnimatedActorCullingPreservesPixelsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "actor visibility", Width: 320, Height: 320, Hidden: true, Debug: true})
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
	player, err := BuildCharacter(assets, Appearance{Race: 0, Job: 0, Helmet: -1, WeaponType: 1, WeaponModel: 0}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	if len(player.attachments) == 0 {
		t.Fatal("weapon fixture missing")
	}
	other, err := BuildCharacter(assets, Appearance{Race: 1, Gender: 1, Job: 2, Helmet: -1}, "002_run")
	if err != nil {
		t.Fatal(err)
	}
	mob, err := assets.BuildMonster(2009)
	if err != nil {
		t.Fatal(err)
	}
	vehicle, err := assets.BuildVehicle(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	actors := []*Character{player, other, mob.Character, vehicle.Character}
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	skin, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer skin.Delete()
	s.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .1, 120)
	s.Ambient = 1
	totalBefore, totalAfter := 0, 0
	var beforeCPU, afterCPU time.Duration
	for frame := 0; frame < 6; frame++ {
		for _, a := range actors {
			a.Update(.13)
		}
		eye := render.V3{0, 2, -5}
		angle := float64(frame) * math.Pi / 3
		s.View = render.LookAtLH(eye, eye.Add(render.V3{float32(math.Sin(angle)), -.05, float32(math.Cos(angle))}), render.V3{0, 1, 0})
		for _, clip := range [][4]float32{{}, {0, 1, 0, 0}} {
			s.ClipPlane = clip
			draw := func(cull bool) ([]byte, int, time.Duration) {
				gl.Viewport(0, 0, 320, 320)
				gl.DepthMask(true)
				gl.ClearColor(.08, .1, .13, 1)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				s.Begin()
				render.ResetDrawCalls()
				start := time.Now()
				frustum := render.ViewFrustum(s.View, s.Proj)
				for i := 0; i < 48; i++ {
					a := actors[i%len(actors)]
					theta := float64(i) * 2 * math.Pi / 48
					radius := float32(8 + i%3*9)
					y := float32(0)
					if i%7 == 0 {
						y = -4
					}
					a.World = render.Basis(render.V3{radius * float32(math.Sin(theta)), y, radius * float32(math.Cos(theta))}, render.V3{0, 0, 1}, render.V3{0, 1, 0})
					if !cull || a.Visible(frustum, clip) {
						a.Draw(skin, s)
					}
				}
				elapsed := time.Since(start)
				count := render.DrawCalls()
				p := make([]byte, 320*320*4)
				gl.ReadPixels(0, 0, 320, 320, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
				return p, count, elapsed
			}
			before, bc, bt := draw(false)
			after, ac, at := draw(true)
			if !bytes.Equal(before, after) {
				t.Fatalf("frame %d clip %v changed visible pixels", frame, clip)
			}
			if ac >= bc {
				t.Fatalf("no draw reduction at frame %d: %d -> %d", frame, bc, ac)
			}
			totalBefore += bc
			totalAfter += ac
			beforeCPU += bt
			afterCPU += at
		}
	}
	t.Logf("12 mixed actor views: draws %d -> %d, CPU submission %s -> %s, identical pixels", totalBefore, totalAfter, beforeCPU, afterCPU)
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}

func TestActorCullingPreservesShadowsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "offscreen shadow", Width: 256, Height: 256, Hidden: true, Debug: true})
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
	actor, err := BuildCharacter(assets, Appearance{Helmet: -1}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	actor.Update(.2)
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	skin, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer skin.Delete()
	shadow, err := render.NewShadowMap()
	if err != nil {
		t.Fatal(err)
	}
	defer shadow.Delete()
	w := &World{WLD: &world.WLD{MapSize: 128}, side: 65, step: 2, heights: make([]float32, 65*65)}
	ground := render.NewMesh([]render.Vertex{{Pos: render.V3{0, 0, 0}}, {Pos: render.V3{128, 0, 0}}, {Pos: render.V3{128, 0, 128}}, {Pos: render.V3{0, 0, 128}}}, []uint32{0, 1, 2, 0, 2, 3})
	defer ground.Delete()
	s.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .1, 120)
	s.Ambient = 1
	eye := render.V3{64, 8, 53}
	s.View = render.LookAtLH(eye, render.V3{64, 0, 64}, render.V3{0, 1, 0})
	f := render.ViewFrustum(s.View, s.Proj)
	draw := func(cull, withShadow bool) []byte {
		gl.Viewport(0, 0, 256, 256)
		gl.DepthMask(true)
		gl.ClearColor(.3, .3, .3, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		s.DrawSolid(ground, [4]float32{.7, .7, .7, 1})
		if !cull || actor.Visible(f, [4]float32{}) {
			actor.Draw(skin, s)
		}
		if withShadow {
			actor.DrawShadow(shadow, skin, s, w)
		}
		p := make([]byte, 256*256*4)
		gl.ReadPixels(0, 0, 256, 256, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
		return p
	}
	visibleShadows, offscreen := 0, 0
	actor.World = render.Basis(render.V3{64, 0, 64}, render.V3{0, 0, 1}, render.V3{0, 1, 0})
	if !actor.fitShadow().valid {
		t.Fatal("real actor shadow fit failed")
	}
	const iterations = 1000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		actor.shadowFitReady = false
		actor.fitShadow()
	}
	recomputed := time.Since(start)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		actor.fitShadow()
	}
	cached := time.Since(start)
	t.Logf("real actor fit %d times: %s recomputed, %s cached", iterations, recomputed, cached)
	for z := float32(54); z < 78; z += 2 {
		for x := float32(52); x < 76; x += 2 {
			actor.World = render.Basis(render.V3{x, 0, z}, render.V3{0, 0, 1}, render.V3{0, 1, 0})
			if !actor.Visible(f, [4]float32{}) {
				offscreen++
			}
			base := draw(false, false)
			actor.shadowFitReady = false
			actor.shadowReceiverReady = false
			original := draw(false, true)
			if !bytes.Equal(base, original) {
				visibleShadows++
			}
			culled := draw(true, true)
			if !bytes.Equal(original, culled) {
				t.Fatalf("actor culling changed shadow pixels at %v,%v", x, z)
			}
		}
	}
	if visibleShadows == 0 || offscreen == 0 {
		t.Fatalf("insufficient coverage: %d visible shadows, %d offscreen actors", visibleShadows, offscreen)
	}
	t.Logf("144 positions: %d visible shadows, %d offscreen actors, identical pixels", visibleShadows, offscreen)
}
