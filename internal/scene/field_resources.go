package scene

import (
	"fmt"
	"path"
	"slices"
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
	clear(s.objectMarks)
	for _, id := range s.global {
		s.objectMarks[id] = true
	}
	sections := map[int]bool{}
	for _, c := range s.cells {
		for _, id := range c.objects {
			s.objectMarks[id] = true
		}
		for _, p := range c.parts {
			sections[p.section] = true
		}
	}
	s.drawObjects = s.drawObjects[:0]
	clear(s.resourceMarks)
	for id, marked := range s.objectMarks {
		if marked {
			s.drawObjects = append(s.drawObjects, id)
			s.resourceMarks[s.objects[id].resourceID] = true
		}
	}
	if s.loader != nil {
		for k := range s.loader.pinned {
			if k.kind != fieldTexture {
				if id, ok := s.resourceIndex[k.file]; ok {
					s.resourceMarks[id] = true
				}
			}
		}
	}
	for file, m := range s.assets.models {
		if s.resourceAlive(file) {
			continue
		}
		if m != nil {
			m.delete()
		}
		delete(s.assets.models, file)
		delete(s.batches, modelBatchKey{file: file})
		delete(s.batches, modelBatchKey{file: file, mani: true})
	}
	for file, v := range s.vanis {
		if !s.resourceAlive(file) {
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
	if s.loader != nil {
		for k := range s.loader.pinned {
			if k.kind == fieldTexture {
				keep(s.assets.textures["world:"+k.file])
			}
		}
	}
	for file, t := range s.assets.textures {
		if s.loader != nil && strings.HasPrefix(file, "world:") && s.loader.pinned[fieldAssetKey{fieldTexture, strings.TrimPrefix(file, "world:")}] {
			continue
		}
		if t != nil && used[t] {
			continue
		}
		if t != nil {
			t.Delete()
		}
		delete(s.assets.textures, file)
	}
	s.prepareDrawEntries()
}

// prepareDrawEntries resolves resources and static bounds once per residency
// change. Rendering does not hash asset paths or sort model keys per placement.
func (s *fieldStream) prepareDrawEntries() {
	if s.batches == nil {
		s.batches = map[modelBatchKey]*fieldModelBatch{}
	}
	for _, b := range s.batches {
		b.worlds = b.worlds[:0]
	}
	clear(s.drawEntries)
	clear(s.modelBatches)
	s.drawEntries = s.drawEntries[:0]
	s.modelBatches = s.modelBatches[:0]
	s.batchPreparation++
	if s.resourceIndex == nil || s.resourceObjectCount != len(s.objects) {
		s.indexResources()
	}
	for i := range s.resources {
		resource := &s.resources[i]
		resource.model, resource.vani = s.assets.models[resource.file], s.vanis[resource.file]
		if resource.model == nil {
			resource.batches = [2]*fieldModelBatch{}
		}
		if resource.model != nil {
			for pass := range 2 {
				key := modelBatchKey{file: resource.file, mani: pass == 1}
				batch := s.batches[key]
				if batch == nil {
					batch = &fieldModelBatch{key: key}
					s.batches[key] = batch
				}
				batch.model = resource.model
				resource.batches[pass] = batch
			}
		}
	}
	for _, id := range s.drawObjects {
		o := &s.objects[id]
		entry := fieldDrawEntry{object: o}
		resource := &s.resources[o.resourceID]
		if o.vani {
			entry.vani = resource.vani
			if entry.vani == nil {
				continue
			}
			entry.center, entry.radius = transformPoint(o.center, o.world), o.radius*o.scale
		} else {
			m := resource.model
			if m == nil {
				continue
			}
			pass := 0
			if o.mani {
				pass = 1
			}
			batch := resource.batches[pass]
			entry.batch = batch
			if !o.drawBoundsValid {
				o.drawCenter = transformPoint(m.center, o.world)
				o.drawRadius = m.radius * o.scale
				o.drawBoundsValid = true
			}
			entry.center, entry.radius = o.drawCenter, o.drawRadius
			if batch.prepared != s.batchPreparation {
				batch.prepared = s.batchPreparation
				s.modelBatches = append(s.modelBatches, batch)
			}
		}
		s.drawEntries = append(s.drawEntries, entry)
	}
	s.prepareSectors()
	slices.SortFunc(s.modelBatches, func(a, b *fieldModelBatch) int {
		if a.key.mani != b.key.mani {
			if a.key.mani {
				return -1
			}
			return 1
		}
		return strings.Compare(a.key.file, b.key.file)
	})
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
				if (!clipCulling || render.ClipBoxVisible(r.ClipPlane, p.minimum, p.maximum)) && frustum.SphereVisible(p.center, p.radius) && !r.OccludedBox(p.minimum, p.maximum) {
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

	s.lodInstances, s.submittedTriangles = 0, 0
	for _, batch := range s.modelBatches {
		batch.worlds = batch.worlds[:0]
		for i := range batch.lodWorlds {
			batch.lodWorlds[i] = batch.lodWorlds[i][:0]
		}
	}
	for _, asset := range s.vanis {
		if asset != nil {
			asset.worlds = asset.worlds[:0]
		}
	}
	s.updateSectors(r, frustum, clipCulling)
	for i := range s.drawEntries {
		entry := &s.drawEntries[i]
		if !s.disableSectors && !s.drawSectors[entry.sector].visible {
			continue
		}
		s.objectTests++
		o := entry.object
		center, radius, transform := entry.center, entry.radius, o.world
		if o.rotor != nil {
			transform = o.rotor.transform(s.world.elapsed)
			center = transformPoint(entry.batch.model.center, transform)
		}
		if clipCulling && !render.ClipSphereVisible(r.ClipPlane, center, radius) || !frustum.SphereVisible(center, radius) || r.OccludedSphere(center, radius) {
			continue
		}
		if entry.vani != nil {
			entry.vani.worlds = append(entry.vani.worlds, transform)
		} else {
			entry.lod = entry.chooseLOD(r, center, radius)
			if r.StaticObjectLOD && !r.Shadow && r.ClipPlane == [4]float32{} {
				o.drawLOD = entry.lod
			}
			if entry.lod == 0 {
				entry.batch.worlds = append(entry.batch.worlds, transform)
			} else {
				entry.batch.lodWorlds[entry.lod-1] = append(entry.batch.lodWorlds[entry.lod-1], transform)
				s.lodInstances++
			}
		}
	}
	// Stable native material order, precomputed when residency changes.
	for _, batch := range s.modelBatches {
		groups := [3][]render.M4{batch.worlds, batch.lodWorlds[0], batch.lodWorlds[1]}
		for level, worlds := range groups {
			if len(worlds) == 0 {
				continue
			}
			if len(worlds) > 1 {
				r.SetInstances(worlds)
			}
			for i := range batch.model.Parts {
				p := &batch.model.Parts[i]
				mesh := p.meshAtLOD(level)
				s.submittedTriangles += int(mesh.Count) / 3 * len(worlds)
				if len(worlds) == 1 {
					r.DrawModel(mesh, p.Tex, worlds[0], render.NativeWorldAlphaRef(batch.key.mani), false)
				} else {
					r.DrawInstanced(mesh, p.Tex, render.NativeWorldAlphaRef(batch.key.mani))
				}
			}
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
	for _, p := range w.dg {
		p.mesh.Delete()
	}
	w.dg = nil
	w.dgOrder = nil
	w.dungeonOccluders = nil
	w.dungeonInstances = nil
	w.occlusion = nil
	if w.stream == nil {
		return
	}
	s := w.stream
	if s.loader != nil {
		s.loader.close()
	}
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
			m.delete()
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
