package world

import (
	"fmt"
	"math"
)

// WTR follows Parsec Wtr and Game.exe 0x5b0670. The native 0x5b0490
// divides model UVs by TileSize; 0x5b0500 assigns ARGB to the vertices;
// 0x5b0610 advances the texture using FrameMillis.
type WTR struct {
	TileSize    float32
	ARGB        uint32
	FrameMillis int32
	Textures    []string
}

func ParseWTR(b []byte) (*WTR, error) {
	r := &reader{b: b}
	w := &WTR{TileSize: r.f32(), ARGB: r.u32(), FrameMillis: r.i32()}
	if r.err != nil {
		return nil, r.err
	}
	if math.IsNaN(float64(w.TileSize)) || math.IsInf(float64(w.TileSize), 0) || w.TileSize <= 0 || w.FrameMillis < 0 {
		return nil, fmt.Errorf("wtr: invalid scale or frame interval")
	}
	n := r.count(256)
	if n > 4096 {
		return nil, fmt.Errorf("wtr: too many frames %d", n)
	}
	for i := 0; i < n && r.err == nil; i++ {
		w.Textures = append(w.Textures, r.fixed(256))
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.off != len(b) {
		return nil, fmt.Errorf("wtr: %d trailing bytes", len(b)-r.off)
	}
	return w, nil
}

func (w *WTR) FrameAt(seconds float64) int {
	if w.FrameMillis <= 0 || len(w.Textures) == 0 || seconds <= 0 {
		return 0
	}
	return int(math.Mod(math.Floor(seconds*1000/float64(w.FrameMillis)), float64(len(w.Textures))))
}
