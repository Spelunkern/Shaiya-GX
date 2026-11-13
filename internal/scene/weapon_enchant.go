package scene

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type weaponEnchantLayer struct {
	definition effect.SEFFEmitter
	texture    *render.Texture
	worlds     []render.M4
	colors     [][4]float32
}
type weaponEnchantRecipe struct{ layers []weaponEnchantLayer }
type weaponEnchant struct {
	recipe *weaponEnchantRecipe
	seed   uint32
	record int32
}

func (a *Assets) weaponEnchantRecipe(record int32) *weaponEnchantRecipe {
	if !a.weaponSEFFTried {
		a.weaponSEFFTried = true
		a.weaponRecipes = map[int32]*weaponEnchantRecipe{}
		raw, err := a.Arc.ReadFile("Effect/Weapon.Seff")
		if err == nil {
			a.weaponSEFF, err = effect.ParseSEFF(raw)
		}
		if err != nil {
			log.Printf("Weapon.Seff: %v", err)
		}
	}
	if a.weaponSEFF == nil {
		return nil
	}
	if recipe, ok := a.weaponRecipes[record]; ok {
		return recipe
	}
	a.weaponRecipes[record] = nil
	definitions := a.weaponSEFF.Records[record]
	if len(definitions) == 0 {
		return nil
	}
	r := &weaponEnchantRecipe{}
	for _, d := range definitions {
		if !weaponSEFFSupported(d) {
			log.Printf("Weapon.Seff record %d: unsupported emitter", record)
			return nil
		}
		name := path.Base(strings.ReplaceAll(d.Texture, `\`, "/"))
		texture := a.worldTexture("Effect/DDS/" + name)
		if texture == nil {
			log.Printf("Weapon.Seff record %d: missing texture %q", record, name)
			return nil
		}
		r.layers = append(r.layers, weaponEnchantLayer{definition: d, texture: texture})
	}
	if a.weaponQuad == nil {
		a.weaponQuad = render.NewMesh([]render.Vertex{
			{Pos: render.V3{-.5, -.5, 0}, UV: [2]float32{0, 1}},
			{Pos: render.V3{.5, -.5, 0}, UV: [2]float32{1, 1}},
			{Pos: render.V3{.5, .5, 0}, UV: [2]float32{1, 0}},
			{Pos: render.V3{-.5, .5, 0}, UV: [2]float32{0, 0}},
		}, []uint32{0, 1, 2, 0, 2, 3})
	}
	a.weaponRecipes[record] = r
	return r
}

func weaponSEFFSupported(d effect.SEFFEmitter) bool {
	shape := d.Shape == 0 || d.Shape == 1 || d.Shape == 2 || d.Shape == 3 || d.Shape == 6 || d.Shape == 7 || d.Shape == 8
	return shape && d.Blend == 0 && d.Local && d.Rate > 0 && d.Rate <= 1000 && d.LifetimeMS > 0 && d.HalfSize > 0
}

// Fixed hash streams keep births stable across draws and camera passes.
// Positions remain local: every live particle follows the current weapon pose.
func enchantRandom(seed uint32) uint32 {
	seed ^= seed >> 16
	seed *= 0x7feb352d
	seed ^= seed >> 15
	seed *= 0x846ca68b
	return seed ^ (seed >> 16)
}

func weaponParticle(d effect.SEFFEmitter, age, born float64, seed uint32) (render.V3, float32, float32) {
	random := func(n uint32) uint32 {
		seed = enchantRandom(seed + 0x9e3779b9)
		if n == 0 {
			return 0
		}
		return seed % n
	}
	speedSteps := uint32(math.Abs(math.Trunc(float64(d.Speed) * 100)))
	speed := float32(random(speedSteps)) * .01
	if d.Speed < 0 {
		speed = -speed
	}
	direction := render.V3{float32(random(200)) - 100, float32(random(200)) - 100, float32(random(200)) - 100}.Norm()
	pos := render.V3{}
	extent := float32(math.Abs(float64(d.Extent)))
	switch d.Shape {
	case 1:
		pos = direction.Scale(float32(random(uint32(extent*100))) * .002)
	case 2, 3:
		for axis := range 3 {
			factor := float32(.0002)
			if axis == int(d.Shape)-2 {
				factor = .002
			}
			pos[axis] = (float32(random(uint32(extent*100))) - extent*50) * factor
		}
		direction = pos.Norm()
	case 6, 7, 8:
		angle := float32((born + min(age, float64(1000/d.Rate)/1000)) * float64(d.OrbitSpeed))
		sn, cs := float32(math.Sin(float64(angle))), float32(math.Cos(float64(angle)))
		radius := extent * .02
		switch d.Shape {
		case 6:
			pos = render.V3{0, cs * radius, sn * radius}
		case 7:
			pos = render.V3{cs * radius, 0, -sn * radius}
		case 8:
			pos = render.V3{cs * radius, sn * radius, 0}
		}
	}
	pos = pos.Add(render.V3(d.Origin)).Add(direction.Scale(speed * float32(age)))
	pos[1] += d.VerticalSpeed * .1 * float32(age)
	sizeSteps := int(math.Abs(math.Trunc(float64(d.SizeRandomness) * 100)))
	size := 2 * d.HalfSize
	if sizeSteps > 0 {
		size *= 1 + float32(int(random(uint32(sizeSteps*2)))-sizeSteps)*.01
	}
	return pos, size, weaponParticleAlpha(d, age)
}

func weaponParticleAlpha(d effect.SEFFEmitter, age float64) float32 {
	hold := float64(d.LifetimeMS) / 1000
	initial := min(1, max(0, float64(d.Alpha)))
	if age <= hold {
		return float32(min(1, initial+age*3))
	}
	return float32(max(0, min(1, initial+hold*3)-(age-hold)*2))
}

func (c *Character) drawWeaponEnchant(s *render.Scene3D, a *attachment, matrix render.M4) {
	if a.enchant == nil || s.Shadow || s.HideWeaponEffects {
		return
	}
	viewPosition := matrix.Mul(s.View)
	if viewPosition[12]*viewPosition[12]+viewPosition[13]*viewPosition[13]+viewPosition[14]*viewPosition[14] > 50*50 {
		return
	}
	// One instanced draw per authored layer, never one draw per particle.
	for i := range a.enchant.recipe.layers {
		layer := &a.enchant.recipe.layers[i]
		d := layer.definition
		interval := float64(1000/d.Rate) / 1000
		hold := float64(d.LifetimeMS) / 1000
		life := hold + min(1, float64(d.Alpha)+hold*3)/2
		last := int64(math.Floor(c.enchantTime / interval))
		first := max(int64(1), last-min(int64(math.Ceil(life/interval)), 1999))
		layer.worlds, layer.colors = layer.worlds[:0], layer.colors[:0]
		for birth := first; birth <= last; birth++ {
			born := float64(birth) * interval
			age := c.enchantTime - born
			seed := enchantRandom(a.enchant.seed + uint32(i)*977 + uint32(birth)*6151)
			pos, size, alpha := weaponParticle(d, age, born, seed)
			if alpha <= 0 {
				continue
			}
			pos = transformPoint(pos, matrix)
			m := effectBaseBasis(0, s.View, render.Identity())
			for axis := range 3 {
				m[axis] *= size
				m[4+axis] *= size
				m[8+axis] *= size
			}
			m[12], m[13], m[14] = pos[0], pos[1], pos[2]
			layer.worlds = append(layer.worlds, m)
			layer.colors = append(layer.colors, [4]float32{float32(d.Color[0]) / 255, float32(d.Color[1]) / 255, float32(d.Color[2]) / 255, alpha})
		}
		if len(layer.worlds) > 0 {
			s.DrawEffectBatch(c.assets.weaponQuad, layer.texture, layer.worlds, layer.colors, 4, 1, true)
		}
	}
}

func (c *Character) newWeaponEnchant(typ, level uint8) *weaponEnchant {
	record, ok := effect.WeaponEnchantRecord(typ, level)
	if !ok {
		return nil
	}
	recipe := c.assets.weaponEnchantRecipe(record)
	if recipe == nil {
		c.assets.Missing[fmt.Sprintf("weapon enchant %d", record)] = true
		return nil
	}
	return &weaponEnchant{recipe: recipe, seed: rand.Uint32(), record: record}
}

// WeaponEnchantRecords exposes successfully attached catalogue IDs for diagnostics.
func (c *Character) WeaponEnchantRecords() []int32 {
	if c == nil {
		return nil
	}
	var records []int32
	for _, a := range c.attachments {
		if a.enchant != nil {
			records = append(records, a.enchant.record)
		}
	}
	return records
}
