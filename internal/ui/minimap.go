package ui

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const MinimapWidth, MinimapHeight = 196, 197

// Projection follows Phoenix tools/minimap_data.py: raw X grows right, raw
// Z grows north (up). Image and map sizes supply their actual calibration.
func MapPosition(x, z, mapSize float32) (u, v float32, ok bool) {
	if mapSize <= 0 {
		return
	}
	u, v = x/mapSize, 1-z/mapSize
	return u, v, u >= 0 && u <= 1 && v >= 0 && v <= 1
}

func MapWindow(u, v, span float32) UV {
	span = min(1, max(.001, span))
	half := span / 2
	u, v = min(1-half, max(half, u)), min(1-half, max(half, v))
	return UV{u - half, v - half, u + half, v + half}
}

func MapPoint(u, v float32, view UV) (float32, float32, bool) {
	if view.U1 <= view.U0 || view.V1 <= view.V0 {
		return 0, 0, false
	}
	x, y := (u-view.U0)/(view.U1-view.U0), (v-view.V0)/(view.V1-view.V0)
	return x, y, x >= 0 && x <= 1 && y >= 0 && y <= 1
}

// Anchoring is responsive delivery placement; native artwork is unscaled here.
func MinimapOrigin(w int) (int, int) { return max(0, w-210), 14 }

// Native full-map artwork (0x4c0720); Phoenix build_unified_ui_frames.py
// recovers the same 16px edges, 43px top and 24px bottom around 512px imagery.
func FullMapRect(w, h int) (x, y int, scale float32) {
	scale = min(float32(1), float32(max(1, w-20))/544, float32(max(1, h-20))/579)
	return (w - int(544*scale)) / 2, (h - int(579*scale)) / 2, scale
}

type MapAction struct {
	Toggle, Close, World bool
	Zoom                 int
}

type Minimap struct {
	WorldAvailable, WorldReturn                                         bool
	worldButton, mapButton                                              Button
	Position, FullPosition                                              WindowPosition
	Caption, MapName                                                    string
	plate                                                               *render.Texture
	frame, arrow, party, questStart, questEnd, top, bottom, left, right *render.Texture
	buttons                                                             [3]Button
	close                                                               Button
}

func NewMinimap(t Textures) (*Minimap, error) {
	ld := &loader{t: t}
	m := &Minimap{frame: ld.get("main_map.png"), arrow: ld.get("arrow.png"), party: ld.get("radar_party.png"),
		questStart: ld.get("radar_queststart.png"), questEnd: ld.get("radar_questend.png"),
		top: ld.get("map_up.png"), bottom: ld.get("map_bottom.png"),
		left: ld.get("map_left.png"), right: ld.get("map_right.png")}
	m.plate = ld.get("main_servertime.png")
	tex := ld.get("main_map_button.png")
	for i, angle := range []float64{-45, -27, 40} {
		pos := [2]int{int(math.Round(68 + 84*math.Cos(angle*math.Pi/180))), int(math.Round(68 + 84*math.Sin(angle*math.Pi/180)))}
		b := &m.buttons[i]
		*b = Button{Tex: tex, OffX: pos[0], OffY: pos[1], DrawW: 32, DrawH: 32, HitW: 28, HitH: 28}
		for state, col := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(col) / 4, float32(i) / 4, float32(col+1) / 4, float32(i+1) / 4}
		}
	}
	m.close = nativeClose(ld.get("talk1_close_button.png"), 0, 0)
	m.worldButton = nativeClose(ld.get("wm_button_w.png"), 0, 0)
	m.mapButton = nativeClose(ld.get("wm_button_m.png"), 0, 0)
	// 0x4c3990: 32px artwork cells with independent 22px hit boxes.
	for _, b := range []*Button{&m.worldButton, &m.mapButton} {
		b.DrawW, b.DrawH = 32, 32
		for i, state := range []int{0, 1, 2, 2, 0} {
			b.UV[i] = UV{float32(state) / 4, 0, float32(state+1) / 4, 1}
		}
	}
	return m, ld.err
}

