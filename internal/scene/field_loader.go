package scene

import (
	"context"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"path"
	"strings"
	"sync"
	"time"
)

type fieldAssetKind uint8

const (
	fieldTexture fieldAssetKind = iota
	fieldSMOD
	fieldVANI
	fieldTextureIndex
)

type fieldAssetKey struct {
	kind fieldAssetKind
	file string
}
type fieldAssetJob struct {
	key        fieldAssetKey
	generation uint64
}
type fieldAssetResult struct {
	textureIndex map[string]string
	lods         [][]staticLODCPU
	job          fieldAssetJob
	texture      img.Texture
	smod         *world.SMOD
	vani         *world.VANI
	err          error
}
type fieldAssetState struct{ result *fieldAssetResult }

// Workers own file reads and CPU decoding only. States, GPU uploads and caches
// belong exclusively to the render thread. Four jobs bound in-flight work.
type fieldLoader struct {
	decodedModels int
	slowestUpload time.Duration
	slowestFile   string
	cancel        context.CancelFunc
	workers       sync.WaitGroup
	jobs          chan fieldAssetJob
	results       chan fieldAssetResult
	generation    uint64
	outstanding   int
	states        map[fieldAssetKey]*fieldAssetState
	pinned        map[fieldAssetKey]bool
}

func newFieldLoader(arc *archive.Archive) *fieldLoader {
	ctx, cancel := context.WithCancel(context.Background())
	l := &fieldLoader{cancel: cancel, jobs: make(chan fieldAssetJob, 4), results: make(chan fieldAssetResult, 4)}
	l.reset()
	for range 2 {
		l.workers.Add(1)
		go func() {
			defer l.workers.Done()
			for {
				var job fieldAssetJob
				select {
				case <-ctx.Done():
					return
				case job = <-l.jobs:
				}
				r := fieldAssetResult{job: job}
				var b []byte
				var err error
				if job.key.kind == fieldTextureIndex {
					r.textureIndex = textureFileIndex(arc)
				} else {
					b, err = arc.ReadFile(job.key.file)
				}
				if err == nil {
					switch job.key.kind {
					case fieldTexture:
						r.texture, err = img.DecodeResource(job.key.file, b)
					case fieldSMOD:
						r.smod, err = world.ParseSMOD(b)
						if err == nil {
							r.lods = buildStaticLODs(r.smod)
						}
					case fieldVANI:
						r.vani, err = world.ParseVANI(b)
					}
				}
				r.err = err
				select {
				case <-ctx.Done():
					return
				case l.results <- r:
				}
			}
		}()
	}
	return l
}
func (l *fieldLoader) close() { l.cancel(); l.workers.Wait() }
func (l *fieldLoader) reset() {
	l.generation++
	l.decodedModels = 0
	l.states = map[fieldAssetKey]*fieldAssetState{}
	l.pinned = map[fieldAssetKey]bool{}
}

