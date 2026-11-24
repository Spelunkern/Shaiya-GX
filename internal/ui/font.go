// Package ui is the client's 2D interface: the original windows rebuilt
// from their Game.exe constructors (docs/ui.md), drawn with render.Sprites
// in screen pixels like the D3D9 client.
package ui

import (
	"fmt"
	"image"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Font uses a fast Latin-1 atlas and shaped Unicode fallback. Height follows D3DXCreateFont with a
// positive height: the cell (ascent + descent) is Height pixels.
type Font struct {
	tex           *render.Texture
	glyphs        [256]glyph
	outlineGlyphs [256]glyph
	hasOutline    bool
	unicode       *unicodeFont
	Ascent        int
	Height        int
	atlasW        float32
	atlasH        float32
	subtleOutline bool
}

type glyph struct {
	x, y, w, h int // atlas rect
	ox, oy     int // offset from pen (x) and cell top (y)
	adv        int
}

// LoadFont loads data/fonts resources beside the game (or SHAIYA_DATA for fixtures).
func LoadFont(cellHeight int) (*Font, error) {
	source, err := externalFontSource()
	if err != nil {
		return nil, err
	}
	return source.Load(cellHeight)
}

func loadFontData(data []byte, cellHeight int, pixelSize bool) (*Font, error) {
	ft, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("font: %w", err)
	}
	// Scale so ascent+descent == cellHeight (GDI cell height).
	probe, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: 100, DPI: 72})
	if err != nil {
		return nil, err
	}
	m := probe.Metrics()
	probe.Close()
	cell := float64(m.Ascent+m.Descent) / 64
	size := 100 * float64(cellHeight) / cell
	if pixelSize {
		size = float64(cellHeight)
	}
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	defer face.Close()
	height := cellHeight
	if pixelSize {
		height = (face.Metrics().Ascent + face.Metrics().Descent).Round()
	}
	f := &Font{Height: height, Ascent: face.Metrics().Ascent.Round()}
	f.unicode, err = newUnicodeFont(data, float32(size), height, f.Ascent)
	if err != nil {
		return nil, err
	}
	const aw, ah = 512, 512
	atlas := image.NewAlpha(image.Rect(0, 0, aw, ah))
	px, py, rowH := 1, 1, 0
	for c := 32; c < 256; c++ {
		r := rune(c) // Latin-1
		dr, mask, mp, adv, ok := face.Glyph(fixed.P(0, f.Ascent), r)
		if !ok {
			continue
		}
		g := glyph{adv: adv.Round(), w: dr.Dx(), h: dr.Dy(), ox: dr.Min.X, oy: dr.Min.Y}
		if g.w > 0 && g.h > 0 {
			if px+g.w+1 > aw {
				px, py, rowH = 1, py+rowH+1, 0
			}
			if py+g.h+1 > ah {
				return nil, fmt.Errorf("font atlas full")
			}
			draw.Draw(atlas, image.Rect(px, py, px+g.w, py+g.h), mask, mp, draw.Src)
			g.x, g.y = px, py
			px += g.w + 1
			rowH = max(rowH, g.h)
		}
		f.glyphs[c] = g
	}
	px, py, rowH = 1, max(256, py+rowH+1), 0
	sample := func(src glyph, sx, sy int) uint8 {
		if sx < 0 || sy < 0 || sx >= src.w || sy >= src.h {
			return 0
		}
		return atlas.Pix[(src.y+sy)*aw+src.x+sx]
	}
	for c := 32; c < 256; c++ {
		src := f.glyphs[c]
		if src.w <= 0 || src.h <= 0 {
			f.outlineGlyphs[c] = glyph{adv: src.adv}
			continue
		}
		ow, oh := src.w+2, src.h+2
		if px+ow+1 > aw {
			px, py, rowH = 1, py+rowH+1, 0
		}
		if py+oh+1 > ah {
			return nil, fmt.Errorf("font outline atlas full")
		}
		f.outlineGlyphs[c] = glyph{x: px, y: py, w: ow, h: oh, ox: src.ox - 1, oy: src.oy - 1, adv: src.adv}
		for dy := 0; dy < oh; dy++ {
			ly := dy - 1
			for dx := 0; dx < ow; dx++ {
				lx := dx - 1
				a := outlineUnionAlpha(sample(src, lx+1, ly), sample(src, lx-1, ly), sample(src, lx, ly+1), sample(src, lx, ly-1))
				atlas.Pix[(py+dy)*aw+px+dx] = a
			}
		}
		px += ow + 1
		rowH = max(rowH, oh)
	}
	f.hasOutline = true
	rgba := &img.RGBA{W: aw, H: ah, Pix: make([]byte, aw*ah*4)}
	for i, a := range atlas.Pix {
		rgba.Pix[i*4], rgba.Pix[i*4+1], rgba.Pix[i*4+2], rgba.Pix[i*4+3] = 255, 255, 255, a
	}
	f.tex = render.Upload(img.Texture{RGBA: rgba})
	f.atlasW, f.atlasH = aw, ah
	return f, nil
}

