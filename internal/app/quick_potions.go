package app

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type quickPotionView struct {
	dragging       bool
	pressX, pressY int
	Position       ui.WindowPosition
	bindings       [2]sdata.ItemKey
	profile        string
	held           int
}

func (a *App) potionConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "shaiya-go", "quick-potions.json")
}
func (a *App) ensureQuickPotions() {
	if a.state == nil || a.wv == nil {
		return
	}
	v := &a.quickPotions
	profile := a.localActionsKey()
	if v.profile == profile {
		return
	}
	v.profile, v.bindings, v.held = profile, [2]sdata.ItemKey{}, -1
	data, err := os.ReadFile(a.potionConfigPath())
	if err != nil {
		return
	}
	var saved map[string][2]sdata.ItemKey
	if json.Unmarshal(data, &saved) == nil {
		v.bindings = saved[profile]
	}
}
func (a *App) saveQuickPotions() {
	file := a.potionConfigPath()
	if file == "" {
		return
	}
	saved := map[string][2]sdata.ItemKey{}
	if data, err := os.ReadFile(file); err == nil {
		if err = json.Unmarshal(data, &saved); err != nil {
			log.Printf("quick potions: %v", err)
			return
		}
	}
	if saved == nil {
		saved = map[string][2]sdata.ItemKey{}
	}
	saved[a.quickPotions.profile] = a.quickPotions.bindings
	data, err := json.Marshal(saved)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(file), 0700)
	}
	if err == nil {
		var f *os.File
		f, err = os.CreateTemp(filepath.Dir(file), ".quick-potions-*")
		if err == nil {
			defer os.Remove(f.Name())
			_, err = f.Write(data)
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				err = os.Rename(f.Name(), file)
			}
		}
	}
	if err != nil {
		log.Printf("quick potions: %v", err)
	}
}
func (a *App) quickPotionItem(cell int) ([2]uint8, proto.InventoryItem, bool) {
	binding := a.quickPotions.bindings[cell]
	best := [2]uint8{255, 255}
	var item proto.InventoryItem
	for key, it := range a.state.Inventory {
		if (key[0] == 0 || key[0] > 5) || it.Count == 0 || it.Type != binding.Type || it.TypeID != binding.TypeID {
			continue
		}
		if key[0] < best[0] || key[0] == best[0] && key[1] < best[1] {
			best, item = key, it
		}
	}
	return best, item, best[0] != 255
}
func (a *App) useQuickPotion(cell int) {
	if key, _, ok := a.quickPotionItem(cell); ok {
		a.useInventoryItem(key)
	}
}
func (a *App) updateQuickPotionDrop(in *platform.Input, w, h int) bool {
	a.ensureQuickPotions()
	v := &a.quickPotions
	if in.HasKey(platform.KeyEscape) || !in.MouseDown && !in.Clicked {
		v.dragging, v.held = false, -1
	}
	if v.held >= 0 && (in.MouseDown || in.Clicked) {
		if absInt(in.MouseX-v.pressX) > 4 || absInt(in.MouseY-v.pressY) > 4 {
			v.dragging = true
		}
		if v.dragging {
			a.wv.uiLeft = -2
			if in.Clicked {
				x, y := ui.QuickPotionOrigin(v.Position, w, h)
				cell := ui.QuickPotionCell(in.MouseX, in.MouseY, x, y)
				if cell < 0 {
					v.bindings[v.held] = sdata.ItemKey{}
					a.saveQuickPotions()
				} else if cell != v.held {
					v.bindings[cell], v.bindings[v.held] = v.bindings[v.held], v.bindings[cell]
					a.saveQuickPotions()
				}
				v.dragging, v.held = false, -1
			}
			return true
		}
	}

	if !a.inventory.dragging || !in.Clicked {
		return false
	}
	x, y := ui.QuickPotionOrigin(a.quickPotions.Position, w, h)
	if ui.QuickPotionCell(in.MouseX, in.MouseY, x, y) < 0 {
		return false
	}
	return a.updateQuickPotions(in, w, h)
}
func (a *App) updateQuickPotions(in *platform.Input, w, h int) bool {
	if a.wv == nil || a.state == nil {
		return false
	}
	a.ensureQuickPotions()
	x, y := ui.QuickPotionOrigin(a.quickPotions.Position, w, h)
	cell := ui.QuickPotionCell(in.MouseX, in.MouseY, x, y)
	if a.inventory.dragging && in.Clicked && cell >= 0 && !a.barCovered(in, w, h) {
		if it, ok := a.state.Inventory[a.inventory.source]; ok && quickPotionType(it.Type) {
			a.quickPotions.bindings[cell] = sdata.ItemKey{Type: it.Type, TypeID: it.TypeID}
			a.saveQuickPotions()
		}
		a.inventory.dragging, a.inventory.selected = false, false
		return true
	}
	handled := false
	for _, r := range in.Chars {
		if r == 'z' || r == 'Z' {
			a.useQuickPotion(0)
			handled = true
		}
		if r == 'x' || r == 'X' {
			a.useQuickPotion(1)
			handled = true
		}
	}
	if a.barCovered(in, w, h) {
		return handled
	}
	if !pointInside(in, x, y, 94, 54) {
		return handled
	}

	if in.Pressed {
		a.quickPotions.held = cell
		a.quickPotions.pressX, a.quickPotions.pressY = in.MouseX, in.MouseY
	}
	if in.Clicked {
		if cell >= 0 && a.quickPotions.held == cell {
			a.useQuickPotion(cell)
		}
		a.quickPotions.held = -1
	}
	return handled || in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.Wheel != 0
}
func (a *App) drawQuickPotions(w, h int) {
	a.ensureQuickPotions()
	var slots [2]ui.ActionSlot
	for i, binding := range a.quickPotions.bindings {
		if binding.Type == 0 {
			continue
		}
		d := a.itemsItem(binding.Type, binding.TypeID)
		if d == nil {
			continue
		}
		if file, uv, ok := ui.ItemIcon(binding.Type, d.Icon); ok {
			slots[i].Icon, slots[i].UV = a.assets.Texture("interface/"+file), uv
		}
		if _, it, ok := a.quickPotionItem(i); ok {
			slots[i].Count = it.Count
			slots[i].Cooldown = a.itemCooldown(it)
			slots[i].CooldownDuration = itemReuseSeconds(d.ReqIg)
			slots[i].ReadyFlash = cooldownFlash(a.inventory.cooldowns[d.ReqIg], a.clock)
		}
	}
	x, y := ui.QuickPotionOrigin(a.quickPotions.Position, w, h)
	ui.DrawQuickPotions(a.sprites, a.font, a.assets.Texture("interface/QuickPotion.png"), a.white, x, y, slots)
	if cell := ui.QuickPotionCell(a.lastMouseX, a.lastMouseY, x, y); cell >= 0 && !a.barCovered(&platform.Input{MouseX: a.lastMouseX, MouseY: a.lastMouseY}, w, h) {
		binding := a.quickPotions.bindings[cell]
		if d := a.itemsItem(binding.Type, binding.TypeID); d != nil {
			_, it, owned := a.quickPotionItem(cell)
			if owned {
				a.drawItemDetails(d, &it, it.Count, min(a.lastMouseX+18, w-296), max(0, y-200), w, h, "")
			}
		}
	}

}

func quickPotionType(kind uint8) bool {
	switch kind {
	case 25, 44, 100, 101, 102:
		return true
	}
	return false
}
