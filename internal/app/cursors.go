package app

import (
	"fmt"
	"log"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

// LoadCursors keeps the original CUR artwork, dimensions, alpha/AND mask and
// hotspot. No converted game assets are stored in the repository or package.
func (a *App) LoadCursors(window *platform.Window) *platform.Cursors {
	c := platform.NewCursors(window)
	for i := 0; i < 7; i++ {
		path := fmt.Sprintf("Cursor/%02d.cur", i+1)
		b, err := a.arc.ReadFile(path)
		if err == nil {
			var cursor *img.Cursor
			cursor, err = img.DecodeCUR(b)
			if err == nil {
				c.Load(i, cursor.Image.Pix, cursor.Image.W, cursor.Image.H, cursor.HotX, cursor.HotY)
			}
		}
		if err != nil {
			log.Printf("cursor %s: %v", path, err)
		}
	}
	c.Set(0)
	return c
}

// Native 0x41fde0: default, attack, pickup, interaction, drag, talk, aim.
func (a *App) CursorIndex(w, h int) int {
	if a.screen != screenWorld || a.wv == nil || a.state == nil {
		return 0
	}
	if a.skills.drag.active || a.inventory.dragging || a.wv.localActionDrag.active {
		return 4
	}
	in := &platform.Input{MouseX: a.lastMouseX, MouseY: a.lastMouseY, Clicked: true}
	if c := a.playerContext; c.open && pointInside(in, c.x, c.y, c.width, c.height) {
		return 0
	}
	if a.barCovered(in, w, h) || a.menuOpen || a.chatOpen || a.wv.mapView.full {
		return 0
	}
	if a.actionBar != nil && a.actionBar.SlotAt(in.MouseX, in.MouseY, w, h) >= 0 {
		return 3
	}
	if in.MouseY >= h-64 || in.MouseX < 220 && in.MouseY < 100 {
		return 0
	}
	if a.minimap != nil {
		x, y := a.minimap.Origin(w, h)
		if pointInside(in, x, y, 196, 197) {
			return 0
		}
	}
	if a.skills.assign != nil || len(a.wv.approachSkill) > 0 {
		return 6
	}
	if a.pickGroundLabel(in, w, h) != 0 || a.pickGroundModel(in, w, h) != 0 {
		return 2
	}
	if a.pickNPC(in, w, h) != 0 {
		return 5
	}
	if id := a.pickMob(in, w, h); id != 0 {
		if entity := a.state.Entities[id]; entity.Kind == game.KindUser && entity.Shape != nil {
			if own, ok := a.selfSlot(); ok && own.Race/2 == entity.Shape.Race/2 {
				return 0
			}
		}
		return 1
	}
	return 0
}
