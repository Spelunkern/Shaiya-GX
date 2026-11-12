//go:build integration

package render

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"runtime"
	"testing"
)

// A surface a millimeter in front must win independently of submission order.
// At map distances conventional D24 quantizes both surfaces to one depth.
func TestMapSurfaceDepthPrecisionGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "depth precision", Width: 64, Height: 64, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = Init(true); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	mesh := NewMesh([]Vertex{{Pos: V3{-100, -100, 0}}, {Pos: V3{100, -100, 0}}, {Pos: V3{0, 100, 0}}}, []uint32{0, 1, 2})
	defer mesh.Delete()
	s.View = Identity()
	s.Proj = PerspectiveFovLH(1, 1, .1, 600)
	target, err := NewWaterTarget(64, 64)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	var pixel [4]byte
	for _, reversed := range []bool{false, true} {
		failures := 0
		s.SetReverseDepth(reversed)
		for sample := 0; sample < 32; sample++ {
			distance := []float32{50, 100, 200, 350}[sample%4] + float32(sample/4)*.07
			target.Capture(V3{}, func() {
				gl.Viewport(0, 0, 64, 64)
				gl.DepthMask(true)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				s.Begin()
				for i, z := range []float32{distance - .001, distance} {
					m := Identity()
					m[14] = z
					tint := [4]float32{1, 0, 0, 1}
					if i == 1 {
						tint = [4]float32{0, 0, 1, 1}
					}
					s.setWorld(m)
					gl.Uniform1i(s.uMode, 5)
					gl.Uniform4fv(s.uTint, 1, &tint[0])
					gl.Uniform1f(s.uRef, -1)
					s.blend(false)
					mesh.Draw()
				}
				gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
			})
			if pixel[0] != 255 || pixel[2] != 0 {
				failures++
			}
		}

		if reversed && failures != 0 {
			t.Errorf("reverse depth lost %d of 32 front surfaces", failures)
		}
		if !reversed && failures == 0 {
			t.Fatal("conventional control did not reproduce depth loss")
		}
		t.Logf("reverse=%v, incorrect front surfaces=%d/32", reversed, failures)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
