package app

import (
	"cmp"
	"fmt"
	"log"
	"math"
	"slices"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/pvprank"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type playerPlate struct {
	gm    bool
	kills uint32
	party uint8 // 1 party member, 2 party leader (jang): the 16x16 coin left of the name
}

type remoteActor struct {
	climbing        bool
	entity          *game.Entity
	position        remotePosition
	gestureSerial   uint64
	moveSerial      uint64
	moveMode        uint8
	moveX, moveZ    float32
	walkUntil       float64
	jumpMoving      bool
	dead            bool
	swimming        bool
	shape           proto.UserShape
	hideCloak       bool
	hideHelmet      bool
	character       *scene.Character
	ghost           *scene.Character // standing translucent spirit of a dead player
	appearance      scene.Appearance
	mount           *scene.Vehicle
	mountShape      uint8
	mountLoaded     bool
	mountCasting    bool
	travelDirection render.V3
	facing          render.V3
}

// The native 0x0501 low bits encode walk/run, not a reliable stop flag.
// Infer slow locomotion from successive positions and expire it if updates stop.
func (r *remoteActor) observeMovement(e *game.Entity, now float64) (walking, moving bool) {
	if r.moveSerial != e.MoveSerial {
		dx, dz := e.X-r.moveX, e.Z-r.moveZ
		r.jumpMoving = dx*dx+dz*dz > 0.000001
		if r.jumpMoving {
			r.travelDirection = render.V3{dx, 0, dz}.Norm()
		}
		if e.Motion&0x3f == 0 && r.moveMode == 0 && dx*dx+dz*dz > 0.000001 {
			r.walkUntil = now + .7
		} else {
			r.walkUntil = 0
		}
		r.moveSerial, r.moveMode, r.moveX, r.moveZ = e.MoveSerial, e.Motion&0x3f, e.X, e.Z
	}
	walking = e.Motion&0x3f == 0 && r.walkUntil > now
	// The jump flag is also sent for stationary vertical motion. It does not
	// by itself mean the actor is running horizontally.
	if e.Motion&proto.MotionFlag80 != 0 && e.Motion&0x3f == 4 {
		return false, r.jumpMoving
	}
	return walking, e.Motion&0x3f != 0 || walking
}

