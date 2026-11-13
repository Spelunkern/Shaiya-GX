//go:build integration

package scene

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestLoginManiAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "mani", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	w, err := LoadDungeon(NewAssets(arc), "world/login3.wld")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if len(w.rotors) == 0 {
		t.Fatal("login has no animated MAni instances")
	}
	if !w.HasShadowReceivers() {
		t.Fatal("login dungeon did not index DG shadow receivers")
	}
	triangle := w.dgShadow.triangles[0]
	center := triangle.vertices[0].Add(triangle.vertices[1]).Add(triangle.vertices[2]).Scale(1.0 / 3)
	if _, indices := w.shadowReceiver(center[0]-1, center[2]-1, center[0]+1, center[2]+1, center[1]); len(indices) == 0 {
		t.Fatal("real DG triangle not found in shadow receiver index")
	}
	t.Logf("login DG shadow receivers: %d triangles", len(w.dgShadow.triangles))
	if len(w.vanis) == 0 {
		t.Fatal("login has no VANI assets")
	}
	renderer, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	renderer.View, renderer.Proj = render.Identity(), render.Identity()
	renderer.Begin()
	w.vanis[0].draw(renderer, 0)
	if w.vanis[0].frame != 0 {
		t.Fatal("VANI did not draw its initial frame")
	}
	w.vanis[0].draw(renderer, .2)
	if w.vanis[0].frameMilliseconds != 33 || w.vanis[0].frame != 6 {
		t.Fatal("VANI did not advance to the expected frame")
	}
	before := w.rotors[0].transform(w.elapsed)
	w.Update(1)
	if after := w.rotors[0].transform(w.elapsed); before == after {
		t.Fatal("MAni did not advance")
	}
	t.Logf("login animated MAni instances: %d", len(w.rotors))
}

func TestVANIUploadCostGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "VANI upload", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("Entity/VAni/eagle.vani")
	if err != nil {
		t.Fatal(err)
	}
	data, err := world.ParseVANI(raw)
	if err != nil {
		t.Fatal(err)
	}
	m := data.Meshes[0]
	vertices := make([]render.Vertex, len(m.Frames[0]))
	for i, v := range m.Frames[0] {
		vertices[i] = render.Vertex{Pos: render.V3(v.Pos), Normal: render.V3(v.Normal), UV: v.UV}
	}
	indices := make([]uint32, 0, len(m.Faces)*3)
	for _, face := range m.Faces {
		indices = append(indices, uint32(face[0]), uint32(face[1]), uint32(face[2]))
	}
	old := render.NewStreamMesh()
	defer old.Delete()
	updated := render.NewAnimatedMesh(len(vertices), indices)
	defer updated.Delete()
	const iterations = 1000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		old.Stream(vertices, indices)
	}
	oldTime := time.Since(start)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		updated.UpdateVertices(vertices)
	}
	newTime := time.Since(start)
	t.Logf("eagle mesh: %d vertices, %d indices; %d uploads: stream=%s vertex-only=%s; bytes/frame %d -> %d", len(vertices), len(indices), iterations, oldTime, newTime, len(vertices)*40+len(indices)*4, len(vertices)*40)
}

func TestStreamedFieldManiResidencyGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "field mani", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	for mapID := 1; mapID <= 100; mapID++ {
		stem := fmt.Sprint(mapID)
		if _, ok := arc.Lookup("world/" + stem + ".wld"); !ok {
			continue
		}
		w, err := loadStreamedField(NewAssets(arc), stem)
		if err != nil {
			continue
		}
		var object *cellObject
		for i := range w.stream.objects {
			if w.stream.objects[i].rotor != nil {
				object = &w.stream.objects[i]
				break
			}
		}
		if object == nil {
			w.Close()
			continue
		}
		position := render.V3{object.world[12], object.world[13], object.world[14]}
		for frame := 0; frame < 500 && !w.Ready(); frame++ {
			w.UpdateVisible(position, 200)
		}
		if !w.Ready() {
			w.Close()
			t.Fatal("field streaming did not converge")
		}
		if w.stream.assets.models[object.file] == nil {
			w.Close()
			t.Fatal("resident rotor model not loaded")
		}
		before := object.rotor.transform(w.elapsed)
		w.Update(1)
		if object.rotor.transform(w.elapsed) == before {
			w.Close()
			t.Fatal("streamed rotor did not advance")
		}
		far := position.Add(render.V3{10000, 0, 10000})
		w.UpdateVisible(far, 200)
		for frame := 0; frame < 500 && !w.Ready(); frame++ {
			w.UpdateVisible(far, 200)
		}
		if _, resident := w.stream.assets.models[object.file]; resident {
			w.Close()
			t.Fatal("distant rotor model remained resident")
		}
		t.Logf("map %d: MAni %s loaded and animated through streaming", mapID, object.file)
		w.Close()
		return
	}
	t.Fatal("no streamed map with an animated MAni instance found")
}

func TestStreamedFieldVANIResidencyGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "field vani", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	for mapID := 1; mapID <= 100; mapID++ {
		stem := fmt.Sprint(mapID)
		if _, ok := arc.Lookup("world/" + stem + ".wld"); !ok {
			continue
		}
		w, err := loadStreamedField(NewAssets(arc), stem)
		if err != nil {
			continue
		}
		var object *cellObject
		for i := range w.stream.objects {
			if w.stream.objects[i].vani {
				object = &w.stream.objects[i]
				break
			}
		}
		if object == nil {
			w.Close()
			continue
		}
		position := render.V3{object.world[12], object.world[13], object.world[14]}
		for frame := 0; frame < 500 && !w.Ready(); frame++ {
			w.UpdateVisible(position, 200)
		}
		asset := w.stream.vanis[object.file]
		if !w.Ready() || asset == nil {
			w.Close()
			t.Fatal("nearby VANI did not load")
		}
		renderer, err := render.NewScene3D()
		if err != nil {
			w.Close()
			t.Fatal(err)
		}
		renderer.View, renderer.Proj = render.Identity(), render.Identity()
		renderer.Begin()
		asset.worlds = append(asset.worlds, object.world)
		asset.draw(renderer, .2)
		if asset.frame != 2%asset.count {
			w.Close()
			t.Fatal("VANI did not animate")
		}
		renderer.Delete()
		far := position.Add(render.V3{10000, 0, 10000})
		w.UpdateVisible(far, 200)
		for frame := 0; frame < 500 && !w.Ready(); frame++ {
			w.UpdateVisible(far, 200)
		}
		if _, ok := w.stream.vanis[object.file]; ok {
			w.Close()
			t.Fatal("distant VANI remained resident")
		}
		t.Logf("map %d: VANI %s loaded, animated and released", mapID, object.file)
		w.Close()
		return
	}
	t.Fatal("no field with VANI found")
}
