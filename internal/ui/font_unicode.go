package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	textRender "github.com/go-text/render"
	textFont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/segmenter"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/unicode/norm"
)

type unicodeLayout struct {
	runs  []shaping.Output
	width int
}
type unicodeTexture struct {
	mono               *render.Texture
	pad, width, height int
	bytes              int
}
type unicodeFont struct {
	primary        *textFont.Face
	size           float32
	height, ascent int
	segmenter      shaping.Segmenter
	shaper         shaping.HarfbuzzShaper
	layouts        map[string]*unicodeLayout
	layoutOrder    []string
	textures       map[string]*unicodeTexture
	textureOrder   []string
	bytes          int
}
type oneFace struct{ face *textFont.Face }

func (o oneFace) ResolveFace(r rune) *textFont.Face { return o.face }
func newUnicodeFont(data []byte, size float32, height, ascent int) (*unicodeFont, error) {
	face, err := textFont.ParseTTF(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("unicode font: %w", err)
	}
	return &unicodeFont{primary: face, size: size, height: height, ascent: ascent, layouts: map[string]*unicodeLayout{}, textures: map[string]*unicodeTexture{}}, nil
}
func needsUnicode(s string) bool {
	for _, r := range s {
		if r > 255 {
			return true
		}
	}
	return false
}
func (u *unicodeFont) layout(s string) *unicodeLayout {
	if l := u.layouts[s]; l != nil {
		return l
	}
	l := &unicodeLayout{}
	rs := []rune(norm.NFC.String(s))
	if len(rs) == 0 {
		return l
	}
	// Shape all scripts with the sole embedded Arial face. Missing glyphs stay
	// missing; UTF-8 transport and grapheme-safe editing are independent.
	input := shaping.Input{Text: rs, RunEnd: len(rs), Face: u.primary, Size: fixed.Int26_6(u.size * 64)}
	for _, run := range u.segmenter.Split(input, oneFace{u.primary}) {
		l.runs = append(l.runs, u.shaper.Shape(run))
	}
	if len(l.runs) > 0 {
		direction := l.runs[0].Direction
		var wrapper shaping.LineWrapper
		wrapper.Prepare(shaping.WrapConfig{Direction: direction}, rs, shaping.NewSliceIterator(l.runs))
		wrapped, _ := wrapper.WrapNextLine(math.MaxInt)
		l.runs = append([]shaping.Output(nil), wrapped.Line...)
		sort.SliceStable(l.runs, func(i, j int) bool { return l.runs[i].VisualIndex < l.runs[j].VisualIndex })
	}
	var advance fixed.Int26_6
	for _, run := range l.runs {
		advance += run.Advance
	}
	l.width = max(0, advance.Ceil())
	if len(u.layoutOrder) >= 512 {
		delete(u.layouts, u.layoutOrder[0])
		u.layoutOrder = u.layoutOrder[1:]
	}
	u.layouts[s] = l
	u.layoutOrder = append(u.layoutOrder, s)
	return l
}
func (u *unicodeFont) fit(s string, maxW int) string {
	if maxW <= 0 || u.layout(s).width <= maxW {
		return s
	}
	var seg segmenter.Segmenter
	seg.InitWithString(s)
	it := seg.GraphemeIterator()
	end := 0
	rs := []rune(s)
	for it.Next() {
		g := it.Grapheme()
		next := g.Offset + len(g.Text)
		if u.layout(string(rs[:next])).width > maxW {
			break
		}
		end = next
	}
	return string(rs[:end])
}
func (u *unicodeFont) rasterize(s string) *unicodeTexture {
	l := u.layout(s)
	pad := int(math.Ceil(float64(u.size*2))) + 2
	width, height := max(1, l.width+pad*2), max(1, u.height+pad*2)
	mono := image.NewNRGBA(image.Rect(0, 0, width, height))
	x := fixed.I(pad)
	renderer := textRender.Renderer{FontSize: u.size, Color: color.White}
	for _, run := range l.runs {
		renderer.DrawShapedRunAt(run, mono, x.Round(), pad+u.ascent)
		x += run.Advance
	}
	t := &unicodeTexture{pad: pad, width: width, height: height}
	t.mono = render.Upload(img.Texture{RGBA: &img.RGBA{W: width, H: height, Pix: mono.Pix}})
	t.bytes = width * height * 4
	return t
}
func (u *unicodeFont) draw(s *render.Sprites, text string, x, y, scale float32, tint [4]float32, maxW int) {
	if maxW <= 0 || maxW > 4096 {
		maxW = 4096
	}
	text = u.fit(text, maxW)
	if text == "" {
		return
	}
	t := u.textures[text]
	if t == nil {
		for len(u.textureOrder) > 0 && (len(u.textureOrder) >= 256 || u.bytes > 16<<20) {
			s.Flush()
			old := u.textureOrder[0]
			u.textureOrder = u.textureOrder[1:]
			v := u.textures[old]
			v.mono.Delete()
			u.bytes -= v.bytes
			delete(u.textures, old)
		}
		t = u.rasterize(text)
		u.textures[text] = t
		u.textureOrder = append(u.textureOrder, text)
		u.bytes += t.bytes
	}
	draw := func(tex *render.Texture, c [4]float32) {
		s.Draw(tex, x-float32(t.pad)*scale, y-float32(t.pad)*scale, float32(t.width)*scale, float32(t.height)*scale, 0, 0, 1, 1, c)
	}
	draw(t.mono, tint)
}
func (u *unicodeFont) delete() {
	for _, t := range u.textures {
		t.mono.Delete()
	}
	clear(u.textures)
	u.textureOrder = nil
	u.bytes = 0
}
