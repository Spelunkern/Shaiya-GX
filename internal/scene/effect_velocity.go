package scene

import "github.com/Spelunkern/Shaiya-GX/internal/data/effect"

// Native 0x55de50/0x55d270: Parsec's Opacity keys drive dv/dt = v*a(t).
// State starts at zero (0x55afc0), advances at most one key per tick and
// retains the last slope. This is separate from the RGBA alpha curve.
type velocityCurve struct {
	key          int
	value, slope float32
}

func (c *velocityCurve) step(keys []effect.OpacityFrame, age, dt float32) float32 {
	if c.key < len(keys) && age > keys[c.key].Time {
		if c.key+1 < len(keys) {
			a, b := keys[c.key], keys[c.key+1]
			c.value = a.Opacity
			c.slope = float32((float64(b.Opacity) - float64(a.Opacity)) / (float64(b.Time) - float64(a.Time)))
		}
		c.key++
	}
	c.value += dt * c.slope
	return c.value
}
