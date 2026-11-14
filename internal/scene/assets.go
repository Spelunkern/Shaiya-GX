// Package scene builds drawable worlds from the original data: field
// terrain with its layer masks and lightmap, dungeon geometry and the WLD
// object instances. Rendering rules follow docs/apariencia.md.
package scene

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Assets caches GPU textures and models loaded from the archive.
type Assets struct {
	monsterEffectLibrary *effect.Library
	monsterEffectTried   bool
	cloaks               map[string]*cloakGeometry
	Arc                  *archive.Archive
	textures             map[string]*render.Texture
	models               map[string]*Model
	parts                map[string]*skinPart
	clips                map[string]*Clip
	Missing              map[string]bool
	byStem               map[string]string
	monsters             []sdata.Monster
	creatures            []model.Creature
	npcVisuals           []model.Creature
	vehicles             [4][]model.Creature
	itemTables           map[string]*model.ITM
	itemParts            map[string]*Part
	groundModels         [4]*GroundModel
	groundAttempted      [4]bool
	effectTemplates      map[string]*effectTemplate
	effectOrder          []string
	effectQuad           *render.Mesh
	effectMirrorQuad     *render.Mesh
	weaponSEFF           *effect.SEFF
	weaponSEFFTried      bool
	weaponRecipes        map[int32]*weaponEnchantRecipe
	weaponQuad           *render.Mesh
}

func NewAssets(a *archive.Archive) *Assets {
	return &Assets{Arc: a, textures: map[string]*render.Texture{}, models: map[string]*Model{}, parts: map[string]*skinPart{}, clips: map[string]*Clip{}, Missing: map[string]bool{}}
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
func (s *Assets) Texture(file string) *render.Texture { return s.texture(file, false) }

func (s *Assets) worldTexture(file string) *render.Texture { return s.texture(file, true) }

func (s *Assets) texture(file string, world bool) *render.Texture {
	key := strings.ToLower(file)
	if world {
		key = "world:" + key
	}
	if t, ok := s.textures[key]; ok {
		return t
	}
	var tex *render.Texture
	if b, err := s.Arc.ReadFile(file); err == nil {
		if t, err := img.DecodeResource(file, b); err == nil {
			if world {
				tex = render.UploadWorld(t)
			} else {
				tex = render.Upload(t)
			}
		}
	}
	if tex == nil {
		s.Missing[file] = true
	}
	s.textures[key] = tex
	return tex
}

// Model albedo uses WRAP, as Phoenix account_ep5.gdshader and its native
// material adapter. SMOD/3DC UVs intentionally extend beyond the unit square;
// CLAMP stretches the last texel over whole roofs, walls and model surfaces.
// UI atlases, terrain masks and lightmaps retain Texture's clamp addressing.
func (s *Assets) modelTexture(file string) *render.Texture {
	t := s.Texture(file)
	if t != nil {
		t.SetWrap(true)
	}
	return t
}

// Part is one textured mesh of a model.
type Part struct {
	lods     []staticMeshLOD
	Mesh     *render.Mesh
	Tex      *render.Texture
	Alpha    bool
	boundMin render.V3
	boundMax render.V3
	bounded  bool
}

type Model struct {
	occluders    [][3]render.V3
	lodErrors    [2]float32
	lodAvailable [2]bool
	colliders    []world.CollisionMesh
	center       render.V3
	radius       float32
	Parts        []Part
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
	return s.uploadSMOD(key, sm)
}

func (s *Assets) uploadSMOD(key string, sm *world.SMOD, preparedLODs ...[][]staticLODCPU) (*Model, error) {
	m := &Model{colliders: sm.Colliders}
	var lo, hi render.V3
	hasBounds := false
	for meshIndex, mesh := range sm.Meshes {
		v := make([]render.Vertex, len(mesh.Vertices))
		for i, x := range mesh.Vertices {
			point := render.V3(x.Pos)
			if !hasBounds {
				lo, hi, hasBounds = point, point, true
			}
			for axis := range 3 {
				lo[axis] = min(lo[axis], point[axis])
				hi[axis] = max(hi[axis], point[axis])
			}
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
			p.Tex = s.worldTexture("Entity/Texture/" + DDSName(path.Base(strings.ReplaceAll(mesh.Texture, `\`, "/"))))
			p.Alpha = p.Tex != nil
			if p.Tex != nil {
				p.Tex.SetWrap(true)
			}
		}
		if len(preparedLODs) > 0 && meshIndex < len(preparedLODs[0]) {
			p.lods = make([]staticMeshLOD, 2)
			for level, lod := range preparedLODs[0][meshIndex] {
				if len(lod.indices) == 0 {
					continue
				}
				p.lods[level] = staticMeshLOD{mesh: render.NewMesh(lod.vertices, lod.indices), error: lod.error}
				m.lodAvailable[level] = true
				m.lodErrors[level] = max(m.lodErrors[level], lod.error)
			}
		}
		if p.Tex == nil || p.Tex.Opaque {
			for i := 0; i < len(idx); i += 3 {
				tri := [3]render.V3{v[idx[i]].Pos, v[idx[i+1]].Pos, v[idx[i+2]].Pos}
				if triangleArea2(tri) >= 2 {
					m.occluders = append(m.occluders, tri)
				}
			}
		}
		m.Parts = append(m.Parts, p)
	}
	sort.Slice(m.occluders, func(i, j int) bool { return triangleArea2(m.occluders[i]) > triangleArea2(m.occluders[j]) })
	if len(m.occluders) > 32 {
		m.occluders = append([][3]render.V3(nil), m.occluders[:32]...)
	}
	m.center = lo.Add(hi).Scale(.5)
	m.radius = hi.Sub(lo).Len() * .5
	s.models[key] = m
	return m, nil
}

// optionalTexture loads a texture that may legitimately be absent (field
// masks exist only for the layers a section uses); absence is not logged.
func (s *Assets) optionalTexture(file string) *render.Texture {
	if _, ok := s.Arc.Lookup(file); !ok {
		return nil
	}
	return s.worldTexture(file)
}

// LoadMap loads world/<id>.wld as a field or a dungeon.
func LoadMap(a *Assets, id uint16) (*World, error) {
	file := fmt.Sprintf("world/%d.wld", id)
	b, err := a.Arc.ReadFile(file)
	if err != nil {
		return nil, err
	}
	if len(b) >= 3 && string(b[:3]) == "DUN" {
		return LoadDungeon(a, file)
	}
	return loadStreamedField(a, fmt.Sprint(id))
}
