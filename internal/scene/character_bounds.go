package scene

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Every weighted vertex lies in the convex hull of its transformed bone
// positions. Bounding each bone's influencing vertices and merging the
// transformed boxes therefore bounds the complete skinned mesh.
type boneBounds struct {
	min, max render.V3
	used     bool
}

func makeBoneBounds(vertices []render.SkinVertex, bones int) ([]boneBounds, bool) {
	count := min(bones, render.MaxBones)
	if count == 0 {
		count = 1
	}
	out := make([]boneBounds, count)
	for _, v := range vertices {
		if math.IsNaN(float64(v.Weight)) || math.IsInf(float64(v.Weight), 0) || v.Weight < 0 || v.Weight > 1 {
			return nil, false
		}
		for _, x := range v.Pos {
			if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
				return nil, false
			}
		}
		indices := [2]int{int(v.Bone0), int(v.Bone1)}
		weights := [2]float32{v.Weight, 1 - v.Weight}
		for i, weight := range weights {
			if weight == 0 {
				continue
			}
			bone := min(max(indices[i], 0), count-1)
			box := &out[bone]
			if !box.used {
				box.min, box.max, box.used = v.Pos, v.Pos, true
			} else {
				for axis := range 3 {
					box.min[axis] = min(box.min[axis], v.Pos[axis])
					box.max[axis] = max(box.max[axis], v.Pos[axis])
				}
			}
		}
	}
	return out, true
}

func unionTransformedBox(lo, hi *render.V3, valid *bool, minv, maxv render.V3, m render.M4) bool {
	if m[3] != 0 || m[7] != 0 || m[11] != 0 || m[15] != 1 {
		return false
	}
	center := minv.Add(maxv).Scale(.5)
	half := maxv.Sub(minv).Scale(.5)
	world := transformPoint(center, m)
	for axis := range 3 {
		extent := float32(math.Abs(float64(m[axis])))*half[0] + float32(math.Abs(float64(m[4+axis])))*half[1] + float32(math.Abs(float64(m[8+axis])))*half[2]
		if math.IsNaN(float64(world[axis])) || math.IsInf(float64(world[axis]), 0) || math.IsNaN(float64(extent)) || math.IsInf(float64(extent), 0) {
			return false
		}
		a, b := world[axis]-extent, world[axis]+extent
		if !*valid {
			lo[axis], hi[axis] = a, b
		} else {
			lo[axis] = min(lo[axis], a)
			hi[axis] = max(hi[axis], b)
		}
	}
	*valid = true
	return true
}

// WorldBounds bounds skinned parts and equipment in the current pose. Unknown
// geometry returns invalid so callers can use their conservative fallback.
func (c *Character) WorldBounds() (render.V3, render.V3, bool) {
	if c == nil || c.clip == nil {
		return render.V3{}, render.V3{}, false
	}
	if !c.boundReady || c.boundClip != c.clip || c.boundTime != c.Time || c.boundWorld != c.World || c.boundMounted != c.Mounted {
		finals := c.preparePose()
		lo, hi := render.V3{}, render.V3{}
		has, valid := false, true
		for _, p := range c.parts {
			if !p.bounded {
				valid = false
				break
			}
			for i, box := range p.bounds {
				if !box.used {
					continue
				}
				m := c.World
				if len(p.pose) > 0 {
					if i >= len(p.pose) {
						valid = false
						break
					}
					m = p.pose[i].Mul(c.World)
				}
				if !unionTransformedBox(&lo, &hi, &has, box.min, box.max, m) {
					valid = false
					break
				}
			}
			if !valid {
				break
			}
		}
		if valid && !c.Mounted {
			for _, a := range c.attachments {
				if a.bone < 0 || a.bone >= len(finals) {
					continue
				}
				if a.part == nil || !a.part.bounded {
					valid = false
					break
				}
				m := a.extra.Mul(finals[a.bone]).Mul(c.World)
				if !unionTransformedBox(&lo, &hi, &has, a.part.boundMin, a.part.boundMax, m) {
					valid = false
					break
				}
			}
		}
		c.boundClip, c.boundTime, c.boundWorld, c.boundMounted = c.clip, c.Time, c.World, c.Mounted
		c.boundReady, c.boundValid = true, valid && has
		if c.boundValid {
			for axis := range 3 {
				lo[axis] -= .01
				hi[axis] += .01
			}
			c.boundMin, c.boundMax = lo, hi
		}
	}
	return c.boundMin, c.boundMax, c.boundValid
}

func (c *Character) Visible(frustum render.Frustum, clip [4]float32) bool {
	// Posing every actor just to cull it dominated crowded dungeons. A box
	// from the last posed bounds, moved with the actor and padded well beyond
	// any swing between poses, rejects actors that are clearly out of view
	// without posing them; anything that may be visible gets exact bounds.
	if lo, hi, ok := c.previousBoundsEnvelope(); ok && !(render.ClipBoxVisible(clip, lo, hi) && frustum.BoxVisible(lo, hi)) {
		return false
	}
	lo, hi, valid := c.WorldBounds()
	return !valid || render.ClipBoxVisible(clip, lo, hi) && frustum.BoxVisible(lo, hi)
}

// envelopeScale and envelopePadding size the culling envelope around the
// last posed bounds (limbs, weapons and wings reaching beyond them).
const (
	envelopeScale   = 1.5
	envelopePadding = 2
)

// RayMayHit reports whether a ray can reach the actor: false only when it
// misses the envelope of the last posed bounds, so callers may skip exact
// (posed) bounds without changing their result.
func (c *Character) RayMayHit(origin, direction render.V3) bool {
	lo, hi, ok := c.previousBoundsEnvelope()
	if !ok {
		return true
	}
	return rayHitsBox(origin, direction, lo, hi)
}

// rayHitsBox is a slab test for a ray (t >= 0) against an axis-aligned box.
func rayHitsBox(origin, direction, lo, hi render.V3) bool {
	near, far := float32(0), float32(math.MaxFloat32)
	for axis := range 3 {
		if direction[axis] == 0 {
			if origin[axis] < lo[axis] || origin[axis] > hi[axis] {
				return false
			}
			continue
		}
		t0, t1 := (lo[axis]-origin[axis])/direction[axis], (hi[axis]-origin[axis])/direction[axis]
		if t0 > t1 {
			t0, t1 = t1, t0
		}
		near, far = max(near, t0), min(far, t1)
		if near > far {
			return false
		}
	}
	return true
}

// previousBoundsEnvelope is a cube around the last valid posed bounds,
// translated by the actor's movement since then; ok is false before the
// first valid pose or for a different clip or mount state.
func (c *Character) previousBoundsEnvelope() (render.V3, render.V3, bool) {
	if c == nil || !c.boundReady || !c.boundValid || c.boundClip != c.clip || c.boundMounted != c.Mounted {
		return render.V3{}, render.V3{}, false
	}
	moved := render.V3{c.World[12] - c.boundWorld[12], c.World[13] - c.boundWorld[13], c.World[14] - c.boundWorld[14]}
	center := c.boundMin.Add(c.boundMax).Scale(.5).Add(moved)
	radius := c.boundMax.Sub(c.boundMin).Len()*.5*envelopeScale + envelopePadding
	reach := render.V3{radius, radius, radius}
	return center.Sub(reach), center.Add(reach), true
}
