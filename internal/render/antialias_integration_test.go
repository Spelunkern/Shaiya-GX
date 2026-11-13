//go:build integration

package render

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"runtime"
	"testing"
)

func TestAntialiasWaterResolveGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, e := platform.Open(platform.Config{Width: 128, Height: 128, Hidden: true})
	if e != nil {
		t.Fatal(e)
	}
	defer win.Close()
	if e = Init(false); e != nil {
		t.Fatal(e)
	}
	scene, e := NewScene3D()
	if e != nil {
		t.Fatal(e)
	}
	defer scene.Delete()
	target, e := NewWaterTarget(128, 128)
	if e != nil {
		t.Fatal(e)
	}
	defer target.Delete()
	for _, samples := range []int{0, 2, 4} {
		target.Capture(V3{}, func() {
			var ms *MSAATarget
			if samples > 0 {
				ms, e = NewMSAATarget(128, 128, samples)
				if e != nil {
					t.Fatal(e)
				}
				defer ms.Delete()
				ms.Begin()
			}
			scene.CaptureWaterBackground()
			if ms != nil {
				ms.Resolve()
			}
		})
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("samples=%d GL error %#x", samples, code)
		}
	}
}

func TestTemporalAAHistoryDepthAndStateGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, e := platform.Open(platform.Config{Width: 128, Height: 128, Hidden: true})
	if e != nil {
		t.Fatal(e)
	}
	defer win.Close()
	if e = Init(false); e != nil {
		t.Fatal(e)
	}
	baseline := GPUBytes()
	frame, e := NewWaterTarget(128, 128)
	if e != nil {
		t.Fatal(e)
	}
	taa, e := NewTemporalAA()
	if e != nil {
		t.Fatal(e)
	}
	shader, e := Program(`#version 450 core
 uniform vec2 offset;uniform vec2 jitter;out vec2 uv;
 void main(){vec2 p[3]=vec2[3](vec2(-.8,-.8),vec2(.8,-.8),vec2(.8,.8));uv=p[gl_VertexID];gl_Position=vec4(uv+offset+jitter,.4,1);}`, `#version 450 core
 in vec2 uv;out vec4 frag;void main(){frag=vec4(.2,.8,.4,1);}`)
	if e != nil {
		t.Fatal(e)
	}
	var vao uint32
	gl.CreateVertexArrays(1, &vao)
	gl.ClipControl(gl.LOWER_LEFT, gl.ZERO_TO_ONE)
	for i := 0; i < 12; i++ {
		jitter := taa.NextJitter(128, 128)
		frame.Capture(V3{}, func() {
			gl.UseProgram(shader)
			gl.BindVertexArray(vao)
			gl.Enable(gl.DEPTH_TEST)
			gl.DepthFunc(gl.ALWAYS)
			gl.DepthMask(true)
			gl.Uniform2f(gl.GetUniformLocation(shader, gl.Str("jitter\x00")), jitter[0], jitter[1])
			if i >= 8 {
				gl.Uniform2f(gl.GetUniformLocation(shader, gl.Str("offset\x00")), .7, 0)
			}
			gl.DrawArrays(gl.TRIANGLES, 0, 3)
			// Deliberately keep a different read framebuffer to check restoration.
			gl.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)
			if e = taa.Apply(Identity(), Identity(), jitter); e != nil {
				t.Fatal(e)
			}
			var read, draw, program, depth int32
			gl.GetIntegerv(gl.READ_FRAMEBUFFER_BINDING, &read)
			gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &draw)
			gl.GetIntegerv(gl.CURRENT_PROGRAM, &program)
			gl.GetIntegerv(gl.DEPTH_FUNC, &depth)
			if read != 0 || uint32(draw) != frame.fbo || uint32(program) != shader || depth != gl.ALWAYS {
				t.Fatal("TAA leaked GL state")
			}
			if i == 11 {
				gl.BindFramebuffer(gl.READ_FRAMEBUFFER, frame.fbo)
				pixel := [4]byte{}
				gl.ReadPixels(20, 20, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
				if pixel[0] > 2 || pixel[1] > 2 || pixel[2] > 2 {
					t.Fatal("moving object left history trail", pixel)
				}
			}
		})
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("frame%d GL error %#x", i, code)
		}
	}
	if !taa.valid {
		t.Fatal("history never became valid")
	}
	taa.Reset()
	if taa.valid {
		t.Fatal("history reset ignored")
	}
	taa.Delete()
	frame.Delete()
	gl.DeleteVertexArrays(1, &vao)
	gl.DeleteProgram(shader)
	if GPUBytes() != baseline {
		t.Fatal("temporal resources leaked GPU memory", GPUBytes(), baseline)
	}
}
