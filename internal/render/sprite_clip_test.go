package render

import "testing"

func TestClipRectKeepsGlyphVerticesAndUVsInsidePanel(t *testing.T) {
	tex := &Texture{}
	s := &Sprites{tex: tex, cap: 16}
	restore := s.ClipRect(10, 20, 30, 40)
	s.Draw(tex, 5, 15, 20, 20, 0, 0, 1, 1, [4]float32{1, 1, 1, 1})
	if len(s.verts) != 48 {
		t.Fatal("missing clipped quad", len(s.verts))
	}
	for i := 0; i < len(s.verts); i += 8 {
		x, y := s.verts[i], s.verts[i+1]
		if x < 10 || x > 40 || y < 20 || y > 60 {
			t.Fatal("glyph escaped panel", x, y)
		}
	}
	if s.verts[2] != .25 || s.verts[3] != .25 {
		t.Fatal("clipping moved glyph texture", s.verts[:4])
	}
	before := len(s.verts)
	s.Draw(tex, 0, 0, 5, 5, 0, 0, 1, 1, [4]float32{})
	if len(s.verts) != before {
		t.Fatal("off-panel glyph was queued")
	}
	restore()
	if s.clipped {
		t.Fatal("clip leaked into other UI")
	}
	s.Draw(tex, 0, 0, 5, 5, 0, 0, 1, 1, [4]float32{})
	if len(s.verts) == before {
		t.Fatal("clip restore discarded unrelated UI")
	}
}
