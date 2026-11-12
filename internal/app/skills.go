package app

import (
	"fmt"
	"log"
	"sort"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type skillsView struct {
	open, pending       bool
	drag                shortcutDrag
	assignMode, editing bool
	assign              *proto.QuickSlot
	cooldowns           map[uint16]float64
	offset              int
	category            uint8
	deadline            float64
}

func (a *App) loadSkills() {
	if a.skillDefs != nil {
		return
	}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{}
	b, err := a.arc.ReadFile("Character/Skill.SData")
	if err == nil {
		var defs map[sdata.SkillKey]sdata.Skill
		defs, err = sdata.ParseSkills(b)
		if err == nil {
			a.skillDefs = defs
		}
	}
	if err != nil {
		log.Printf("Skill.SData: %v", err)
	}
}

func (a *App) availableSkills() []sdata.SkillKey {
	a.loadSkills()
	var own proto.CharacterSlot
	for _, c := range a.state.Slots {
		if c.ID == a.state.SelfID {
			own = c
		}
	}
	learned := map[uint16]uint8{}
	for _, k := range a.state.Skills {
		learned[k.ID] = max(learned[k.ID], k.Level)
	}
	var keys []sdata.SkillKey
	for key, d := range a.skillDefs {
		if key.Level != learned[key.ID]+1 || key.Level > 3 || d.Show == 0 || own.Class >= 6 || d.Classes[own.Class] == 0 {
			continue
		}
		// Presentation filter from EnableLearnSkill/Rapture; server validates requests.
		country := d.Country
		allowed := country > 5 || country == 0 && own.Race == 0 || country == 1 && own.Race == 1 || country == 2 && own.Race < 2 || country == 3 && own.Race == 3 || country == 4 && own.Race == 2 || country == 5 && own.Race >= 2
		if !allowed {
			continue
		}
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].ID < keys[j].ID })
	return keys
}

// skillBar uses server assignments when present; otherwise the first ten
// active learned skills are a temporary, deterministic bar.
func (a *App) skillBar() [ui.TotalActionSlots]*proto.LearnedSkill {
	a.loadSkills()
	var bar [ui.TotalActionSlots]*proto.LearnedSkill
	for _, q := range a.state.QuickSlots {
		if q.Bar < ui.ActionBars && q.Slot < ui.ActionSlots && q.Bag == 100 {
			if k, ok := a.state.Skills[uint8(q.Number)]; ok && q.Number < 256 {
				cp := k
				bar[int(q.Bar)*ui.ActionSlots+int(q.Slot)] = &cp
			}
		}
	}
	if a.state.SkillBarReady || len(a.state.QuickSlots) > 0 {
		return bar
	}
	numbers := make([]int, 0, len(a.state.Skills))
	for n := range a.state.Skills {
		numbers = append(numbers, int(n))
	}
	sort.Ints(numbers)
	i := 0
	for _, n := range numbers {
		k := a.state.Skills[uint8(n)]
		if d, ok := a.skillDefs[sdata.SkillKey{ID: k.ID, Level: k.Level}]; ok && d.Effect == 0 {
			continue
		}
		cp := k
		bar[i] = &cp
		i++
		if i == ui.ActionSlots {
			break
		}
	}
	return bar
}

