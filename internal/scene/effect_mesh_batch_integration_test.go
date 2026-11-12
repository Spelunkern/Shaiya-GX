//go:build integration

package scene

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestMeshEffectBatchPreservesFramesAndOrderGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "3DE batch", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View, s.Proj = render.Identity(), render.Identity()
	texture := render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 255}}})
	defer texture.Delete()
	points := [][3]float32{{-1, -1, 0}, {1, -1, 0}, {0, 1, 0}}
	mesh := &effect.Mesh3DE{MaxKeyframe: 10, Indices: []uint16{0, 1, 2}}
	for _, p := range points {
		mesh.Vertices = append(mesh.Vertices, effect.MeshVertex{Position: p})
	}
	for _, key := range []int32{0, 10} {
		frame := effect.MeshFrame{Key: key}
		for _, p := range points {
			p[0] += float32(key) / 20
			frame.Vertices = append(frame.Vertices, effect.FrameVertex{Position: p})
		}
		mesh.Frames = append(mesh.Frames, frame)
	}
	geometry := newEffectGeometry(mesh)
	definition := &effect.Emitter{BaseAxis: 3, DstBlend: 4, SrcBlend: 5, Colors: []effect.ColorFrame{{RGBA: [4]float32{1, .5, .2, .5}}}}
	particles := []effectParticle{{frame: 0, pos: render.V3{-.2, 0, .5}, life: 1}, {frame: 0, pos: render.V3{0, 0, .5}, life: 1}, {frame: 5, pos: render.V3{.2, 0, .5}, life: 1}, {frame: 5, pos: render.V3{0, .2, .5}, life: 1}, {frame: 0, pos: render.V3{0, -.2, .5}, life: 1}}
	x := &Effect{emitters: []effectEmitter{{definition: definition, geometry: geometry, textures: []*render.Texture{texture}, particles: particles}}, basis: render.Identity()}
	draw := func(batched bool) ([]byte, int) {
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.ClearColor(.1, .1, .1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		render.ResetDrawCalls()
		if batched {
			x.Draw(s)
		} else {
			s.Begin()
			for _, particle := range particles {
				m := render.Identity()
				m[0], m[5], m[10] = .3, .3, .3
				m[12], m[13], m[14] = particle.pos[0], particle.pos[1], particle.pos[2]
				s.DrawEffectMesh(geometry.verticesAt(particle.frame), geometry.indices, texture, m, [4]float32{1, 127.0 / 255, 51.0 / 255, 127.0 / 255}, 4, 5)
			}
			s.End()
		}
		calls := render.DrawCalls()
		pixels := make([]byte, 64*64*4)
		gl.ReadPixels(0, 0, 64, 64, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels, calls
	}
	before, n := draw(false)
	after, m := draw(true)
	if !bytes.Equal(before, after) {
		t.Fatal("animated mesh batch changed pixels or draw order")
	}
	if n != 5 || m != 3 {
		t.Fatalf("calls = %d -> %d, expected 5 -> 3", n, m)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
