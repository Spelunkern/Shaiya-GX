// Package scene builds drawable worlds from the original data: field
// terrain with its layer masks and lightmap, dungeon geometry and the WLD
// object instances. Rendering rules follow docs/apariencia.md.
package scene

import (
	"fmt"
	"path"
	"strings"

	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
	img "github.com/Spelunkern/shaiya-go-client/internal/data/image"
	"github.com/Spelunkern/shaiya-go-client/internal/data/world"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
)

// Assets caches GPU textures and models loaded from the archive.
type Assets struct {
	Arc      *archive.Archive
	textures map[string]*render.Texture
	models   map[string]*Model
	Missing  map[string]bool
}

func NewAssets(a *archive.Archive) *Assets {
	return &Assets{Arc: a, textures: map[string]*render.Texture{}, models: map[string]*Model{}, Missing: map[string]bool{}}
}

// DDSName applies the data's convention: models and worlds name their
// textures ".tga" but the files are ".dds".
func DDSName(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	if ext := path.Ext(name); ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	return name + ".dds"
}

// Texture loads dir/name once; a missing texture is remembered and nil.
func (s *Assets) Texture(file string) *render.Texture {
	key := strings.ToLower(file)
	if t, ok := s.textures[key]; ok {
		return t
	}
	var tex *render.Texture
	if b, err := s.Arc.ReadFile(file); err == nil {
		if t, err := img.Decode(b); err == nil {
			tex = render.Upload(t)
		}
	}
	if tex == nil {
		s.Missing[file] = true
	}
	s.textures[key] = tex
	return tex
}

// Part is one textured mesh of a model.
type Part struct {
	Mesh  *render.Mesh
	Tex   *render.Texture
	Alpha bool
}

type Model struct {
	Parts []Part
}

// SMOD loads a static model; textures come from Entity/Texture.
func (s *Assets) SMOD(file string) (*Model, error) {
	key := strings.ToLower(file)
	if m, ok := s.models[key]; ok {
		return m, nil
	}
	b, err := s.Arc.ReadFile(file)
	if err != nil {
		return nil, err
	}
	sm, err := world.ParseSMOD(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	m := &Model{}
	for _, mesh := range sm.Meshes {
		v := make([]render.Vertex, len(mesh.Vertices))
		for i, x := range mesh.Vertices {
			v[i] = render.Vertex{Pos: render.V3(x.Pos), Normal: render.V3(x.Normal), UV: x.UV}
		}
		idx := make([]uint32, 0, len(mesh.Faces)*3)
		for _, f := range mesh.Faces {
			if int(f[0]) < len(v) && int(f[1]) < len(v) && int(f[2]) < len(v) {
				idx = append(idx, uint32(f[0]), uint32(f[1]), uint32(f[2]))
			}
		}
		p := Part{Mesh: render.NewMesh(v, idx)}
		if mesh.Texture != "" {
			p.Tex = s.Texture("Entity/Texture/" + DDSName(path.Base(strings.ReplaceAll(mesh.Texture, `\`, "/"))))
			p.Alpha = p.Tex != nil
		}
		m.Parts = append(m.Parts, p)
	}
	s.models[key] = m
	return m, nil
}