func (a *App) updateSkills(in *platform.Input, w, h int) bool {
	v := &a.skills
	for _, r := range in.Chars {
		if r == 'k' || r == 'K' {
			a.toggleSkills(in.Shift)
			return true
		}
	}
	x, y := a.skillBook.Origin(w, h)
	captured := v.open && panelPointer(in, x, y, ui.SkillBookWidth, ui.SkillBookHeight)
	if captured {
		a.actionBar.Hover = -1
	}
	if !captured && a.updateActions(in, w, h) {
		return true
	}
	if !v.open {
		return false
	}
	if in.HasKey(platform.KeyEscape) {
		v.open = false
		v.assign = nil
		return true
	}
	keys := a.visibleSkills()
	if in.Pressed {
		if row, ok := ui.SkillBookHit(in.MouseX-x, in.MouseY-y); ok && v.offset+row < len(keys) {
			if k, ok := a.learnedSkill(keys[v.offset+row].ID); ok && a.skillDefs[sdata.SkillKey{ID: k.ID, Level: k.Level}].Effect != 0 {
				v.drag = shortcutDrag{tracking: true, x: in.MouseX, y: in.MouseY, from: -1, reference: proto.QuickSlot{Bag: 100, Number: uint16(k.Number)}, skill: k}
			}
		}
	}
	enabled := a.learnableRows(keys)
	action := a.skillBook.Update(in, w, h, int(v.category), v.offset, len(keys), enabled)
	if action.Close {
		v.open = false
		v.assign = nil
		return true
	}
	if uint8(action.Category) != v.category {
		v.assign = nil
	}
	v.category, v.offset = uint8(action.Category), action.Offset
	if a.state.Self.Dead || v.pending {
		return captured
	}
	if v.assignMode && action.Select >= 0 {
		index := v.offset + action.Select
		if index < len(keys) {
			key := keys[index]
			for _, k := range a.state.Skills {
				if k.ID == key.ID {
					v.assign = &proto.QuickSlot{Bag: 100, Number: uint16(k.Number)}
					break
				}
			}
		}
	} else if action.Learn < 0 && action.Select >= 0 && v.offset+action.Select < len(keys) {
		if k, ok := a.learnedSkill(keys[v.offset+action.Select].ID); ok && a.skillDefs[sdata.SkillKey{ID: k.ID, Level: k.Level}].Effect != 0 && a.world != nil {
			if err := a.world.Send(a.learnedSkillPacket(k)); err != nil {
				a.backToLogin("Unable to use skill.")
				return true
			}
		}
	} else if action.Learn >= 0 {
		index := v.offset + action.Learn
		if index < len(keys) && a.canLearn(keys[index]) {
			key := keys[index]
			if err := a.world.Send(proto.LearnSkillPacket(key.ID, key.Level)); err != nil {
				a.backToLogin("Unable to learn skill.")
				return true
			}
			v.pending, v.deadline = true, a.clock+10
		}
	}

	return captured
}

func (a *App) updateActions(in *platform.Input, w, h int) bool {
	if a.state.Self.Dead {
		return false
	}
	pointer := *in
	if a.barCovered(in, w, h) {
		pointer.MouseX, pointer.MouseY = -10000, -10000
		pointer.Pressed, pointer.Clicked, pointer.MouseDown = false, false, false
		pointer.RightPressed, pointer.RightDown = false, false
		pointer.Wheel = 0
	}
	click, captured := a.actionBar.Update(&pointer, w, h)
	handled := captured
	for _, index := range in.Actions {
		if index >= ui.ActionSlots && index < ui.TotalActionSlots {
			a.useAction(index)
			handled = true
			if a.state == nil {
				return true
			}
		}
	}
	if click >= 0 {
		a.useAction(click)
		return true
	}
	keypad := in.KeypadChars
	for _, r := range in.Chars {
		if len(keypad) > 0 && r == rune(keypad[0]) {
			keypad = keypad[1:]
			continue
		}
		index := int(r - '1')
		if r == '0' {
			index = 9
		}
		if index >= 0 && index < ui.ActionSlots {
			handled = true
			a.useAction(index)
			if a.state == nil {
				return true
			}
		}
	}
	return handled
}

