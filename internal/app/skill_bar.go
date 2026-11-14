package app

import (
	"fmt"
	"sort"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) visibleSkills() []sdata.SkillKey {
	a.loadSkills()
	keys := a.availableSkills()
	if a.skills.assignMode {
		keys = nil
	}
	present := map[uint16]bool{}
	for _, key := range keys {
		present[key.ID] = true
	}
	for _, k := range a.state.Skills {
		key := sdata.SkillKey{ID: k.ID, Level: k.Level}
		if d, ok := a.skillDefs[key]; ok && !present[key.ID] && (!a.skills.assignMode || d.Effect != 0) {
			keys = append(keys, key)
			present[key.ID] = true
		}
	}
	filtered := keys[:0]
	for _, key := range keys {
		if a.skillDefs[key].Show == a.skills.category+1 {
			filtered = append(filtered, key)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })
	return filtered
}

// Presentation only. The server remains authoritative for learning rules.
func (a *App) canLearn(key sdata.SkillKey) bool {
	if a.skills.pending || a.state.Self.Dead {
		return false
	}
	d, ok := a.skillDefs[key]
	own, hasOwn := a.selfSlot()
	if !ok || !hasOwn || key.Level > 3 || d.RequiredLevel == 0 || d.RequiredLevel > own.Level || uint16(d.Points) > a.state.Details.SkillPoint {
		return false
	}
	for _, k := range a.state.Skills {
		if k.ID == key.ID && k.Level >= key.Level {
			return false
		}
	}
	switch own.Mode {
	case 0:
		return d.Grow == 0
	case 1:
		return d.Grow <= 1
	case 2:
		return d.Grow != 3
	}
	return true
}

func replaceAction(slots []proto.QuickSlot, index uint8, source *proto.QuickSlot) []proto.QuickSlot {
	out := make([]proto.QuickSlot, 0, len(slots)+1)
	for _, q := range slots {
		if int(q.Bar)*ui.ActionSlots+int(q.Slot) != int(index) {
			out = append(out, q)
		}
	}
	if source != nil {
		q := *source
		q.Bar, q.Slot = index/ui.ActionSlots, index%ui.ActionSlots
		out = append(out, q)
	}
	return out
}

// Rapture persists 0x010B without an acknowledgement. Show the requested bar
// after a successful send; the next character login replaces it with server data.
func (a *App) updateBarEditing(in *platform.Input, w, h int) bool {
	v := &a.skills
	if a.actionBar.LayoutActive() || !a.barCovered(in, w, h) {
		if captured, changed := a.actionBar.Layout(in, w, h); captured {
			if changed {
				a.saveUILayout()
			}
			return true
		}
	}
	if a.barCovered(in, w, h) {
		a.actionBar.Hover = -1
		return false
	}
	indexAt := a.actionBar.SlotAt(in.MouseX, in.MouseY, w, h)

	if in.Pressed && indexAt >= 0 && v.assign == nil && !in.Shift {
		q, local := a.shortcutAt(indexAt)
		if q != nil || local != 0 {
			v.drag = shortcutDrag{tracking: true, x: in.MouseX, y: in.MouseY, from: indexAt, local: local}
			if q != nil {
				v.drag.reference = *q
			}
		}
	}
	if a.state.Self.Dead || (!in.Shift && v.assign == nil && !v.editing) {
		return false
	}
	index, captured := a.actionBar.Update(in, w, h)
	if in.Pressed && captured {
		v.editing = true
	}
	if in.Clicked {
		v.editing = false
	}
	if index < 0 {
		return captured
	}
	// 0x010B replaces the complete server bar. Never send a partial snapshot.
	if !a.state.SkillBarReady {
		return true
	}
	source := v.assign
	if source == nil && in.Shift && a.inventory.open && a.inventory.selected {
		key := a.inventory.source
		if _, ok := a.state.Inventory[key]; ok {
			source = &proto.QuickSlot{Bag: key[0], Number: uint16(key[1])}
		}
	}
	slots := replaceAction(a.state.QuickSlots, uint8(index), source)
	p, err := proto.SkillBarPacket(slots)
	if err == nil && a.world != nil {
		err = a.world.Send(p)
	}
	if err != nil {
		a.backToLogin(fmt.Sprintf("Unable to save action bar: %v", err))
		return true
	}
	if a.world != nil {
		a.state.QuickSlots = slots
		a.state.SkillBarReady = true
		if a.wv != nil && a.wv.localActions[index] != 0 {
			a.wv.localActions[index] = 0
			a.saveLocalActions()
		}
	}
	v.assign = nil
	return true
}

func (a *App) learnableRows(keys []sdata.SkillKey) (enabled [ui.SkillBookRows]bool) {
	for row := range enabled {
		index := a.skills.offset + row
		if index < len(keys) {
			enabled[row] = !a.skills.assignMode && a.canLearn(keys[index])
		}
	}
	return
}