func (m *Minimap) Update(in *platform.Input, w, h int, full bool) MapAction {
	var action MapAction
	if full {
		x, y, scale := m.FullRect(w, h)
		m.close.OriginX, m.close.OriginY = x+int(515*scale), y+int(scale)
		m.close.DrawW, m.close.DrawH = int(22*scale), int(22*scale)
		m.close.HitW, m.close.HitH = int(22*scale), int(22*scale)
		action.Close = m.close.Update(in)
		if m.WorldAvailable {
			b := &m.worldButton
			if m.WorldReturn {
				b = &m.mapButton
			}
			b.OriginX, b.OriginY = x+int(490*scale), y+int(scale)
			b.DrawW, b.DrawH, b.HitW, b.HitH = int(32*scale), int(32*scale), int(22*scale), int(22*scale)
			action.World = b.Update(in)
		}
		return action
	}
	x, y := m.Origin(w, h)
	for i := range m.buttons {
		m.buttons[i].OriginX, m.buttons[i].OriginY = x, y
		if m.buttons[i].Update(in) {
			switch i {
			case 0:
				action.Zoom = 1
			case 1:
				action.Zoom = -1
			case 2:
				action.Toggle = true
			}
		}
	}
	return action
}

type MapMarkerKind uint8

const (
	MapParty MapMarkerKind = iota
	MapQuestStart
	MapQuestEnd
)

type MapMarker struct {
	U, V float32
	Kind MapMarkerKind
}

func (m *Minimap) drawMarkers(s *render.Sprites, markers []MapMarker, view UV, x, y, side, size float32, circular bool) {
	for _, mark := range markers {
		px, py, ok := MapPoint(mark.U, mark.V, view)
		if !ok || px*side < size/2 || py*side < size/2 || (1-px)*side < size/2 || (1-py)*side < size/2 {
			continue
		}
		dx, dy := px-.5, py-.5
		if circular && dx*dx+dy*dy > (.5-size/side/2)*(.5-size/side/2) {
			continue
		}
		texture := m.party
		switch mark.Kind {
		case MapQuestStart:
			texture = m.questStart
		case MapQuestEnd:
			texture = m.questEnd
		}
		s.Draw(texture, x+px*side-size/2, y+py*side-size/2, size, size, 0, 0, 1, 1, white)
	}
}

func (m *Minimap) Draw(s *render.Sprites, f *Font, image, solid *render.Texture, w, h int, view UV, u, v, heading float32, mapID uint16, rawX, rawZ float32, markers []MapMarker) {
	x, y := m.Origin(w, h)
	tex, tint := image, white
	if tex == nil {
		tex, tint = solid, [4]float32{.03, .04, .05, 1}
	}
	s.DrawCircle(tex, float32(x+16), float32(y+16), 136, view.U0, view.V0, view.U1, view.V1, tint)
	if image != nil {
		m.drawMarkers(s, markers, view, float32(x+16), float32(y+16), 136, 12, true)
		px, py, ok := MapPoint(u, v, view)
		dx, dy := px-.5, py-.5
		if ok && dx*dx+dy*dy < .24 {
			s.DrawRotated(m.arrow, float32(x+16)+px*136, float32(y+16)+py*136, 16, 16, heading, 0, 0, 1, 1, white)
		}
	}
	s.DrawImage(m.frame, float32(x+4), float32(y+4))
	s.Draw(m.plate, float32(x+9), float32(y+157), 150, 40, 0, 0, 150.0/256, 40.0/64, white)
	for i := range m.buttons {
		m.buttons[i].OriginX, m.buttons[i].OriginY = x, y
		m.buttons[i].Draw(s)
	}
	label := m.Caption
	if label == "" {
		label = "--:--"
	}
	scale := float32(1)
	if width := f.Width(label); width > 134 {
		scale = 134 / float32(width)
	}
	f.DrawScaled(s, label, float32(x+84)-float32(f.Width(label))*scale/2, float32(y+168), scale, [4]float32{.95, .87, .66, 1}, 0)
}