func remoteLocomotion(w *scene.World, e *game.Entity) string {
	moving := e.Motion&0x3f != 0
	if w.SwimmingAt(e.X, e.Y, e.Z) {
		if moving {
			return "007_swim"
		}
		return "006_swnormal"
	}
	if e.Sitting {
		return "012_sit"
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
	settings := a.clientSettings()
	for id, e := range a.state.Entities {
		if e.Kind != game.KindUser || e.Shape == nil {
			continue
		}
		sh := *e.Shape
		if form := a.state.PlayerForms[id]; form.Kind == proto.FormDisguise {
			sh = form.Shape
		}
		hideCloak := settings.HideCloak && sh.Equipment[7][0] != 0
		hideHelmet := settings.HideHelmet && sh.Equipment[0][0] != 0
		cached := v.actors[id]
		newEntity := cached == nil || cached.entity != e
		if newEntity {
			cached = &remoteActor{entity: e, moveSerial: e.MoveSerial, moveMode: e.Motion & 0x3f, moveX: e.X, moveZ: e.Z}
			v.actors[id] = cached
		}
		if newEntity || cached.shape != sh || cached.hideCloak != hideCloak || cached.hideHelmet != hideHelmet {
			slot := proto.CharacterSlot{Race: sh.Race, Gender: sh.Gender, Class: sh.Class, Face: sh.Face, Hair: sh.Hair, Height: sh.Height}
			for i, it := range sh.Equipment {
				slot.EquipType[i], slot.EquipTypeID[i] = it[0], it[1]
			}
			slot.Cloak = sh.Cloak
			appearance := a.appearance(&slot)
			appearance.WeaponEnchant, appearance.OffhandEnchant = sh.Equipment[5][2], sh.Equipment[6][2]
			actor, err := scene.BuildCharacter(a.assets, appearance, "000_normal")
			if err != nil {
				log.Printf("player %d model: %v", id, err)
			}
			// Cosmetics can replace the model, but must not reset which
			// gestures have already been consumed for this entity.
			cached.character, cached.shape = actor, sh
			cached.ghost, cached.appearance = nil, appearance
			cached.hideCloak, cached.hideHelmet = hideCloak, hideHelmet
			cached.dead, cached.mountCasting = false, false
		}
		c := cached.character
		if c == nil {
			continue
		}
		walking, moving := cached.observeMovement(e, a.clock)
		if moving {
			cancelGesture(c)
		}
		if e.Dead && !cached.dead {
			c.PlayAction(9, true)
		}
		if !e.Dead && cached.dead {
			c.ReleaseAction()
		}
		cached.dead = e.Dead
		if cached.gestureSerial != e.GestureSerial {
			cached.gestureSerial = e.GestureSerial
			if action, ok := proto.LocomotionAction(e.Gesture); ok && !e.Dead && !e.Mounted && !c.PlayingAction(8) {
				if c.PlayAction(action, false) && action == 8 {
					a.stopCastEffects(id)
				}
			}
			if action, ok := proto.GestureAction(e.Gesture); ok && !e.Dead && !e.Mounted && !moving && !c.PlayingAction(8) {
				c.PlayAction(action, false)
			}
		}
		if e.Motion == 4|proto.MotionFlag80 {
			if _, _, casting := c.CastTiming(); casting {
				c.CancelCast()
				a.stopCastEffects(id)
			}
		}
		swimming := v.world.SwimmingAt(e.X, e.Y, e.Z)
		if swimming && !cached.swimming && !e.Dead && !e.Mounted {
			c.ReleaseActionExceptJump()
		}
		cached.swimming = swimming
		clip := remoteLocomotion(v.world, e)
		if !moving && !swimming && !e.Sitting {
			clip = "000_normal"
		}
		if walking && (!e.Sitting || swimming) {
			if v.world.SwimmingAt(e.X, e.Y, e.Z) {
				clip = "007_swim"
			} else {
				clip = "001_walk"
			}
		}
		if e.ClimbMode != 0 && !e.Dead && !e.Mounted {
			if !cached.climbing {
				c.ReleaseAction()
				c.CancelCast()
			}
			clip = "018_ladder"
		}
		cached.climbing = e.ClimbMode != 0
		casting := e.Vehicle.State == proto.VehicleCasting
		if casting && !cached.mountCasting {
			c.PlayCast(85, float32(e.Vehicle.CastMS)/1000)
		}
		if !casting && cached.mountCasting {
			c.CancelCast()
		}
		cached.mountCasting = casting
		c.Mounted = e.Mounted
		if e.Mounted {
			c.SetMountedClip(moving || e.Motion&proto.MotionFlag80 != 0 || cached.mount.PlayingJump(), scene.VehicleAlternate(e.MountShape))
		} else {
			c.SetClip(clip)
		}
		c.Update(float32(a.frameDT))
		if e.ClimbMode == 1 && !c.ActionActive() {
			c.Time -= float32(a.frameDT)
		}
		angle := float64(e.Dir) / 65536 * 2 * math.Pi
		fwd := render.V3{float32(math.Sin(angle)), 0, float32(math.Cos(angle))}
		if c.ActionActive() && cached.facing.Len() > 0 {
			fwd = cached.facing
		} else {
			cached.facing = render.V3{}
		}
		scale := (scene.Appearance{Height: sh.Height}).ScaleMatrix()
		pos := cached.position.update(e, a.clock)
		if e.ClimbMode != 0 && !e.Dead && !e.Mounted {
			pos = v.world.LadderVisual(pos, -1)
		}
		if d, ok := v.dashes[id]; ok {
			pos = render.V3(d.Position(a.skillPhysicsTime() - d.Started))
			cached.position.pos, cached.position.from, cached.position.to = pos, pos, pos
			cached.position.duration = 0
			fwd = render.V3(d.To).Sub(render.V3(d.From))
			fwd[1] = 0
			fwd = fwd.Norm()
		}
		if e.Mounted && e.Vehicle.State != proto.VehiclePassenger {
			if !cached.mountLoaded || cached.mountShape != e.MountShape {
				cached.mountShape, cached.mountLoaded = e.MountShape, true
				var err error
				cached.mount, err = a.assets.BuildVehicle(sh.Race, e.MountShape)
				if err != nil {
					log.Printf("player %d mount: %v", id, err)
				}
			}
			if cached.mount != nil {
				reverse := e.Motion&0x3f == 2
				if moving && cached.travelDirection.Len() > 0 {
					reverse = reverse || cached.travelDirection.Dot(fwd) < -.5
					if !reverse {
						fwd = cached.travelDirection
					}
				}
				cached.mount.SetMotion(moving, walking, reverse, e.Motion&proto.MotionFlag80 != 0)
				cached.mount.Update(float32(a.frameDT))
				cached.mount.World = render.Basis(pos, fwd.Scale(-1), render.V3{0, 1, 0})
				pos = cached.mount.Seat()
				cached.mount.PrepareEffects(float32(a.frameDT), a.clientSettings().SkillEffects, true, &a.vehicle.effectBudget)
			}
		} else {
			cached.mount, cached.mountLoaded = nil, false
		}
		c.World = scale.Mul(render.Basis(pos, fwd.Scale(-1), render.V3{0, 1, 0}))
		a.prepareSpirit(cached, e)
	}
}

// prepareSpirit follows 0x41b300/0x419f60: when the death clip ends, the body
// stays in its last frame and the character itself stands up again as a
// translucent spirit. Only other players' actors get one; the own character
// keeps showing just its body.
func (a *App) prepareSpirit(cached *remoteActor, e *game.Entity) {
	body := cached.character
	if !e.Dead || !body.PlayingAction(9) || !body.ActionFinished() {
		if !e.Dead {
			cached.ghost = nil
		}
		return
	}
	if cached.ghost == nil {
		ghost, err := scene.BuildCharacter(a.assets, cached.appearance, "000_normal")
		if err != nil {
			log.Printf("player %d spirit: %v", e.ID, err)
			return
		}
		cached.ghost = ghost
	}
	g := cached.ghost
	g.Concealed, g.FadeOpacity, g.Mounted = true, render.GhostOpacity, false
	g.SetClip("000_normal")
	g.Update(float32(a.frameDT))
	g.World = body.World
}

func (a *App) drawRemotePlayers() {
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
	v := a.wv
	v.visiblePlayers, v.visiblePlayerCloth = v.visiblePlayers[:0], v.visiblePlayerCloth[:0]
	v.visiblePlayerMounts = v.visiblePlayerMounts[:0]
	for id, e := range a.state.Entities {
		if e.Kind != game.KindUser || e.Shape == nil {
			continue
		}
		cached := a.wv.actors[id]
		if cached == nil || cached.character == nil {
			continue
		}
		var mount *scene.Vehicle
		cached.character.Concealed = e.Concealed
		if e.Mounted && cached.mount != nil {
			cached.mount.Concealed = e.Concealed
			if cached.mount.Visible(frustum, a.scene3d.ClipPlane) {
				mount = cached.mount
			}
		}
		v.visiblePlayers = append(v.visiblePlayers, cached.character)
		v.visiblePlayerMounts = append(v.visiblePlayerMounts, mount)
		if cached.ghost != nil && e.Dead {
			v.visiblePlayers = append(v.visiblePlayers, cached.ghost)
			v.visiblePlayerMounts = append(v.visiblePlayerMounts, nil)
		}
	}
	scene.CullCrowdCharacters(v.visiblePlayers, frustum, a.scene3d)
	for _, character := range v.visiblePlayers {
		if character.HasCloak() {
			v.visiblePlayerCloth = append(v.visiblePlayerCloth, character)
		}
	}
	scene.PrepareCrowdCloth(v.visiblePlayerCloth, a.scene3d)
	for _, mount := range v.visiblePlayerMounts {
		if mount != nil {
			mount.Draw(a.skinner, a.scene3d)
			mount.DrawEffects(a.scene3d)
		}
	}
	v.crowdCharacters = v.crowdCharacters[:0]
	for _, character := range v.visiblePlayers {
		if character != nil {
			v.crowdCharacters = append(v.crowdCharacters, character)
		}
	}
	v.crowdBatch.DrawMin(a.skinner, a.scene3d, v.crowdCharacters, 4)
	clear(v.crowdCharacters)
	v.crowdCharacters = v.crowdCharacters[:0]
	if a.scene3d.ClipPlane == ([4]float32{}) {
		v.anyPlayerVisible = len(v.visiblePlayers) > 0
	}
	clear(v.visiblePlayers)
	clear(v.visiblePlayerMounts)
	clear(v.visiblePlayerCloth)
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
func (a *App) npcTypeLabel(kind uint8) string {
	switch kind {
	case 1:
		return a.tr(5154)
	case 2:
		return a.tr(5160)
	case 3:
		return a.tr(5161)
	case 4:
		return a.tr(12228)
	case 6:
		return a.tr(5165)
	case 8:
		return a.tr(5166)
	case 10:
		return a.tr(5167)
	case 11:
		return a.tr(5168)
	case 12:
		return a.tr(12229)
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
	return name
}

// Native 0x0044c0cf..0x0044c125: the guild caption has its own tint,
// independent of character name/GM/faction colours. The directory identifies
// masters by character name; own-guild role data also works before it arrives.
func guildCaptionColor(s *game.State, guildID, characterID uint32, name string) [4]float32 {
	leader := false
	if s != nil && guildID != 0 {
		if guildID == s.Guild.ID && characterID == s.SelfID && s.Guild.Rank != 0 {
			leader = s.Guild.Rank == 1
		} else {
			master := s.Guild.Directory[guildID].Master
			leader = master != "" && master == name
			if guildID == s.Guild.ID {
				if member, ok := s.Guild.Members[characterID]; ok && member.Rank != 0 {
					leader = member.Rank == 1
				}
			}
		}
	}
	if leader {
		return ui.RGBColor(0xE0AFFA)
	}
	return ui.RGBColor(0xE4E4E4)
}

const guildCaptionOffset = 15 // Native 0x0044c100 subtracts float 0x006bc894.

// The ground packet exposes an item stack count. For gold, Rapture sends a
// fixed count of one and does not disclose the actual amount until pickup.
func (a *App) groundItemName(it proto.GroundItem) string {
	if it.Type == 26 && it.TypeID == 0 {
		return a.tr(5226)
	}
	name := a.msg.Format(12299, map[string]string{"v": fmt.Sprint(it.Type), "v2": fmt.Sprint(it.TypeID)})
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

// playerNameplateAnchor uses the projected actor origin from the native
// player pass. GX stores the feet position, 0.9 below the native base.
func (a *App) playerNameplateAnchor(id uint32, actor *scene.Character, race uint8, low bool, vehicle proto.VehicleState) (render.V3, int) {
	pos := render.V3{actor.World[12], actor.World[13], actor.World[14]}
	if form := a.wv.transforms[id]; actor.Hidden && form != nil && form.actor != nil {
		return form.actor.MonsterPlateAnchor(), 0
	}
	if actor.Mounted {
		if mount := a.driverMount(vehicle.Driver); mount != nil {
			top := mount.MonsterPlateAnchor()
			if vehicle.State == proto.VehiclePassenger {
				pos[1] = top[1]
				return pos, -15
			}
			return top, 0
		}
	}
	height := float32(2.2)
	if low {
		height = 1.5
	}
	if race != 3 {
		height -= .2
	}
	pos[1] += height
	return pos, 0
}

func (a *App) catalogNameSuffix(e *game.Entity) string {
	if !a.showCatalogIDs {
		return ""
	}
	if e.Kind == game.KindNpc {
		return fmt.Sprintf(" (%d %d)", e.NpcType, e.TypeID)
	}
	return fmt.Sprintf(" (%d)", e.TypeID)
}

func (a *App) captureNameDepth() {
	if a.frameTarget != nil {
		if id, dw, dh := a.frameTarget.DepthTexture(); id != 0 && dw == a.physW && dh == a.physH {
			a.sprites.UseNameDepthTexture(id, dw, dh, a.scene3d.ReverseDepth)
			return
		}
	}
	a.sprites.CaptureNameDepth(a.physW, a.physH)
}

// drawPlayerNames uses the original world anchors and fixed pixel overlays.
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
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
	v.plateIcons = v.plateIcons[:0]
	captured := false
	if v.anyMobVisible || v.anyPlayerVisible {
		a.captureNameDepth()
		captured = true
	}
	// A model can be visible while its label anchor is off screen. Preserve
	// scene depth for clicking the mesh in that case too.
	if !captured {
		for _, batch := range v.groundBatches {
			if len(batch) > 0 {
				a.captureNameDepth()
				captured = true
				break
			}
		}
	}
	if !captured {
		frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
		for _, mob := range v.mobs {
			if mob != nil && mob.actor != nil && mob.actor.Visible(frustum, a.scene3d.ClipPlane) {
				a.captureNameDepth()
				captured = true
				break
			}
		}
	}
	if !captured {
		frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
		for _, player := range v.actors {
			if player != nil && player.character != nil && player.character.Visible(frustum, a.scene3d.ClipPlane) {
				a.captureNameDepth()
				captured = true
				break
			}
		}
	}
	// Overlay text starts at the projected Y, centered on the projected X.
	label := func(name string, pos render.V3, tint [4]float32, offset int, plate playerPlate, marker, suffix string) {
		if name == "" {
			return
		}
		x, y, depth, ok := projectLabel(pos, vp, w, h)
		if !ok {
			return
		}
		nameWidth := a.font.Width(name)
		width := nameWidth + a.font.Width(suffix)
		left, top := x-width/2, y-offset
		if a.scene3d.OccludedLabel(pos, left, top, width, a.font.Height, w, h) {
			return
		}
		if !captured {
			a.captureNameDepth()
			captured = true
		}
		a.sprites.NameDepth(depth)
		if plate.gm {
			tint = ui.RGBColor(0x3B6CFF)
		}
		a.font.DrawName(a.sprites, name, left, top, tint)
		if suffix != "" {
			a.font.DrawName(a.sprites, suffix, left+nameWidth, top, white)
		}
		outerLeft, outerRight := left, left+width
		rank, ranked := pvprank.Index(plate.kills)
		if plate.party != 0 {
			file := "interface/party_jang.png"
			if plate.party == 1 {
				file = "interface/party_member.png"
			}
			if tex := a.assets.Texture(file); tex != nil {
				outerLeft = left - 11 // 0x44bdb6: left of name by 5+6 pixels.
				if ranked {
					outerLeft -= 6 // Match the early PvP-rank offset before it is compensated.
				}
				v.plateIcons = append(v.plateIcons, nameplateIcon{tex, float32(outerLeft), float32(top - 3), 16, 16, 0, 0, 1, 1, depth})
			}
		}
		if ranked {
			if grade := a.assets.Texture("interface/grade2.png"); grade != nil {
				outerRight += 1 + 32
				u0, v0 := float32(rank/16)*.5, float32(rank%16)/16
				v.plateIcons = append(v.plateIcons, nameplateIcon{grade, float32(left + width + 1), float32(top - 3), 32, 32, u0, v0, u0 + .5, v0 + 1./16, depth})
			}
		}
		if marker != "" {
			dot := a.assets.Texture(marker)
			if dot == nil {
				dot = a.assets.Texture("interface/name_select.png")
			}
			if dot != nil {
				for _, edge := range []int{outerLeft - 17, outerRight + 1} {
					v.plateIcons = append(v.plateIcons, nameplateIcon{dot, float32(edge), float32(top - 3), 16, 16, 0, 0, 1, 1, depth})
				}
			}
		}
	}
	const neutralMarker = "interface/mon_not.png"
	targetMarker := func(id uint32) string {
		if id != 0 && a.state.Target == id {
			return neutralMarker
		}
		return ""
	}
	slot, hasSlot := a.selfSlot()
	if hasSlot && v.self != nil && settings.OwnName {
		pos, offset := a.playerNameplateAnchor(a.state.SelfID, v.self, slot.Race, a.state.Self.Dead || a.state.Self.Sitting, a.state.Self.Vehicle)
		if v.selfGhost != nil && a.state.Self.Dead {
			// Same split as other players: name on the body, full plate on the spirit.
			label(slot.Name, pos, white, offset, playerPlate{}, "", "")
			pos, offset = a.playerNameplateAnchor(a.state.SelfID, v.selfGhost, slot.Race, false, a.state.Self.Vehicle)
		}
		label(slot.Name, pos, white, offset, playerPlate{gm: a.state.Self.IsGM, kills: a.state.Details.Kills, party: a.partyBadgeKind(a.state.SelfID)}, targetMarker(a.state.SelfID), "")
		label(guildCaption(a.state, a.state.Guild.ID), pos, guildCaptionColor(a.state, a.state.Guild.ID, a.state.SelfID, slot.Name), offset+guildCaptionOffset, playerPlate{}, "", "")
	}
	hoveredMob := uint32(0)
	if a.hoveredMob != 0 && a.hoveredMobX == a.lastMouseX && a.hoveredMobY == a.lastMouseY {
		hoveredMob = a.hoveredMob
	}
	mobMarkers := [...]string{neutralMarker, "interface/mon_fire.png", "interface/mon_water.png", "interface/mon_ground.png", "interface/mon_wind.png"}
	for _, e := range a.state.Entities {
		switch e.Kind {
		case game.KindUser:
			actor := v.actors[e.ID]
			if !settings.PlayerNames || actor == nil || actor.character == nil || e.Shape == nil {
				continue
			}
			name := actor.shape.Name
			pos, offset := a.playerNameplateAnchor(e.ID, actor.character, actor.shape.Race, e.Dead || e.Sitting, e.Vehicle)
			tint := white
			if hasSlot && slot.Race/2 != e.Shape.Race/2 {
				tint = ui.RGBColor(0xff0000)
			}
			plate, plateMarker := playerPlate{gm: e.IsGM, kills: e.PvPKills, party: a.partyBadgeKind(e.ID)}, targetMarker(e.ID)
			if actor.ghost != nil && e.Dead {
				// The corpse copy keeps only the name (0x41b300 does not copy the
				// guild or badges); the standing spirit carries the full plate.
				label(name, pos, tint, offset, playerPlate{}, "", "")
				pos, offset = a.playerNameplateAnchor(e.ID, actor.ghost, actor.shape.Race, false, e.Vehicle)
				label(name, pos, tint, offset, plate, plateMarker, "")
			} else {
				label(name, pos, tint, offset, plate, plateMarker, "")
			}
			caption := guildCaption(a.state, e.GuildID)
			if caption == "" && e.GuildID != 0 {
				caption = e.Shape.Guild
			}
			label(caption, pos, guildCaptionColor(a.state, e.GuildID, e.ID, name), offset+guildCaptionOffset, playerPlate{}, "", "")
		case game.KindMob:
			actor := v.mobs[e.ID]
			if (!settings.MonsterNames || e.ID != a.state.Target && e.ID != hoveredMob) || e.Dead || actor == nil || actor.actor == nil || actor.pos.Sub(v.pos).Len() > 100 {
				continue
			}
			mob := actor.actor
			marker := ""
			if element := mob.Definition.Element; e.ID == a.state.Target && int(element) < len(mobMarkers) {
				marker = mobMarkers[element]
			}
			label(mob.Name, mob.MonsterPlateAnchor(), ui.RGBColor(ui.MobNameRGB(mob.Level, slot.Level)), 0, playerPlate{}, marker, a.catalogNameSuffix(e))
		case game.KindNpc:
			if actor := v.npcs[e.ID]; actor != nil && actor.actor != nil {
				pos := actor.actor.NPCPlateAnchor(e.NpcType, e.TypeID)
				// The MON height can sit inside the mesh; the depth-tested plate
				// would then vanish behind its own NPC, so keep it above the crown.
				if e.NpcType != 12 {
					if clo, chi, ok := actor.actor.CoarseBounds(); ok {
						if pos[1] < chi[1]+.15 && frustum.BoxVisible(clo, chi) {
							if _, hi, ok := actor.actor.WorldBounds(); ok && pos[1] < hi[1]+.15 {
								pos[1] = hi[1] + .15
							}
						}
					} else if _, hi, ok := actor.actor.WorldBounds(); ok && pos[1] < hi[1]+.15 {
						pos[1] = hi[1] + .15
					}
				}
				// The notice board's plate is hidden by its own board when seen
				// from the front: raise it well above the board.
				if a.auctionBoard(e.ID) {
					pos[1] += noticeBoardPlateRise
				}
				label(actor.actor.Name, pos, ui.RGBColor(0xffff00), 0, playerPlate{}, targetMarker(e.ID), a.catalogNameSuffix(e))
				label(a.npcTypeLabel(e.NpcType), pos, [4]float32{.35, .9, 1, 1}, a.font.Height-2, playerPlate{}, "", "")
			}
		}
	}
	if len(v.plateIcons) > 1 {
		slices.SortFunc(v.plateIcons, func(a, b nameplateIcon) int { return cmp.Compare(a.tex.ID, b.tex.ID) })
	}
	for _, icon := range v.plateIcons {
		a.sprites.NameDepth(icon.depth)
		a.sprites.Draw(icon.tex, icon.x, icon.y, icon.w, icon.h, icon.u0, icon.v0, icon.u1, icon.v1, white)
	}
	clear(v.plateIcons)
	v.plateIcons = v.plateIcons[:0]
	a.sprites.NameDepth(-1)
}

// noticeBoardPlateRise lifts the notice board name plate, in world units.
const noticeBoardPlateRise = 1.5

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// partyBadgeKind is the name-plate coin of 0x44bd44: yourself by your own
// group state (leader or raid role 0 -> jang), anyone else by the 0x0520 badge
// the server broadcasts, with your own party's roster as the fallback.
func (a *App) partyBadgeKind(id uint32) uint8 {
	p := &a.state.Party
	if id == a.state.SelfID {
		switch {
		case len(p.Members) == 0:
			return 0
		case p.Leader == id:
			return 2
		}
		return 1
	}
	if e := a.state.Entities[id]; e != nil && e.PartyBadge != 0 {
		return min(e.PartyBadge, 2)
	}
	if p.Member(id) != nil {
		if p.Leader == id {
			return 2
		}
		return 1
	}
	return 0
}
