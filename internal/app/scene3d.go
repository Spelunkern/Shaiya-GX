package app

import (
	"log"
	"math"

	"github.com/Spelunkern/shaiya-go-client/internal/data/sdata"
	"github.com/Spelunkern/shaiya-go-client/internal/proto"

	"github.com/Spelunkern/shaiya-go-client/internal/render"
	"github.com/Spelunkern/shaiya-go-client/internal/scene"
)

// Character select camera (0x46a5a0, full-body case; Phoenix
// tools/reference/ep5_character_selection.json).
var (
	selectEye    = render.V3{275.0299987792969, 1.9570000171661377, 180.1959991455078}
	selectTarget = render.V3{271.09698486328125, 1.8660000562667847, 178.51499938964844}
)

// selectWorld returns select_a (Light, map 100) or select_b (Fury, 101).
func (a *App) selectWorld(fury bool) *scene.World {
	stem := "select_a"
	if fury {
		stem = "select_b"
	}
	if w, ok := a.worlds[stem]; ok {
		return w
	}
	w, err := scene.LoadField(a.assets, stem)
	if err != nil {
		log.Printf("scene %s: %v", stem, err)
	}
	a.worlds[stem] = w
	for f := range a.assets.Missing {
		log.Printf("scene %s: missing %s", stem, f)
		delete(a.assets.Missing, f)
	}
	return w
}

func (a *App) drawWorld(w *scene.World, eye, target render.V3, width, height int) {
	if w == nil || a.scene3d == nil {
		return
	}
	s := a.scene3d
	s.View = render.LookAtLH(eye, target, render.V3{0, 1, 0})
	aspect := float32(width) / float32(height)
	s.Proj = render.PerspectiveFovLH(math.Pi/4, aspect, 0.1, 400)
	s.SkyProj = render.PerspectiveFovLH(math.Pi/4, aspect, 10, 100000)
	s.FogColor, s.FogStart, s.FogEnd = w.FogColor, w.FogStart, w.FogEnd
	// Select maps light (Phoenix account_ep5_render, X mirrored back to raw).
	s.LightDir = render.V3{-0.513, -1, 0.858}
	s.Begin()
	w.Sky.Update(float32(a.frameDT))
	w.Sky.Draw(s, eye)
	w.Draw(s)
	if actor := a.selectActor(w); actor != nil {
		actor.Update(float32(a.frameDT))
		actor.Draw(a.skinner, s)
	}
	s.End()
}

// Actor placement in the select maps (0x46b020; Phoenix reference).
var (
	selectActorPos = render.V3{272.1020848632812, 1.7059999704360962, 179.5852963378906}
	selectActorFwd = render.V3{1.301, 0, 0.233}
)

func (a *App) itemModel(typ, id uint8) (uint8, bool) {
	if typ == 0 {
		return 0, false
	}
	if a.items == nil {
		a.items = map[sdata.ItemKey]*sdata.Item{}
		if b, err := a.arc.ReadFile("Item/Item.SData"); err == nil {
			if it, err := sdata.ParseItems(b); err == nil {
				a.items = it
			} else {
				log.Printf("Item.SData: %v", err)
			}
		}
	}
	it := a.items[sdata.ItemKey{Type: typ, TypeID: id}]
	if it == nil {
		return 0, false
	}
	return it.Model, true
}

// appearance maps a 0x0101 slot to the parts of 0x417220 (equipment slots
// 0 helmet, 1 upper, 2 lower, 3 hands, 4 feet).
func (a *App) appearance(c *proto.CharacterSlot) scene.Appearance {
	ap := scene.Appearance{Race: c.Race, Gender: c.Gender, Job: c.Class, Face: c.Face, Hair: c.Hair,
		Height: c.Height, Helmet: -1, ShowHelmet: true}
	m := func(slot int) uint8 { v, _ := a.itemModel(c.EquipType[slot], c.EquipTypeID[slot]); return v }
	if v, ok := a.itemModel(c.EquipType[0], c.EquipTypeID[0]); ok {
		ap.Helmet = int(v)
	}
	ap.Upper, ap.Lower, ap.Hand, ap.Foot = m(1), m(2), m(3), m(4)
	return ap
}

// selectActor returns the posed actor of the selected slot.
func (a *App) selectActor(w *scene.World) *scene.Character {
	sel := a.chars.Selected
	if sel < 0 || a.chars.Chars[sel] == nil || w == nil {
		return nil
	}
	ch := a.chars.Chars[sel]
	if a.actor != nil && a.actorID == ch.ID {
		return a.actor
	}
	ap := a.appearance(ch)
	c, err := scene.BuildCharacter(a.assets, ap, "019_select")
	if err != nil {
		log.Printf("character %s: %v", ch.Name, err)
	}
	for f := range a.assets.Missing {
		log.Printf("character %s: missing %s", ch.Name, f)
		delete(a.assets.Missing, f)
	}
	pos := selectActorPos
	pos[1] = w.Height(pos[0], pos[2])
	sc := ap.HeightScale()
	scale := render.M4{sc, 0, 0, 0, 0, sc, 0, 0, 0, 0, sc, 0, 0, 0, 0, 1}
	// Character meshes face -Z in model space: point that axis along the
	// actor direction (0x46b020 stores it at +0x1c).
	c.World = scale.Mul(render.Basis(pos, selectActorFwd.Scale(-1), render.V3{0, 1, 0}))
	a.actor, a.actorID = c, ch.ID
	return c
}