func (m *Minimap) DrawFull(s *render.Sprites, f *Font, image, solid *render.Texture, w, h int, view UV, u, v, heading float32, mapID uint16, markers []MapMarker) {
	x, y, scale := m.FullRect(w, h)
	mapX, mapY, side := float32(x)+16*scale, float32(y)+43*scale, 512*scale
	s.Draw(solid, mapX, mapY, side, side, 0, 0, 1, 1, [4]float32{.03, .04, .05, .98})
	if image != nil {
		s.Draw(image, mapX, mapY, side, side, view.U0, view.V0, view.U1, view.V1, white)
		m.drawMarkers(s, markers, view, mapX, mapY, side, 12*scale, false)
		if px, py, ok := MapPoint(u, v, view); ok {
			s.DrawRotated(m.arrow, mapX+px*side, mapY+py*side, 20*scale, 20*scale, heading, 0, 0, 1, 1, white)
		}
	} else {
		f.Draw(s, "Map image unavailable", int(mapX+12), int(mapY+12), white, 0)
	}
	s.Draw(m.top, float32(x), float32(y), 544*scale, 43*scale, 0, 0, 544.0/1024, 43.0/64, white)
	s.Draw(m.bottom, float32(x), mapY+side, 544*scale, 24*scale, 0, 0, 544.0/1024, 24.0/32, white)
	s.Draw(m.left, float32(x), mapY, 16*scale, side, 0, 0, 1, 1, white)
	s.Draw(m.right, mapX+side, mapY, 16*scale, side, 0, 0, 1, 1, white)
	m.close.OriginX, m.close.OriginY = x+int(515*scale), y+int(scale)
	m.close.DrawW, m.close.DrawH = int(22*scale), int(22*scale)

	m.close.Draw(s)
	if m.WorldAvailable {
		b := &m.worldButton
		if m.WorldReturn {
			b = &m.mapButton
		}
		b.OriginX, b.OriginY = x+int(490*scale), y+int(scale)
		b.DrawW, b.DrawH = int(32*scale), int(32*scale)
		b.Draw(s)
	}
	label := m.MapName
	if label == "" {
		label = fmt.Sprintf("Map %d", mapID)
	}
	f.Draw(s, label, x+int(24*scale), y+int(14*scale), white, max(1, int(450*scale)))
}

func (m *Minimap) Origin(w, h int) (int, int) {
	x, y := MinimapOrigin(w)
	return m.Position.Origin(w, h, MinimapWidth, MinimapHeight, x, y)
}
func (m *Minimap) FullRect(w, h int) (int, int, float32) {
	x, y, scale := FullMapRect(w, h)
	x, y = m.FullPosition.Origin(w, h, int(544*scale), int(579*scale), x, y)
	return x, y, scale
}

func (m *Minimap) DragHandleAt(mx, my, w, h int) bool {
	x, y := m.Origin(w, h)
	for _, b := range m.buttons {
		if mx >= x+b.OffX && mx < x+b.OffX+b.HitW && my >= y+b.OffY && my < y+b.OffY+b.HitH {
			return false
		}
	}
	dx, dy := mx-x-84, my-y-84
	return dx*dx+dy*dy <= 76*76 || mx >= x+9 && mx < x+159 && my >= y+157 && my < y+197 || mx >= x+35 && mx < x+110 && my >= y && my < y+20
}

func (m *Minimap) DrawWorldMarker(s *render.Sprites, x, y, scale, alpha float32) {
	s.Draw(m.party, x-8*scale, y-8*scale, 16*scale, 16*scale, 0, 0, 1, 1, [4]float32{1, 1, 1, alpha})
}
