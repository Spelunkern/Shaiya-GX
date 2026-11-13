package ui

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const QuestWidth, QuestHeight, QuestRows = 510, 488, 7

type QuestLog struct {
	Position                                       WindowPosition
	frame                                          *render.Texture
	alarm, line                                    *render.Texture
	listScroll, descriptionScroll, objectiveScroll panelScroll
	Ready                                          []bool
	textLines, objectiveLines, objectiveOffset     int
	hoverRow                                       int
	close, quit                                    Button
	held                                           int
	textOffset                                     int
}

func (q *QuestLog) ResetDescriptionScroll() { q.textOffset, q.objectiveOffset = 0, 0 }

func NewQuestLog(t Textures) (*QuestLog, error) {
	ld := &loader{t: t}
	q := &QuestLog{frame: ld.get("quest.tga"), alarm: ld.get("quest_alarm.tga"), line: ld.get("quest_line.tga"), held: -1, hoverRow: -1}
	scroll := ld.get("scrollbar.tga")
	q.listScroll = newPanelScroll(scroll, 227, 52, 237)
	q.objectiveScroll = newPanelScroll(scroll, 227, 320, 130)
	q.descriptionScroll = newPanelScroll(scroll, 488, 52, 397)
	q.close = nativeClose(ld.get("talk1_close_button.tga"), 480, 7)
	q.quit = Button{Tex: ld.get("quest_button.tga"), OffX: 402, OffY: 453, DrawW: 128, DrawH: 32, HitW: 80, HitH: 26, UV: FourRowUV}
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
	if selected && in.MouseX >= x+18 && in.MouseX < x+224 && in.MouseY >= y+320 && in.MouseY < y+450 && in.Wheel != 0 {
		if in.Wheel < 0 {
			q.objectiveOffset += 3
		} else {
			q.objectiveOffset = max(0, q.objectiveOffset-3)
		}
	}
	hit := -1
	if in.MouseX >= x+18 && in.MouseX < x+227 && in.MouseY >= y+53 && in.MouseY < y+53+QuestRows*31 {
		hit = (in.MouseY - y - 53) / 31
	}
	q.hoverRow = hit
	scroll = q.listScroll.Update(in, x, y, count, QuestRows, scroll)
	if selected {
		q.textOffset = q.descriptionScroll.Update(in, x, y, q.textLines, 18, q.textOffset)
		q.objectiveOffset = q.objectiveScroll.Update(in, x, y, q.objectiveLines, 6, q.objectiveOffset)
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
		top := y + 65 + i*31
		tint := [4]float32{130.0 / 255, 128.0 / 255, 120.0 / 255, 1}
		if i == q.hoverRow {
			tint = [4]float32{.667, .667, .667, 1}
		}
		if offset+i == selected {
			tint = white
		}
		if i > 0 && q.line != nil {
			s.Draw(q.line, float32(x+46), float32(y+53+i*31), 256, 2, 0, 0, 1, 1, white)
		}
		if q.alarm != nil {
			v := float32(0)
			if offset+i < len(q.Ready) && q.Ready[offset+i] {
				v = .5
			}
			s.Draw(q.alarm, float32(x+26), float32(top-2), 16, 16, 0, v, 1, v+.5, white)
		}
		f.Draw(s, names[offset+i], x+46, top, tint, 170)
	}
	f.Draw(s, title, x+279+(207-min(207, f.Width(title)))/2, y+73, white, 207)
	lines := wrapLines(f, strings.TrimSpace(description), 196)
	q.textLines = len(lines)
	q.textOffset = min(q.textOffset, max(0, len(lines)-18))
	for i, line := range lines[q.textOffset:min(len(lines), q.textOffset+18)] {
		f.Draw(s, line, x+279, y+105+i*18, white, 196)
	}
	objectives := wrapLines(f, strings.TrimSpace(objective), 188)
	q.objectiveLines = len(objectives)
	q.objectiveOffset = min(q.objectiveOffset, max(0, len(objectives)-6))
	for i, line := range objectives[q.objectiveOffset:min(len(objectives), q.objectiveOffset+6)] {
		f.Draw(s, line, x+24, y+331+i*18, white, 188)
	}
	q.listScroll.Draw(s, x, y, len(names), QuestRows, offset)
	q.descriptionScroll.Draw(s, x, y, len(lines), 18, q.textOffset)
	q.objectiveScroll.Draw(s, x, y, len(objectives), 6, q.objectiveOffset)

	q.close.OriginX, q.close.OriginY = x, y
	q.close.Draw(s)
	q.quit.OriginX, q.quit.OriginY = x, y
	q.quit.Draw(s)
}
