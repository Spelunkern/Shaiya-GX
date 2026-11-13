package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const WarehouseWidth, WarehouseHeight, WarehousePageSize = 333, 380, 40

// Bank frame and tabs: ctor 0x45fad0, input 0x45fc50, draw 0x45ffc0.
type Warehouse struct {
	Position    WindowPosition
	frame, bag  *render.Texture
	tabs        [2]*render.Texture
	close       Button
	guildSwitch Button
	hover, held int
}

func NewWarehouse(t Textures) (*Warehouse, error) {
	ld := &loader{t: t}
	v := &Warehouse{frame: ld.get("bank.png"), bag: ld.get("inven_bag.png"), hover: -1, held: -1}
	v.tabs = [2]*render.Texture{ld.get("store_slot_button1.png"), ld.get("store_slot_button2.png")}
	v.close = nativeClose(ld.get("talk1_close_button.png"), 309, 5)
	v.guildSwitch = Button{Tex: ld.get("guild_button2_blank.png"), OffX: 237, OffY: 82, DrawW: 128, DrawH: 32, HitW: 75, HitH: 21, UV: FourRowUV}
	return v, ld.err
}

func (v *Warehouse) UpdateGuildSwitch(in *platform.Input, w, h int, enabled bool) bool {
	x, y := v.Origin(w, h)
	v.guildSwitch.OriginX, v.guildSwitch.OriginY, v.guildSwitch.Disabled = x, y, !enabled
	return v.guildSwitch.Update(in)
}

func (v *Warehouse) DrawGuildSwitch(s *render.Sprites, f *Font, w, h int, enabled, guild bool) {
	x, y := v.Origin(w, h)
	v.guildSwitch.OriginX, v.guildSwitch.OriginY, v.guildSwitch.Disabled = x, y, !enabled
	v.guildSwitch.Draw(s)
	label := "Guild"
	if guild {
		label = "Personal"
	}
	f.Draw(s, label, x+245, y+85, white, 64)
}
func (v *Warehouse) Origin(w, h int) (int, int) {
	x, y := min(32, max(0, w-WarehouseWidth)), min(82, max(0, h-WarehouseHeight))
	if v == nil {
		return x, y
	}
	return v.Position.Origin(w, h, WarehouseWidth, WarehouseHeight, x, y)
}
func WarehouseCell(page int, slot uint8) (int, int, bool) {
	i := int(slot) - page*WarehousePageSize
	if page < 0 || page >= 6 || i < 0 || i >= WarehousePageSize {
		return 0, 0, false
	}
	return 15 + i%8*39, 146 + i/8*39, true
}
func WarehouseHit(x, y, page int) (uint8, bool) {
	for i := 0; i < WarehousePageSize; i++ {
		slot := uint8(page*WarehousePageSize + i)
		cx, cy, ok := WarehouseCell(page, slot)
		if ok && x >= cx && x < cx+32 && y >= cy && y < cy+32 {
			return slot, true
		}
	}
	return 0, false
}
func (v *Warehouse) Update(in *platform.Input, w, h, page int) (int, bool) {
	x, y := v.Origin(w, h)
	v.close.OriginX, v.close.OriginY = x, y
	closed := v.close.Update(in)
	v.hover = -1
	if in.MouseX >= x+11 && in.MouseX < x+323 && in.MouseY >= y+114 && in.MouseY < y+142 {
		v.hover = (in.MouseX - x - 11) / 52
	}
	if in.Pressed {
		v.held = v.hover
	}
	if in.Clicked {
		if v.hover >= 0 && v.held == v.hover {
			page = v.hover
		}
		v.held = -1
	}
	return page, closed
}
func (v *Warehouse) Draw(s *render.Sprites, f *Font, w, h, page int, name, welcome string, portrait *render.Texture) {
	x, y := v.Origin(w, h)
	s.Draw(v.frame, float32(x), float32(y), 512, 512, 0, 0, 1, 1, white)
	if portrait != nil {
		s.Draw(portrait, float32(x+7), float32(y+7), 64, 64, 0, 0, 1, 1, white)
	}
	f.Draw(s, name, x+90, y+22, white, 210)
	f.Draw(s, welcome, x+90, y+46, white, 210)
	for i := 0; i < 6; i++ {
		sheet, dy, gx, gu := 1, 120, 16, float32(.5)
		if i == page {
			sheet, dy, gx, gu = 0, 114, 14, 0
		}
		u := float32(0)
		if i == v.hover {
			u = .5
		}
		s.Draw(v.tabs[sheet], float32(x+11+i*52), float32(y+dy), 64, 32, u, float32(i)/8, u+.5, float32(i+1)/8, white)
		s.Draw(v.bag, float32(x+11+i*52+gx), float32(y+dy), 32, 32, gu, 0, gu+.5, 1, white)
	}
	v.close.OriginX, v.close.OriginY = x, y
	v.close.Draw(s)
}
