package scene

import (
	"fmt"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

// Bone indices: 0x417220/0x552400/0x410920, verified by the native
// ep5_weapon_bindings.json probes. IT2 archetypes follow Parsec ItmCharacterCode.
var itemBones = map[string][4]int{
	"humf": {24, 14, 10, 0}, "huwf": {27, 17, 13, 1}, "humm": {24, 14, 10, 2}, "huwm": {27, 17, 13, 3},
	"elmr": {22, 14, 10, 4}, "elwr": {26, 18, 14, 5}, "elmm": {26, 17, 10, 6}, "elwm": {26, 18, 14, 7},
	"vimm": {22, 14, 10, 8}, "viwm": {28, 21, 16, 9}, "vimr": {24, 14, 10, 10}, "viwr": {30, 20, 16, 11},
	"demf": {24, 14, 10, 12}, "dewf": {28, 18, 14, 13}, "demr": {24, 14, 10, 14}, "dewr": {28, 18, 14, 15},
}

type attachment struct {
	enchant *weaponEnchant
	part    *Part
	bone    int
	extra   render.M4
	alpha   int32
}

func itemBaseType(typ uint8) uint8 {
	switch typ {
	case 34, 69, 75, 84, 90:
		return 19
	}
	if family := effect.WeaponFamily(typ); family != 0 {
		return family
	}
	return typ
}

func (s *Assets) itemPart(typ, index uint8) (*Part, *model.ITM, *model.ItemModel, error) {
	file := fmt.Sprintf("Item/%02d.ITM", typ)
	if _, ok := s.Arc.Lookup(file); !ok {
		file = fmt.Sprintf("Item/%02d.ITM", itemBaseType(typ))
	}
	if s.itemTables == nil {
		s.itemTables = map[string]*model.ITM{}
		s.itemParts = map[string]*Part{}
	}
	table := s.itemTables[file]
	if table == nil {
		b, err := s.Arc.ReadFile(file)
		if err != nil {
			return nil, nil, nil, err
		}
		table, err = model.ParseITM(b)
		if err != nil {
			return nil, nil, nil, err
		}
		s.itemTables[file] = table
	}
	if int(index) >= len(table.Records) {
		return nil, nil, nil, fmt.Errorf("%s model %d out of range", file, index)
	}
	rec := &table.Records[index]
	if rec.Mesh < 0 || int(rec.Mesh) >= len(table.Meshes) || rec.Texture < 0 || int(rec.Texture) >= len(table.Textures) {
		return nil, nil, nil, fmt.Errorf("%s model %d has no valid mesh/texture", file, index)
	}
	mesh := path.Base(strings.ReplaceAll(table.Meshes[rec.Mesh], `\`, "/"))
	texture := path.Base(strings.ReplaceAll(table.Textures[rec.Texture], `\`, "/"))
	key := fmt.Sprintf("%s:%s:%d", mesh, texture, rec.Alpha)
	if p := s.itemParts[key]; p != nil {
		return p, table, rec, nil
	}
	b, err := s.Arc.ReadFile("Item/3DO/" + mesh)
	if err != nil {
		return nil, nil, nil, err
	}
	obj, err := world.Parse3DO(b)
	if err != nil {
		return nil, nil, nil, err
	}
	vertices := make([]render.Vertex, len(obj.Vertices))
	for i, v := range obj.Vertices {
		vertices[i] = render.Vertex{Pos: render.V3(v.Pos), Normal: render.V3(v.Normal), UV: v.UV}
	}
	indices := make([]uint32, 0, len(obj.Faces)*3)
	for _, f := range obj.Faces {
		if int(f[0]) < len(vertices) && int(f[1]) < len(vertices) && int(f[2]) < len(vertices) {
			indices = append(indices, uint32(f[0]), uint32(f[1]), uint32(f[2]))
		}
	}
	p := &Part{Mesh: render.NewMesh(vertices, indices), Tex: s.modelTexture("Item/DDS/" + DDSName(texture)), Alpha: rec.Alpha == 1}
	if len(vertices) > 0 {
		p.boundMin, p.boundMax, p.bounded = vertices[0].Pos, vertices[0].Pos, true
		for _, vertex := range vertices[1:] {
			for axis := range 3 {
				p.boundMin[axis] = min(p.boundMin[axis], vertex.Pos[axis])
				p.boundMax[axis] = max(p.boundMax[axis], vertex.Pos[axis])
			}
		}
	}
	s.itemParts[key] = p
	return p, table, rec, nil
}

func (c *Character) attachItem(typ, index uint8, shield bool, enchant uint8) {
	if typ == 0 {
		return
	}
	part, table, record, err := c.assets.itemPart(typ, index)
	if err != nil {
		c.assets.Missing[err.Error()] = true
		return
	}
	bones, ok := itemBones[c.prefix]
	if !ok {
		return
	}
	base := itemBaseType(typ)
	bone := bones[0]
	if base == 11 || base == 13 {
		bone = bones[1]
	}
	if shield {
		bone = bones[2]
	}
	extra := render.Identity()
	reversed := func() render.M4 { m := render.Identity(); m[0], m[5], m[12] = -1, -1, 0.04; return m }
	if base == 9 {
		extra = reversed()
	}
	count := 1
	if base == 5 || base == 15 {
		count = 2
	}
	for i := 0; i < count; i++ {
		if i == 1 {
			bone = bones[1]
			extra = reversed()
		}
		if table.Signature == "IT2" {
			binding := record.Bindings[bones[3]][i]
			bone = int(binding.Bone)
			extra = render.QuatMatrix(binding.Rotation, binding.Position)
		}
		c.attachments = append(c.attachments, attachment{part: part, bone: bone, extra: extra, alpha: record.Alpha, enchant: c.newWeaponEnchant(typ, enchant)})
	}
}

func (c *Character) drawAttachments(s *render.Scene3D, finals []render.M4) {
	if len(c.attachments) == 0 {
		return
	}
	s.Begin()
	for _, a := range c.attachments {
		if a.bone < 0 || a.bone >= len(finals) {
			continue
		}
		matrix := a.extra.Mul(finals[a.bone]).Mul(c.World)
		ref := float32(-1)
		if a.alpha == 0 {
			ref = 0.5
		}
		s.DrawModel(a.part.Mesh, a.part.Tex, matrix, ref, a.part.Alpha)
	}
	for i := range c.attachments {
		a := &c.attachments[i]
		if a.enchant != nil && a.bone >= 0 && a.bone < len(finals) {
			c.drawWeaponEnchant(s, a, a.extra.Mul(finals[a.bone]).Mul(c.World))
		}
	}
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}
