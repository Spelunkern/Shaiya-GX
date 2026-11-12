package render

import (
	"math"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// DrawCircle clips one minimap texture to a screen-space circle, like the
// Phoenix MinimapHud MAP_SHADER. UVs remain in the source map's coordinate space.
func (s *Sprites) DrawCircle(t *Texture, x, y, diameter, u0, v0, u1, v1 float32, tint [4]float32) {
	s.Flush()
	radius := diameter / 2
	gl.Uniform3f(s.uCircle, x+radius, y+radius, radius)
	s.Draw(t, x, y, diameter, diameter, u0, v0, u1, v1, tint)
	s.Flush()
	gl.Uniform3f(s.uCircle, 0, 0, 0)
}

// DrawRotated rotates an ordinary atlas quad clockwise around its screen centre.
func (s *Sprites) DrawRotated(t *Texture, cx, cy, w, h, angle, u0, v0, u1, v1 float32, tint [4]float32) {
	s.Draw(t, cx-w/2, cy-h/2, w, h, u0, v0, u1, v1, tint)
	sine, cosine := math.Sincos(float64(angle))
	sn, cs := float32(sine), float32(cosine)
	for i := len(s.verts) - 6*floatsPerVertex; i < len(s.verts); i += floatsPerVertex {
		x, y := s.verts[i]-cx, s.verts[i+1]-cy
		s.verts[i], s.verts[i+1] = cx+x*cs-y*sn, cy+x*sn+y*cs
	}
}