func (a *App) drawSkills(w, h int) {
	a.drawActionBar(w, h)
	if !a.skills.open {
		return
	}
	x, y := a.skillBook.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.SkillBookWidth, ui.SkillBookHeight)
	keys := a.visibleSkills()
	a.skills.offset = max(0, min(a.skills.offset, max(0, ((len(keys)+1)/2-9)*2)))
	a.skillBook.Draw(a.sprites, a.font, w, h, int(a.skills.category), a.skills.offset, len(keys), a.state.Details.SkillPoint, a.learnableRows(keys))
	for row := 0; row < ui.SkillBookRows && a.skills.offset+row < len(keys); row++ {
		key := keys[a.skills.offset+row]
		d := a.skillDefs[key]
		cx, cy, _ := ui.SkillBookRow(row)
		iconDef := d
		for _, learned := range a.state.Skills {
			if learned.ID == key.ID {
				iconDef = a.skillDefs[sdata.SkillKey{ID: key.ID, Level: learned.Level}]
				break
			}
		}
		if file, uv, ok := ui.SkillIcon(iconDef.Icon); ok {
			if tex := a.assets.Texture("interface/" + file); tex != nil {
				tint := white
				if _, known := a.learnedSkill(key.ID); !known {
					tint = [4]float32{.45, .45, .45, .65}
				}
				a.sprites.Draw(tex, float32(x+cx), float32(y+cy), 32, 32, uv.U0, uv.V0, uv.U1, uv.V1, tint)
			}
		}
		name := iconDef.Name
		if q := a.skills.assign; q != nil && a.state.Skills[uint8(q.Number)].ID == key.ID {
			name = "> " + name
		}
		known := false
		if _, ok := a.learnedSkill(key.ID); ok {
			known = true
		}
		nameY, nameTint := y+cy+1, gray
		if known {
			nameY, nameTint = y+cy+8, white
		}
		a.font.DrawScaled(a.sprites, name, float32(x+cx+41), float32(nameY), 11.0/13, nameTint, 96)
		detail := fmt.Sprintf("Lv %d | %d pts", d.RequiredLevel, d.Points)
		tint := white
		if a.skills.assignMode {
			detail = fmt.Sprintf("Skill Lv %d", key.Level)
		} else if !a.canLearn(key) {
			tint = gray
		}
		if _, learned := a.learnedSkill(key.ID); !learned {
			a.font.DrawScaled(a.sprites, detail, float32(x+cx+41), float32(y+cy+20), 10.0/13, tint, 104)
		}
	}

	text := "Drag learned skills to an action bar."
	if a.state.LastLearn.ID != 0 {
		text = "Skill learned."
	}
	switch a.state.LastLearn.Result {
	case 1:
		text = "Not enough skill points."
	case 2:
		text = "Your level is too low."
	case 3:
		text = "This skill is not available for this character."
	}
	if a.skills.pending {
		text = "Waiting for server..."
	}
	if a.skills.assignMode {
		text = "Drag learned skills to an action bar."
	}
	a.font.Draw(a.sprites, text, x+32, y+ui.SkillBookHeight+4, white, 301)
	if row, ok := ui.SkillBookHit(a.lastMouseX-x, a.lastMouseY-y); ok && a.skills.offset+row < len(keys) {
		key := keys[a.skills.offset+row]
		if k, ok := a.learnedSkill(key.ID); ok {
			key.Level = k.Level
		}
		a.queueSkillTooltip(key)
	}

}

// Both keyboard and mouse resolve the same server-assigned slot.
func (a *App) actionPacket(index int) []byte {
	if index < 0 || index >= ui.TotalActionSlots || a.state.Self.Dead {
		return nil
	}
	for _, q := range a.state.QuickSlots {
		if int(q.Bar)*ui.ActionSlots+int(q.Slot) == index && q.Bag <= 5 && q.Number < 24 {
			if a.inventory.pending || a.shop.pending {
				return nil
			}
			if _, ok := a.state.Inventory[[2]uint8{q.Bag, uint8(q.Number)}]; !ok {
				return nil
			}
			return proto.UseItemPacket(q.Bag, uint8(q.Number))
		}
	}
	if k := a.skillBar()[index]; k != nil {
		return a.learnedSkillPacket(*k)
	}
	return nil
}

