package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

// Autoplay injects the input a player would type to go from the login
// screen into the world (N2 check of the windowed flow).
type Autoplay struct {
	User, Password string
	StopAtSelect   bool
	Slot           int
	CastSlot       uint8 // 0 disables; 1..10 activates an assigned slot.
	WaitResult     bool  // Capture flight after 0x0511/0x0517 instead of preparation.
	step           int
	wait           float64
}

// Inject adds this frame's synthetic input; it returns true once in world.
func (p *Autoplay) Inject(a *App, in *platform.Input, dt float64) bool {
	p.wait -= dt
	if p.wait > 0 {
		return false
	}
	p.wait = 0.2
	switch {
	case a.screen == screenLogin && p.step == 0:
		in.Chars = append(in.Chars, []rune(p.User)...)
		in.Keys = append(in.Keys, platform.KeyTab)
		p.step = 1
	case a.screen == screenLogin && p.step == 1:
		in.Chars = append(in.Chars, []rune(p.Password)...)
		in.Keys = append(in.Keys, platform.KeyEnter)
		p.step = 2
	case a.screen == screenServers && p.step == 2:
		in.Keys = append(in.Keys, platform.KeyEnter)
		p.step = 3
	case a.state != nil && a.state.Phase == game.PhaseCharSelect && p.step == 3:
		if p.StopAtSelect {
			if p.Slot >= 0 && p.Slot < len(a.chars.Chars) && a.chars.Chars[p.Slot] != nil {
				a.chars.Selected = p.Slot
			}
			p.wait = 2 // let the actor load and animate
			p.step = 5
			return false
		}
		in.Keys = append(in.Keys, platform.KeyEnter) // first character is preselected
		p.step = 4
	case p.step == 5:
		return true
	case a.state != nil && a.state.Phase == game.PhaseInWorld:
		if !a.SceneReady() {
			return false
		}
		if p.CastSlot > 0 && p.CastSlot <= 10 {
			if p.step == 4 {
				in.Keys = append(in.Keys, platform.KeyTarget)
				p.step = 6
				return false
			}
			if p.step == 6 {
				e := a.state.Entities[a.state.Target]
				if e == nil || e.Dead || e.HealthKnown && e.HP <= 0 {
					in.Keys = append(in.Keys, platform.KeyTarget)
					return false
				}
				if !e.HealthKnown {
					return false
				}
				// A restored cooldown may still be running after the fixture's
				// previous session. The server silently ignores an early cast request.
				skill := a.skillBar()[p.CastSlot-1]
				if skill == nil || a.skills.cooldowns[skill.ID] > a.clock {
					return false
				}
				key := rune('0')
				if p.CastSlot < 10 {
					key = rune('0' + p.CastSlot)
				}
				in.Chars = append(in.Chars, key)
				p.step = 7
				if p.WaitResult {
					p.wait = 0
				}
				return false
			}
			if p.WaitResult {
				p.wait = 0
				if a.state.LastSkill.Source != a.state.SelfID {
					return false
				}
			} else if a.state.LastCast.Source != a.state.SelfID {
				return false
			}
		}
		return true
	}
	return false
}
