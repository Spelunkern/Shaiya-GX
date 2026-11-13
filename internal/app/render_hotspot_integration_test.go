//go:build integration

package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/go-gl/gl/v4.5-core/gl"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

// Reproduce the reported field spot with advancing effect/animation time at
// desktop resolution. A zero-time, low-resolution capture is not a frame budget.
func TestProfileMapOneHotspotGL(t *testing.T) {
	if os.Getenv("GX_PROFILE_HOTSPOT") != "1" {
		t.Skip("explicit desktop scene profile")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1920, Height: 1040, Hidden: true, VSync: false})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
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
	if os.Getenv("SHAIYA_TEST_DISPOSABLE") == "1" {
		st, conn, _, _ := testworld.Enter(t)
		a.state, a.world, a.screen = st, conn, screenWorld
		a.enterWorld()
	} else if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	a.gameMenu.Settings.VisualDistance = 0
	v := a.wv
	if a.world == nil {
		v.place(render.V3{996.458, 12.76, 787.152})
	}
	v.facing = render.V3{0, 0, 1}
	v.resetCamera()
	a.frameDT = 0
	for i := 0; !a.SceneReady(); i++ {
		if i > 500 {
			t.Fatal("warmup")
		}
		a.Draw(1920, 1040)
	}
	a.frameDT = 1.0 / 15
	v.sendT = 1e9
	for range 120 {
		a.clock += a.frameDT
		a.pumpWorld()
		a.updateWorld(&platform.Input{}, a.frameDT)
		a.Draw(1920, 1040)
	}
	dir := filepath.Join("..", "..", "work", "render-hotspot")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	profile, err := os.Create(filepath.Join(dir, "baseline.cpu"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.StartCPUProfile(profile); err != nil {
		t.Fatal(err)
	}
	captures := make([]*image.RGBA, 0, 4)
	for angle := 0; angle < 4; angle++ {
		yaw := float64(angle) * math.Pi / 2
		v.facing = render.V3{float32(math.Sin(yaw)), 0, float32(math.Cos(yaw))}
		v.resetCamera()
		for range 10 {
			a.Draw(1920, 1040)
		}
		gl.Finish()
		start := time.Now()
		calls := 0
		for range 30 {
			a.clock += a.frameDT
			a.pumpWorld()
			a.updateWorld(&platform.Input{}, a.frameDT)
			a.Draw(1920, 1040)
			calls += render.DrawCalls()
		}
		gl.Finish()
		t.Logf("heading%d total%.2fms/frame draws%d VRAM%.1fMiB CPU%.2fms GPUtimestamp%.2fms entities%d", angle, float64(time.Since(start))/float64(30*time.Millisecond), calls/30, float64(render.GPUBytes())/1048576, a.performance.cpuMS, a.performance.gpuMS, len(a.state.Entities))
		im := image.NewRGBA(image.Rect(0, 0, 1920, 1040))
		gl.ReadPixels(0, 0, 1920, 1040, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&im.Pix[0]))
		captures = append(captures, im)
	}
	pprof.StopCPUProfile()
	profile.Close()
	for i, im := range captures {
		// Flip the GL rows only after profiling has stopped.
		out := image.NewRGBA(im.Bounds())
		for y := 0; y < 1040; y++ {
			copy(out.Pix[y*out.Stride:(y+1)*out.Stride], im.Pix[(1039-y)*im.Stride:(1040-y)*im.Stride])
		}
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("heading-%d.png", i)))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, out)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}

}