func outlineUnionAlpha(a1, a2, a3, a4 uint8) uint8 {
	if a1 == 0 && a2 == 0 && a3 == 0 && a4 == 0 {
		return 0
	}
	if a1 == 255 || a2 == 255 || a3 == 255 || a4 == 255 {
		return 255
	}
	remain := (1 - float64(a1)/255) * (1 - float64(a2)/255) * (1 - float64(a3)/255) * (1 - float64(a4)/255)
	return uint8((1-remain)*255 + .5)
}

// Width is the advance of s in pixels.
func (f *Font) Width(s string) int {
	if f.unicode != nil && needsUnicode(s) {
		return f.unicode.layout(s).width
	}
	w := 0
	for _, r := range s {
		if r < 256 {
			w += f.glyphs[r].adv
		}
	}
	return w
}

// DrawOutlined follows Phoenix's black-outlined fixed-pixel nameplates.
func (f *Font) DrawOutlined(s *render.Sprites, text string, x, y int, tint [4]float32) {
	if !f.hasOutline || (f.unicode != nil && needsUnicode(text)) {
		for _, offset := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			f.drawText(s, text, float32(x+offset[0]), float32(y+offset[1]), 1, [4]float32{0, 0, 0, tint[3]}, 0, false)
		}
		f.drawText(s, text, float32(x), float32(y), 1, tint, 0, false)
		return
	}
	f.drawGlyphs(s, &f.outlineGlyphs, text, float32(x), float32(y), 1, [4]float32{0, 0, 0, tint[3]}, 0)
	f.drawGlyphs(s, &f.glyphs, text, float32(x), float32(y), 1, tint, 0)
}

// DrawName uses the US EP4 single foreground and four one-pixel outline passes.
func (f *Font) DrawName(s *render.Sprites, text string, x, y int, tint [4]float32) {
	f.DrawOutlined(s, text, x, y, tint)
}

// Draw renders UTF-8 text with its cell top-left at (x, y), clipped to
// maxW pixels when maxW > 0.
func (f *Font) Draw(s *render.Sprites, text string, x, y int, rgba [4]float32, maxW int) {
	f.DrawScaled(s, text, float32(x), float32(y), 1, rgba, maxW)
}

// DrawScaled keeps tooltip text and frame together on small viewports.
func (f *Font) DrawScaled(s *render.Sprites, text string, x, y, scale float32, rgba [4]float32, maxW int) {
	if f.subtleOutline && (rgba[0] != 0 || rgba[1] != 0 || rgba[2] != 0) && rgba[3] > 0 {
		edge := .8 * scale
		ink := [4]float32{0, 0, 0, rgba[3] * .30}
		f.drawText(s, text, x-edge, y, scale, ink, maxW, true)
		f.drawText(s, text, x+edge, y, scale, ink, maxW, true)
		f.drawText(s, text, x, y-edge, scale, ink, maxW, true)
		f.drawText(s, text, x, y+edge, scale, ink, maxW, true)
	}
	f.drawText(s, text, x, y, scale, rgba, maxW, false)
}

func (f *Font) drawText(s *render.Sprites, text string, x, y, scale float32, rgba [4]float32, maxW int, monochromeOnly bool) {
	if f.unicode != nil && needsUnicode(text) {
		f.unicode.draw(s, text, x, y, scale, rgba, maxW, monochromeOnly)
		return
	}
	f.drawGlyphs(s, &f.glyphs, text, x, y, scale, rgba, maxW)
}

func (f *Font) drawGlyphs(s *render.Sprites, glyphs *[256]glyph, text string, x, y, scale float32, rgba [4]float32, maxW int) {
	pen := 0
	for _, r := range text {
		if r >= 256 {
			r = '?'
		}
		g := glyphs[r]
		if maxW > 0 && pen+g.adv > maxW {
			return
		}
		if g.w > 0 {
			s.Draw(f.tex, x+float32(pen+g.ox)*scale, y+float32(g.oy)*scale, float32(g.w)*scale, float32(g.h)*scale, float32(g.x)/f.atlasW, float32(g.y)/f.atlasH, float32(g.x+g.w)/f.atlasW, float32(g.y+g.h)/f.atlasH, rgba)
		}
		pen += g.adv
	}
}

func (f *Font) Delete() {
	if f != nil && f.unicode != nil {
		f.unicode.delete()
	}
	if f != nil && f.tex != nil {
		f.tex.Delete()
		f.tex = nil
	}
}
