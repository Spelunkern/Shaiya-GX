// Package ui is the client's 2D interface: the original windows rebuilt
// from their Game.exe constructors (docs/ui.md), drawn with render.Sprites
// in screen pixels like the D3D9 client.
package ui

import (
	"fmt"
	"image"
	"image/draw"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	img "github.com/Spelunkern/shaiya-go-client/internal/data/image"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
)

// Font is a Latin-1 glyph atlas. Height follows D3DXCreateFont with a
// positive height: the cell (ascent + descent) is Height pixels.
type Font struct {
	tex    *render.Texture
	glyphs [256]glyph
	Ascent int
	Height int
	atlasW float32
	atlasH float32
}

type glyph struct {
	x, y, w, h int // atlas rect
	ox, oy     int // offset from pen (x) and cell top (y)
	adv        int
}

// DefaultFontPath is wine's metric-compatible Tahoma; SHAIYA_FONT overrides.
func DefaultFontPath() string {
	if p := os.Getenv("SHAIYA_FONT"); p != "" {
		return p
	}
	return "/usr/share/wine/fonts/tahoma.ttf"
}

// LoadFont rasterizes the face so that its cell is cellHeight pixels.
func LoadFont(path string, cellHeight int) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("font: %w", err)
	}
	ft, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("font %s: %w", path, err)
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
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	defer face.Close()
	f := &Font{Height: cellHeight, Ascent: face.Metrics().Ascent.Round()}
	const aw, ah = 512, 256
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
	rgba := &img.RGBA{W: aw, H: ah, Pix: make([]byte, aw*ah*4)}
	for i, a := range atlas.Pix {
		rgba.Pix[i*4], rgba.Pix[i*4+1], rgba.Pix[i*4+2], rgba.Pix[i*4+3] = 255, 255, 255, a
	}
	f.tex = render.Upload(img.Texture{RGBA: rgba})
	f.atlasW, f.atlasH = aw, ah
	return f, nil
}

// Width is the advance of s in pixels.
func (f *Font) Width(s string) int {
	w := 0
	for _, r := range s {
		if r < 256 {
			w += f.glyphs[r].adv
		}
	}
	return w
}

// Draw renders Latin-1 text with its cell top-left at (x, y), clipped to
// maxW pixels when maxW > 0.
func (f *Font) Draw(s *render.Sprites, text string, x, y int, rgba [4]float32, maxW int) {
	pen := x
	for _, r := range text {
		if r >= 256 {
			r = '?'
		}
		g := f.glyphs[r]
		if maxW > 0 && pen+g.adv-x > maxW {
			return
		}
		if g.w > 0 {
			s.Draw(f.tex, float32(pen+g.ox), float32(y+g.oy), float32(g.w), float32(g.h),
				float32(g.x)/f.atlasW, float32(g.y)/f.atlasH, float32(g.x+g.w)/f.atlasW, float32(g.y+g.h)/f.atlasH, rgba)
		}
		pen += g.adv
	}
}
