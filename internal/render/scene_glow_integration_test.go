//go:build integration

package render

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"runtime"
	"testing"
)

func TestDefaultGlowWeightsDepthAndUIIsolationGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "default lighting", Width: 128, Height: 128, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = Init(true); err != nil {
		t.Fatal(err)
	}
	baseline := GPUBytes()
	target, err := NewWaterTarget(128, 128)
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewSceneGlow()
	if err != nil {
		target.Delete()
		t.Fatal(err)
	}
	defer func() {
		g.Delete()
		target.Delete()
		if GPUBytes() != baseline {
			t.Errorf("glow leaked VRAM accounting: %d -> %d", baseline, GPUBytes())
		}
	}()
	for _, size := range [][2]int32{{128, 128}, {127, 65}, {128, 128}} {
		for _, tc := range []struct{ input, want byte }{{0, 0}, {64, 75}, {128, 172}, {255, 255}} {
			target.Capture(V3{}, func() {
				gl.Viewport(0, 0, size[0], size[1])
				v := float32(tc.input) / 255
				gl.ClearColor(v, v, v, 1)
				gl.ClearDepth(.375)
				gl.DepthMask(true)
				gl.Disable(gl.SCISSOR_TEST)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				gl.Enable(gl.DEPTH_TEST)
				gl.Enable(gl.BLEND)
				gl.BlendEquation(gl.FUNC_REVERSE_SUBTRACT)
				gl.Enable(gl.CULL_FACE)
				gl.Enable(gl.CLIP_DISTANCE0)
				if err := g.Apply(); err != nil {
					t.Fatal(err)
				}
				var p [4]byte
				gl.ReadPixels(size[0]/2, size[1]/2, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
				if p != [4]byte{tc.want, tc.want, tc.want, 255} {
					t.Fatalf("size=%v input=%d got=%v want=%d", size, tc.input, p, tc.want)
				}
				var depth float32
				gl.ReadPixels(size[0]/2, size[1]/2, 1, 1, gl.DEPTH_COMPONENT, gl.FLOAT, gl.Ptr(&depth))
				if depth != .375 {
					t.Fatal("glow changed scene depth", depth)
				}
				for _, cap := range []uint32{gl.DEPTH_TEST, gl.BLEND, gl.CULL_FACE, gl.CLIP_DISTANCE0} {
					if !gl.IsEnabled(cap) {
						t.Fatalf("lost GL state %#x", cap)
					}
				}
				var equation int32
				gl.GetIntegerv(gl.BLEND_EQUATION_RGB, &equation)
				if equation != gl.FUNC_REVERSE_SUBTRACT {
					t.Fatal("lost blend equation", equation)
				}
				// HUD is drawn afterwards and must retain its original color.
				gl.Enable(gl.SCISSOR_TEST)
				gl.Scissor(0, 0, 2, 2)
				red := [4]float32{1, 0, 0, 1}
				gl.ClearBufferfv(gl.COLOR, 0, &red[0])
				gl.Disable(gl.SCISSOR_TEST)
				gl.ReadPixels(0, 0, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
				if p != [4]byte{255, 0, 0, 255} {
					t.Fatal("glow altered subsequent UI", p)
				}
				gl.Disable(gl.CLIP_DISTANCE0)
				gl.Disable(gl.CULL_FACE)
				gl.Disable(gl.BLEND)
				gl.BlendEquation(gl.FUNC_ADD)
			})
		}
	}
	// A two-by-two source impulse maps to one half-resolution pixel. Check
	// the authored kernel directly, independently of the final blend.
	target.Capture(V3{}, func() {
		gl.Disable(gl.SCISSOR_TEST)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		gl.Enable(gl.SCISSOR_TEST)
		gl.Scissor(64, 64, 2, 2)
		white := [4]float32{1, 1, 1, 1}
		gl.ClearBufferfv(gl.COLOR, 0, &white[0])
		gl.Disable(gl.SCISSOR_TEST)
		if err := g.Apply(); err != nil {
			t.Fatal(err)
		}
	})
	kernel := make([]byte, 64*64*4)
	gl.GetTextureImage(g.textures[1], 0, gl.RGBA, gl.UNSIGNED_BYTE, int32(len(kernel)), gl.Ptr(&kernel[0]))
	for _, sample := range []struct {
		x, y int
		want byte
	}{{32, 32, 35}, {31, 32, 44}, {33, 32, 44}, {32, 31, 44}, {32, 33, 44}, {31, 31, 0}} {
		if got := kernel[(sample.y*64+sample.x)*4]; got != sample.want {
			t.Fatalf("kernel (%d,%d)=%d want%d", sample.x, sample.y, got, sample.want)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("glow GL error %#x", code)
	}
}
