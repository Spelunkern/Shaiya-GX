package scene

import (
	"log"
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func (s *fieldStream) indexObjects() {
	type sphere struct {
		center render.V3
		radius float32
		valid  bool
	}
	bounds := map[string]sphere{}
	add := func(file string, m render.M4, rotor *maniRotor, vani, mani bool) {
		b, known := bounds[file]
		if !known {
			raw, err := s.assets.Arc.ReadPrefix(file, 16)
			if err == nil {
				center, radius, e := world.SMODBounds(raw)
				err = e
				b = sphere{render.V3(center), radius, e == nil}
			}
			if err != nil {
				log.Printf("field bounds %s: %v", file, err)
			}
			bounds[file] = b
		}
		if !b.valid {
			return
		}
		center := render.V3{m[12], m[13], m[14]}
		radius := b.radius
		if rotor != nil {
			// A complete rotation can move an off-center mesh anywhere inside
			// this sphere; keep all those cells resident throughout its cycle.
			radius += b.center.Len()
		} else {
			for axis := 0; axis < 3; axis++ {
				center[axis] += b.center[0]*m[axis] + b.center[1]*m[4+axis] + b.center[2]*m[8+axis]
			}
		}
		scale := sphereScale(m)
		radius *= scale
		id := len(s.objects)
		s.objects = append(s.objects, cellObject{file: file, world: m, rotor: rotor, vani: vani, mani: mani, center: b.center, radius: b.radius, scale: scale})
		x0, x1 := int(math.Floor(float64((center[0]-radius)/s.size()))), int(math.Floor(float64((center[0]+radius)/s.size())))
		z0, z1 := int(math.Floor(float64((center[2]-radius)/s.size()))), int(math.Floor(float64((center[2]+radius)/s.size())))
		if x1-x0 > 64 || z1-z0 > 64 {
			s.global = append(s.global, id)
			return
		}
		for z := z0; z <= z1; z++ {
			for x := x0; x <= x1; x++ {
				s.index[cellKey{x, z}] = append(s.index[cellKey{x, z}], id)
			}
		}
	}
	for _, g := range groupDirs {
		group := g.get(s.world.WLD)
		for _, inst := range group.Instances {
			if inst.Asset < 0 || int(inst.Asset) >= len(group.Assets) {
				continue
			}
			name := group.Assets[inst.Asset]
			if !strings.EqualFold(path.Ext(name), ".smod") {
				continue
			}
			file := strings.ToLower(g.dir + "/" + name)
			m := render.Basis(render.V3(inst.Pos), render.V3(inst.Fwd), render.V3(inst.Up))
			add(file, m, nil, false, false)
		}
	}
	cache := make(map[int32]world.Mani)
	for _, instance := range s.world.WLD.Manis {
		if instance.Building < 0 || int(instance.Building) >= len(s.world.WLD.Buildings.Assets) {
			continue
		}
		file := strings.ToLower("Entity/Building/" + s.world.WLD.Buildings.Assets[instance.Building])
		base := render.Basis(render.V3(instance.Pos), render.V3(instance.Fwd), render.V3(instance.Up))
		mani := readMani(s.assets, s.world.WLD, instance.Mani, cache)
		var rotor *maniRotor
		if mani.Rotate && mani.Speed != 0 && render.V3(mani.Axis).Len() > 0 {
			rotor = &maniRotor{base: base, axis: render.V3(mani.Axis), speed: mani.Speed}
		}
		add(file, base, rotor, false, true)
	}
	for _, group := range []world.Group{s.world.WLD.VaniA, s.world.WLD.VaniB} {
		for _, instance := range group.Instances {
			if instance.Asset < 0 || int(instance.Asset) >= len(group.Assets) {
				continue
			}
			name := path.Base(strings.ReplaceAll(group.Assets[instance.Asset], `\`, "/"))
			file := strings.ToLower("Entity/VAni/" + name)
			base := render.Basis(render.V3(instance.Pos), render.V3(instance.Fwd), render.V3(instance.Up))
			add(file, base, nil, true, false)
		}
	}
}
