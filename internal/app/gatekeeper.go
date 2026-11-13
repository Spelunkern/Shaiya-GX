package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type gatekeeperView struct {
	npc         uint32
	index       uint8
	destination sdata.NPCGate
	pending     bool
}

func (a *App) updateGatekeeper(in *platform.Input, w, h int, e *game.Entity) bool {
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	x, y := a.gateUI.Origin(w, h)
	busy := a.shop.pending || a.inventory.pending
	choice, close := a.gateUI.Update(in, w, h, def.Gates, busy)
	if (close || in.HasKey(platform.KeyEscape)) && !busy {
		if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
			a.backToLogin("Unable to close NPC conversation.")
			return true
		}
		a.shop.npcID = 0
		return true
	}
	if choice >= 0 && a.gateConfirm != nil {
		g := def.Gates[choice]
		a.gate = gatekeeperView{npc: e.ID, index: uint8(choice), destination: g}
		label := g.Name
		if label == "" {
			label = fmt.Sprintf("map %d", g.MapID)
		}
		a.gateConfirm.Open(fmt.Sprintf("Travel to %s? Listed price: %d gold.", label, g.Cost), true)
		return true
	}
	return panelPointer(in, x, y, ui.GatekeeperWidth, ui.GatekeeperHeight)
}
func (a *App) updateGateConfirm(in *platform.Input, w, h int) {
	if a.gateConfirm.Update(in, w, h) != ui.MessageOK {
		return
	}
	v := &a.gate
	e := a.state.Entities[v.npc]
	if e == nil || e.NpcType != 2 || v.index >= 3 || a.shop.npcID != v.npc || a.shop.questMode || a.state.Self.Dead || a.shop.pending || a.inventory.pending ||
		a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}].Gates[v.index] != v.destination {
		a.questMessage("Gatekeeper destination is no longer available.")
		return
	}
	dx, dz := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
	if dx*dx+dz*dz >= 900 {
		a.questMessage("Move closer to the gatekeeper.")
		return
	}
	if err := a.world.Send(proto.GatekeeperPacket(v.npc, v.index)); err != nil {
		a.backToLogin("Unable to request travel.")
		return
	}
	v.pending = true
	a.shop.pending, a.shop.deadline = true, a.clock+10
}
func (a *App) confirmedGatekeeper(p []byte) {
	v := &a.gate
	if !v.pending || proto.Op(p) != proto.Gatekeeper {
		return
	}
	r, err := proto.DecodeGatekeeperResult(p)
	if err != nil {
		return
	}
	v.pending, a.shop.pending = false, false
	if r.Result == 0 {
		a.questMessage("Gatekeeper refused travel.")
		return
	}
	if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
		a.backToLogin("Unable to close gatekeeper conversation.")
		return
	}
	a.shop.npcID = 0
}
func (a *App) drawGatekeeper(w, h int, e *game.Entity, def sdata.NPC) {
	portrait := a.assets.Texture(fmt.Sprintf("interface/NpcFace/%d.png", def.Model))
	a.gateUI.Currency = a.msg.Get(0x146a)
	a.gateUI.Draw(a.sprites, a.font, w, h, def.Name, questText(def.Welcome), portrait, def.Gates, a.shop.pending || a.inventory.pending)
}
