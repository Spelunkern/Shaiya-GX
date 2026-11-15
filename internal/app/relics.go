package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// relicNoticeMessage follows Game.exe 0x5959a0 for 0x2103 (owners 0
// neutral, 1 light, 2 fury; faction 0 light, 1 fury): 463 "Destoyed <p>."
// when the reader's country took the relic, 462 "<p> is destroyed!" when
// it lost it; a neutral relic taken by the other country says nothing.
func relicNoticeMessage(oldOwner, newOwner, faction uint8) (int, bool) {
	switch {
	case oldOwner == 0 && newOwner == 1 && faction == 0,
		oldOwner == 0 && newOwner == 2 && faction == 1,
		oldOwner == 1 && newOwner == 2 && faction == 1,
		oldOwner == 2 && newOwner == 1 && faction == 0:
		return 463, true
	case oldOwner == 1 && newOwner == 2 && faction == 0,
		oldOwner == 2 && newOwner == 1 && faction == 1:
		return 462, true
	}
	return 0, false
}

// The notice uses message type 23 on the relic maps (0, 18, 30, 70, 81)
// and 22 elsewhere (0x5959b7).
func relicNoticeStyle(mapID uint16) uint8 {
	switch mapID {
	case 0, 18, 30, 70, 81:
		return 23
	}
	return 22
}

// relicMapMarkers adds the current map's relics, under the other markers.
func relicMapMarkers(s *game.State, mapSize float32, markers []ui.MapMarker) []ui.MapMarker {
	for _, r := range s.Relics {
		kind, ok := ui.RelicMarkerKind(r.Owner)
		if !ok {
			continue
		}
		if u, v, ok := ui.MapPosition(r.X, r.Z, mapSize); ok {
			markers = append(markers, ui.MapMarker{U: u, V: v, Kind: kind})
		}
	}
	return markers
}
