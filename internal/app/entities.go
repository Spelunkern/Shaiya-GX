package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type remoteActor struct {
	entity        *game.Entity
	position      remotePosition
	gestureSerial uint64
	moveSerial    uint64
	moveMode      uint8
	moveX, moveZ  float32
	walkUntil     float64
	dead          bool
	shape         proto.UserShape
	character     *scene.Character
	mount         *scene.Vehicle
	mountShape    uint8
	mountLoaded   bool
	facing        render.V3
}

// The native 0x0501 low bits encode walk/run, not a reliable stop flag.
// Infer slow locomotion from successive positions and expire it if updates stop.
func (r *remoteActor) observeMovement(e *game.Entity, now float64) (walking, moving bool) {
	if r.moveSerial != e.MoveSerial {
		dx, dz := e.X-r.moveX, e.Z-r.moveZ
		if e.Motion&0x3f == 0 && r.moveMode == 0 && dx*dx+dz*dz > 0.000001 {
			r.walkUntil = now + .7
			cancelGesture(r.character)
		} else {
			r.walkUntil = 0
		}
		r.moveSerial, r.moveMode, r.moveX, r.moveZ = e.MoveSerial, e.Motion&0x3f, e.X, e.Z
	}
	walking = e.Motion&0x3f == 0 && r.walkUntil > now
	return walking, e.Motion&0x3f != 0 || walking
}

func remoteLocomotion(w *scene.World, e *game.Entity) string {
	if e.Sitting {
		return "012_sit"
	}
	moving := e.Motion&0x3f != 0
	if w.SwimmingAt(e.X, e.Y, e.Z) {
		if moving {
			return "007_swim"
		}
		return "006_swnormal"
	}
	if moving {
		switch e.Motion & 0x3f {
		case 2:
			return "003_bstep"
		case 3:
			return "004_lstep"
		case 4:
			if e.Motion&proto.MotionFlag80 == 0 {
				return "005_rstep"
			}
		}
		return "002_run"
	}
	return "000_normal"
}

