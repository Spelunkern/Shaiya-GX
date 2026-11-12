package effect

import "math"

// Parameters maps the raw EFT fields through Game.exe 0x559890 to the
// runtime consumed by 0x55de50. Parsec's blend and timing names differ.
type Parameters struct {
	SourceBlend, DestinationBlend                int32
	LifeMin, LifeMax, RateMin, RateMax, Duration float32
	VelocityMin, VelocityMax, PositionSpread     [3]float32
}

func (e *Emitter) Parameters() Parameters {
	return Parameters{e.DstBlend, e.SrcBlend, e.InitialDelay, e.Unk13, e.Unk14, e.Unk12, e.Unk16, e.Spread1, e.Spread2, e.OffsetFrame}
}

// LinearSupported checks particle physics and facing. Geometry support must
// also be checked when loading a referenced 3DE mesh.
func (e *Emitter) LinearSupported() bool {
	// The native velocity curve divides consecutive key times. Keep malformed
	// curves unsupported rather than feeding non-finite values to the renderer.
	for i := 1; i < len(e.Opacity); i++ {
		a, b := e.Opacity[i-1], e.Opacity[i]
		if b.Time <= a.Time || math.Abs((float64(b.Opacity)-float64(a.Opacity))/(float64(b.Time)-float64(a.Time))) > math.MaxFloat32 {
			return false
		}
	}
	p := e.Parameters()
	colorOK := len(e.Colors) == 0 || len(e.Colors) >= 2 && e.Colors[0].Time == 0 && e.Colors[1].Time > 0
	// Without forces or a velocity curve, zero-life particles can also move
	// indefinitely. Their spatial bound is handled separately by the renderer.
	zeroVelocity := e.Spread1 == [3]float32{} && e.Spread2 == [3]float32{}
	stationary := p.LifeMin == 0 && p.LifeMax == 0 && e.Unk20 == 0 && e.Unk21 == 0 && (zeroVelocity || len(e.Opacity) == 0) && colorOK
	if stationary && len(e.Colors) >= 2 {
		for i := range e.Colors[0].RGBA {
			if math.Abs((float64(e.Colors[1].RGBA[i])-float64(e.Colors[0].RGBA[i]))/float64(e.Colors[1].Time)) > math.MaxFloat32 {
				stationary = false
			}
		}
	}
	if stationary && len(e.Sub3) >= 2 && (e.Sub3[0].Time != 0 || e.Sub3[1].Time <= 0) {
		stationary = false
	}
	spin := e.RotationEnabled == 0 || e.RotationAxis >= 1 && e.RotationAxis <= 3
	initial := e.Unk31 == 0 && e.Unk32 == 0 || e.Unk30 >= 1 && e.Unk30 <= 3
	return e.Unk06 >= 0 && e.Unk06 <= 3 && e.BaseAxis >= 0 && e.BaseAxis <= 3 && spin && initial && len(e.Textures) > 0 && !math.IsNaN(float64(e.FrameDelay)) && !math.IsInf(float64(e.FrameDelay), 0) && (p.LifeMin > 0 && p.LifeMax > 0 || stationary) && p.SourceBlend >= 0 && p.SourceBlend <= 12 && p.DestinationBlend >= 0 && p.DestinationBlend <= 12
}

// UsesMesh follows 0x55cb10: +0x17C (Parsec Unknown10) forces the billboard
// branch even when a mesh index is present. +0x100 (Unknown8) loops its clock.
func (e *Emitter) UsesMesh() bool { return e.Mesh >= 0 && e.Unk10 == 0 }
