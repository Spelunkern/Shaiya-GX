package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// A click tests the original mesh only once, then the captured frame depth
// rejects objects hidden behind terrain or other geometry.
func (a *App) pickGroundModel(in *platform.Input, w, h int) uint32 {
	if !in.Clicked || a.wv == nil || a.scene3d == nil || a.sprites == nil || a.assets == nil || w <= 0 || h <= 0 || in.MouseX < 0 || in.MouseX >= w || in.MouseY < 0 || in.MouseY >= h {
		return 0
	}
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	inv := vp.Inverse()
	x := (float32(in.MouseX)+.5)*2/float32(w) - 1
	y := 1 - (float32(in.MouseY)+.5)*2/float32(h)
	unproject := func(z float32) render.V3 {
		cw := x*inv[3] + y*inv[7] + z*inv[11] + inv[15]
		return render.V3{(x*inv[0] + y*inv[4] + z*inv[8] + inv[12]) / cw, (x*inv[1] + y*inv[5] + z*inv[9] + inv[13]) / cw, (x*inv[2] + y*inv[6] + z*inv[10] + inv[14]) / cw}
	}
	origin, far := unproject(0), unproject(1)
	dir := far.Sub(origin).Norm()
	best, bestID, bestDepth := float32(math.MaxFloat32), uint32(0), float32(0)
	for id, it := range a.state.GroundItems {
		pos := groundItemPosition(a.wv.world, it)
		if pos.Sub(a.wv.pos).Len() > 50 {
			continue
		}
		m := a.assets.GroundModel(scene.GroundItemModel(it.Type))
		if m == nil {
			continue
		}
		offset := pos.Sub(origin)
		projection := offset.Dot(dir)
		if projection < -m.Radius || offset.Dot(offset)-projection*projection > m.Radius*m.Radius {
			continue
		}
		d, hit := m.RayIntersection(origin.Sub(pos), dir)
		if !hit || d > best || d == best && bestID != 0 && id > bestID {
			continue
		}
		_, _, depth, ok := projectLabel(origin.Add(dir.Scale(d)), vp, w, h)
		if !ok {
			continue
		}
		best, bestID, bestDepth = d, id, depth
	}
	if bestID != 0 && a.sprites.NameVisibleAt(in.MouseX, in.MouseY, w, h, bestDepth) {
		return bestID
	}
	return 0
}
