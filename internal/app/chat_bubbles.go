package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"strings"
)

type chatBubble struct {
	speaker uint32
	text    string
	until   float64
}

// Game.exe 0x40fbc0 replaces one actor's text and restarts its five-second timer.
// Only authoritative ordinary-chat echoes create speech bubbles.
func (a *App) confirmedChatBubble(op proto.Opcode) {
	if op != proto.ChatNormal || a.wv == nil || len(a.state.Chat) == 0 {
		return
	}
	line := a.state.Chat[len(a.state.Chat)-1]
	if line.Op != op || line.SenderID == 0 || strings.TrimSpace(line.Text) == "" {
		return
	}
	for i := range a.chatBubbles {
		if a.chatBubbles[i].speaker == line.SenderID {
			a.chatBubbles[i] = chatBubble{line.SenderID, line.Text, a.clock + 5}
			return
		}
	}

	a.chatBubbles = append(a.chatBubbles, chatBubble{line.SenderID, line.Text, a.clock + 5})
}
func (a *App) drawChatBubbles(w, h int) {
	if a.wv == nil || a.scene3d == nil {
		return
	}
	f := a.chatFont()
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	active := a.chatBubbles[:0]
	for _, b := range a.chatBubbles {
		if a.clock >= b.until {
			continue
		}
		pos, exists := a.effectPosition(b.speaker)
		if !exists {
			continue
		}
		active = append(active, b)
		if pos.Sub(a.wv.pos).Len() > 50 {
			continue
		}
		// Native 0x41c830 adds 1.5 world units to the actor's logical origin.
		pos = pos.Add(render.V3{0, 1.5, 0})
		x, y, _, ok := projectLabel(pos, vp, w, h)
		if !ok {
			continue
		}
		// Rig heights differ from the original actor origin. Keep the native
		// frame clear of the posed crown and its nameplate, including mounts.
		if actor := a.playerActor(b.speaker); actor != nil {
			if head, valid := actor.NameAnchor(); valid {
				if hx, hy, _, visible := projectLabel(head, vp, w, h); visible {
					x = hx
					y = min(y, hy-f.Height-24)
				}
			}
		}
		width := f.Width(b.text)
		left, top := x-width/2-10, y-10
		a.sprites.Draw(a.white, float32(left), float32(top), float32(width+20), 30, 0, 0, 1, 1, [4]float32{72.0 / 255, 72.0 / 255, 72.0 / 255, 128.0 / 255})
		f.Draw(a.sprites, b.text, left+10, top+(30-f.Height)/2, white, 0)
	}
	a.chatBubbles = active
}
