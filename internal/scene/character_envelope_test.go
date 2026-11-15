package scene

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func envelopeCharacter() *Character {
	c := &Character{World: render.Identity()}
	c.boundReady, c.boundValid = true, true
	c.boundWorld = render.Identity()
	c.boundMin, c.boundMax = render.V3{-1, 0, -1}, render.V3{1, 2, 1}
	return c
}

func TestEnvelopeFollowsMovementAndCoversBounds(t *testing.T) {
	c := envelopeCharacter()
	lo, hi, ok := c.previousBoundsEnvelope()
	if !ok || lo[0] > -1 || hi[1] < 2 || hi[0] < 1 {
		t.Fatalf("envelope %v %v does not cover the bounds", lo, hi)
	}
	c.World[12], c.World[14] = 50, -20 // moved without a new pose
	lo, hi, _ = c.previousBoundsEnvelope()
	if lo[0] > 49 || hi[0] < 51 || lo[2] > -21 || hi[2] < -19 {
		t.Fatalf("envelope %v %v did not follow the actor", lo, hi)
	}
	c.Mounted = true
	if _, _, ok := c.previousBoundsEnvelope(); ok {
		t.Fatal("a mount change must force exact bounds")
	}
	if _, _, ok := (&Character{}).previousBoundsEnvelope(); ok {
		t.Fatal("no pose yet must force exact bounds")
	}
}

func TestRayMayHitOnlyRejectsClearMisses(t *testing.T) {
	c := envelopeCharacter()
	if !c.RayMayHit(render.V3{0, 1, -20}, render.V3{0, 0, 1}) {
		t.Fatal("ray straight at the actor rejected")
	}
	if c.RayMayHit(render.V3{0, 1, -20}, render.V3{0, 0, -1}) {
		t.Fatal("ray pointing away accepted")
	}
	if c.RayMayHit(render.V3{30, 1, -20}, render.V3{0, 0, 1}) {
		t.Fatal("ray passing far to the side accepted")
	}
	if !(&Character{}).RayMayHit(render.V3{30, 1, -20}, render.V3{0, 0, 1}) {
		t.Fatal("without a pose the ray must be allowed")
	}
}
