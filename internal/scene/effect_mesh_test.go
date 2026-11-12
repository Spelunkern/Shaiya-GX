package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"os"
	"testing"
)

func TestMeshClockUsesNativeLoopAndMaximum(t *testing.T) {
	if got := advanceMeshFrame(0, 23, false, .5); got != 15 {
		t.Fatalf("default rate=%g", got)
	}
	if got := advanceMeshFrame(22, 23, false, .1); got != 23 {
		t.Fatalf("held key=%g", got)
	}
	if got := advanceMeshFrame(22, 23, true, .1); got != 0 {
		t.Fatalf("loop must discard overshoot, got %g", got)
	}
	if got := advanceMeshFrame(22, 23, true, 1.0/30); got != 23 {
		t.Fatalf("exact maximum wrapped early: %g", got)
	}
}
func TestMeshSnapshotsInterpolatePositionAndUVWithoutChangingSource(t *testing.T) {
	m := &effect.Mesh3DE{Vertices: []effect.MeshVertex{{}}, MaxKeyframe: 10, Frames: []effect.MeshFrame{
		{Key: 0, Vertices: []effect.FrameVertex{{Position: [3]float32{2, 0, 0}, UV: [2]float32{0, 1}}}},
		{Key: 10, Vertices: []effect.FrameVertex{{Position: [3]float32{12, 4, 0}, UV: [2]float32{1, 0}}}},
	}}
	g := newEffectGeometry(m)
	v := g.verticesAt(5)[0]
	if v.Pos[0] != 7 || v.Pos[1] != 2 || v.UV != [2]float32{.5, .5} {
		t.Fatalf("snapshot=%+v", v)
	}
	g.verticesAt(10)
	v = g.verticesAt(0)[0]
	if v.Pos[0] != 2 || v.UV != [2]float32{0, 1} || m.Frames[0].Vertices[0].Position[0] != 2 {
		t.Fatal("shared scratch contaminated the next particle or its source")
	}
	// The declared maximum can precede the last key (real cylinder01: 23/24).
	if got := g.verticesAt(9)[0].Pos[0]; got != 11 {
		t.Fatalf("last interval=%g", got)
	}
}
func TestMeshSamplingPreservesNativeDenseKeyWindow(t *testing.T) {
	m := &effect.Mesh3DE{Vertices: []effect.MeshVertex{{}}, Frames: []effect.MeshFrame{
		{Key: 0, Vertices: []effect.FrameVertex{{}}},
		{Key: 1, Vertices: []effect.FrameVertex{{Position: [3]float32{10, 0, 0}}}},
		{Key: 2, Vertices: []effect.FrameVertex{{}}},
	}}
	// 0x5575e0 searches using frame-2, so 1.5 still uses the 0->1 pair.
	if got := newEffectGeometry(m).verticesAt(1.5)[0].Pos[0]; got != 15 {
		t.Fatalf("native window changed: %g", got)
	}
}

func TestRealAnimatedEffectMesh(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	raw, err := a.ReadFile("Effect/3DE/vertical021.3DE")
	if err != nil {
		t.Fatal(err)
	}
	m, err := effect.Parse3DE(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Frames) != 16 || m.MaxKeyframe != 30 {
		t.Fatal("unexpected animated fixture")
	}
	g := newEffectGeometry(m)
	start := append([]render.Vertex(nil), g.verticesAt(0)...)
	mid := g.verticesAt(15)
	changed := 0
	for i := range start {
		if start[i].Pos != mid[i].Pos || start[i].UV != mid[i].UV {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("authored vertex/UV animation stayed frozen")
	}
	t.Logf("%d/%d vertices change at key 15", changed, len(start))
}
