package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Native 0x43b650 picks a camera-facing rectangle, 0.6 units wide and
// one unit tall, independently of the small dropped-item mesh. Captured depth
// still prevents selecting loot through world geometry.
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
	right := render.V3{a.scene3d.View[0], 0, a.scene3d.View[8]}.Norm()
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
		d, hit := groundPickupIntersection(origin, dir, right, pos)
		if !hit || d > best || d == best && bestID != 0 && id > bestID {
			continue
		}
		// Do not let the item's own front surface occlude its selection plane.
		visiblePoint := origin.Add(dir.Scale(max(0, d-m.Radius)))
		_, _, depth, ok := projectLabel(visiblePoint, vp, w, h)
		if !ok {
			continue
		}
		best, bestID, bestDepth = d, id, depth
	}
	if bestID != 0 && a.nameVisibleAt(in.MouseX, in.MouseY, w, h, bestDepth) {
		return bestID
	}
	return 0
}

// The original uses two triangles in a vertical billboard, not mesh triangles.
func groundPickupIntersection(origin, direction, right, position render.V3) (float32, bool) {
	normal := (render.V3{0, 1, 0}).Cross(right)
	denom := direction.Dot(normal)
	if float32(math.Abs(float64(denom))) < 1e-7 {
		return 0, false
	}
	distance := position.Sub(origin).Dot(normal) / denom
	if distance < 0 {
		return 0, false
	}
	hit := origin.Add(direction.Scale(distance)).Sub(position)
	return distance, hit[1] >= 0 && hit[1] <= 1 && float32(math.Abs(float64(hit.Dot(right)))) <= .3
}
