package scene

import (
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func (s *fieldStream) loadMaterial(sx, sz int) Section {
	field := fmt.Sprintf("world/Field/%s/%s_%d%d", s.stem, s.stem, sz, sx)
	sec := Section{}
	for i, l := range s.world.WLD.Layers {
		var tile *render.Texture
		if strings.TrimSpace(l.Texture) != "" {
			tile = s.assets.worldTexture("Terrain/Detail/" + DDSName(path.Base(l.Texture)))
			if tile != nil {
				tile.SetWrap(true)
			}
		}
		sec.Layers = append(sec.Layers, TerrainLayer{Tile: tile, TileSize: l.TileSize, Mask: s.assets.optionalTexture(fmt.Sprintf("%s_A%d.dds", field, i))})
	}
	sec.Lightmap = s.assets.optionalTexture(field + "_L.dds")
	return sec
}

// trim keeps only resources reachable from resident cells (plus the sky).
// It runs after a residency change, never while issuing draw calls.
func (s *fieldStream) trim() {
	ids := map[int]bool{}
	for _, id := range s.global {
		ids[id] = true
	}
	sections := map[int]bool{}
	for _, c := range s.cells {
		for _, id := range c.objects {
			ids[id] = true
		}
		for _, p := range c.parts {
			sections[p.section] = true
		}
	}
	s.drawObjects = s.drawObjects[:0]
	alive := map[string]bool{}
	for id := range ids {
		s.drawObjects = append(s.drawObjects, id)
		alive[s.objects[id].file] = true
	}
	sort.Ints(s.drawObjects)
	for file, m := range s.assets.models {
		if alive[file] {
			continue
		}
		if m != nil {
			for _, p := range m.Parts {
				p.Mesh.Delete()
			}
		}
		delete(s.assets.models, file)
		delete(s.batches, modelBatchKey{file: file})
		delete(s.batches, modelBatchKey{file: file, mani: true})
	}
	for file, v := range s.vanis {
		if !alive[file] {
			if v != nil {
				v.close()
			}
			delete(s.vanis, file)
		}
	}
	for id := range s.materials {
		if !sections[id] {
			delete(s.materials, id)
		}
	}
	used := map[*render.Texture]bool{}
	keep := func(t *render.Texture) {
		if t != nil {
			used[t] = true
		}
	}
	if sky := s.world.Sky; sky != nil {
		keep(sky.DomeTex)
		for _, t := range sky.Clouds {
			keep(t)
		}
	}
	for _, m := range s.assets.models {
		if m != nil {
			for _, p := range m.Parts {
				keep(p.Tex)
			}
		}
	}
	for _, v := range s.vanis {
		if v != nil {
			for _, part := range v.parts {
				keep(part.texture)
			}
		}
	}
	for _, sec := range s.materials {
		keep(sec.Lightmap)
		for _, l := range sec.Layers {
			keep(l.Tile)
			keep(l.Mask)
		}
	}
	for file, t := range s.assets.textures {
		if t != nil && used[t] {
			continue
		}
		if t != nil {
			t.Delete()
		}
		delete(s.assets.textures, file)
	}
}
func (s *fieldStream) draw(r *render.Scene3D) {
	s.drawWithClipCulling(r, true)
}

func (s *fieldStream) drawWithClipCulling(r *render.Scene3D, clipCulling bool) {
	frustum := render.ViewFrustum(r.View, r.Proj)
	for _, batch := range s.terrainBatches {
		batch.ranges = batch.ranges[:0]
	}
	s.visibleTerrain = s.visibleTerrain[:0]
	for _, key := range s.order {
		if c := s.cells[key]; c != nil {
			for _, p := range c.parts {
				if (!clipCulling || render.ClipBoxVisible(r.ClipPlane, p.minimum, p.maximum)) && frustum.SphereVisible(p.center, p.radius) {
					if batch := s.terrainBatches[p.batch]; batch != nil {
						if len(batch.ranges) == 0 {
							s.visibleTerrain = append(s.visibleTerrain, batch)
						}
						batch.ranges = append(batch.ranges, p.rangeInBatch)
					}
				}
			}
		}
	}
	for _, batch := range s.visibleTerrain {
		drawTerrainBatch(r, batch)
	}
	if s.batches == nil {
		s.batches = map[modelBatchKey][]render.M4{}
	}
	for file, worlds := range s.batches {
		s.batches[file] = worlds[:0]
	}
	for _, asset := range s.vanis {
		if asset != nil {
			asset.worlds = asset.worlds[:0]
		}
	}
	s.batchFiles = s.batchFiles[:0]
	for _, id := range s.drawObjects {
		o := s.objects[id]
		if o.vani {
			if asset := s.vanis[o.file]; asset != nil && (!clipCulling || render.ClipSphereVisible(r.ClipPlane, transformPoint(o.center, o.world), o.radius*o.scale)) && frustum.SphereVisible(transformPoint(o.center, o.world), o.radius*o.scale) {
				asset.worlds = append(asset.worlds, o.world)
			}
			continue
		}
		m := s.assets.models[o.file]
		if m == nil {
			continue
		}
		transform := o.world
		if o.rotor != nil {
			transform = o.rotor.transform(s.world.elapsed)
		}
		center, radius := transformPoint(m.center, transform), m.radius*o.scale
		if clipCulling && !render.ClipSphereVisible(r.ClipPlane, center, radius) || !frustum.SphereVisible(center, radius) {
			continue
		}
		key := modelBatchKey{file: o.file, mani: o.mani}
		worlds := s.batches[key]
		if len(worlds) == 0 {
			s.batchFiles = append(s.batchFiles, key)
		}
		s.batches[key] = append(worlds, transform)
	}
	// Visibility of the first placement must not reorder entire model batches:
	// equal-depth surfaces otherwise swap materials when unrelated copies cull.
	slices.SortFunc(s.batchFiles, func(a, b modelBatchKey) int {
		if a.mani != b.mani {
			if a.mani {
				return -1
			}
			return 1
		}
		return strings.Compare(a.file, b.file)
	})
	for _, file := range s.batchFiles {
		m, worlds := s.assets.models[file.file], s.batches[file]
		if len(worlds) == 1 {
			for _, p := range m.Parts {
				r.DrawModel(p.Mesh, p.Tex, worlds[0], render.NativeWorldAlphaRef(file.mani), false)
			}
			continue
		}
		r.SetInstances(worlds)
		for _, p := range m.Parts {
			r.DrawInstanced(p.Mesh, p.Tex, render.NativeWorldAlphaRef(file.mani))
		}
	}
	for _, asset := range s.vanis {
		if asset != nil {
			asset.draw(r, s.world.elapsed)
		}
	}
}

// Close releases the field's owned resources. Eager menu/dungeon worlds keep
// their existing shared caches; they are not streaming worlds.
func (w *World) Close() {
	if w == nil {
		return
	}
	if w.water != nil {
		w.water.Delete()
		w.water = nil
	}
	for _, v := range w.vanis {
		v.close()
	}
	w.vanis = nil
	if w.mapFX != nil {
		w.mapFX.close()
		w.mapFX = nil
	}
	if w.stream == nil {
		return
	}
	s := w.stream
	for _, v := range s.vanis {
		if v != nil {
			v.close()
		}
	}
	for _, batch := range s.terrainBatches {
		batch.mesh.Delete()
	}
	for _, m := range s.assets.models {
		if m != nil {
			for _, p := range m.Parts {
				p.Mesh.Delete()
			}
		}
	}
	if w.Sky != nil {
		w.Sky.Dome.Delete()
		w.Sky.quad.Delete()
	}
	for _, t := range s.assets.textures {
		if t != nil {
			t.Delete()
		}
	}
	w.stream, w.Sky = nil, nil
}
