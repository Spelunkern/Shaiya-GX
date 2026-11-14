package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/itemstack"
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
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
	settings := a.clientSettings()
	s.DetailBias = float32(2 - settings.Details)
	s.HideWeaponEffects = !settings.SkillEffects
	nav := &a.chars.Navigation
	if !a.chars.Busy {
		nav.Advance(a.frameDT, 1)
	}
	if nav.Face && a.chars.Selected >= 0 {
		if ch := a.chars.Chars[a.chars.Selected]; ch != nil {
			eye, target = selectionFaceCamera(ch.Race, ch.Height)
		}
	}
	s.SetCameraView(eye, target, render.V3{0, 1, 0})
	aspect := projectionAspect(width, height)
	s.Proj = render.PerspectiveFovLH(math.Pi/4, aspect, 0.1, 400)
	s.SkyProj = render.PerspectiveFovLH(math.Pi/4, aspect, 10, 100000)
	s.FogColor, s.FogStart, s.FogEnd = w.FogColor, w.FogStart, w.FogEnd
	// Select maps light (Phoenix account_ep5_render, X mirrored back to raw).
	s.LightDir = render.V3{-0.513, -1, 0.858}
	s.Begin()
	w.Update(float32(a.frameDT))
	w.Sky.Draw(s, eye)
	w.Draw(s)
	if actor := a.selectActor(w); actor != nil {
		// Selection rotates the actor while the scenery camera stays fixed.
		pos := selectActorPos
		pos[1] = w.Height(pos[0], pos[2])
		scale := a.actorAppearance.HeightScale()
		m := render.Identity()
		m[0], m[5], m[10] = scale, scale, scale
		actor.World = m.Mul(render.Basis(pos, rotatePreview(selectActorFwd.Scale(-1), nav.Angle), render.V3{0, 1, 0}))
		actor.Update(float32(a.frameDT))
		if !actor.ActionActive() {
			actor.SetClip("000_normal")
		}
		actor.Draw(a.skinner, s)
	}
	w.DrawWater(s)
	w.DrawMapEffects(s, eye)
	s.End()
}

// Face pose from 0x46a5a0; family and height adjust the native eye level.
func selectionFaceCamera(race, height uint8) (render.V3, render.V3) {
	y := float32(2.406)
	if height < 5 {
		y += [5]float32{-.1, -.05, 0, .05, .1}[height]
	}
	if race == 3 {
		y += .3
	}
	return render.V3{273.0897, y, 179.83809}, render.V3{272.1887, y, 179.40509}
}

func creationFaceHeight(race, height uint8) float32 {
	y := float32(0)
	if height < 5 {
		y = [5]float32{-.05, -.02, 0, .05, .1}[height]
	}
	if race == 3 {
		y += .2
	}
	return y
}

func rotatePreview(v render.V3, angle float64) render.V3 {
	c, s := float32(math.Cos(angle)), float32(math.Sin(angle))
	return render.V3{v[0]*c - v[2]*s, v[1], v[0]*s + v[2]*c}
}

// Actor placement in the select maps (0x46b020; Phoenix reference).
var (
	selectActorPos = render.V3{272.1020848632812, 1.7059999704360962, 179.5852963378906}
	selectActorFwd = render.V3{1.301, 0, 0.233}
)

func (a *App) itemsItem(typ, id uint8) *sdata.Item {
	if typ == 0 {
		return nil
	}
	if a.items == nil {
		a.items = map[sdata.ItemKey]*sdata.Item{}
		if b, err := a.arc.ReadFile("Item/Item.SData"); err == nil {
			if it, err := sdata.ParseItems(b); err == nil {
				a.items = it
				// GX stackability follows the server definitions, rather than
				// the legacy byte-sized client catalog.
				if raw, err := a.arc.ReadFile("item-stack-limits.csv"); err == nil {
					if limits, err := itemstack.ParseLimits(raw); err == nil {
						for key, item := range a.items {
							if n, ok := limits[uint32(key.Type)*1000+uint32(key.TypeID)]; ok {
								item.Count = n
							}
						}
					} else {
						log.Printf("item-stack-limits.csv: %v", err)
					}
				}

			} else {
				log.Printf("Item.SData: %v", err)
			}
		}
	}
	return a.items[sdata.ItemKey{Type: typ, TypeID: id}]
}

func (a *App) itemModel(typ, id uint8) (uint8, bool) {
	it := a.itemsItem(typ, id)
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
	ap.WeaponType, ap.WeaponModel = c.EquipType[5], m(5)
	ap.ShieldType, ap.ShieldModel = c.EquipType[6], m(6)
	ap.CloakType, ap.CloakModel = c.EquipType[7], m(7)
	if a.state != nil && c.ID == a.state.SelfID && a.state.InventoryReady {
		ap.WeaponEnchant = weaponEnchantFromCraft(a.state.Inventory[[2]uint8{0, 5}].Craft)
		ap.OffhandEnchant = weaponEnchantFromCraft(a.state.Inventory[[2]uint8{0, 6}].Craft)
	}
	if upper := a.itemsItem(c.EquipType[1], c.EquipTypeID[1]); upper != nil {
		fit := uint8(upper.ReqVG) - 1
		if fit < 8 {
			ap.CloakFit = fit
		}
	}
	settings := a.clientSettings()
	ap.ShowHelmet = !settings.HideHelmet
	if settings.HideCloak {
		ap.CloakType = 0
	}
	return ap
}

func weaponEnchantFromCraft(craft string) uint8 {
	if _, ok := parseCraftValues(craft); !ok {
		return 0
	}
	level := (craft[18]-'0')*10 + craft[19] - '0'
	if level >= 50 {
		return 0
	}
	return level
}

// selectActor returns the posed actor of the selected slot.
func (a *App) selectActor(w *scene.World) *scene.Character {
	sel := a.chars.Selected
	if sel < 0 || a.chars.Chars[sel] == nil || w == nil {
		return nil
	}
	ch := a.chars.Chars[sel]
	ap := a.appearance(ch)
	if a.actor != nil && a.actorID == ch.ID && a.actorAppearance == ap {
		return a.actor
	}
	c, err := scene.BuildCharacter(a.assets, ap, "000_normal")
	if err != nil {
		log.Printf("character %s: %v", ch.Name, err)
	}
	c.PlayAction(19, false) // 0x46b020: selection entrance, followed by idle.
	for f := range a.assets.Missing {
		log.Printf("character %s: missing %s", ch.Name, f)
		delete(a.assets.Missing, f)
	}
	a.actor, a.actorID = c, ch.ID
	a.actorAppearance = ap
	return c
}

// loginScene loads the login3 backdrop once.
func (a *App) loginBackdrop() *scene.LoginScene {
	if a.login3 == nil && !a.login3Failed {
		ls, err := scene.LoadLoginScene(a.assets)
		if err != nil {
			log.Printf("login scene: %v", err)
			a.login3Failed = true
			return nil
		}
		for f := range a.assets.Missing {
			log.Printf("login scene: missing %s", f)
			delete(a.assets.Missing, f)
		}
		a.login3 = ls
	}
	return a.login3
}
