package scene

import (
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// The LH perspective matrix stores the far plane in m[14]/(1-m[10]).
// Invalid or non-perspective matrices keep a placement eligible.
func mapEffectInRange(proj render.M4, distanceSquared, radius float32) bool {
	reach, valid := mapEffectCameraReach(proj)
	return mapEffectWithinReach(reach, valid, distanceSquared, radius)
}

func mapEffectCameraReach(proj render.M4) (float32, bool) {
	denominator := 1 - proj[10]
	if denominator >= 0 || proj[14] >= 0 || proj[0] == 0 || proj[5] == 0 {
		return 0, false
	}
	far := proj[14] / denominator
	if far <= 0 || math.IsInf(float64(far), 0) || math.IsNaN(float64(far)) {
		return 0, false
	}
	x, y := float64(1/proj[0]), float64(1/proj[5])
	return far * float32(math.Sqrt(1+x*x+y*y)), true
}

func mapEffectWithinReach(cameraReach float32, valid bool, distanceSquared, radius float32) bool {
	if !valid {
		return true
	}
	reach := cameraReach + radius
	return distanceSquared <= reach*reach
}

type mapEffect struct {
	position render.V3
	world    render.M4
	sequence int
	radius   float32
	instance *Effect
}

type mapEffects struct {
	assets    *Assets
	file      string
	library   *effect.Library
	instances []mapEffect
	active    []int
	failed    map[int]bool
}

func (w *World) loadMapEffects(parent *Assets) {
	if len(w.WLD.Effects) == 0 || w.WLD.EffectFile == "" {
		return
	}
	name := path.Base(strings.ReplaceAll(w.WLD.EffectFile, `\`, "/"))
	file := "Effect/" + name
	data, err := parent.Arc.ReadFile(file)
	if err != nil {
		return
	}
	lib, err := effect.ParseEFT(data)
	if err != nil || lib.Format != "EFT" {
		return
	}
	f := &mapEffects{assets: NewAssets(parent.Arc), file: file, library: lib, failed: make(map[int]bool)}
	bounds := effectSequenceBounds(parent, lib)
	for _, placement := range w.WLD.Effects {
		if placement.Effect < 0 || int(placement.Effect) >= len(lib.Sequences) {
			continue
		}
		pos := render.V3(placement.Pos)
		f.instances = append(f.instances, mapEffect{position: pos, world: render.Basis(pos, render.V3(placement.Fwd), render.V3(placement.Up)), sequence: int(placement.Effect), radius: bounds[placement.Effect]})
	}
	w.mapFX = f
}

// EffectCamera identifies a camera whose visible effects must remain resident.
type EffectCamera struct {
	Eye              render.V3
	View, Projection render.M4
}

// PrepareMapEffects advances effects once and keeps the union of camera views.
// Instances outside every camera's range are released; drawing is a separate
// operation and may be repeated without advancing animation.
func (w *World) PrepareMapEffects(s *render.Scene3D, eye render.V3, additional ...EffectCamera) {
	if w == nil || w.mapFX == nil {
		return
	}
	f := w.mapFX
	type camera struct {
		eye        render.V3
		frustum    render.Frustum
		reach      float32
		validReach bool
	}
	var local [2]camera
	cameras := local[:0]
	reach, valid := mapEffectCameraReach(s.Proj)
	cameras = append(cameras, camera{eye: eye, frustum: render.ViewFrustum(s.View, s.Proj), reach: reach, validReach: valid})
	for _, c := range additional {
		reach, valid := mapEffectCameraReach(c.Projection)
		cameras = append(cameras, camera{eye: c.Eye, frustum: render.ViewFrustum(c.View, c.Projection), reach: reach, validReach: valid})
	}
	changed := false
	f.active = f.active[:0]
	for i := range f.instances {
		placed := &f.instances[i]
		inRange, visible := false, false
		for _, camera := range cameras {
			distance := placed.position.Sub(camera.eye)
			if mapEffectWithinReach(camera.reach, camera.validReach, distance.Dot(distance), placed.radius) {
				inRange = true
				visible = visible || camera.frustum.SphereVisible(placed.position, placed.radius)
				if visible {
					break
				}
			}
		}
		if !inRange {
			changed = changed || placed.instance != nil
			placed.instance = nil
			continue
		}
		if placed.instance == nil && visible && !f.failed[placed.sequence] {
			instance, err := loadEffectLibrary(f.assets, f.file, placed.sequence, f.library)
			changed = true
			if err != nil {
				f.failed[placed.sequence] = true
				continue
			}
			instance.SetTransform(placed.world)
			placed.instance = instance
		}
		if placed.instance == nil {
			continue
		}
		placed.instance.Update(w.frameDT)
		f.active = append(f.active, i)
	}
	if changed {
		f.trimTextures()
	}
}

// RenderMapEffects uses the current camera without changing effect time or
// residency. PrepareMapEffects must include every camera used in the frame.
func (w *World) RenderMapEffects(s *render.Scene3D) {
	if w == nil || w.mapFX == nil {
		return
	}
	frustum := render.ViewFrustum(s.View, s.Proj)
	for _, i := range w.mapFX.active {
		placed := &w.mapFX.instances[i]
		if placed.instance != nil && render.ClipSphereVisible(s.ClipPlane, placed.position, placed.radius) && frustum.SphereVisible(placed.position, placed.radius) && !s.OccludedSphere(placed.position, placed.radius) {
			placed.instance.Draw(s)
		}
	}
}

// DrawMapEffects is the single-camera convenience path used by menu scenes.
func (w *World) DrawMapEffects(s *render.Scene3D, eye render.V3) {
	w.PrepareMapEffects(s, eye)
	w.RenderMapEffects(s)
}

// Templates and live instances may share textures, even after a template is
// evicted. Keep both sets; unused resources can be released after drawing.
func (f *mapEffects) trimTextures() {
	used := make(map[*render.Texture]bool)
	keep := func(emitters []effectEmitter) {
		for _, emitter := range emitters {
			for _, texture := range emitter.textures {
				if texture != nil {
					used[texture] = true
				}
			}
		}
	}
	for _, template := range f.assets.effectTemplates {
		keep(template.emitters)
	}
	for _, placement := range f.instances {
		if placement.instance != nil {
			keep(placement.instance.emitters)
		}
	}
	for name, texture := range f.assets.textures {
		if texture != nil && used[texture] {
			continue
		}
		if texture != nil {
			texture.Delete()
		}
		delete(f.assets.textures, name)
	}
}

func (f *mapEffects) close() {
	if f.assets.effectMirrorQuad != nil {
		f.assets.effectMirrorQuad.Delete()
	}
	if f.assets.effectQuad != nil {
		f.assets.effectQuad.Delete()
	}
	for _, texture := range f.assets.textures {
		if texture != nil {
			texture.Delete()
		}
	}
}
