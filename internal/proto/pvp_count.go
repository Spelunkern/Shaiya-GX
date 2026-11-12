package proto

import "fmt"

// PvPCountUpdate is 0x020E: index 0..3 is kills, deaths, wins, losses.
// Game.exe 0x58f030 and Rapture buildKillCountUpdate both read u8 + u32.
type PvPCountUpdate struct {
	Index uint8
	Value uint32
}

func DecodePvPCountUpdate(p []byte) (PvPCountUpdate, error) {
	r := NewReader(p)
	v := PvPCountUpdate{Index: r.U8(), Value: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("pvp count: %d trailing bytes", r.Remaining())
	}
	return v, nil
}

// VisiblePvPKills is 0x0522: character ID and current kills (two u32s).
type VisiblePvPKills struct{ ID, Kills uint32 }

func DecodeVisiblePvPKills(p []byte) (VisiblePvPKills, error) {
	r := NewReader(p)
	v := VisiblePvPKills{ID: r.U32(), Kills: r.U32()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 {
		return v, fmt.Errorf("visible pvp kills: %d trailing bytes", r.Remaining())
	}
	return v, nil
}