func (a *App) learnedSkillPacket(k proto.LearnedSkill) []byte {
	target := a.state.Target
	def := a.skillDefs[sdata.SkillKey{ID: k.ID, Level: k.Level}]
	if def.Target == 2 {
		target = a.state.SelfID
	}
	userTarget := target == a.state.SelfID || a.state.Party.Member(target) != nil
	if e := a.state.Entities[target]; e != nil && e.Kind == game.KindUser {
		userTarget = true
	}
	return proto.UseSkillPacket(k.Number, target, userTarget)
}
func (a *App) learnedSkill(id uint16) (proto.LearnedSkill, bool) {
	for _, k := range a.state.Skills {
		if k.ID == id {
			return k, true
		}
	}
	return proto.LearnedSkill{}, false
}
func (a *App) drawSkillIcon(k proto.LearnedSkill, x, y int) {
	d := a.skillDefs[sdata.SkillKey{ID: k.ID, Level: k.Level}]
	if file, uv, ok := ui.SkillIcon(d.Icon); ok {
		if tex := a.assets.Texture("interface/" + file); tex != nil {
			a.sprites.Draw(tex, float32(x), float32(y), 32, 32, uv.U0, uv.V0, uv.U1, uv.V1, white)
		}
	}
}

func (a *App) useAction(index int) {
	if a.world == nil {
		return
	}
	if action, ok := a.localAction(index); ok {
		a.activateSubAction(action)
		return
	}
	if !a.state.Self.Dead && !a.inventory.pending && !a.shop.pending {
		for _, q := range a.state.QuickSlots {
			if int(q.Bar)*ui.ActionSlots+int(q.Slot) == index && q.Bag > 0 && q.Bag <= 5 && q.Number < 24 && a.openAppearanceEditor([2]uint8{q.Bag, uint8(q.Number)}) {
				return
			}
		}
	}
	if p := a.actionPacket(index); p != nil {
		if err := a.world.Send(p); err != nil {
			a.backToLogin("Unable to use action.")
		}
	}
}

func (a *App) drawActionBar(w, h int) {
	var slots [ui.TotalActionSlots]ui.ActionSlot
	for i, k := range a.skillBar() {
		if k == nil {
			continue
		}
		d := a.skillDefs[sdata.SkillKey{ID: k.ID, Level: k.Level}]
		slot := &slots[i]
		slot.Cooldown = max(0, a.skills.cooldowns[k.ID]-a.clock)
		slot.Label, slot.Level = fmt.Sprintf("%s Lv%d | MP %d SP %d", d.Name, k.Level, d.MP, d.SP), k.Level
		if file, uv, ok := ui.SkillIcon(d.Icon); ok {
			slot.Icon, slot.UV = a.assets.Texture("interface/"+file), uv
		}
	}
	for _, q := range a.state.QuickSlots {
		if q.Bar >= ui.ActionBars || q.Slot >= ui.ActionSlots || q.Bag > 5 || q.Number >= 24 {
			continue
		}
		it := a.state.Inventory[[2]uint8{q.Bag, uint8(q.Number)}]
		slot := &slots[int(q.Bar)*ui.ActionSlots+int(q.Slot)]
		slot.Label, slot.Count = "Empty", it.Count
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			slot.Label = fmt.Sprintf("%s x%d", d.Name, it.Count)
			if file, uv, ok := ui.ItemIcon(it.Type, d.Icon); ok {
				slot.Icon, slot.UV = a.assets.Texture("interface/"+file), uv
			}
		}
	}
	for i := range slots {
		if action, ok := a.localAction(i); ok {
			u, v := float32(action%4)/4, float32(action/4)/4
			slots[i] = ui.ActionSlot{Icon: a.assets.Texture("interface/icon/icon_sub.tga"), UV: ui.UV{U0: u, V0: v, U1: u + .25, V1: v + .25}, Label: subActionNames[action]}
		}
	}
	a.actionBar.Draw(a.sprites, a.font, a.white, w, h, slots, false)
	a.queueActionTooltip(w, h)
}
