package app

import (
	"fmt"

	"github.com/Spelunkern/shaiya-go-client/internal/game"
	"github.com/Spelunkern/shaiya-go-client/internal/proto"
)

// Preview puts a screen in a fixed state for offscreen screenshots.
func (a *App) Preview(name string) error {
	switch name {
	case "login":
		a.login.ID.Text, a.login.Password.Text = "admin1", "secret"
	case "login-settled":
		a.login.ID.Text, a.login.Password.Text = "admin1", "secret"
		if ls := a.loginBackdrop(); ls != nil {
			ls.Camera.Skip()
		}
	case "servers":
		a.servers.Servers = []proto.Server{
			{ID: 1, Name: "Rapture", Users: 10, Capacity: 100},
			{ID: 2, Name: "Busy", Users: 50, Capacity: 100},
			{ID: 3, Name: "Normal", Users: 40, Capacity: 100},
			{ID: 4, Name: "Full", Users: 90, Capacity: 100},
			{ID: 5, Name: "Closed", Status: 2},
		}
		a.servers.Selected = 0
		a.screen = screenServers
	case "chars":
		a.state = game.New()
		a.state.Phase = game.PhaseCharSelect
		a.state.Slots[0] = proto.CharacterSlot{Slot: 0, ID: 1, Name: "Interloper", Level: 60, Race: 0, Class: 0, Mode: 2, MapID: 42, Hair: 1, Face: 1, Height: 2}
		a.state.Slots[1] = proto.CharacterSlot{Slot: 1, ID: 2, Name: "Valiant", Level: 60, Class: 1, Mode: 2, MapID: 1}
		a.state.Slots[3] = proto.CharacterSlot{Slot: 3, ID: 4, Name: "Sylvan", Level: 12, Class: 4, Mode: 1, MapID: 3}
		a.screen = screenCharacters
		a.chars.SetChars(a.state.Slots)
		a.world = nil
		return nil
	default:
		return fmt.Errorf("unknown preview %q", name)
	}
	return nil
}
