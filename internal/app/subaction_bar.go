package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Rapture rejects social-action quickslots. Keep these UI-only shortcuts on
// the local machine; items and skills remain in the server's quickslot list.
type subActionDrag struct {
	tracking, active             bool
	source, x, y, mouseX, mouseY int
}

var subActionNames = [15]string{"Sit / Stand", "Walk / Run", "Auto Run", "Pick Up", "Attack", "Victory", "Beg", "Love", "Laugh", "Clap", "Greet", "Start", "Forgive", "Provoke", "Insult"}

type localActionStore map[string][ui.TotalActionSlots]uint8

func localActionsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "shaiya-go", "local-actions.json"), nil
}
func (a *App) localActionsKey() string {
	return fmt.Sprintf("%s:%d/%d", a.cfg.LoginAddr, a.cfg.WorldPort, a.state.SelfID)
}
func readLocalActions(path string) (localActionStore, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return localActionStore{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m localActionStore
	if err = json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = localActionStore{}
	}
	return m, nil
}
func (a *App) loadLocalActions() {
	if a.wv == nil || a.state == nil || a.state.SelfID == 0 {
		return
	}
	path, err := localActionsPath()
	if err != nil {
		log.Printf("load local actions: %v", err)
		return
	}
	m, err := readLocalActions(path)
	if err != nil {
		log.Printf("load local actions: %v", err)
		return
	}
	slots := m[a.localActionsKey()]
	for i, v := range slots {
		if v <= 15 {
			a.wv.localActions[i] = v
		}
	}
	a.reconcileLocalActions()
}

// A server assignment takes precedence and permanently removes a conflicting
// local shortcut so it cannot unexpectedly return if that slot is later cleared.
func (a *App) reconcileLocalActions() {
	if a.wv == nil || a.state == nil || !a.state.SkillBarReady {
		return
	}
	changed := false
	for _, q := range a.state.QuickSlots {
		if q.Bar < ui.ActionBars && q.Slot < ui.ActionSlots && a.wv.localActions[int(q.Bar)*ui.ActionSlots+int(q.Slot)] != 0 {
			a.wv.localActions[int(q.Bar)*ui.ActionSlots+int(q.Slot)] = 0
			changed = true
		}
	}
	if changed {
		a.saveLocalActions()
	}
}
func (a *App) saveLocalActions() {
	if a.wv == nil || a.state == nil || a.state.SelfID == 0 {
		return
	}
	path, err := localActionsPath()
	var m localActionStore
	if err == nil {
		m, err = readLocalActions(path)
	}
	if err == nil {
		m[a.localActionsKey()] = a.wv.localActions
		err = os.MkdirAll(filepath.Dir(path), 0700)
	}
	if err == nil {
		var data []byte
		data, err = json.Marshal(m)
		if err == nil {
			var f *os.File
			f, err = os.CreateTemp(filepath.Dir(path), ".local-actions-*")
			if err == nil {
				defer os.Remove(f.Name())
				_, err = f.Write(data)
				if e := f.Close(); err == nil {
					err = e
				}
				if err == nil {
					err = os.Rename(f.Name(), path)
				}
			}
		}
	}
	if err != nil {
		log.Printf("save local actions: %v", err)
	}
}
func (a *App) localAction(index int) (int, bool) {
	if a.wv == nil || a.state == nil || index < 0 || index >= len(a.wv.localActions) {
		return 0, false
	}
	v := a.wv.localActions[index]
	if v == 0 || v > 15 {
		return 0, false
	}
	for _, q := range a.state.QuickSlots {
		if int(q.Bar)*ui.ActionSlots+int(q.Slot) == index {
			return 0, false
		}
	}
	return int(v - 1), true
}
func (a *App) updateLocalActionDrag(in *platform.Input, w, h int) bool {
	if a.wv == nil {
		return false
	}
	d := &a.wv.localActionDrag
	if !d.tracking {
		return false
	}
	if !a.subActionsOpen || a.state == nil || a.state.Self.Dead {
		*d = subActionDrag{}
		return false
	}
	if !d.active && (absInt(in.MouseX-d.x) > 4 || absInt(in.MouseY-d.y) > 4) && (in.MouseDown || in.Clicked) {
		d.active = true
		a.subActionsUI.CancelHeld()
		a.wv.uiLeft = -2
	}
	d.mouseX, d.mouseY = in.MouseX, in.MouseY
	if !d.active {
		if in.Clicked || !in.MouseDown {
			*d = subActionDrag{}
		}
		return false
	}
	if in.Clicked || !in.MouseDown {
		if in.Clicked {
			if index := a.actionBar.SlotAt(in.MouseX, in.MouseY, w, h); index >= 0 && a.world != nil && a.state.SkillBarReady {
				slots := replaceAction(a.state.QuickSlots, uint8(index), nil)
				p, err := proto.SkillBarPacket(slots)
				if err == nil {
					err = a.world.Send(p)
				}
				if err != nil {
					*d = subActionDrag{}
					a.backToLogin("Unable to save action bar.")
					return true
				}
				a.state.QuickSlots = slots
				a.state.SkillBarReady = true
				a.wv.localActions[index] = uint8(d.source + 1)
				a.saveLocalActions()
				a.playUI()
			}
		}
		*d = subActionDrag{}
	}
	return true
}

func (a *App) drawLocalActionDrag() {
	if a.wv == nil || !a.wv.localActionDrag.active {
		return
	}
	d := a.wv.localActionDrag
	if tex := a.assets.Texture("interface/icon/icon_sub.png"); tex != nil {
		u, v := float32(d.source%4)/4, float32(d.source/4)/4
		a.sprites.Draw(tex, float32(d.mouseX-16), float32(d.mouseY-16), 32, 32, u, v, u+.25, v+.25, white)
	}
}
