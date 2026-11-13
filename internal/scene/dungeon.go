package scene

import (
	"fmt"
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
	out := &World{WLD: w, FogColor: render.V3{w.FogColor[0] / 255, w.FogColor[1] / 255, w.FogColor[2] / 255},
		FogStart: w.FogStart, FogEnd: w.FogEnd}
	if err := out.loadDG(a, w.Layout); err != nil {
		return nil, err
	}
	out.addObjects(a)
	out.addManis(a)
	out.addVanis(a)
	out.loadMapEffects(a)
	return out, nil
}

type dgPart struct {
	mesh     *render.Mesh
	tex      *render.Texture
	lightmap *render.Texture
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
				w.dg = append(w.dg, dgPart{mesh: render.NewMesh(v, idx), tex: tex, lightmap: lm(m.Lightmap)})
			}
		}
	})
	return nil
}
