package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"os"
	"testing"
)

func TestWorldMapPreviewFactionAndBack(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("resources")
	}
	arc, err := archive.OpenDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.arc = arc
	a.minimap, _ = ui.NewMinimap(inputTextures{})
	a.wv.mapID = 1
	a.wv.mapView.full = true
	v := a.mapState()
	a.loadWorldMap(v)
	x, y, scale := a.minimap.FullRect(1280, 800)
	a.updateWorldMap(&platform.Input{MouseX: x + 100, MouseY: y + 100, RightPressed: true}, 1280, 800)
	if !v.atlas.overview {
		t.Fatal("right click did not open overview")
	}
	click := func(mapID int) {
		t.Helper()
		for _, r := range v.atlas.regions {
			if r.Map != mapID {
				continue
			}
			in := &platform.Input{MouseX: x + int((16+float32(r.Left+r.Right)/2)*scale), MouseY: y + int((43+float32(r.Top+r.Bottom)/2)*scale), Pressed: true, MouseDown: true}
			a.updateWorldMap(in, 1280, 800)
			in.Pressed, in.MouseDown, in.Clicked = false, false, true
			a.updateWorldMap(in, 1280, 800)
			return
		}
		t.Fatal("missing map", mapID)
	}
	click(19)
	if v.atlas.preview != 19 {
		t.Fatal("friendly map did not preview", v.atlas)
	}
	a.updateWorldMap(&platform.Input{MouseX: x + 100, MouseY: y + 100, RightPressed: true}, 1280, 800)
	if v.atlas.preview != -1 || !v.atlas.overview {
		t.Fatal("preview did not return to overview")
	}
	click(2)
	if !v.atlas.warning || v.atlas.preview != -1 {
		t.Fatal("opposing map was not blocked", v.atlas)
	}
	click(1)
	if v.atlas.overview || a.wv.mapID != 1 {
		t.Fatal("current region did not return to local map")
	}
}
