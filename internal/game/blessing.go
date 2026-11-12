package game

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

type BlessingState struct {
	Country                   uint8
	Count, RemainingMS        uint32
	Known                     bool
	entryInit, entryBroadcast bool
}

func (b *BlessingState) apply(op proto.Opcode, value proto.BlessingUpdate, loading bool) {
	// Rapture sends a real initial snapshot and later an unconditional zero
	// placeholder in the same loading sequence. A real intervening broadcast
	// always wins, including a reset to zero. See docs/discrepancias.md #5.
	if op == proto.BlessInit && loading && b.entryInit && !b.entryBroadcast && b.Known && b.Country == value.Country && b.Count != 0 && value.Count == 0 && value.RemainingMS == 0 {
		return
	}
	if op == proto.CountryKillCount {
		if b.Country != value.Country {
			b.RemainingMS = 0
		}
		if value.Count == 0 {
			b.RemainingMS = 0
		}
		if loading {
			b.entryBroadcast = true
		}
	} else {
		b.RemainingMS = value.RemainingMS
		if loading {
			b.entryInit = true
		}
	}
	b.Country, b.Count, b.Known = value.Country, value.Count, true
}
