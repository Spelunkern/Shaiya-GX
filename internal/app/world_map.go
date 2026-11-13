package app

import (
	"fmt"
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type worldMapView struct {
	loaded, overview, warning bool
	hover, pressed, preview   int
	regions                   []world.WorldMapRegion
}

func (a *App) loadWorldMap(v *mapView) {
	if v.atlas.loaded {
		return
	}
	v.atlas.loaded = true
	v.atlas.hover, v.atlas.pressed, v.atlas.preview = -1, -1, -1
	if a.arc == nil {
		return
	}
	raw, err := a.arc.ReadFile("WorldMap.cfg")
	if err == nil {
		v.atlas.regions, err = world.ParseWorldMap(raw)
	}
	if err != nil {
		log.Printf("World Map: %v", err)
	}
}
func (a *App) updateWorldMap(in *platform.Input, w, h int) bool {
	v := a.mapState()
	atlas := &v.atlas
	if len(atlas.regions) == 0 {
		return false
	}
	x, y, scale := a.minimap.FullRect(w, h)
	captured := panelPointer(in, x, y, int(544*scale), int(579*scale))
	if !atlas.overview {
		if in.RightPressed && pointInside(in, x, y, int(544*scale), int(579*scale)) {
			atlas.overview = true
			atlas.preview = -1
			return true
		}
		return false
	}
	if in.RightPressed && pointInside(in, x, y, int(544*scale), int(579*scale)) {
		if atlas.preview >= 0 {
			atlas.preview = -1
		} else {
			atlas.overview = false
		}
		atlas.warning = false
		return true
	}
	if atlas.preview >= 0 {
		return captured
	}
	mx, my := (float32(in.MouseX-x)-16*scale)/scale, (float32(in.MouseY-y)-43*scale)/scale
	previous := atlas.hover
	atlas.hover = -1
	for i, r := range atlas.regions {
		if r.Contains(mx, my) {
			atlas.hover = i
			break
		}
	}
	if previous != atlas.hover {
		atlas.warning = false
	}
	if in.Pressed {
		atlas.pressed = atlas.hover
	}
	if in.Clicked && atlas.hover >= 0 && atlas.hover == atlas.pressed {
		atlas.pressed = -1
		r := atlas.regions[atlas.hover]
		if r.Map == int(a.wv.mapID) {
			atlas.overview = false
			return true
		}
		if r.Map < 0 {
			return true
		}
		own, _ := a.selfSlot()
		if !r.Available(own.Race >= 2) {
			atlas.warning = true
			return true
		}
		file := fmt.Sprintf("interface/MiniMap/%d.png", r.Map)
		if _, ok := a.arc.Lookup(file); ok {
			atlas.preview = r.Map
		}
	}
	return captured
}
func (a *App) drawWorldMap(w, h int) {
	v := a.mapState()
	atlas := &v.atlas
	a.minimap.MapName = a.msg.Get(0x1530)
	if a.minimap.MapName == "" {
		a.minimap.MapName = "World Map"
	}
	file := "interface/wm_map.png"
	if atlas.preview >= 0 {
		file = fmt.Sprintf("interface/MiniMap/%d.png", atlas.preview)
		a.minimap.MapName = ui.MapName(a.msg, uint16(atlas.preview))
	}
	a.minimap.DrawFull(a.sprites, a.font, a.assets.Texture(file), a.white, w, h, ui.UV{U1: 1, V1: 1}, -1, -1, 0, a.wv.mapID, nil)
	if atlas.preview >= 0 {
		return
	}
	x, y, scale := a.minimap.FullRect(w, h)
	mx, my := float32(x)+16*scale, float32(y)+43*scale
	if atlas.hover >= 0 {
		r := atlas.regions[atlas.hover]
		if r.Map >= 0 {
			if tex := a.assets.Texture(fmt.Sprintf("interface/wm_e%d.png", r.Map)); tex != nil {
				a.sprites.Draw(tex, mx+float32(r.ImageX)*scale, my+float32(r.ImageY)*scale, float32(tex.W)*scale, float32(tex.H)*scale, 0, 0, 1, 1, white)
			}
			if tex := a.assets.Texture(fmt.Sprintf("interface/wm_f%d.png", r.Map)); tex != nil {
				a.sprites.Draw(tex, mx, my+10*scale, float32(tex.W)*scale, float32(tex.H)*scale, 0, 0, 1, 1, white)
			}
		}
	}
	for _, r := range atlas.regions {
		if r.Map == int(a.wv.mapID) {
			a.minimap.DrawWorldMarker(a.sprites, mx+float32(r.Left+r.Right)/2*scale, my+float32(r.Top+r.Bottom)/2*scale, scale, 1)
			break
		}
	}
	if tex := a.assets.Texture("interface/wm_letter.png"); tex != nil {
		a.sprites.Draw(tex, mx, my, 512*scale, 512*scale, 0, 0, 1, 1, white)
	}
	if atlas.warning {
		if tex := a.assets.Texture("interface/wm_warning.png"); tex != nil {
			a.sprites.Draw(tex, mx, float32(y)+523*scale, 512*scale, 32*scale, 0, 0, 1, 1, white)
		}
	}
}