// prepareRemotePlayers advances animation and resolves mounts once per frame.
// Drawing can then be repeated by camera-dependent passes without advancing time.
func (a *App) prepareRemotePlayers() {
	v := a.wv
	if v.actors == nil {
		v.actors = map[uint32]*remoteActor{}
	}
	for id := range v.actors {
		if a.state.Entities[id] == nil {
			delete(v.actors, id)
		}
	}
	for id, e := range a.state.Entities {
		if e.Kind != game.KindUser || e.Shape == nil {
			continue
		}
		sh := *e.Shape
		cached := v.actors[id]
		if cached == nil || cached.shape != sh || cached.entity != e {
			var gestureSerial uint64
			var position remotePosition
			moveSerial, moveMode, moveX, moveZ, walkUntil := e.MoveSerial, e.Motion&0x3f, e.X, e.Z, float64(0)
			if cached != nil && cached.entity == e {
				position = cached.position
				gestureSerial = cached.gestureSerial
				moveSerial, moveMode, moveX, moveZ, walkUntil = cached.moveSerial, cached.moveMode, cached.moveX, cached.moveZ, cached.walkUntil
			}
			slot := proto.CharacterSlot{Race: sh.Race, Gender: sh.Gender, Class: sh.Class, Face: sh.Face, Hair: sh.Hair, Height: sh.Height}
			for i, it := range sh.Equipment {
				slot.EquipType[i], slot.EquipTypeID[i] = it[0], it[1]
			}
			appearance := a.appearance(&slot)
			appearance.WeaponEnchant, appearance.OffhandEnchant = sh.Equipment[5][2], sh.Equipment[6][2]
			actor, err := scene.BuildCharacter(a.assets, appearance, "000_normal")
			if err != nil {
				log.Printf("player %d model: %v", id, err)
			}
			cached = &remoteActor{entity: e, shape: sh, character: actor, position: position, gestureSerial: gestureSerial, moveSerial: moveSerial, moveMode: moveMode, moveX: moveX, moveZ: moveZ, walkUntil: walkUntil}
			v.actors[id] = cached
		}
		c := cached.character
		if c == nil {
			continue
		}
		walking, moving := cached.observeMovement(e, a.clock)
		if e.Dead && !cached.dead {
			c.PlayAction(9, true)
		}
		if !e.Dead && cached.dead {
			c.ReleaseAction()
		}
		cached.dead = e.Dead
		if cached.gestureSerial != e.GestureSerial {
			cached.gestureSerial = e.GestureSerial
			if action, ok := proto.LocomotionAction(e.Gesture); ok && !e.Dead && !e.Mounted {
				c.PlayAction(action, false)
			}
			if action, ok := proto.GestureAction(e.Gesture); ok && !e.Dead && !e.Mounted && !moving {
				c.PlayAction(action, false)
			}
		}
		clip := remoteLocomotion(v.world, e)
		if walking && !e.Sitting {
			if v.world.SwimmingAt(e.X, e.Y, e.Z) {
				clip = "007_swim"
			} else {
				clip = "001_walk"
			}
		}
		c.Mounted = e.Mounted
		if e.Mounted {
			c.SetMountedClip(moving, scene.VehicleAlternate(e.MountShape))
		} else {
			c.SetClip(clip)
		}
		c.Update(float32(a.frameDT))
		angle := float64(e.Dir) / 65536 * 2 * math.Pi
		fwd := render.V3{float32(math.Sin(angle)), 0, float32(math.Cos(angle))}
		if c.ActionActive() && cached.facing.Len() > 0 {
			fwd = cached.facing
		} else {
			cached.facing = render.V3{}
		}
		scale := render.Identity()
		scale[5] = (scene.Appearance{Height: sh.Height}).HeightScale()
		pos := cached.position.update(e, a.clock)
		if e.Mounted {
			if !cached.mountLoaded || cached.mountShape != e.MountShape {
				cached.mountShape, cached.mountLoaded = e.MountShape, true
				var err error
				cached.mount, err = a.assets.BuildVehicle(sh.Race, e.MountShape)
				if err != nil {
					log.Printf("player %d mount: %v", id, err)
				}
			}
			if cached.mount != nil {
				cached.mount.SetMoving(moving)
				cached.mount.Update(float32(a.frameDT))
				cached.mount.World = render.Basis(pos, fwd.Scale(-1), render.V3{0, 1, 0})
				pos = cached.mount.Seat()
			}
		} else {
			cached.mount, cached.mountLoaded = nil, false
		}
		c.World = scale.Mul(render.Basis(pos, fwd.Scale(-1), render.V3{0, 1, 0}))
	}
}

func (a *App) drawRemotePlayers() {
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.Proj)
	for id, e := range a.state.Entities {
		if e.Kind != game.KindUser || e.Shape == nil {
			continue
		}
		cached := a.wv.actors[id]
		if cached == nil || cached.character == nil {
			continue
		}
		if e.Mounted && cached.mount != nil {
			if cached.mount.Visible(frustum, a.scene3d.ClipPlane) {
				cached.mount.Draw(a.skinner, a.scene3d)
			}
			a.drawActorShadow(cached.mount.Character)
		}
		if cached.character.Visible(frustum, a.scene3d.ClipPlane) {
			cached.character.Draw(a.skinner, a.scene3d)
		}
		a.drawActorShadow(cached.character)
	}
}

// projectName maps a world point to the UI's top-left pixel coordinates.
func projectName(p render.V3, vp render.M4, w, h int) (int, int, bool) {
	x, y, _, ok := projectLabel(p, vp, w, h)
	return x, y, ok
}

