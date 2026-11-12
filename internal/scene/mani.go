package scene

import (
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type maniRotor struct {
	model *Model
	base  render.M4
	axis  render.V3
	speed float32
}

func (r maniRotor) transform(elapsed float64) render.M4 {
	a := r.axis.Norm()
	if a.Len() == 0 || r.speed == 0 {
		return r.base
	}
	t := math.Mod(elapsed*float64(r.speed)*30, 2*math.Pi)
	c, s := float32(math.Cos(t)), float32(math.Sin(t))
	x, y, z := a[0], a[1], a[2]
	d := 1 - c
	rotation := render.M4{
		c + x*x*d, x*y*d + z*s, x*z*d - y*s, 0,
		y*x*d - z*s, c + y*y*d, y*z*d + x*s, 0,
		z*x*d + y*s, z*y*d - x*s, c + z*z*d, 0,
		0, 0, 0, 1,
	}
	return rotation.Mul(r.base)
}

func readMani(a *Assets, w *world.WLD, id int32, cache map[int32]world.Mani) world.Mani {
	if id < 0 || int(id) >= len(w.ManiAssets) {
		return world.Mani{}
	}
	if mani, ok := cache[id]; ok {
		return mani
	}
	name := path.Base(strings.ReplaceAll(w.ManiAssets[id], `\`, "/"))
	var mani world.Mani
	if data, err := a.Arc.ReadFile("Entity/Mani/" + name); err == nil {
		mani, _ = world.ParseMani(data)
	}
	cache[id] = mani
	return mani
}

func (w *World) addManis(a *Assets) {
	cache := make(map[int32]world.Mani)
	for _, instance := range w.WLD.Manis {
		if instance.Building < 0 || int(instance.Building) >= len(w.WLD.Buildings.Assets) {
			continue
		}
		model, err := a.SMOD("Entity/Building/" + w.WLD.Buildings.Assets[instance.Building])
		if err != nil {
			continue
		}
		base := render.Basis(render.V3(instance.Pos), render.V3(instance.Fwd), render.V3(instance.Up))
		mani := readMani(a, w.WLD, instance.Mani, cache)
		if mani.Rotate && mani.Speed != 0 && render.V3(mani.Axis).Len() > 0 {
			w.rotors = append(w.rotors, maniRotor{model: model, base: base, axis: render.V3(mani.Axis), speed: mani.Speed})
		} else {
			w.Objects = append(w.Objects, Instance{Model: model, World: base, Mani: true})
		}
	}
}
