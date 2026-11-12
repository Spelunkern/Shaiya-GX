package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Geometry and its scratch vertices are shared by templates. Sampling and draws
// run sequentially on the GL owner thread; source frames remain immutable.
type effectGeometry struct {
	source   *effect.Mesh3DE
	vertices []render.Vertex
	indices  []uint32
}

func newEffectGeometry(m *effect.Mesh3DE) *effectGeometry {
	m.Trailing = nil // The native loader ignores bytes after the first object.
	g := &effectGeometry{source: m, vertices: make([]render.Vertex, len(m.Vertices)), indices: make([]uint32, len(m.Indices))}
	for i, v := range m.Vertices {
		g.vertices[i] = render.Vertex{Pos: render.V3(v.Position), UV: v.UV}
	}
	for i, v := range m.Indices {
		g.indices[i] = uint32(v)
	}
	return g
}

// 0x557470 uses the default rate code 5 set at 0x55cf20: 30 keys/second.
// Unknown8 is read as the loop flag. Crossing the maximum resets to zero,
// discarding overshoot; a nonlooping clock holds the declared maximum.
func advanceMeshFrame(frame float32, maximum int32, loop bool, dt float32) float32 {
	frame += dt * 30
	if frame > float32(maximum) {
		if loop {
			return 0
		}
		return float32(maximum)
	}
	return frame
}

// 0x5575e0 interpolates complete position/UV snapshots. Its key search uses
// a two-key lookbehind window, including extrapolation for densely spaced keys.
func (g *effectGeometry) verticesAt(frame float32) []render.Vertex {
	fs := g.source.Frames
	if len(fs) == 0 {
		return g.vertices
	}
	index := 0
	for index < len(fs)-1 && frame-2 >= float32(fs[index].Key) {
		index++
	}
	if frame < float32(fs[index].Key) {
		index = max(0, index-1)
	}
	a := &fs[index]
	b := a
	t := float32(0)
	if frame < float32(fs[len(fs)-1].Key) && index+1 < len(fs) {
		b = &fs[index+1]
		if b.Key > a.Key {
			t = (frame - float32(a.Key)) / float32(b.Key-a.Key)
		}
	}
	for i := range g.vertices {
		av, bv := a.Vertices[i], b.Vertices[i]
		for j := range av.Position {
			g.vertices[i].Pos[j] = av.Position[j] + (bv.Position[j]-av.Position[j])*t
		}
		for j := range av.UV {
			g.vertices[i].UV[j] = av.UV[j] + (bv.UV[j]-av.UV[j])*t
		}
	}
	return g.vertices
}
