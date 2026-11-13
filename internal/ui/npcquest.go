package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"strings"
)

const NPCQuestWidth = 256
const NPCQuestRows = 6

type NPCQuest struct {
	Position                    WindowPosition
	list, take, complete        *render.Texture
	close, action, finish, back Button
	serviceQuest, serviceReturn Button
	held                        int
	textOffset                  int
}

func (v *NPCQuest) ResetDescriptionScroll() { v.textOffset = 0 }

type NPCQuestAction struct {
	Row, Offset         int
	Close, Back, Accept bool
}

func NewNPCQuest(t Textures) (*NPCQuest, error) {
	ld := &loader{t: t}
	v := &NPCQuest{list: ld.get("talk1_bg.png"), take: ld.get("quest_take.png"), complete: ld.get("quest_complete.png"), held: -1}
	v.close = nativeClose(ld.get("talk1_close_button.png"), 228, 5)
	v.action = Button{Tex: ld.get("quest_take_button.png"), OffX: 53, DrawW: 128, DrawH: 32, HitW: 67, HitH: 27, UV: FourRowUV}
	v.finish = Button{Tex: ld.get("talk_button.png"), OffX: 53, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: [5]UV{{0, 0, .25, .5}, {.25, 0, .5, .5}, {.5, 0, .75, .5}, {.5, 0, .75, .5}, {.75, 0, 1, .5}}}
	v.serviceQuest = Button{Tex: ld.get("talk_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: [5]UV{{0, .5, .25, 1}, {.25, .5, .5, 1}, {.5, .5, .75, 1}, {.5, .5, .75, 1}, {.75, .5, 1, 1}}}
	v.serviceReturn = Button{Tex: ld.get("talk_blank_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: v.finish.UV}
	v.back = Button{Tex: ld.get("cancel_button.png"), OffX: 145, DrawW: 128, DrawH: 32, HitW: 67, HitH: 27, UV: FourRowUV}
	return v, ld.err
}

// ServiceButton switches between a service and the NPC's quest conversation.
func (v *NPCQuest) ServiceButton(in *platform.Input, x, y int, returning bool) bool {
	b := &v.serviceQuest
	if returning {
		b = &v.serviceReturn
	}
	b.OriginX, b.OriginY = x, y
	return b.Update(in)
}
func (v *NPCQuest) DrawServiceButton(s *render.Sprites, f *Font, x, y int, returning bool, label string) {
	b := &v.serviceQuest
	if returning {
		b = &v.serviceReturn
	}
	b.OriginX, b.OriginY = x, y
	b.Draw(s)
	if returning {
		f.Draw(s, label, x+8, y+6, white, 54)
	}
}
func NPCQuestOrigin(w, h int) (int, int) {
	return min(32, max(0, w-NPCQuestWidth)), min(82, max(0, h-512))
}
func (v *NPCQuest) Origin(w, h int) (int, int) {
	x, y := MarketOrigin(w, h)
	return v.Position.Origin(w, h, NPCQuestWidth, 512, x, y)
}
func (v *NPCQuest) Update(in *platform.Input, w, h, count, offset int, detail, ending bool, choices int) NPCQuestAction {
	x, y := v.Origin(w, h)
	height := 391
	if detail {
		height = 512
	}
	v.close.OriginX, v.close.OriginY = x, y
	v.action.OriginX, v.action.OriginY = x, y
	v.finish.OriginX, v.finish.OriginY = x, y
	v.back.OriginX, v.back.OriginY = x, y
	v.action.OffY = height - 46
	v.finish.OffY = height - 46
	v.back.OffY = height - 46
	out := NPCQuestAction{Row: -1, Offset: max(0, min(offset, max(0, count-NPCQuestRows)))}
	out.Close = v.close.Update(in)
	if detail {
		bottom := y + 437
		if choices > 0 {
			bottom = y + 300
		}
		if in.MouseX >= x+20 && in.MouseX < x+235 && in.MouseY >= y+110 && in.MouseY < bottom && in.Wheel != 0 {
			if in.Wheel < 0 {
				v.textOffset += 3
			} else {
				v.textOffset = max(0, v.textOffset-3)
			}
		}
		if choices > 0 {
			row := -1
			if in.MouseX >= x+20 && in.MouseX < x+235 && in.MouseY >= y+310 && in.MouseY < y+310+choices*40 {
				row = (in.MouseY - y - 310) / 40
			}
			if in.Pressed {
				v.held = row
			}
			if in.Clicked {
				if row >= 0 && row == v.held {
					out.Row = row
				}
				v.held = -1
			}
			out.Back = v.back.Update(in)
			return out
		}
		if ending {
			out.Accept = v.finish.Update(in)
		} else {
			out.Accept = v.action.Update(in)
		}
		out.Back = v.back.Update(in)
		return out
	}
	if in.MouseX >= x+16 && in.MouseX < x+240 && in.MouseY >= y+142 && in.MouseY < y+310 && in.Wheel != 0 {
		d := 1
		if in.Wheel < 0 {
			d = -1
		}
		out.Offset = max(0, min(max(0, count-NPCQuestRows), out.Offset-d))
	}
	row := -1
	if in.MouseX >= x+18 && in.MouseX < x+238 && in.MouseY >= y+142 && in.MouseY < y+142+NPCQuestRows*26 {
		row = (in.MouseY - y - 142) / 26
	}
	if in.Pressed {
		v.held = row
	}
	if in.Clicked {
		if row >= 0 && row == v.held && out.Offset+row < count {
			out.Row = out.Offset + row
		}
		v.held = -1
	}
	return out
}
func (v *NPCQuest) Draw(s *render.Sprites, f *Font, w, h int, name, greeting string, portrait *render.Texture, entries []string, offset int, detail, ending bool, title, description string, choices []string) {
	x, y := v.Origin(w, h)
	height := 391
	frame := v.list
	if detail {
		height = 512
		frame = v.take
		if ending {
			frame = v.complete
		}
	}
	s.Draw(frame, float32(x), float32(y), 256, float32(height), 0, 0, 1, float32(height)/512, white)
	if portrait != nil {
		s.Draw(portrait, float32(x+7), float32(y+7), 64, 64, 0, 0, 1, 1, white)
	}
	f.Draw(s, name, x+77, y+22, white, 160)
	if !detail {
		for i, line := range wrapLines(f, greeting, 210) {
			if i >= 3 {
				break
			}
			f.Draw(s, line, x+20, y+85+i*15, white, 210)
		}
		for i := 0; i < NPCQuestRows && offset+i < len(entries); i++ {
			f.Draw(s, entries[offset+i], x+22, y+146+i*26, white, 210)
		}
	} else {
		f.Draw(s, title, x+25, y+86, white, 205)
		lines := wrapLines(f, description, 205)
		limit := 23
		if len(choices) > 0 {
			limit = 13
		}
		v.textOffset = min(v.textOffset, max(0, len(lines)-limit))
		for i, line := range lines[v.textOffset:min(len(lines), v.textOffset+limit)] {
			f.Draw(s, line, x+25, y+115+i*14, white, 205)
		}
		v.action.OriginX, v.action.OriginY = x, y
		v.finish.OriginX, v.finish.OriginY = x, y
		v.back.OriginX, v.back.OriginY = x, y
		v.action.OffY = height - 46
		v.finish.OffY = height - 46
		v.back.OffY = height - 46
		if len(choices) > 0 {
			for i, choice := range choices {
				for line, text := range strings.SplitN(choice, "\n", 2) {
					f.Draw(s, text, x+25, y+313+i*40+line*14, white, 200)
				}
			}
		} else if !ending {
			v.action.Draw(s)
		} else {
			v.finish.Draw(s)
		}
		v.back.Draw(s)
	}
	v.close.OriginX, v.close.OriginY = x, y
	v.close.Draw(s)
}
