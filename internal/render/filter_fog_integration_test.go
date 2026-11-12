//go:build integration

package render

import (
	"bytes"
	"os"
	"runtime"
	"testing"
	"time"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestNativeTextureFogAndAsyncMetricsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "fog/filter probe", Width: 64, Height: 64, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := Init(true); err != nil {
		t.Fatal(err)
	}
	baseline := GPUBytes()
	compressed := func(blocks int, color uint16) []byte {
		p := make([]byte, blocks*8)
		for i := 0; i < blocks; i++ {
			p[i*8], p[i*8+1] = byte(color), byte(color>>8)
			p[i*8+2], p[i*8+3] = byte(color), byte(color>>8)
		}
		return p
	}
	blue, red := compressed(4, 31), compressed(1, 0xf800)
	tex := Upload(img.Texture{DDS: &img.DDS{W: 8, H: 8, Format: img.FormatDXT1, Levels: [][]byte{blue, red}}})
	var levels int32
	gl.GetTextureParameteriv(tex.ID, gl.TEXTURE_IMMUTABLE_LEVELS, &levels)
	if levels != 2 {
		t.Fatal("native DDS mip allocation changed", levels)
	}
	original := make([]byte, len(blue))
	gl.GetCompressedTextureImage(tex.ID, 0, int32(len(original)), gl.Ptr(&original[0]))
	if !bytes.Equal(original, blue) {
		t.Fatal("world filtering overwrote authored DDS level")
	}
	tex.Delete()
	if GPUBytes() != baseline {
		t.Fatal("texture VRAM estimate did not release storage")
	}
	solid := func(value byte) *Texture {
		return Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{value, value, value, 255}}})
	}
	tile, mask, light := solid(255), solid(255), solid(64)
	defer tile.Delete()
	defer mask.Delete()
	defer light.Delete()
	var filter int32
	gl.GetTextureParameteriv(mask.ID, gl.TEXTURE_MIN_FILTER, &filter)
	if filter != gl.LINEAR {
		t.Fatal("world filtering leaked into UI/mask upload")
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	s.View, s.Proj = Identity(), Identity()
	s.FogColor = V3{.2, .4, .6}
	s.FogStart, s.FogEnd = 0, .5
	meter := NewFrameMeter()
	defer meter.Delete()
	gl.Viewport(0, 0, 64, 64)
	for _, test := range []struct {
		depth float32
		want  [3]byte
	}{{.5, [3]byte{51, 102, 153}}, {.25, [3]byte{58, 83, 109}}} {
		vertices := []Vertex{{Pos: V3{-1, -1, test.depth}}, {Pos: V3{1, -1, test.depth}}, {Pos: V3{1, 1, test.depth}}, {Pos: V3{-1, 1, test.depth}}}
		mesh := NewMesh(vertices, []uint32{0, 1, 2, 0, 2, 3})
		ResetDrawCalls()
		meter.Begin()
		s.Begin()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.DrawTerrainLayer(mesh, mask, tile, light, 1, true)
		meter.End()
		var pixel [4]byte
		gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
		for i, want := range test.want {
			delta := int(pixel[i]) - int(want)
			if delta < -1 || delta > 1 {
				t.Fatal("fog/lightmap order wrong", test, pixel)
			}
		}
		if DrawCalls() != 1 {
			t.Fatal("draw call counter", DrawCalls())
		}
		mesh.Delete()
	}

	gl.Flush()
	deadline := time.Now().Add(2 * time.Second)
	for meter.Milliseconds() == 0 && time.Now().Before(deadline) {
		meter.Begin()
		meter.End()
		gl.Flush()
		time.Sleep(time.Millisecond)
	}
	if meter.Milliseconds() <= 0 {
		t.Fatal("GPU timing did not resolve asynchronously")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("fog/filter/metrics GL error %#x", code)
	}
}
