package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func nativeClothFixture(mode int) *clothPhysics {
	rest := make([]render.V3, 50)
	for i := range rest {
		r, col := i/5, i%5
		rest[i] = render.V3{float32(float64(col) * .07), float32(2 - float64(r)*.10), float32(.002 * float64(col*col))}
	}
	c := newClothPhysics(rest, 5, 10, mode)
	for i := range c.positions {
		if i < 5 {
			c.pins[i] = true
			continue
		}
		c.positions[i] = render.V3{float32(float64(i%5)*.07 + .01*math.Sin(float64(i))), float32(2 - float64(i/5)*.10 - .01), float32(.002*float64((i%5)*(i%5)) + .015)}
		c.velocity[i] = render.V3{float32(math.Sin(float64(i)) * 3), float32(math.Cos(float64(i)) * 2), float32(.1 * float64(i))}
	}
	return c
}

func TestClothModeZeroMatchesNativeSyntheticKernel(t *testing.T) {
	c := nativeClothFixture(0)
	if len(c.links) != 274 {
		t.Fatalf("native directed graph: %d", len(c.links))
	}
	c.force(render.V3{.01, 0, .02})
	for _, row := range []struct {
		i     int
		force render.V3
	}{{5, render.V3{9.5560026, -6.043276, -.6290529}}, {8, render.V3{-41.49614, 11.968446, -8.968021}}, {26, render.V3{-28.491669, -39.787678, -1.4317019}}, {49, render.V3{4.9066935, 7.011084, .5689576}}} {
		if delta := c.forces[row.i].Sub(row.force).Len(); delta > .001 {
			t.Fatalf("particle %d: force %v expected %v (difference %v)", row.i, c.forces[row.i], row.force, delta)
		}
	}
}

// Expected values are one-step outputs from the original x86 kernel on a
// synthetic 5x10 grid, not samples from a proprietary character mesh.
func TestClothIntegrationMatchesBothNativeModes(t *testing.T) {
	for _, row := range []struct {
		mode, index        int
		position, velocity render.V3
	}{
		{0, 5, render.V3{-.0109320693, 1.89022326, .0152437091}, render.V3{-2.68565273, .446458876, .48741895}},
		{0, 26, render.V3{.0784845054, 1.49024904, .0182856843}, render.V3{1.71784198, .498085022, 2.57136583}},
		{1, 5, render.V3{-.0109132016, 1.89026773, .0152394138}, render.V3{-2.64791775, .53543311, .47882748}},
		{1, 26, render.V3{.0784485489, 1.4902308, .0182842519}, render.V3{1.64592314, .46156472, 2.5685029}},
	} {
		c := nativeClothFixture(row.mode)
		c.stepWithWind(.0005, render.V3{.01, 0, .02})
		if c.positions[row.index].Sub(row.position).Len() > .00001 || c.velocity[row.index].Sub(row.velocity).Len() > .0001 {
			t.Fatalf("mode %d particle %d: got %v / %v, want %v / %v", row.mode, row.index, c.positions[row.index], c.velocity[row.index], row.position, row.velocity)
		}
	}
}
