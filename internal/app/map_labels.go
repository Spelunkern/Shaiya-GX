package app

import (
	"fmt"
	"path"
	"strings"

	gametext "github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type mapLabelView struct {
	world         *scene.World
	texts         gametext.SysMsg
	inside        map[int]bool
	banner        string
	bannerTexture *render.Texture
	bannerLoaded  string
	shownAt       float64
	nextCheck     float64
}

func (a *App) drawMapLabels(w, h int) {
	if a.wv == nil || a.wv.world == nil || a.wv.world.WLD == nil {
		return
	}
	v := &a.mapLabels
	if v.world != a.wv.world {
		*v = mapLabelView{world: a.wv.world, inside: map[int]bool{}}
		if raw, err := a.arc.ReadFile(fmt.Sprintf("world/%d.txt", a.wv.mapID)); err == nil {
			v.texts = gametext.ParseSysMsg(raw)
		}
	}
	areas := v.world.WLD.NamedAreas
	if a.clock >= v.nextCheck {
		v.nextCheck = a.clock + .1
		bestVolume := float32(1e30)
		for i, area := range areas {
			if area.Mode != 2 {
				continue
			}
			lo, hi := render.V3(area.Box.Min), render.V3(area.Box.Max)
			p := a.wv.pos
			inside := p[0] >= lo[0] && p[0] <= hi[0] && p[1] >= lo[1] && p[1] <= hi[1] && p[2] >= lo[2] && p[2] <= hi[2]
			volume := (hi[0] - lo[0]) * (hi[1] - lo[1]) * (hi[2] - lo[2])
			if inside && !v.inside[i] && volume < bestVolume {
				v.banner, v.shownAt, bestVolume = area.Text, a.clock, volume
			}
			v.inside[i] = inside
		}
	}
	elapsed := a.clock - v.shownAt
	if v.banner != "" && elapsed >= 0 && elapsed < 4.5 {
		if v.bannerLoaded != v.banner {
			name := path.Base(strings.ReplaceAll(v.banner, `\`, "/"))
			name = strings.TrimSuffix(name, path.Ext(name)) + ".png"
			v.bannerTexture = a.assets.Texture("interface/worldname/" + name)
			v.bannerLoaded = v.banner
		}
		if tex := v.bannerTexture; tex != nil {
			alpha := float32(min(elapsed/.25, 1, (4.5-elapsed)/.65))
			a.sprites.Draw(tex, float32(w-tex.W)/2, 72, float32(tex.W), float32(tex.H), 0, 0, 1, 1, [4]float32{1, 1, 1, alpha})
		}
	}
	if a.wv.mapView.full || a.scene3d == nil {
		return
	}
	origin, dir, ok := clickRay(a.lastMouseX, a.lastMouseY, w, h, a.scene3d.View.Mul(a.scene3d.Proj))
	if !ok {
		return
	}
	best := float32(50)
	text := ""
	for _, area := range areas {
		if area.Mode != 0 {
			continue
		}
		label := v.texts.Get(area.TextID)
		if label == "" {
			continue
		}
		margin := render.V3{2, 1, 2}
		if distance, hit := rayBox(origin, dir, render.V3(area.Box.Min).Sub(margin), render.V3(area.Box.Max).Add(margin)); hit && distance < best {
			best, text = distance, label
		}
	}
	if text != "" {
		x, y := min(a.lastMouseX+18, max(0, w-a.font.Width(text)-12)), min(a.lastMouseY+18, h-a.font.Height-8)
		a.font.DrawOutlined(a.sprites, text, x, y, white)
	}
}
