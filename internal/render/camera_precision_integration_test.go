//go:build integration

package render

import (
	"bytes"
	"math"
	"os"
	"runtime"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
)

// Moving an otherwise identical scene to map coordinates must not change its
// rasterization. The camera-relative pose is identical at every sampled time.
func TestCameraRelativeTexturePrecisionGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "camera precision", Width: 128, Height: 128, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := Init(true); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	k, err := NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer k.Delete()
	pixels := make([]byte, 128*128*4)
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			i := (y*128 + x) * 4
			v := byte(0)
			if (x+y)%2 == 0 {
				v = 255
			}
			copy(pixels[i:i+4], []byte{v, v, v, 255})
		}
	}
	tex := Upload(img.Texture{RGBA: &img.RGBA{W: 128, H: 128, Pix: pixels}})
	defer tex.Delete()
	s.Proj = PerspectiveFovLH(math.Pi/4, 1, .1, 600)
	s.Ambient = 1
	static := NewStreamMesh()
	defer static.Delete()
	renderAt := func(origin V3, frame int, mode int) []byte {
		gl.Viewport(0, 0, 128, 128)
		gl.DepthMask(true)
		gl.ClearDepth(1)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.SetCameraView(origin.Add(V3{2, 1, -4}), origin, V3{0, 1, 0})
		s.Begin()
		world := Basis(origin, V3{0, 0, 1}, V3{0, 1, 0})
		offset := float32(frame) * .000013
		vertices := []Vertex{{Pos: V3{-1 + offset, -1, .017}, Normal: V3{0, 1, 0}, UV: [2]float32{0, 0}}, {Pos: V3{1 + offset, -1, .017}, Normal: V3{0, 1, 0}, UV: [2]float32{1, 0}}, {Pos: V3{1 + offset, 1, .017}, Normal: V3{0, 1, 0}, UV: [2]float32{1, 1}}, {Pos: V3{-1 + offset, 1, .017}, Normal: V3{0, 1, 0}, UV: [2]float32{0, 1}}}
		indices := []uint32{0, 1, 2, 0, 2, 3}
		if mode == 1 {
			skin := make([]SkinVertex, len(vertices))
			for i, v := range vertices {
				skin[i] = SkinVertex{Pos: v.Pos, Normal: v.Normal, UV: v.UV, Weight: 1}
			}
			mesh := NewSkinMesh(skin, indices)
			k.Begin(s, 1)
			k.SetPose(world, []M4{Identity()})
			k.Draw(mesh, tex, -1)
		} else {
			static.Stream(vertices, indices)
			if mode == 2 {
				s.SetInstances([]M4{world, world})
				s.DrawInstanced(static, tex, -1)
			} else {
				s.DrawModel(static, tex, world, -1, false)
			}
		}
		out := make([]byte, 128*128*4)
		gl.ReadPixels(0, 0, 128, 128, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&out[0]))
		return out
	}
	for mode := 0; mode < 3; mode++ {
		changed := 0
		for frame := 0; frame < 16; frame++ {
			near := renderAt(V3{}, frame, mode)
			far := renderAt(V3{1703, 80, 1632}, frame, mode)
			if !bytes.Equal(near, far) {
				for i := range near {
					if near[i] != far[i] {
						changed++
					}
				}
			}
		}
		if changed != 0 {
			t.Errorf("map translation changed textured rasterization: mode=%d channels=%d", mode, changed)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("camera precision GL error %#x", code)
	}
}
