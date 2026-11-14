package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// DetailBias zero is the unchanged default. Lower optional presets trim only
// decorative vegetation, never buildings, terrain, actors or collision data.
func (e *fieldDrawEntry) detailVisible(s *render.Scene3D, center render.V3, radius float32) bool {
	if s.DetailBias < 1 || s.Shadow || e.object.detailClass == 0 {
		return true
	}
	v := s.View
	p := render.V3{center[0]*v[0] + center[1]*v[4] + center[2]*v[8] + v[12], center[0]*v[1] + center[1]*v[5] + center[2]*v[9] + v[13], center[0]*v[2] + center[1]*v[6] + center[2]*v[10] + v[14]}
	distance := max(float32(0), p.Len()-radius)
	if e.object.detailClass == 2 {
		limit := float32(300)
		if s.DetailBias >= 2 {
			limit = 160
		}
		return radius > 32 || distance <= limit
	}
	limit := float32(120)
	if s.DetailBias >= 2 {
		limit = 60
	}
	if distance > limit {
		return false
	}
	if distance <= 25 {
		return true
	}
	// Stable source-index density avoids random flicker when the camera moves.
	if s.DetailBias >= 2 {
		return e.object.detailRank == 0
	}
	return e.object.detailRank%2 == 0
}
