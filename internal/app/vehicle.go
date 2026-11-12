package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"strings"
)

type vehicleView struct {
	pending  bool
	deadline float64
}

func (a *App) requestVehicle() {
	if a.state == nil || a.world == nil || a.state.Self.Dead || a.vehicle.pending || a.inventory.pending || a.shop.pending {
		return
	}
	if !a.state.Self.Mounted {
		item, ok := a.state.Inventory[[2]uint8{0, 13}]
		if !ok || item.Type != 42 {
			a.questMessage("Equip a mount first.")
			return
		}
	}
	if err := a.world.Send(proto.UseVehiclePacket()); err != nil {
		a.backToLogin("Unable to use mount.")
		return
	}
	a.vehicle.pending, a.vehicle.deadline = true, a.clock+10
}
func (a *App) vehicleCommand(text string) bool {
	if !strings.EqualFold(strings.TrimSpace(text), "/mount") {
		return false
	}
	a.requestVehicle()
	return true
}
func (a *App) confirmedVehicle(p []byte) {
	if !a.vehicle.pending || proto.Op(p) != proto.UseVehicle {
		return
	}
	r, err := proto.DecodeVehicleResult(p)
	if err != nil {
		return
	}
	a.vehicle.pending = false
	if r.Result == 0 {
		a.questMessage("Mount request was refused.")
	}
}
