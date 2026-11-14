package render

// ClipRect clips queued axis-aligned quads in CPU space, including glyph UVs.
// It adds no draw calls or GL state changes. Restore before drawing other UI.
func (s *Sprites) ClipRect(x, y, w, h float32) func() {
	previous, enabled := s.clipRect, s.clipped
	rect := [4]float32{x, y, x + w, y + h}
	if enabled {
		rect = [4]float32{max(rect[0], previous[0]), max(rect[1], previous[1]), min(rect[2], previous[2]), min(rect[3], previous[3])}
	}
	s.clipRect, s.clipped = rect, true
	return func() { s.clipRect, s.clipped = previous, enabled }
}
