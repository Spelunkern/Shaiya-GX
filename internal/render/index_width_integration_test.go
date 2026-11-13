//go:build integration

package render

import (
	"bytes"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"runtime"
	"testing"
)

func TestCompactIndicesDirectInstancedAndBoundaryGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := Init(false); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View, s.Proj, s.Ambient = Identity(), Identity(), 1
	vertices := []Vertex{{Pos: V3{-.6, -.6, .5}}, {Pos: V3{.6, -.6, .5}}, {Pos: V3{0, .6, .5}}}
	idx := []uint32{0, 1, 2}
	wide := NewStreamMesh()
	defer wide.Delete()
	wide.Stream(vertices, idx)
	narrow := NewMesh(vertices, idx)
	defer narrow.Delete()
	if narrow.indexType != gl.UNSIGNED_SHORT || narrow.bytes != uint64(3*vertexSize+6) {
		t.Fatal("compact storage not used")
	}
	draw := func(m *Mesh, instanced bool) []byte {
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		if instanced {
			s.SetInstances([]M4{Identity(), Identity()})
			s.DrawInstanced(m, nil, -1)
		} else {
			s.DrawModel(m, nil, Identity(), -1, false)
		}
		p := make([]byte, 64*64*4)
		gl.ReadPixels(0, 0, 64, 64, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
		return p
	}
	for _, instanced := range []bool{false, true} {
		if !bytes.Equal(draw(wide, instanced), draw(narrow, instanced)) {
			t.Fatal("index width changed pixels", instanced)
		}
	}
	for _, v := range []uint32{65535, 65536} {
		var buffer uint32
		gl.CreateBuffers(1, &buffer)
		kind, size := uploadMeshIndices(buffer, []uint32{v})
		if v == 65535 && (kind != gl.UNSIGNED_SHORT || size != 2) || v == 65536 && (kind != gl.UNSIGNED_INT || size != 4) {
			t.Fatal("boundary overflow", v, kind, size)
		}
		gl.DeleteBuffers(1, &buffer)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