func projectLabel(p render.V3, vp render.M4, w, h int) (int, int, float32, bool) {
	x := p[0]*vp[0] + p[1]*vp[4] + p[2]*vp[8] + vp[12]
	y := p[0]*vp[1] + p[1]*vp[5] + p[2]*vp[9] + vp[13]
	z := p[0]*vp[2] + p[1]*vp[6] + p[2]*vp[10] + vp[14]
	cw := p[0]*vp[3] + p[1]*vp[7] + p[2]*vp[11] + vp[15]
	if cw <= 0 || z < 0 || z > cw || x < -cw || x > cw || y < -cw || y > cw {
		return 0, 0, 0, false
	}
	return int((x/cw + 1) * float32(w) / 2), int((1 - y/cw) * float32(h) / 2), z / cw, true
}

// Phoenix uses the original NPC categories as a second line, omitting the
// generic, gambling-house and animal categories.
func npcTypeLabel(kind uint8) string {
	switch kind {
	case 1:
		return "Merchant"
	case 2:
		return "GateKeeper"
	case 3:
		return "Blacksmith"
	case 4:
		return "PvPManager"
	case 6:
		return "Warehouse"
	case 8:
		return "Guard"
	case 10:
		return "Apprentice"
	case 11:
		return "GuildMaster"
	case 12:
		return "DeadNpc"
	default:
		return ""
	}
}

// Character entry contains only a guild ID. Resolve names exclusively from
// Rapture's confirmed own-guild data or its directory, never guess unknown IDs.
func guildCaption(s *game.State, guildID uint32) string {
	if s == nil || guildID == 0 {
		return ""
	}
	name := ""
	if guildID == s.Guild.ID {
		name = s.Guild.Name
	}
	if name == "" {
		name = s.Guild.Directory[guildID].Name
	}
	if name == "" {
		return ""
	}
	return "<" + name + ">"
}

// The ground packet exposes an item stack count. For gold, Rapture sends a
// fixed count of one and does not disclose the actual amount until pickup.
func (a *App) groundItemName(it proto.GroundItem) string {
	if it.Type == 26 && it.TypeID == 0 {
		return "Gold"
	}
	name := fmt.Sprintf("Item %d/%d", it.Type, it.TypeID)
	if a.arc != nil || a.items != nil {
		if def := a.itemsItem(it.Type, it.TypeID); def != nil && def.Name != "" {
			name = def.Name
		}
	}
	if it.Count > 1 {
		name += fmt.Sprintf(" (x%d)", it.Count)
	}
	return name
}

// Rapture's monster drops may carry Y=0 even on elevated terrain. Preserve
// server position for interaction, but never draw their label below the field.
func groundItemPosition(w *scene.World, it proto.GroundItem) render.V3 {
	y := it.Y
	if w != nil && w.HasTerrain() && it.X >= 0 && it.Z >= 0 && it.X <= float32(w.WLD.MapSize) && it.Z <= float32(w.WLD.MapSize) {
		y = max(y, w.Height(it.X, it.Z))
	}
	return render.V3{it.X, y, it.Z}
}

func groundItemLabelPosition(w *scene.World, it proto.GroundItem) render.V3 {
	return groundItemPosition(w, it).Add(render.V3{0, .3, 0})
}

