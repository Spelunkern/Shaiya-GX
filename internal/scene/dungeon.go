package scene

import (
	"fmt"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// FindTexture resolves a texture by base name anywhere in the archive
// (DG textures live under Entity/Texture or the dungeon folders), .tga
// names loading the .dds file.
func (s *Assets) FindTexture(name string) *render.Texture {
	if s.byStem == nil {
		s.byStem = map[string]string{}
		s.Arc.Walk(func(p string, f *archive.File) {
			l := strings.ToLower(p)
			if strings.HasSuffix(l, ".dds") {
				stem := strings.TrimSuffix(path.Base(l), ".dds")
				if _, dup := s.byStem[stem]; !dup || strings.HasPrefix(l, "entity/texture/") {
					s.byStem[stem] = p
				}
			}
		})
	}
	stem := strings.ToLower(strings.TrimSuffix(DDSName(path.Base(strings.ReplaceAll(name, `\`, "/"))), ".dds"))
	if p, ok := s.byStem[stem]; ok {
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
