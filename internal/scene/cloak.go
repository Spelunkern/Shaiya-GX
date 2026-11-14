package scene

import (
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type cloakGeometry struct {
	record   model.CloakRecord
	vertices []render.Vertex
	indices  []uint32
	mesh     *render.Mesh
	shoulder *skinPart
	texture  *render.Texture
}
type cloakInstance struct {
	geometry    *cloakGeometry
	physics     *clothPhysics
	vertices    []render.Vertex
	initialized bool
	world       render.M4
	elapsed     float32
}

// HasCloak reports successfully loaded native cloth, including Model zero.
func (c *Character) HasCloak() bool { return c != nil && c.cloak != nil }

func cloakPrefix(a Appearance) string {
	family := "f"
	switch a.Race {
	case 0:
		if a.Job >= 4 {
			family = "m"
		}
	case 1, 2:
		family = "r"
		if a.Job >= 4 {
			family = "m"
		}
	case 3:
		if a.Job >= 2 {
			family = "r"
		}
	}
	sex := "m"
	if a.Gender == 1 {
		sex = "w"
	}
	return racePrefix[a.Race%4] + sex + family
}

// Native 0x414580 selects PC by the chest's ReqVg-1 (0..7), then indexes
// the record and texture by the cloak's zero-based Model byte.
func (s *Assets) cloakGeometry(a Appearance) (*cloakGeometry, error) {
	prefix, folder := cloakPrefix(a), raceFolders[a.Race%4]
	file := fmt.Sprintf("cloak/Character/%s/%d_%s.PC", folder, a.CloakFit+1, prefix)
	key := fmt.Sprintf("%s:%d", file, a.CloakModel)
	if g := s.cloaks[key]; g != nil {
		return g, nil
	}
	raw, err := s.Arc.ReadFile(file)
	if err != nil {
		return nil, err
	}
	table, err := model.ParseCloakPC(raw)
	if err != nil {
		return nil, err
	}
	if int(a.CloakModel) >= len(table.Records) {
		return nil, fmt.Errorf("cloak: model %d outside PC", a.CloakModel)
	}
	record := table.Records[a.CloakModel]
	if record.Mode != 0 && record.Mode != 1 {
		return nil, fmt.Errorf("cloak: unsupported native mode %d", record.Mode)
	}
	raw, err = s.Arc.ReadFile(fmt.Sprintf("cloak/Character/%s/3DC/%s", folder, path.Base(table.Bodies[record.Body])))
	if err != nil {
		return nil, err
	}
	mesh, err := model.ParseCloth3DC(raw)
	if err != nil {
		return nil, err
	}
	if len(mesh.Vertices) != record.Columns*record.Rows {
		return nil, fmt.Errorf("cloak: grid/mesh mismatch")
	}
	short := strings.ToUpper(racePrefix[a.Race%4])
	raw, err = s.Arc.ReadFile("cloak/ClothTexture/CLOTH_TEXTHRE_" + short + ".CTL")
	if err != nil {
		return nil, err
	}
	textures, err := model.ParseClothTextures(raw)
	if err != nil {
		return nil, err
	}
	if record.Texture < 0 || record.Texture >= len(textures) {
		return nil, fmt.Errorf("cloak: invalid texture")
	}
	g := &cloakGeometry{record: record, vertices: make([]render.Vertex, len(mesh.Vertices))}
	for i, v := range mesh.Vertices {
		g.vertices[i] = render.Vertex{Pos: render.V3(v.Pos), Normal: render.V3(v.Normal), UV: v.UV}
	}
	for _, f := range mesh.Faces {
		g.indices = append(g.indices, uint32(f[0]), uint32(f[1]), uint32(f[2]))
	}
	g.shoulder, err = s.skinFile(fmt.Sprintf("cloak/Character/%s/3DC/Static/%s", folder, path.Base(table.Shoulders[record.Shoulder])))
	if err != nil {
		return nil, err
	}
	g.texture = s.modelTexture("cloak/ClothTexture/" + short + "/" + path.Base(textures[record.Texture]))
	if g.texture == nil {
		return nil, fmt.Errorf("cloak: missing texture")
	}
	g.mesh = render.NewAnimatedMesh(len(g.vertices), g.indices)
	if s.cloaks == nil {
		s.cloaks = map[string]*cloakGeometry{}
	}
	s.cloaks[key] = g
	return g, nil
}
func (c *Character) attachCloak(a Appearance) {
	if a.CloakType != 24 && a.CloakType != 39 {
		return
	}
	g, err := c.assets.cloakGeometry(a)
	if err != nil {
		c.assets.Missing[err.Error()] = true
		return
	}
	cp := *g.shoulder
	cp.tex, cp.cut = g.texture, true
	c.parts = append(c.parts, &cp)
	rest := make([]render.V3, len(g.vertices))
	for i, v := range g.vertices {
		rest[i] = v.Pos
	}
	p := newClothPhysics(rest, g.record.Columns, g.record.Rows, g.record.Mode)
	for _, pin := range g.record.Pins {
		p.pins[pin.Vertex] = true
	}
	c.cloak = &cloakInstance{geometry: g, physics: p, vertices: append([]render.Vertex(nil), g.vertices...)}
}
func (c *cloakInstance) update(world render.M4, finals []render.M4) {
	p := c.physics
	if !c.initialized || (render.V3{world[12] - c.world[12], world[13] - c.world[13], world[14] - c.world[14]}).Len() > 20 {
		for i, v := range p.rest {
			p.positions[i] = transformPoint(v, world)
			p.velocity[i] = render.V3{}
		}
		c.initialized = true
	}
	c.world = world
	for _, pin := range c.geometry.record.Pins {
		if pin.Bone < len(finals) {
			p.positions[pin.Vertex] = transformPoint(render.V3(pin.Offset), finals[pin.Bone].Mul(world))
			p.velocity[pin.Vertex] = render.V3{}
		}
	}
	// 0x584ba0: ceil(frameSeconds*200) steps of 0.0005 seconds.
	steps := min(50, int(math.Ceil(float64(c.elapsed)*200)))
	c.elapsed = 0
	if c.geometry.record.Mode == 1 {
		steps *= 3
	}
	for range steps {
		p.step(.0005)
	}
	inverse := world.Inverse()
	for i, v := range p.positions {
		c.vertices[i].Pos = transformPoint(v, inverse)
		c.vertices[i].Normal = render.V3{}
	}
	for i := 0; i < len(c.geometry.indices); i += 3 {
		a, b, d := c.geometry.indices[i], c.geometry.indices[i+1], c.geometry.indices[i+2]
		n := c.vertices[b].Pos.Sub(c.vertices[a].Pos).Cross(c.vertices[d].Pos.Sub(c.vertices[a].Pos))
		if n.Dot(c.geometry.vertices[a].Normal) < 0 {
			n = n.Scale(-1)
		}
		for _, index := range []uint32{a, b, d} {
			c.vertices[index].Normal = c.vertices[index].Normal.Add(n)
		}
	}
	for i := range c.vertices {
		c.vertices[i].Normal = c.vertices[i].Normal.Norm()
	}
}
func (c *cloakInstance) draw(s *render.Scene3D, world render.M4, concealed bool) {
	s.Begin()
	c.geometry.mesh.UpdateVertices(c.vertices)
	if concealed {
		s.DrawModelFaded(c.geometry.mesh, c.geometry.texture, world, .5, render.StealthOpacity)
	} else {
		s.DrawModel(c.geometry.mesh, c.geometry.texture, world, .5, false)
	}
}
