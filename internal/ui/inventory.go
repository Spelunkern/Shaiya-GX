package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const InventoryWidth, InventoryHeight = 256, 486

// Original slots: Game.exe 0x4f6750, table 0x70a2f8 (14 position pairs).
var equipmentCells = [...][2]int{{24, 53}, {24, 93}, {24, 133}, {24, 173}, {24, 213}, {196, 133}, {196, 173}, {196, 213}, {191, 53}, {191, 77}, {215, 77}, {191, 101}, {215, 101}, {215, 53}}

// InventoryOrigin is the default anchor before the player moves the frame.
func InventoryOrigin(w, h int) (int, int) {
	return max(0, w-InventoryWidth-16), min(82, max(0, h-InventoryHeight))
}

func InventoryCell(bag, slot uint8) (x, y, size int, ok bool) {
	if bag == 0 {
		if int(slot) >= len(equipmentCells) {
			return
		}
		p := equipmentCells[slot]
		size = 32
		if slot >= 8 {
			size = 16
		}
		return p[0], p[1], size, true
	}
	if bag > 5 || slot >= 24 {
		return
	}
	return 15 + int(slot%6)*39, 283 + int(slot/6)*39, 32, true
}
func InventoryHit(mx, my int, bag uint8) ([2]uint8, bool) {
	for _, b := range []uint8{0, bag} {
		for slot := uint8(0); slot < 24; slot++ {
			x, y, size, ok := InventoryCell(b, slot)
			if ok && mx >= x && mx < x+size && my >= y && my < y+size {
				return [2]uint8{b, slot}, true
			}
		}
	}
	return [2]uint8{}, false
}

// Inventory uses 0x4f4210's original atlases and 0x4f6750's grid/gold layout.
// Close art is the common native 22px control, also used by the Phoenix reference.
type Inventory struct {
	Position    WindowPosition
	frame, bag  *render.Texture
	tabs        [2]*render.Texture
	close       Button
	hover, held int
}

func NewInventory(t Textures) (*Inventory, error) {
	ld := &loader{t: t}
	v := &Inventory{frame: ld.get("inven.png"), bag: ld.get("inven_bag.png"), hover: -1, held: -1}
	v.tabs = [2]*render.Texture{ld.get("inven_button1.png"), ld.get("inven_button2.png")}
	v.close = nativeClose(ld.get("talk1_close_button.png"), 231, 8)
	return v, ld.err
}
func (v *Inventory) Update(in *platform.Input, w, h int, bag uint8) (uint8, bool) {
	x, y := v.Origin(w, h)
	v.close.OriginX, v.close.OriginY = x, y
	closed := v.close.Update(in)
	v.hover = InventoryBagTab(in.MouseX-x, in.MouseY-y)
	if in.Pressed {
		v.held = v.hover
	}
	if in.Clicked {
		if v.hover >= 0 && v.held == v.hover {
			bag = uint8(v.hover + 1)
		}
		v.held = -1
	}
	return bag, closed
}
func (v *Inventory) Draw(s *render.Sprites, f *Font, w, h int, bag uint8, money uint32, message string) {
	x, y := v.Origin(w, h)
	s.Draw(v.frame, float32(x), float32(y), 256, 486, 0, 0, 1, 486.0/512, white)
	for i := 0; i < 5; i++ {
		selected := uint8(i+1) == bag
		sheet, dy, gx, gu := 1, 257, 16, float32(.5)
		if selected {
			sheet, dy, gx, gu = 0, 251, 14, 0
		}
		u := float32(0)
		if i == v.hover {
			u = .5
		}
		s.Draw(v.tabs[sheet], float32(x+11+i*47), float32(y+dy), 64, 32, u, float32(i)/8, u+.5, float32(i+1)/8, white)
		s.Draw(v.bag, float32(x+11+i*47+gx), float32(y+dy), 32, 32, gu, 0, gu+.5, 1, white)
	}
	v.close.OriginX, v.close.OriginY = x, y
	v.close.Draw(s)
	text := GoldText(money)
	f.Draw(s, text, x+220-f.Width(text), y+460, goldTint(money), 110)
}

func (v *Inventory) Origin(w, h int) (int, int) {
	x, y := InventoryOrigin(w, h)
	if v == nil {
		return x, y
	}
	return v.Position.Origin(w, h, InventoryWidth, InventoryHeight, x, y)
}

// InventoryBagTab identifies native bag tabs for clicks and held-item drags.
func InventoryBagTab(x, y int) int {
	if y >= 251 && y < 279 && x >= 11 && x < 246 {
		return (x - 11) / 47
	}
	return -1
}
