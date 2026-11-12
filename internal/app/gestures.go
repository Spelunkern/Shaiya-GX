package app

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

var gestures = map[string]uint8{
	"/victory": 116, "/beg": 117, "/love": 118, "/laugh": 119, "/clap": 120,
	"/greet": 121, "/start": 122, "/forgive": 123, "/provoke": 124, "/insult": 125,
}

func cancelGesture(actor *scene.Character) {
	if actor.PlayingGesture() {
		actor.ReleaseAction()
	}
}

// 0x560e30 uses sysmsg 5228..5237 and substitutes the actor name into <p>.
// As in the original, an unknown name or missing localized text emits nothing.
func (a *App) confirmedGesture(op proto.Opcode) {
	if op != proto.CharacterMotion || a.state == nil {
		return
	}
	m := a.state.LastMotion
	if _, ok := proto.GestureAction(m.Motion); !ok {
		return
	}
	name := ""
	if m.ID == a.state.SelfID {
		for _, slot := range a.state.Slots {
			if slot.ID == m.ID {
				name = slot.Name
				break
			}
		}
	} else if entity := a.state.Entities[m.ID]; entity != nil && entity.Shape != nil {
		name = entity.Shape.Name
	}
	message := a.msg.Get(5228 + int(m.Motion-116))
	if name != "" && message != "" {
		a.questMessage(strings.ReplaceAll(message, "<p>", name))
	}
}

// These commands expose the ten original social motions until the native
// gesture panel is reconstructed. Only the server echo starts the animation.
func (a *App) gestureCommand(text string) bool {
	motion, ok := gestures[strings.ToLower(strings.TrimSpace(text))]
	if !ok {
		return false
	}
	a.sendGesture(motion)
	return true
}

func (a *App) sendGesture(motion uint8) {
	if a.state == nil || a.wv == nil || a.world == nil {
		return
	}
	if a.state.Self.Dead || a.state.Self.Mounted || a.state.Self.Sitting || a.wv.sitPending || a.wv.self != nil && a.wv.self.PlayingAction(11) {
		a.questMessage("Stand before using a gesture.")
		return
	}
	if a.wv.moving || a.wv.world.SwimmingAt(a.wv.pos[0], a.wv.pos[1], a.wv.pos[2]) {
		a.questMessage("Stop moving before using a gesture.")
		return
	}
	if err := a.world.Send(proto.StopAttackPacket()); err != nil {
		a.backToLogin("Unable to stop attack before gesture.")
		return
	}
	if err := a.world.Send(proto.MotionPacket(motion)); err != nil {
		a.backToLogin("Unable to send gesture.")
	}
}
