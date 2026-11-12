package ui

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const QuestWidth, QuestHeight, QuestRows = 510, 488, 7

type QuestLog struct {
	Position    WindowPosition
	frame       *render.Texture
	close, quit Button
	held        int
	textOffset  int
}

func (q *QuestLog) ResetDescriptionScroll() { q.textOffset = 0 }

func NewQuestLog(t Textures) (*QuestLog, error) {
	ld := &loader{t: t}
	q := &QuestLog{frame: ld.get("quest.tga"), held: -1}
	q.close = nativeClose(ld.get("talk1_close_button.tga"), 480, 7)
	q.quit = Button{Tex: ld.get("quest_button.tga"), OffX: 410, OffY: 456, DrawW: 80, DrawH: 32, HitW: 80, HitH: 26, UV: FourRowUV}
	return q, ld.err
}
func (q *QuestLog) Origin(w, h int) (int, int) {
	return q.Position.Origin(w, h, QuestWidth, QuestHeight, max(0, (w-QuestWidth)/2), max(0, (h-QuestHeight)/2))
}
func (q *QuestLog) Update(in *platform.Input, w, h, count, offset int, selected bool) (row, scroll int, close, quit bool) {
	x, y := q.Origin(w, h)
	q.close.OriginX, q.close.OriginY = x, y
	q.quit.OriginX, q.quit.OriginY = x, y
	q.quit.Disabled = !selected
	close = q.close.Update(in)
	quit = q.quit.Update(in)
	row = -1
	scroll = offset
	if in.MouseX >= x+15 && in.MouseX < x+234 && in.MouseY >= y+54 && in.MouseY < y+282 {
		if in.Wheel != 0 {
			delta := 1
			if in.Wheel < 0 {
				delta = -1
			}
			scroll = max(0, min(max(0, count-QuestRows), offset-delta))
		}
	}
	if selected && in.MouseX >= x+275 && in.MouseX < x+490 && in.MouseY >= y+102 && in.MouseY < y+427 && in.Wheel != 0 {
		if in.Wheel < 0 {
			q.textOffset += 3
		} else {
			q.textOffset = max(0, q.textOffset-3)
		}
	}
	hit := -1
	if in.MouseX >= x+18 && in.MouseX < x+232 && in.MouseY >= y+57 && in.MouseY < y+57+QuestRows*32 {
		hit = (in.MouseY - y - 57) / 32
	}
	if in.Pressed {
		q.held = hit
	}
	if in.Clicked {
		if hit >= 0 && hit == q.held && scroll+hit < count {
			row = scroll + hit
		}
		q.held = -1
	}
	return
}
func (q *QuestLog) Draw(s *render.Sprites, f *Font, w, h int, names []string, offset, selected int, title, description, objective string) {
	x, y := q.Origin(w, h)
	s.Draw(q.frame, float32(x), float32(y), QuestWidth, QuestHeight, 0, 0, 510.0/512, 488.0/512, white)
	for i := 0; i < QuestRows && offset+i < len(names); i++ {
		name := names[offset+i]
		if offset+i == selected {
			name = "> " + name
		}
		f.Draw(s, name, x+22, y+62+i*32, white, 204)
	}
	f.Draw(s, title, x+279, y+72, white, 207)
	lines := wrapLines(f, strings.TrimSpace(description), 207)
	q.textOffset = min(q.textOffset, max(0, len(lines)-23))
	for i, line := range lines[q.textOffset:min(len(lines), q.textOffset+23)] {
		f.Draw(s, line, x+279, y+105+i*14, white, 207)
	}
	for i, line := range wrapLines(f, strings.TrimSpace(objective), 210) {
		if i >= 8 {
			break
		}
		f.Draw(s, line, x+18, y+326+i*14, white, 210)
	}
	q.close.OriginX, q.close.OriginY = x, y
	q.close.Draw(s)
	q.quit.OriginX, q.quit.OriginY = x, y
	q.quit.Draw(s)
}
