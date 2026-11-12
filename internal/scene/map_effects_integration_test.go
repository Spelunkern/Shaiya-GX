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

func TestMapEffectsLoadAndReleaseGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "world effects", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	w, err := loadStreamedField(NewAssets(arc), "1")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if w.mapFX == nil || len(w.mapFX.instances) == 0 {
		t.Fatal("map effects not indexed")
	}
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .1, 1000)
	var target *mapEffect
	for i := range w.mapFX.instances {
		candidate := &w.mapFX.instances[i]
		eye := candidate.position.Add(render.V3{0, 2, -8})
		s.View = render.LookAtLH(eye, candidate.position, render.V3{0, 1, 0})
		w.Update(.2)
		w.DrawMapEffects(s, eye)
		if candidate.instance != nil {
			target = candidate
			break
		}
	}
	if target == nil || target.instance.Time <= 0 {
		t.Fatal("no supported original map effect advanced")
	}
	if len(target.instance.emitters) == 0 || len(target.instance.emitters[0].textures) == 0 {
		t.Fatal("effect has no loaded textures")
	}
	definition := target.instance.emitters[0].definition
	shared := false
	for i := range w.mapFX.library.Effects {
		if definition == &w.mapFX.library.Effects[i] {
			shared = true
			break
		}
	}
	if !shared {
		t.Fatal("world effect did not reuse the parsed library")
	}
	eye := target.position.Add(render.V3{0, 2, -8})
	render.ResetDrawCalls()
	w.Update(.2)
	w.DrawMapEffects(s, eye)
	t.Logf("map effects near sample: %d draw calls", render.DrawCalls())
	previous := -1
	for _, index := range w.mapFX.active {
		if index <= previous || w.mapFX.instances[index].instance == nil {
			t.Fatal("active effects are not in original placement order")
		}
		previous = index
	}
	for i := range w.mapFX.instances {
		present := false
		for _, index := range w.mapFX.active {
			present = present || index == i
		}
		if present != (w.mapFX.instances[i].instance != nil) {
			t.Fatal("active list differs from resident instances")
		}
	}
	draw := func(active bool) []byte {
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.ClearColor(.1, .1, .1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		if active {
			w.RenderMapEffects(s)
		} else {
			frustum := render.ViewFrustum(s.View, s.Proj)
			for i := range w.mapFX.instances {
				placed := &w.mapFX.instances[i]
				if placed.instance != nil && render.ClipSphereVisible(s.ClipPlane, placed.position, placed.radius) && frustum.SphereVisible(placed.position, placed.radius) {
					placed.instance.Draw(s)
				}
			}
		}
		pixels := make([]byte, 64*64*4)
		gl.ReadPixels(0, 0, 64, 64, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	if !bytes.Equal(draw(false), draw(true)) {
		t.Fatal("active effect list changed pixels")
	}
	t.Logf("%d active effects out of %d placements; pixels match full traversal", len(w.mapFX.active), len(w.mapFX.instances))
	// An evicted template must not free a texture still used by a live effect.
	texture := target.instance.emitters[0].textures[0]
	clear(w.mapFX.assets.effectTemplates)
	w.mapFX.trimTextures()
	found := false
	for _, candidate := range w.mapFX.assets.textures {
		if candidate == texture {
			found = true
		}
	}
	if !found {
		t.Fatal("evicted template freed a live effect texture")
	}
	// A reflection camera can retain an effect outside the main camera's
	// residency, and rendering it twice must not advance the simulation.
	w.Update(.2)
	w.PrepareMapEffects(s, target.position.Add(render.V3{10000, 0, 10000}), EffectCamera{Eye: eye, View: s.View, Projection: s.Proj})
	if target.instance == nil {
		t.Fatal("secondary camera did not retain the visible effect")
	}
	if !bytes.Equal(draw(false), draw(true)) {
		t.Fatal("secondary-camera active list changed pixels")
	}
	before := target.instance.Time
	w.RenderMapEffects(s)
	w.RenderMapEffects(s)
	if target.instance.Time != before {
		t.Fatal("repeated camera renders advanced the map effect")
	}
	w.DrawMapEffects(s, target.position.Add(render.V3{10000, 0, 10000}))
	if len(w.mapFX.active) != 0 {
		t.Fatal("active list retained released effects")
	}
	if target.instance != nil {
		t.Fatal("distant effect remained active")
	}
	for _, placement := range w.mapFX.instances {
		if placement.instance != nil {
			t.Fatalf("unexpected live effect after leaving map: sequence=%d radius=%g position=%v", placement.sequence, placement.radius, placement.position)
		}
	}
	if len(w.mapFX.assets.textures) != 0 {
		t.Fatal("unused effect textures were not released")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
	t.Logf("indexed %d original map placements, exercised sequence %d", len(w.mapFX.instances), target.sequence)
}
