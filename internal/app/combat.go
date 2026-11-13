package app

import (
	"fmt"
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Rapture's 0x0205 failure payload contains an explicit reason. Successful
// item and money packets retain their normal inventory handling.
func (a *App) confirmedPickup(p []byte) {
	if proto.Op(p) != proto.AddItem || len(p) != 4 || p[2] != 0 {
		return
	}
	message := "Unable to pick up that item."
	switch p[3] {
	case 0:
		message = "That item is reserved for another player."
	case 1:
		message = "Your inventory is full."
	case 2:
		message = "You do not need that quest item."
	case 3:
		message = "That item is no longer available."
	case 4:
		message = "That item is too far away."
	}
	ids := map[byte]int{0: 202, 1: 238, 3: 239, 4: 233}
	if id := ids[p[3]]; id != 0 && a.msg.Get(id) != "" {
		a.systemMessage(1, id, nil)
	} else {
		a.questMessage(message)
	}
}

// Delivery controls: click a player or mob to select;
// Double click or F requests auto-attack. The server owns range, cadence and damage.
func (a *App) updateCombat(in *platform.Input, w, h int) {
	if a.wv == nil {
		return
	}
	if a.state.Self.Dead {
		a.wv.mobClick = mobClick{}
		for _, r := range in.Chars {
			if (r == 'r' || r == 'R') && a.clock >= a.wv.reviveAt {
				if err := a.world.Send(proto.RevivePacket()); err != nil {
					a.backToLogin("Unable to request resurrection.")
					return
				}
				a.wv.reviveAt = a.clock + 3
			}
		}
		return
	}
	id := a.pickGroundLabel(in, w, h)
	if id == 0 {
		id = a.pickGroundModel(in, w, h)
	}
	if id != 0 {
		a.wv.mobClick = mobClick{}
		if err := a.world.Send(proto.PickupPacket(id)); err != nil {
			a.backToLogin("Unable to request item pickup.")
		}
		return
	}
	if id := a.pickNPC(in, w, h); id != 0 {
		a.wv.mobClick = mobClick{}
		a.openNPC(id)
		return
	}
	if in.Clicked || in.HasKey(platform.KeyTarget) || in.HasKey(platform.KeyPreviousTarget) {
		target := a.pickMob(in, w, h)
		if in.HasKey(platform.KeyPreviousTarget) {
			target = a.state.PreviousTarget
			if e := a.state.Entities[target]; e == nil || e.Dead {
				return
			}
		}
		if e := a.state.Entities[target]; e != nil && e.Kind == game.KindUser && target != a.state.Target {
			if old := a.state.Entities[a.state.Target]; old != nil && old.Kind == game.KindMob {
				if err := a.world.Send(proto.StopAttackPacket()); err != nil {
					a.backToLogin("Unable to stop attack.")
					return
				}
			}
		}
		if e := a.state.Entities[target]; target != a.state.Target && e != nil && e.Kind == game.KindMob {
			if err := a.world.Send(proto.TargetMobPacket(target)); err != nil {
				a.backToLogin("Unable to select target.")
				return
			}
		}
		if target != a.state.Target {
			a.state.PreviousTarget = a.state.Target
		}
		a.state.Target = target
		if in.Clicked {
			a.wv.travel = false
			if target == 0 {
				a.moveToGround(in, w, h)
			}
		}
		if in.Clicked {
			if a.wv.mobClick.double(target, in.MouseX, in.MouseY, a.clock) && !a.attackTarget() {
				return
			}
		} else {
			a.wv.mobClick = mobClick{}
		}
	}
	for _, r := range in.Chars {
		if r == 'x' || r == 'X' {
			if err := a.world.Send(proto.StopAttackPacket()); err != nil {
				a.backToLogin("Unable to stop attack.")
				return
			}
			a.state.Target = 0
			a.wv.mobClick = mobClick{}
			continue
		}
		if r == 'h' || r == 'H' {
			if !a.pickupNearest() {
				return
			}
			continue
		}
		if r != 'f' && r != 'F' {
			continue
		}
		a.wv.mobClick = mobClick{}
		if !a.attackTarget() {
			return
		}
	}
}

func (a *App) pickupNearest() bool {
	id := uint32(0)
	best := float32(50 * 50)
	for k, it := range a.state.GroundItems {
		dx, dz := it.X-a.wv.pos[0], it.Z-a.wv.pos[2]
		d := dx*dx + dz*dz
		if d < best || d == best && (id == 0 || k < id) {
			id, best = k, d
		}
	}
	if id != 0 {
		return a.approachLoot(id)
	}
	return true
}

func (a *App) attackTarget() bool {
	if a.state.Self.Dead {
		a.wv.sitAttack = 0
		return true
	}
	e := a.state.Entities[a.state.Target]
	if e == nil || e.Dead || e.Kind != game.KindMob && (e.Kind != game.KindUser || e.ID == a.state.SelfID) {
		return true
	}
	if a.state.Self.Sitting || a.wv.sitPending || a.wv.self != nil && a.wv.self.PlayingAction(11) {
		a.wv.sitAttack = e.ID
		if a.state.Self.Sitting && !a.wv.sitPending {
			return a.requestPosture(proto.MotionStandUp)
		}
		return true
	}
	a.wv.approachTarget, a.wv.approachSkill = 0, nil
	a.wv.approachNPC = false
	a.wv.travel, a.wv.autoRun = false, false
	if !a.faceRequestedTarget(e.ID) {
		return false
	}
	delta := render.V3{e.X - a.wv.pos[0], 0, e.Z - a.wv.pos[2]}
	if delta.Len() > a.attackReach() {
		a.wv.autoRun = false
		a.wv.approachTarget, a.wv.approachLoot = e.ID, false
		a.wv.approachSkill = nil
		return a.updateActionApproach(&platform.Input{})
	}
	// Rapture decides whether a player is hostile (faction, PvP map, duel,
	// guild duel). A client-side duel-only check would block normal PvP.
	if e != nil && e.Kind == game.KindUser && e.ID != a.state.SelfID && !e.Dead {
		if err := a.world.Send(proto.UserAttackPacket(e.ID)); err != nil {
			a.backToLogin("Unable to attack duel opponent.")
			return false
		}
		return true
	}
	if e != nil && e.Kind == game.KindMob && !e.Dead {
		if err := a.world.Send(proto.MobAttackPacket(e.ID)); err != nil {
			a.backToLogin("Unable to send attack request.")
			return false
		}
	}
	return true
}

func (a *App) drawCombat(w, h int) {
	a.drawCombatNumbers(w, h)
	s := a.state
	if a.clock < a.combatNoticeUntil {
		a.font.Draw(a.sprites, a.combatNotice, w/2-45, h/2-80, white, 0)
	}
	view := ui.HUDState{
		Portrait: a.hudPortrait(w, h),
		HP:       ui.Resource{Current: s.Pools.HP, Max: s.Details.MaxHP},
		MP:       ui.Resource{Current: s.Pools.MP, Max: s.Details.MaxMP},
		SP:       ui.Resource{Current: s.Pools.SP, Max: s.Details.MaxSP},
		Exp:      s.Details.Exp, PrevExp: s.Details.PrevExp, NextExp: s.Details.NextExp,
	}
	if s.Blessing.Known && s.Blessing.Country < 2 {
		if !a.blessingLoaded && a.arc != nil {
			a.blessingLoaded = true
			data, err := a.arc.ReadFile("Character/KillStatus.SData")
			if err == nil {
				var tiers []sdata.BlessingTier
				tiers, err = sdata.ParseBlessing(data)
				if err == nil {
					for country := range a.blessingMax {
						a.blessingMax[country] = sdata.BlessingMaximum(tiers, uint8(country))
					}
				}
			}
			if err != nil {
				log.Printf("blessing data: %v", err)
			}
		}
		view.BlessCount, view.BlessMax, view.BlessKnown = s.Blessing.Count, a.blessingMax[s.Blessing.Country], true
	}
	for _, slot := range s.Slots {
		if slot.ID == s.SelfID {
			view.Name, view.Level = slot.Name, slot.Level
			break
		}
	}
	if e := s.Entities[s.Target]; e != nil && e.Kind == game.KindMob {
		view.TargetName = fmt.Sprintf("Target %d", e.TypeID)
		if def, err := a.assets.MonsterDefinition(e.TypeID); err == nil {
			view.TargetName, view.TargetHP.Max = def.Name, def.HP
			view.TargetColor = ui.RGBColor(ui.MobNameRGB(def.Level, view.Level))
			view.TargetElement = def.Element
		} else if !a.assets.Missing[err.Error()] {
			log.Printf("target definition: %v", err)
			a.assets.Missing[err.Error()] = true
		}
		view.TargetHP.Current = uint32(max(e.HP, 0))
		view.TargetKnown = e.HealthKnown
	} else if e != nil && e.Kind == game.KindUser && e.Shape != nil {
		view.TargetName = e.Shape.Name
		if member := s.Party.Member(e.ID); member != nil {
			view.TargetHP = ui.Resource{Current: member.HP, Max: member.MaxHP}
			view.TargetKnown = true
		}
	}
	a.hud.Draw(a.sprites, a.font, view)
	a.bottom.QuestOpen, a.bottom.GuildOpen = a.questLog.open, a.guild.open
	a.bottom.Draw(a.sprites, a.font, a.white, w, h, view, a.inventory.open, a.skills.open, a.status.open, a.subActionsOpen)
}