// Phoenix player_nameplate.gd / creature_labels.gd / classic_overlay.gdshader:
// posed crown anchors, outlined fixed-pixel text, 50m range and scene occlusion.
func (a *App) drawPlayerNames(w, h int) {
	settings := a.clientSettings()
	a.sprites.InvalidateNameDepth()
	v := a.wv
	if v == nil {
		return
	}
	if len(a.state.Entities) == 0 && len(a.state.GroundItems) == 0 && v.self == nil {
		return
	}
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	captured := false
	// A model can be visible while its label anchor is off screen. Preserve
	// scene depth for clicking the mesh in that case too.
	for _, batch := range v.groundBatches {
		if len(batch) > 0 {
			a.sprites.CaptureNameDepth(w, h)
			captured = true
			break
		}
	}
	if !captured {
		frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.Proj)
		for _, mob := range v.mobs {
			if mob != nil && mob.actor != nil && mob.actor.Visible(frustum, a.scene3d.ClipPlane) {
				a.sprites.CaptureNameDepth(w, h)
				captured = true
				break
			}
		}
	}
	if !captured {
		frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.Proj)
		for _, player := range v.actors {
			if player != nil && player.character != nil && player.character.Visible(frustum, a.scene3d.ClipPlane) {
				a.sprites.CaptureNameDepth(w, h)
				captured = true
				break
			}
		}
	}
	label := func(name string, pos render.V3, tint [4]float32, offset int) {
		if name == "" || pos.Sub(v.pos).Len() > 50 {
			return
		}
		if x, y, depth, ok := projectLabel(pos, vp, w, h); ok {
			if !captured {
				a.sprites.CaptureNameDepth(w, h)
				captured = true
			}
			a.sprites.NameDepth(depth)
			a.font.DrawOutlined(a.sprites, name, x-a.font.Width(name)/2, y-a.font.Height-offset, tint)
		}
	}
	for _, it := range a.state.GroundItems {
		label(a.groundItemName(it), groundItemLabelPosition(v.world, it), white, 0)
	}
	if slot, ok := a.selfSlot(); ok && v.self != nil && settings.OwnName {
		if pos, ok := v.self.NameAnchor(); ok {
			label(slot.Name, pos, white, 0)
			label(guildCaption(a.state, a.state.Guild.ID), pos, [4]float32{.72, .93, .72, 1}, -18)
		}
	}
	for _, e := range a.state.Entities {
		name := ""
		pos := render.V3{}
		valid := false
		tint := white
		switch e.Kind {
		case game.KindUser:
			if !settings.PlayerNames {
				continue
			}
			if actor := v.actors[e.ID]; actor != nil && actor.character != nil && e.Shape != nil {
				name = e.Shape.Name
				pos, valid = actor.character.NameAnchor()
			}
		case game.KindMob:
			if !settings.MonsterNames {
				continue
			}
			if actor := v.mobs[e.ID]; actor != nil && actor.actor != nil {
				name = actor.actor.Name
				pos, valid = actor.actor.NameAnchor()
				if a.state.Target != e.ID {
					// Only the pointed-at or selected monster has a plate in the reference.
					center := actor.pos.Add(pos.Sub(actor.pos).Scale(.5))
					x, y, ok := projectName(center, vp, w, h)
					_, top, topOK := projectName(pos, vp, w, h)
					radius := max(18, min(60, absInt(y-top)))
					dx, dy := a.lastMouseX-x, a.lastMouseY-y
					if !ok || !topOK || dx*dx+dy*dy > radius*radius {
						continue
					}
				}
			}
			if actor := v.mobs[e.ID]; actor != nil && actor.actor != nil {
				player, _ := a.selfSlot()
				tint = ui.RGBColor(ui.MobNameRGB(actor.actor.Level, player.Level))
			}
		case game.KindNpc:
			if actor := v.npcs[e.ID]; actor != nil && actor.actor != nil {
				name = actor.actor.Name
				pos, valid = actor.actor.NameAnchor()
			}
			tint = [4]float32{1, .93, .35, 1}
		}
		if valid {
			if kind := npcTypeLabel(e.NpcType); e.Kind == game.KindNpc && kind != "" {
				label(kind, pos, [4]float32{.55, .85, 1, 1}, 0)
				label(name, pos, tint, 22)
			} else {
				label(name, pos, tint, 0)
				if e.Kind == game.KindMob && a.state.Target == e.ID && pos.Sub(v.pos).Len() <= 50 {
					if x, y, depth, ok := projectLabel(pos, vp, w, h); ok {
						a.sprites.NameDepth(depth)
						if dot := a.assets.Texture("interface/name_select.png"); dot != nil {
							half := a.font.Width(name)/2 + 10
							for _, side := range []int{-1, 1} {
								a.sprites.DrawImage(dot, float32(x+side*half-dot.W/2), float32(y-a.font.Height+(a.font.Height-dot.H)/2))
							}
						}
					}
				}
				if e.Kind == game.KindUser {
					label(guildCaption(a.state, e.GuildID), pos, [4]float32{.72, .93, .72, 1}, -18)
				}
			}
		}
	}
	a.sprites.NameDepth(-1)
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
