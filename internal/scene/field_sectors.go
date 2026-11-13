package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
)

type fieldDrawSector struct {
	minimum, maximum render.V3
	visible          bool
}

// Each placement belongs to one sector, whose bounds contain its whole sphere.
// Rotors use their full swept bound. The fine pass retains native object order.
func (s *fieldStream) prepareSectors() {
	s.drawSectors = s.drawSectors[:0]
	lookup := map[cellKey]int{}
	for i := range s.drawEntries {
		e := &s.drawEntries[i]
		c, r := e.center, e.radius
		if e.object.rotor != nil {
			c = render.V3{e.object.world[12], e.object.world[13], e.object.world[14]}
			r += e.batch.model.center.Len() * e.object.scale
		}
		key := cellKey{int(math.Floor(float64(c[0] / 64))), int(math.Floor(float64(c[2] / 64)))}
		n, ok := lookup[key]
		d := render.V3{r, r, r}
		lo, hi := c.Sub(d), c.Add(d)
		if !ok {
			n = len(s.drawSectors)
			lookup[key] = n
			s.drawSectors = append(s.drawSectors, fieldDrawSector{minimum: lo, maximum: hi})
		} else {
			b := &s.drawSectors[n]
			for a := range 3 {
				b.minimum[a] = min(b.minimum[a], lo[a])
				b.maximum[a] = max(b.maximum[a], hi[a])
			}
		}
		e.sector = n
	}
}
func (s *fieldStream) updateSectors(r *render.Scene3D, f render.Frustum, clip bool) {
	s.objectTests = 0
	for i := range s.drawSectors {
		b := &s.drawSectors[i]
		b.visible = (!clip || render.ClipBoxVisible(r.ClipPlane, b.minimum, b.maximum)) && f.BoxVisible(b.minimum, b.maximum) && !r.OccludedBox(b.minimum, b.maximum)
	}
}
