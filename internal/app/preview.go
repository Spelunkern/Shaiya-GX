package app

import (
	"fmt"

	"github.com/Spelunkern/shaiya-go-client/internal/proto"
)

// Preview puts a screen in a fixed state for offscreen screenshots.
func (a *App) Preview(name string) error {
	switch name {
	case "login":
		a.login.ID.Text, a.login.Password.Text = "admin1", "secret"
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
	default:
		return fmt.Errorf("unknown preview %q", name)
	}
	return nil
}
