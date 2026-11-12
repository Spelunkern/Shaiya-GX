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

func TestWaterOcclusionPreservesPixelsGL(t *testing.T) {
	for _, stem := range []string{"1", "19"} {
		t.Run(stem, func(t *testing.T) { testWaterOcclusionMap(t, stem) })
	}
}

func testWaterOcclusionMap(t *testing.T, stem string) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "water occlusion", Width: 256, Height: 256, Hidden: true, Debug: true})
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
	w, err := loadStreamedField(NewAssets(arc), stem)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if w.water == nil || !w.water.indexed {
		t.Fatal("map water unavailable")
	}
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	target, err := render.NewWaterTarget(128, 128)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	far := max(float32(400), w.FogEnd)
	s.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .1, far)
	s.SkyProj = render.PerspectiveFovLH(math.Pi/4, 1, 10, 100000)
	s.FogColor, s.FogStart, s.FogEnd = w.FogColor, w.FogStart, w.FogEnd
	tangent := float32(math.Tan(math.Pi / 8))
	radius := far * float32(math.Sqrt(float64(1+2*tangent*tangent)))
	var dryEye render.V3
	var dryView render.M4
	found := false
	cellSize := w.stream.size()
	size := float32(w.WLD.MapSize)
	for z := float32(128); z < size-128 && !found; z += 128 {
		for x := float32(128); x < size-128 && !found; x += 128 {
			h := w.Height(x, z)
			if h < 2 {
				continue
			}
			eye := render.V3{x, h + 3, z}
			for turn := 0; turn < 8; turn++ {
				ang := float64(turn) * math.Pi / 4
				view := render.LookAtLH(eye, eye.Add(render.V3{float32(math.Sin(ang)), -1, float32(math.Cos(ang))}), render.V3{0, 1, 0})
				f := render.ViewFrustum(view, s.Proj)
				if !f.BoxVisible(render.V3{}, render.V3{size, 0, size}) {
					continue
				}
				wet := false
				for _, c := range w.water.wetCells {
					px, pz := float32(c.x)*cellSize, float32(c.z)*cellSize
					if f.BoxVisible(render.V3{px - .01, -.01, pz - .01}, render.V3{min(px+cellSize, size) + .01, .01, min(pz+cellSize, size) + .01}) {
						wet = true
						break
					}
				}
				if !wet {
					dryEye, dryView, found = eye, view, true
					break
				}
			}
		}
	}
	if !found {
		t.Fatal("no dry visible-plane viewpoint found")
	}
	draw := func(eye render.V3, view render.M4, optimized bool) ([]byte, int) {
		t.Helper()
		w.UpdateVisible(eye, radius)
		for i := 0; i < 2000 && !w.Ready(); i++ {
			w.UpdateVisible(eye, radius)
		}
		if !w.Ready() {
			t.Fatal("terrain stream not ready")
		}
		w.water.indexed = optimized
		defer func() { w.water.indexed = true }()
		gl.Viewport(0, 0, 256, 256)
		gl.DepthMask(true)
		gl.ClearColor(.13, .17, .2, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.View = view
		s.ClipPlane = [4]float32{}
		s.WaterReflection = nil
		s.Begin()
		render.ResetDrawCalls()
		if w.WaterVisible(view, s.Proj) {
			mirror := render.Identity()
			mirror[5] = -1
			rv := mirror.Mul(view)
			refEye := eye
			refEye[1] = -refEye[1]
			target.Capture(s.FogColor, func() {
				s.View = rv
				s.ClipPlane = [4]float32{0, 1, 0, 0}
				s.Begin()
				if w.Sky != nil {
					w.Sky.Draw(s, refEye)
				}
				w.Draw(s)
			})
			s.View = view
			s.ClipPlane = [4]float32{}
			s.Begin()
			s.WaterReflection = target
		}
		if w.Sky != nil {
			w.Sky.Draw(s, eye)
		}
		w.Draw(s)
		if s.WaterReflection != nil && w.HasWaterNormals() {
			s.CaptureWaterBackground()
		}
		w.DrawWater(s)
		s.WaterReflection = nil
		s.End()
		pixels := make([]byte, 256*256*4)
		gl.ReadPixels(0, 0, 256, 256, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels, render.DrawCalls()
	}
	old, oldCalls := draw(dryEye, dryView, false)
	optimized, newCalls := draw(dryEye, dryView, true)
	if !bytes.Equal(old, optimized) {
		t.Fatal("dry terrain water culling changed visible pixels")
	}
	if newCalls >= oldCalls {
		t.Fatalf("no calls saved: %d vs %d", oldCalls, newCalls)
	}
	t.Logf("dry eye=%v draws %d -> %d, pixels identical; wet cells=%d", dryEye, oldCalls, newCalls, len(w.water.wetCells))
	wetEye := render.V3{600, 2, 620}
	if stem != "1" {
		foundWet := false
		for z := float32(128); z < size-128 && !foundWet; z += 16 {
			for x := float32(128); x < size-128; x += 16 {
				if w.Height(x, z) < -2 {
					wetEye, foundWet = render.V3{x, 2, z}, true
					break
				}
			}
		}
		if !foundWet {
			t.Skip("no interior water sample in this map")
		}
	}
	wetView := render.LookAtLH(wetEye, wetEye.Add(render.V3{0, 0, 5}), render.V3{0, 1, 0})
	old, oldCalls = draw(wetEye, wetView, false)
	optimized, newCalls = draw(wetEye, wetView, true)
	if !bytes.Equal(old, optimized) || oldCalls != newCalls {
		t.Fatalf("visible water changed: draws %d/%d equal=%v", oldCalls, newCalls, bytes.Equal(old, optimized))
	}
	t.Logf("water eye=%v draws %d unchanged", wetEye, newCalls)
	for _, point := range []render.V3{dryEye, wetEye} {
		for turn := 0; turn < 8; turn++ {
			angle := float64(turn) * math.Pi / 4
			view := render.LookAtLH(point, point.Add(render.V3{float32(math.Sin(angle)), -.8, float32(math.Cos(angle))}), render.V3{0, 1, 0})
			before, _ := draw(point, view, false)
			after, _ := draw(point, view, true)
			if !bytes.Equal(before, after) {
				t.Fatalf("water culling changed pixels at %v turn %d", point, turn)
			}
		}
	}
	t.Log("16 additional dry/wet directions preserved all pixels")
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
