package render

// DrawCooldown follows the native 0x4a27d0 eight-triangle square sweep. The
// dark remaining sector unwinds clockwise from the top; no extra art is needed.
func (s *Sprites) DrawCooldown(t *Texture, x, y, size, remaining float32, rgba [4]float32) {
	remaining = min(1, max(0, remaining))
	if remaining == 0 {
		return
	}
	boundary := [9][2]float32{{.5, 0}, {1, 0}, {1, .5}, {1, 1}, {.5, 1}, {0, 1}, {0, .5}, {0, 0}, {.5, 0}}
	progress := (1 - remaining) * 8
	first := int(progress)
	for i := first; i < 8; i++ {
		if s.tex != t || len(s.verts)+3*floatsPerVertex > s.cap*6*floatsPerVertex {
			s.Flush()
			s.tex = t
		}
		a, b := boundary[i], boundary[i+1]
		if i == first {
			f := progress - float32(first)
			a[0] += (b[0] - a[0]) * f
			a[1] += (b[1] - a[1]) * f
		}
		for _, v := range [3][2]float32{{.5, .5}, a, b} {
			s.verts = append(s.verts, x+v[0]*size, y+v[1]*size, .5, .5, rgba[0], rgba[1], rgba[2], rgba[3])
		}
	}
}
