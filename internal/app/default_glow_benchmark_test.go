//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"runtime"
	"testing"
)

// Compare steady frame submission with original map assets and frozen time.
func BenchmarkDefaultGlowOriginalMapWithout(b *testing.B) { benchmarkDefaultGlow(b, false) }
func BenchmarkDefaultGlowOriginalMapWith(b *testing.B)    { benchmarkDefaultGlow(b, true) }

func benchmarkDefaultGlow(b *testing.B, enabled bool) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		b.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "native lighting benchmark", Width: 640, Height: 360, Hidden: true})
	if err != nil {
		b.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		b.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		b.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		b.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-stream"); err != nil {
		b.Fatal(err)
	}
	a.frameDT = 0
	a.sceneGlowFailed = !enabled
	for i := 0; i < 16 || !a.SceneReady(); i++ {
		if i >= 240 {
			b.Fatal("world not ready during benchmark warmup")
		}
		a.Draw(640, 360)
	}
	if enabled && a.sceneGlowFailed {
		b.Fatal("default glow failed during warmup")
	}
	gl.Finish()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Draw(640, 360)
	}
	gl.Finish()
	b.StopTimer()
	if enabled && a.sceneGlowFailed {
		b.Fatal("default glow failed during benchmark")
	}
	b.ReportMetric(float64(render.DrawCalls()), "draws/frame")
	b.ReportMetric(float64(render.GPUBytes()), "gpu-bytes")
	if code := gl.GetError(); code != gl.NO_ERROR {
		b.Fatalf("benchmark GL error %#x", code)
	}
}
