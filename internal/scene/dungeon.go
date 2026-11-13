package scene

import (
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// textureFileIndex resolves native names without GL calls or Assets mutations.
// Live field loaders can build this index on a CPU worker.
func textureFileIndex(arc *archive.Archive) map[string]string {
	index := map[string]string{}
	arc.Walk(func(p string, f *archive.File) {
		l := strings.ToLower(p)
		if strings.HasSuffix(l, ".dds") {
			stem := strings.TrimSuffix(path.Base(l), ".dds")
			old, dup := index[stem]
			// Loose directories have a different traversal order than SAH.
			// Preserve the original canonical dungeon lightmaps over copies
			// nested below world/<map>/dungeon, and keep entity priority.
			if !dup || strings.HasPrefix(l, "entity/texture/") ||
				strings.HasPrefix(l, "world/dungeon/") && !strings.HasPrefix(strings.ToLower(old), "entity/texture/") {
				index[stem] = p
			}
		}
	})
	return index
}
func (s *Assets) findTextureFile(name string) (string, bool) {
	if s.byStem == nil {
		s.byStem = textureFileIndex(s.Arc)
	}
	stem := strings.ToLower(strings.TrimSuffix(DDSName(path.Base(strings.ReplaceAll(name, `\`, "/"))), ".dds"))
	if p, ok := s.byStem[stem]; ok {
		return p, true
	}
	return "", false
}
func (s *Assets) FindTexture(name string) *render.Texture {
	if p, ok := s.findTextureFile(name); ok {
		return s.worldTexture(p)
	}
	s.Missing[name] = true
	return nil
}

// LoadDungeon loads a DUN world: world/dungeon/<layout>.dg geometry (with
// _L<i> lightmaps when present), MAni-placed buildings and the WLD groups.
func LoadDungeon(a *Assets, wldFile string) (*World, error) {
	b, err := a.Arc.ReadFile(wldFile)
	if err != nil {
		return nil, err
	}
	w, err := world.ParseWLD(b)
	if err != nil {
		return nil, err
	}
	if !w.Dungeon {
		return nil, fmt.Errorf("%s is a field", wldFile)
	}
	// DUN has no world fog, even when its WLD contains white field defaults.
	out := &World{WLD: w}
	if err := out.loadDG(a, w.Layout); err != nil {
		return nil, err
	}
	out.addObjects(a)
	out.addManis(a)
	out.addVanis(a)
	out.loadMapEffects(a)
	out.buildDungeonOccluders()
	return out, nil
}

type dgPart struct {
	minimum, maximum render.V3
	occluders        [][3]render.V3
	mesh             *render.Mesh
	tex              *render.Texture
	lightmap         *render.Texture
}

func (w *World) loadDG(a *Assets, layout string) error {
	file := "world/dungeon/" + layout
	b, err := a.Arc.ReadFile(file)
	if err != nil {
		return err
	}
	d, err := world.ParseDG(b)
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	w.dgShadow = buildDungeonShadowIndex(d.Root)
	d.Root.Walk(func(n *world.DGNode) {
		if len(n.Collider.Faces) > 0 {
			w.dgColliders = append(w.dgColliders, n.Collider)
		}
	})
	stem := strings.TrimSuffix(layout, path.Ext(layout))
	lightmaps := map[int32]*render.Texture{}
	lm := func(i int32) *render.Texture {
		if i < 0 || i >= d.Lightmaps {
			return nil
		}
		if t, ok := lightmaps[i]; ok {
			return t
		}
		f := fmt.Sprintf("world/dungeon/%s/%s_L%d.dds", stem, stem, i)
		var t *render.Texture
		if _, ok := a.Arc.Lookup(f); ok {
			t = a.worldTexture(f)
		}
		lightmaps[i] = t
		return t
	}
	type key struct {
		tex, lm *render.Texture
		x, y, z int
	}
	type batch struct {
		vertices []render.Vertex
		indices  []uint32
		tex, lm  *render.Texture
	}
	var lastKey key
	hasKey := false
	var batches []batch
	d.Root.Walk(func(n *world.DGNode) {
		for _, g := range n.Groups {
			var tex *render.Texture
			if g.Texture >= 0 && int(g.Texture) < len(d.Textures) {
				tex = a.FindTexture(d.Textures[g.Texture])
				if tex != nil {
					// DG material UVs tile beyond [0,1]; confirmed against the
					// original Login.wld floor. Lightmap UVs remain clamped.
					tex.SetWrap(true)
				}
			}
			for _, m := range g.Meshes {
				v := make([]render.Vertex, len(m.Vertices))
				for i, x := range m.Vertices {
					v[i] = render.Vertex{Pos: render.V3(x.Pos), Normal: render.V3(x.Normal), UV: x.UV, UV2: x.UV2}
				}
				idx := make([]uint32, 0, len(m.Faces)*3)
				for _, f := range m.Faces {
					if int(f[0]) < len(v) && int(f[1]) < len(v) && int(f[2]) < len(v) {
						idx = append(idx, uint32(f[0]), uint32(f[1]), uint32(f[2]))
					}
				}
				if len(idx) == 0 {
					continue
				}
				// Keep small spatial bounds instead of merging an entire dungeon
				// into one material mesh that cannot be culled behind walls.
				center := render.V3{}
				for _, vertex := range v {
					center = center.Add(vertex.Pos)
				}
				center = center.Scale(1 / float32(len(v)))
				lightmap := lm(m.Lightmap)
				k := key{tex, lightmap, int(math.Floor(float64(center[0] / 32))), int(math.Floor(float64(center[1] / 32))), int(math.Floor(float64(center[2] / 32)))}
				bi := len(batches) - 1
				if !hasKey || k != lastKey {
					bi = len(batches)
					batches = append(batches, batch{tex: tex, lm: lightmap})
					lastKey = k
					hasKey = true
				}
				b := &batches[bi]
				offset := uint32(len(b.vertices))
				b.vertices = append(b.vertices, v...)
				for _, index := range idx {
					b.indices = append(b.indices, index+offset)
				}
			}
		}
	})
	for _, b := range batches {
		p := dgPart{mesh: render.NewMesh(b.vertices, b.indices), tex: b.tex, lightmap: b.lm, minimum: b.vertices[0].Pos, maximum: b.vertices[0].Pos}
		for _, v := range b.vertices {
			for axis := range 3 {
				p.minimum[axis] = min(p.minimum[axis], v.Pos[axis])
				p.maximum[axis] = max(p.maximum[axis], v.Pos[axis])
			}
		}
		if b.tex == nil || b.tex.Opaque {
			for i := 0; i < len(b.indices); i += 3 {
				tri := [3]render.V3{b.vertices[b.indices[i]].Pos, b.vertices[b.indices[i+1]].Pos, b.vertices[b.indices[i+2]].Pos}
				if tri[1].Sub(tri[0]).Cross(tri[2].Sub(tri[0])).Len() >= 2 {
					p.occluders = append(p.occluders, tri)
				}
			}
		}
		w.dg = append(w.dg, p)
	}
	return nil
}
