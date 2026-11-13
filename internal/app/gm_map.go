package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/gm"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// gmMapTeleport sends /goto for a point clicked on the full map, given in map
// image coordinates (u right, v down). The client supplies the terrain height
// because the server keeps no terrain geometry; GX Server checks the GM role,
// map bounds and walkability.
func (a *App) gmMapTeleport(u, v float32) {
	w := a.wv
	if w == nil || w.world == nil || w.world.WLD == nil || a.world == nil || u < 0 || u > 1 || v < 0 || v > 1 {
		return
	}
	size := float32(w.world.WLD.MapSize)
	if size <= 0 {
		return
	}
	x, z := mapPointToWorld(u, v, size)
	text := gm.GotoXZCommand(x, z)
	if w.world.HasTerrain() {
		text = gm.GotoCommand(x, z, w.world.Height(x, z))
	}
	if err := a.world.Send(proto.ChatPacket(proto.ChannelNormal, false, text)); err != nil {
		log.Printf("GM map teleport: %v", err)
	}
}

// mapPointToWorld inverts ui.MapPosition (u = x/size, v = 1 - z/size).
func mapPointToWorld(u, v, size float32) (x, z float32) { return u * size, (1 - v) * size }

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
