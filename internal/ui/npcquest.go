package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"strings"
)

const NPCQuestWidth = 256
const NPCQuestRows = 6

type NPCQuest struct {
	hoverRow                    int
	storyScroll, listScroll     panelScroll
	textLines                   int
	RewardText                  string
	Detail                      bool
	Position                    WindowPosition
	HasService, ActionDisabled  bool
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
	v := &NPCQuest{list: ld.get("talk1_bg.png"), take: ld.get("quest_take.png"), complete: ld.get("quest_complete.png"), held: -1, hoverRow: -1}
	v.storyScroll = newPanelScroll(ld.get("scrollbar.png"), 227, 114, 329)
	v.listScroll = newPanelScroll(ld.get("scrollbar.png"), 227, 142, 166)
	v.close = nativeClose(ld.get("talk1_close_button.png"), 228, 5)
	v.action = Button{Tex: ld.get("quest_take_button.png"), OffX: 53, DrawW: 128, DrawH: 32, HitW: 67, HitH: 27, UV: npcVerticalUV}
	v.finish = Button{Tex: ld.get("talk_button.png"), OffX: 53, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: npcHorizontalUV(0)}
	v.serviceQuest = Button{Tex: ld.get("talk_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: npcHorizontalUV(1)}
	v.serviceReturn = Button{Tex: ld.get("talk_blank_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: v.finish.UV}
	v.back = Button{Tex: ld.get("cancel_button.png"), OffX: 145, DrawW: 128, DrawH: 32, HitW: 67, HitH: 27, UV: npcVerticalUV}
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
	height := 391
	if v.Detail {
		height = 512
	}
	return v.Position.Origin(w, h, NPCQuestWidth, height, x, y)
}
func (v *NPCQuest) Update(in *platform.Input, w, h, count, offset int, detail, ending bool, choices int) NPCQuestAction {
	v.Detail = detail
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
	if !detail {

		v.finish.OffY = 336
	}
	v.action.Disabled, v.finish.Disabled = v.ActionDisabled, detail && v.ActionDisabled
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
		visible := 23
		scrollHeight := 329
		if ending || choices > 0 {
			visible = 13
			scrollHeight = 183
		}
		v.storyScroll.height = scrollHeight
		v.storyScroll.arrows[1].OffY = 114 + scrollHeight - 19
		if v.textLines > 0 {
			v.textOffset = v.storyScroll.Update(in, x, y, v.textLines, visible, v.textOffset)
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
			if !v.HasService {
				out.Back = v.back.Update(in)
			}
			return out
		}
		if ending {
			out.Accept = v.finish.Update(in)
		} else {
			out.Accept = v.action.Update(in)
		}
		if !v.HasService {
			out.Back = v.back.Update(in)
		}
		return out
	}
	if v.finish.Update(in) {
		out.Close = true
	}
	if count > 0 && !v.HasService {
		v.serviceQuest.OriginX, v.serviceQuest.OriginY = x+145, y+336
		if v.serviceQuest.Update(in) {
			out.Row = out.Offset
		}
	}
	if in.MouseX >= x+16 && in.MouseX < x+240 && in.MouseY >= y+142 && in.MouseY < y+310 && in.Wheel != 0 {
		d := 1
		if in.Wheel < 0 {
			d = -1
		}
		out.Offset = max(0, min(max(0, count-NPCQuestRows), out.Offset-d))
	}
	out.Offset = v.listScroll.Update(in, x, y, count, NPCQuestRows, out.Offset)
	row := -1
	if in.MouseX >= x+18 && in.MouseX < x+238 && in.MouseY >= y+142 && in.MouseY < y+142+NPCQuestRows*26 {
		row = (in.MouseY - y - 142) / 26
	}
	v.hoverRow = row
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
	v.Detail = detail
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
	if !detail {
		f.Draw(s, "Choose a quest or service.", x+26, y+89, RGBColor(0xeee4ce), 196)
		v.finish.OriginX, v.finish.OriginY = x, y
		v.finish.OffY = 336
		v.finish.Disabled = false
		v.finish.Draw(s)
		if len(entries) > 0 && !v.HasService {
			v.serviceQuest.OriginX, v.serviceQuest.OriginY = x+145, y+336
			v.serviceQuest.Draw(s)
		}
		for i := 0; i < NPCQuestRows && offset+i < len(entries); i++ {
			tint := RGBColor(0xeee4ce)
			if i == v.hoverRow {
				tint = RGBColor(0xffd67f)
			}
			f.Draw(s, entries[offset+i], x+26, y+146+i*26, tint, 196)
		}
		if len(entries) > NPCQuestRows {
			v.listScroll.Draw(s, x, y, len(entries), NPCQuestRows, offset)
		}
	} else {
		f.Draw(s, title, x+25, y+86, white, 205)
		lines := wrapLines(f, description, 196)
		limit := 23
		if ending || len(choices) > 0 {
			limit = 13
		}
		v.textLines = len(lines)
		v.textOffset = min(v.textOffset, max(0, len(lines)-limit))
		for i, line := range lines[v.textOffset:min(len(lines), v.textOffset+limit)] {
			f.Draw(s, line, x+25, y+115+i*14, white, 205)
		}
		scrollHeight := 329
		if ending || len(choices) > 0 {
			scrollHeight = 183
		}
		v.storyScroll.height = scrollHeight
		v.storyScroll.arrows[1].OffY = 114 + scrollHeight - 19
		if len(lines) > limit {
			v.storyScroll.Draw(s, x, y, len(lines), limit, v.textOffset)
		}
		if ending && len(choices) == 0 && v.RewardText != "" {
			for i, line := range wrapLines(f, v.RewardText, 196) {
				if i >= 8 {
					break
				}
				f.Draw(s, line, x+26, y+310+i*15, RGBColor(0xeee4ce), 196)
			}
		}
		v.action.OriginX, v.action.OriginY = x, y
		v.finish.OriginX, v.finish.OriginY = x, y
		v.back.OriginX, v.back.OriginY = x, y
		v.action.OffY = height - 46
		v.finish.OffY = height - 46
		if !detail {
			v.finish.OffY = 336
		}
		v.action.Disabled, v.finish.Disabled = v.ActionDisabled, detail && v.ActionDisabled
		v.back.OffY = height - 46
		if len(choices) > 0 {
			for i, choice := range choices {
				for line, text := range strings.SplitN(choice, "\n", 2) {
					f.Draw(s, text, x+25, y+313+i*40+line*14, white, 200)
				}
			}
			v.finish.Disabled = true
			v.finish.Draw(s)
		} else if !ending {
			v.action.Draw(s)
		} else {
			v.finish.Draw(s)
		}
		if !v.HasService {
			v.back.Draw(s)
		}
	}
	v.close.OriginX, v.close.OriginY = x, y
	v.close.Draw(s)
}
