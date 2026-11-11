package app

import (
	"github.com/Spelunkern/shaiya-go-client/internal/game"
	"github.com/Spelunkern/shaiya-go-client/internal/platform"
)

// Autoplay injects the input a player would type to go from the login
// screen into the world (N2 check of the windowed flow).
type Autoplay struct {
	User, Password string
	StopAtSelect   bool
	Slot           int
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
		return true
	}
	return false
}
