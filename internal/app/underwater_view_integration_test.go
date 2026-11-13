//go:build integration

package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestUnderwaterViewTransitionsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "underwater view", Width: 256, Height: 256, Hidden: true, Debug: true})
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
	if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	v := a.wv
	var eye render.V3
	found := false
	for x := float32(20); x < float32(v.world.WLD.MapSize)-20 && !found; x += 10 {
		for z := float32(20); z < float32(v.world.WLD.MapSize)-20; z += 10 {
			if floor, deep := v.world.WaterFloor(x, z); deep && floor < -8 && v.world.Height(x, z+10) < -8 {
				eye = render.V3{x, -2, z}
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("no underwater viewpoint")
	}
	if !v.world.HasWaterNormals() {
		t.Fatal("original water height maps were not loaded")
	}
	t.Logf("water sample position: %v", eye)
	draw := func(y float32) []byte {
		v.eye = render.V3{eye[0], y, eye[2]}
		v.pos = v.eye.Add(render.V3{0, -.5, 5})
		// Observe the surface explicitly from either side. A gameplay camera
		// aimed at a swimming actor may look down at the floor instead.
		pitch := float32(-.5)
		if y < 0 {
			pitch = .5
		}
		v.cameraReady = true
		v.cameraEye, v.cameraTarget = v.eye, v.eye.Add(render.V3{0, pitch, 1})
		gl.Viewport(0, 0, 256, 256)
		gl.ClearDepth(1)
		gl.DepthMask(true)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.drawWorldView(256, 256)
		pixels := make([]byte, 256*256*4)
		gl.ReadPixels(0, 0, 256, 256, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	for i := 0; i < 240 && !a.SceneReady(); i++ {
		draw(-2)
	}
	if !a.SceneReady() {
		t.Fatal("field did not finish streaming")
	}
	submerged := draw(-2)
	if v.waterReflection == nil {
		t.Fatal("visible water from below did not capture the above-water scene")
	}
	v.waterReflection.Delete()
	v.waterReflection = nil
	v.waterFailed = true
	withoutAbove := draw(-2)
	v.waterFailed = false
	if bytes.Equal(submerged, withoutAbove) {
		t.Fatal("above-water capture did not affect the underwater surface")
	}
	if again := draw(-2); !bytes.Equal(submerged, again) {
		t.Fatal("restoring the above-water capture changed a stationary scene")
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "water-under-surface.png"), 256, 256); err != nil {
			t.Fatal(err)
		}
	}
	render.ResetDrawCalls()
	draw(-2)
	t.Logf("underwater draw calls with above-water capture: %d", render.DrawCalls())
	color, ok := v.world.UnderwaterFogAt(v.eye)
	if !ok || a.scene3d.FogColor != color || a.scene3d.FogStart != 5 || a.scene3d.FogEnd != 65 {
		t.Fatal("underwater fog was not applied")
	}
	surface := draw(2)
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "water-reflection-field.png"), 256, 256); err != nil {
			t.Fatal(err)
		}
	}
	if v.waterReflection == nil {
		t.Fatal("visible water did not create a reflection target")
	}
	render.ResetDrawCalls()
	draw(2)
	withReflection := render.DrawCalls()
	view := a.scene3d.View
	mirror := render.Identity()
	mirror[5] = -1
	a.scene3d.View = mirror.Mul(view)
	a.scene3d.ClipPlane = [4]float32{0, 1, 0, 0}
	a.scene3d.Begin()
	render.ResetDrawCalls()
	v.world.Draw(a.scene3d)
	worldCalls := render.DrawCalls()
	a.drawGroundItems()
	a.drawSelf()
	a.drawRemotePlayers()
	a.drawMonsters()
	a.drawNPCs()
	actorCalls := render.DrawCalls() - worldCalls
	a.drawCastEffects()
	a.drawResolvedEffects()
	v.world.RenderMapEffects(a.scene3d)
	t.Logf("reflection components: world=%d actors/items=%d effects=%d", worldCalls, actorCalls, render.DrawCalls()-worldCalls-actorCalls)
	a.scene3d.View, a.scene3d.ClipPlane = view, [4]float32{}
	a.scene3d.End()
	v.waterReflection.Delete()
	v.waterReflection = nil
	v.waterFailed = true
	render.ResetDrawCalls()
	simpleWater := draw(2)
	withoutReflection := render.DrawCalls()
	// Turning off advanced effects must match the basic animated-water path,
	// even when a reflection target was allocated on a previous frame.
	a.gameMenu.Settings.Water = false
	v.waterFailed = false
	render.ResetDrawCalls()
	unchecked := draw(2)
	if !bytes.Equal(simpleWater, unchecked) || render.DrawCalls() != withoutReflection {
		t.Fatalf("unchecked Water Surface Effect changed basic water or kept reflection work: %d vs %d draws", render.DrawCalls(), withoutReflection)
	}
	a.gameMenu.Settings.Water = true
	v.waterFailed = false
	if withReflection <= withoutReflection {
		t.Fatalf("reflection did not draw an additional scene: %d vs %d", withReflection, withoutReflection)
	}
	t.Logf("surface draw calls: %d with reflection, %d without", withReflection, withoutReflection)
	if a.scene3d.FogColor != v.world.FogColor || a.scene3d.FogStart != v.world.FogStart || a.scene3d.FogEnd != v.world.FogEnd {
		t.Fatal("WLD fog was not restored")
	}
	if bytes.Equal(submerged, surface) {
		t.Fatal("surface and submerged views are identical")
	}
	if again := draw(-2); !bytes.Equal(submerged, again) {
		t.Fatal("returning underwater changed a stationary scene")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("underwater GL error %#x", code)
	}
}