// EnableAsyncStreaming is used by live gameplay. Deterministic import/render
// tools retain synchronous loading unless they explicitly opt in.
func (w *World) EnableAsyncStreaming() {
	if w.stream != nil && w.stream.loader == nil {
		w.stream.loader = newFieldLoader(w.stream.assets.Arc)
	}
}
func (s *fieldStream) assetCached(k fieldAssetKey) bool {
	switch k.kind {
	case fieldTextureIndex:
		return s.assets.byStem != nil
	case fieldTexture:
		_, ok := s.assets.textures["world:"+k.file]
		return ok
	case fieldSMOD:
		_, ok := s.assets.models[k.file]
		return ok
	case fieldVANI:
		_, ok := s.vanis[k.file]
		return ok
	}
	return false
}
func (s *fieldStream) requestAsset(kind fieldAssetKind, file string) *fieldAssetState {
	l := s.loader
	k := fieldAssetKey{kind, strings.ToLower(file)}
	l.pinned[k] = true
	if s.assetCached(k) {
		return nil
	}
	if state := l.states[k]; state != nil {
		return state
	}
	if (kind == fieldSMOD || kind == fieldVANI) && l.decodedModels > 0 {
		return nil
	}
	if l.outstanding >= 4 {
		return nil
	}
	state := &fieldAssetState{}
	l.states[k] = state
	l.outstanding++
	l.jobs <- fieldAssetJob{k, l.generation}
	return state
}
func (s *fieldStream) pollAssets(deadline time.Time) {
	l := s.loader
	for time.Now().Before(deadline) {
		select {
		case result := <-l.results:
			l.outstanding--
			if result.job.key.kind == fieldTextureIndex {
				s.assets.byStem = result.textureIndex
				continue
			}
			if result.job.generation != l.generation {
				continue
			}
			state := l.states[result.job.key]
			if state == nil {
				continue
			}
			state.result = &result
			if result.job.key.kind == fieldSMOD || result.job.key.kind == fieldVANI {
				l.decodedModels++
			}
			if result.job.key.kind == fieldTexture {
				var tex *render.Texture
				if result.err == nil {
					t := time.Now()
					tex = render.UploadWorld(result.texture)
					if d := time.Since(t); d > l.slowestUpload {
						l.slowestUpload = d
						l.slowestFile = result.job.key.file
					}
				} else {
					s.assets.Missing[result.job.key.file] = true
				}
				s.assets.textures["world:"+result.job.key.file] = tex
				state.result = nil // release decoded pixels immediately after upload
			}
		default:
			return
		}
	}
}
func (s *fieldStream) ensureTexture(file string) bool {
	s.requestAsset(fieldTexture, file)
	return s.assetCached(fieldAssetKey{fieldTexture, strings.ToLower(file)})
}
func modelTextureFile(name string) string {
	return "Entity/Texture/" + DDSName(path.Base(strings.ReplaceAll(name, `\`, "/")))
}
func (s *fieldStream) ensureModel(o cellObject, deadline time.Time) bool {
	kind := fieldSMOD
	if o.vani {
		kind = fieldVANI
	}
	key := fieldAssetKey{kind, strings.ToLower(o.file)}
	state := s.requestAsset(kind, o.file)
	if s.assetCached(key) {
		return true
	}
	if state == nil || state.result == nil {
		return false
	}
	r := state.result
	if r.err != nil {
		s.assets.Missing[o.file] = true
		if o.vani {
			s.vanis[key.file] = nil
		} else {
			s.assets.models[key.file] = nil
		}
		s.loader.decodedModels--
		state.result = nil
		return true
	}
	ready := true
	if r.smod != nil {
		for _, m := range r.smod.Meshes {
			if m.Texture != "" && !s.ensureTexture(modelTextureFile(m.Texture)) {
				ready = false
			}
		}
	}
	if r.vani != nil {
		if s.assets.byStem == nil {
			s.requestAsset(fieldTextureIndex, "")
			return false
		}
		for _, m := range r.vani.Meshes {
			if file, ok := s.assets.findTextureFile(m.Texture); ok && !s.ensureTexture(file) {
				ready = false
			}
		}
	}
	if !ready || !time.Now().Before(deadline) {
		return false
	}
	started := time.Now()
	if o.vani {
		s.vanis[key.file], _ = uploadVani(s.assets, o.file, r.vani)
	} else {
		s.assets.models[key.file], _ = s.assets.uploadSMOD(key.file, r.smod, r.lods)
	}
	if d := time.Since(started); d > s.loader.slowestUpload {
		s.loader.slowestUpload = d
		s.loader.slowestFile = o.file
	}
	s.loader.decodedModels--
	state.result = nil
	return true
}
func (s *fieldStream) prepareCellAssets(key cellKey, deadline time.Time) bool {
	ready := true
	w := s.world
	n := w.sectionsPerSide()
	section := (w.side - 1) / n
	x0, z0 := key.x*terrainCellSamples, key.z*terrainCellSamples
	x1, z1 := min(x0+terrainCellSamples, w.side-1), min(z0+terrainCellSamples, w.side-1)
	for sz := 0; sz < n; sz++ {
		for sx := 0; sx < n; sx++ {
			if x0 < 0 || z0 < 0 || max(x0, sx*section) >= min(x1, (sx+1)*section) || max(z0, sz*section) >= min(z1, (sz+1)*section) {
				continue
			}
			if _, ok := s.materials[sz*n+sx]; ok {
				continue
			}
			stem := fmt.Sprintf("world/Field/%s/%s_%d%d", s.stem, s.stem, sz, sx)
			for i, layer := range w.WLD.Layers {
				if strings.TrimSpace(layer.Texture) != "" && !s.ensureTexture("Terrain/Detail/"+DDSName(path.Base(layer.Texture))) {
					ready = false
				}
				file := fmt.Sprintf("%s_A%d.dds", stem, i)
				if _, ok := s.assets.Arc.Lookup(file); ok && !s.ensureTexture(file) {
					ready = false
				}
			}
			file := stem + "_L.dds"
			if _, ok := s.assets.Arc.Lookup(file); ok && !s.ensureTexture(file) {
				ready = false
			}
		}
	}
	for _, id := range s.index[key] {
		if !s.ensureModel(s.objects[id], deadline) {
			ready = false
		}
	}
	for _, id := range s.global {
		if !s.ensureModel(s.objects[id], deadline) {
			ready = false
		}
	}
	return ready
}
