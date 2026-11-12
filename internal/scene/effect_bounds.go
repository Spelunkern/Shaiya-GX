package scene

import (
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// emitterReach bounds the physics implemented by Effect.Update. Unknown or
// non-finite data must disable culling instead of hiding visible particles.
func emitterReach(d *effect.Emitter, meshRadius float64) float32 {
	p := d.Parameters()
	lifetime := math.Max(float64(p.LifeMin), float64(p.LifeMax))
	origin := render.V3(d.Position).Len() + render.V3(p.PositionSpread).Len()
	var speedSquared, resetSquared float64
	for i := range 3 {
		v := math.Max(math.Abs(float64(p.VelocityMin[i])), math.Abs(float64(p.VelocityMax[i])))
		speedSquared += v * v
		if [3]int32{d.Unk01, d.Unk02, d.Unk03}[i] != 0 {
			resetSquared += v * v
		}
	}
	// Mesh particles with zero life may travel forever, as may orbiting
	// particles with nonzero drift. A linear billboard is reanchored every
	// step, so its maximum offset is one step at maximum authored velocity.
	zeroLife := p.LifeMin == 0 && p.LifeMax == 0
	if zeroLife && speedSquared > 0 && (d.UsesMesh() || d.Unk06 != 0 || d.Unk20 != 0 || d.Unk21 != 0 || len(d.Opacity) != 0) {
		return float32(math.Inf(1))
	}
	var maxValue, maxSlope float64
	for i, key := range d.Opacity {
		maxValue = math.Max(maxValue, math.Abs(float64(key.Opacity)))
		if i > 0 {
			prev := d.Opacity[i-1]
			delta := float64(key.Time - prev.Time)
			if delta <= 0 {
				return float32(math.Inf(1))
			}
			maxSlope = math.Max(maxSlope, math.Abs(float64(key.Opacity-prev.Opacity)/delta))
		}
	}
	acceleration := maxValue + lifetime*maxSlope
	// The runtime integrates v *= 1 + a/60. exp(|a|*t) is an upper bound
	// for its magnitude, including negative multipliers and random resets.
	travel := math.Sqrt(speedSquared) * lifetime * math.Exp(acceleration*lifetime)
	if zeroLife {
		travel = math.Sqrt(speedSquared) / 60
	}
	var force float64
	if d.Unk20 != 0 {
		force += float64(render.V3(d.Trembling).Len())
	}
	if d.Unk21 != 0 {
		force += math.Abs(float64(d.RotationMin))
	}
	if force != 0 || d.Unk06 != 0 {
		// The same authored curve is deterministic for all particles. Bound the
		// fixed-step recurrence directly: arbitrary component resets can add at
		// most their squared maxima, rotations preserve length, and forces and
		// drift add at most their magnitudes. This avoids exponential guesses
		// that keep whole maps resident for ordinary bounded effects.
		if math.IsNaN(lifetime) || math.IsInf(lifetime, 0) || lifetime < 0 {
			return float32(math.Inf(1))
		}
		speed, initial := math.Sqrt(speedSquared), math.Sqrt(speedSquared)
		curve, age := velocityCurve{}, float32(0)
		travel = 0
		for step := 0; ; step++ {
			age += 1.0 / 60
			if float64(age) > lifetime {
				break
			}
			if step >= 60000 {
				return float32(math.Inf(1))
			}
			if resetSquared != 0 {
				speed = math.Sqrt(speed*speed + resetSquared)
			}
			if d.Unk06 == 0 {
				travel += speed / 60
			} else {
				speed += initial / 60
			}
			if len(d.Opacity) != 0 {
				value := curve.step(d.Opacity, age, 1.0/60)
				speed *= math.Abs(1 + float64(value)/60)
			}
			speed = (speed + force/60) * 1.000001
			if d.Unk06 != 0 {
				travel = math.Max(travel, speed)
			}
		}
		if d.Unk06 != 0 {
			travel = math.Max(travel, initial)
		}
	}
	size := .3
	if len(d.Sub3) != 0 {
		size = 0
	}
	for _, key := range d.Sub3 {
		size = math.Max(size, math.Max(math.Abs(float64(key.Unk1)), math.Abs(float64(key.Unk2))))
	}
	reach := 2*float64(origin) + travel + 2*size*meshRadius + 1
	if lifetime < 0 || math.IsNaN(reach) || math.IsInf(reach, 0) || reach > math.MaxFloat32 {
		return float32(math.Inf(1))
	}
	return float32(reach)
}

func effectSequenceBounds(a *Assets, lib *effect.Library) []float32 {
	geometry := make(map[int32]float64)
	out := make([]float32, len(lib.Sequences))
	for i, sequence := range lib.Sequences {
		for _, record := range sequence.Records {
			if record.Effect < 0 || int(record.Effect) >= len(lib.Effects) {
				out[i] = float32(math.Inf(1))
				break
			}
			d := &lib.Effects[record.Effect]
			if !d.LinearSupported() {
				continue
			}
			radius := 1.0
			if d.UsesMesh() {
				var ok bool
				radius, ok = geometry[d.Mesh]
				if !ok {
					radius = math.Inf(1)
					if int(d.Mesh) < len(lib.Meshes) {
						name := path.Base(strings.ReplaceAll(lib.Meshes[d.Mesh], `\`, "/"))
						if raw, err := a.Arc.ReadFile("Effect/3DE/" + name); err == nil {
							if mesh, err := effect.Parse3DE(raw); err == nil {
								radius = 0
								for _, vertex := range mesh.Vertices {
									radius = math.Max(radius, float64(render.V3(vertex.Position).Len()))
								}
								for _, frame := range mesh.Frames {
									for _, vertex := range frame.Vertices {
										radius = math.Max(radius, float64(render.V3(vertex.Position).Len()))
									}
								}
								// Native key interpolation may extrapolate up to two key units.
								radius *= 5
							}
						}
					}
					geometry[d.Mesh] = radius
				}
			}
			out[i] = max(out[i], emitterReach(d, radius))
		}
	}
	return out
}
