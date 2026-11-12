//go:build integration

package app

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestShortcutDragAgainstRapture(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	st, world, send, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state, a.world, a.arc = st, world, arc
	a.skillDefs = nil
	a.loadSkills()
	var skill proto.LearnedSkill
	for _, key := range a.availableSkills() {
		if a.skillDefs[key].Effect != 0 && a.canLearn(key) {
			send(proto.LearnSkillPacket(key.ID, key.Level))
			wait(func() bool { return st.LastLearn.ID == key.ID })
			if st.LastLearn.Result != 0 {
				t.Fatal(st.LastLearn)
			}
			skill = st.Skills[st.LastLearn.Number]
			break
		}
	}
	if skill.ID == 0 {
		t.Fatal("fixture has no learnable active skill")
	}
	a.skills.open = true
	a.skills.category = a.skillDefs[sdata.SkillKey{ID: skill.ID, Level: skill.Level}].Show - 1
	a.skills.offset = 0
	row := -1
	for i, key := range a.visibleSkills() {
		if key.ID == skill.ID {
			row = i
			break
		}
	}
	if row < 0 {
		t.Fatal("learned skill missing in K")
	}
	a.skills.offset = (row / ui.SkillBookRows) * ui.SkillBookRows
	row -= a.skills.offset
	sx, sy := a.skillBook.Origin(1024, 768)
	cx, cy, _ := ui.SkillBookRow(row)
	drag := func(x, y, dx, dy int) {
		t.Helper()
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: dx, MouseY: dy, MouseDown: true}, 1024, 768)
		if !a.skills.drag.active {
			t.Fatal("shortcut did not start dragging")
		}
		a.worldControls(&platform.Input{MouseX: dx, MouseY: dy, Clicked: true}, 1024, 768)
	}
	ax, ay := a.actionBar.SlotOrigin(0, 1024, 768)
	drag(sx+cx+2, sy+cy+2, ax+2, ay+2)
	q, _ := a.shortcutAt(0)
	if q == nil || q.Bag != 100 || q.Number != uint16(skill.Number) {
		t.Fatal("K drop did not bind learned number", q)
	}
	// Both bars remain usable after moving and rotating the destination bar.
	a.skills.open = false
	a.actionBar.Position[1].Vertical = true
	a.actionBar.Position[1].Move(30, 120, 1024, 768, 56, 449)
	bx, by := a.actionBar.SlotOrigin(13, 1024, 768)
	drag(ax+2, ay+2, bx+2, by+2)
	if q, _ := a.shortcutAt(0); q != nil {
		t.Fatal("move left source assigned")
	}
	q, _ = a.shortcutAt(13)
	if q == nil || q.Bag != 100 {
		t.Fatal("second vertical bar rejected drop", q)
	}

	// A filled destination swaps references, including local-only social actions.
	if !a.saveShortcut(1, nil, 11, -1) {
		t.Fatal("could not seed social shortcut")
	}
	lx, ly := a.actionBar.SlotOrigin(1, 1024, 768)
	drag(lx+2, ly+2, bx+2, by+2)
	if q, local := a.shortcutAt(13); q != nil || local != 11 {
		t.Fatal("mixed swap lost local shortcut", q, local)
	}
	if q, _ := a.shortcutAt(1); q == nil || q.Number != uint16(skill.Number) {
		t.Fatal("mixed swap lost learned skill", q)
	}
	drag(bx+2, by+2, lx+2, ly+2)
	if q, _ := a.shortcutAt(13); q == nil || q.Number != uint16(skill.Number) {
		t.Fatal("return swap did not restore skill", q)
	}
	// A drag released in the world keeps its source and sends no use request.
	drag(bx+2, by+2, 500, 300)
	if q, _ := a.shortcutAt(13); q == nil {
		t.Fatal("invalid drop deleted source")
	}
	// The same slot can be invoked through the keypad edge without bar-0 fallback.
	if p := a.actionPacket(13); len(p) == 0 || p[2] != skill.Number {
		t.Fatal("second bar cast lost learned index", p)
	}
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	world.Close()
	persisted, reconnected, _, _ := testworld.Enter(t)
	defer reconnected.Close()
	found := false
	for _, q := range persisted.QuickSlots {
		if q.Bar == 1 && q.Slot == 3 && q.Bag == 100 && q.Number == uint16(skill.Number) {
			found = true
		}
	}
	if !found {
		t.Fatal("second bar assignment was not restored by Rapture", persisted.QuickSlots)
	}
	a.state, a.world = persisted, reconnected
	a.worldControls(&platform.Input{MouseX: bx + 2, MouseY: by + 2, RightPressed: true, RightDown: true}, 1024, 768)
	if q, _ := a.shortcutAt(13); q != nil {
		t.Fatal("right click did not clear shortcut")
	}
	t.Log("K drag, moved vertical second bar, invalid drop, learned-number use, persistence and right-click clear confirmed")
}
