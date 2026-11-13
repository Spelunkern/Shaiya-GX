package app

import (
	"math"
	"path"
	"sort"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type mapAudioState struct {
	world         *world.WLD
	refresh, step float64
	targets       []audio.LoopTarget
}

// WLD names still carry WAV extensions. The portable catalog contains OGG,
// with ambience in Sound/back; resolve existing files without duplicating assets.
func (a *App) audioPath(name string, music bool) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.TrimSuffix(name, path.Ext(name)) + ".ogg"
	folders := []string{"Sound/", "Sound/back/", "Sound/Music/"}
	if music {
		folders = []string{"Sound/Music/", "Sound/back/", "Sound/"}
	}
	for _, folder := range folders {
		if _, ok := a.arc.Lookup(folder + name); ok {
			return folder + name
		}
	}
	return folders[0] + name
}

func smoothAudio(distance, radius float64) float64 {
	if radius <= 0 {
		return 0
	}
	t := min(1, max(0, distance/radius))
	return 1 - t*t*(3-2*t)
}

// Phoenix map_audio.gd: nearest point on each music box, spherical ambient
// emitters, strongest occurrence per file and at most 24 ambient tracks.
func mapAudioTargets(w *world.WLD, pos render.V3) []audio.LoopTarget {
	if w == nil {
		return nil
	}
	wanted := map[string]audio.LoopTarget{}
	offer := func(name string, gain float64, music bool) {
		if name == "" || gain <= .001 {
			return
		}
		key := strings.ToLower(name)
		if music {
			key = "music:" + key
		}
		if gain > wanted[key].Gain {
			wanted[key] = audio.LoopTarget{File: name, Gain: gain, Music: music}
		}
	}
	for _, z := range w.MusicZones {
		if z.Asset < 0 || int(z.Asset) >= len(w.MusicAssets) {
			continue
		}
		d2 := float64(0)
		for i := range 3 {
			lo, hi := min(z.Box.Min[i], z.Box.Max[i]), max(z.Box.Min[i], z.Box.Max[i])
			d := float64(max(lo-pos[i], max(pos[i]-hi, 0)))
			d2 += d * d
		}
		offer(w.MusicAssets[z.Asset], .72*smoothAudio(math.Sqrt(d2), max(48, float64(z.Radius))), true)
	}
	// The map's authored faction field bed is continuous ambience. Point
	// emitters add local detail; they must not gate the entire outdoor soundscape.
	for _, name := range w.SoundAssets {
		stem := strings.ToLower(path.Base(strings.ReplaceAll(name, `\`, "/")))
		if strings.HasPrefix(stem, "bg_league of light_") || strings.HasPrefix(stem, "bg_union of fury_") {
			offer(name, .86, false)
			break
		}
	}
	for _, e := range w.Sounds {
		if e.Asset < 0 || int(e.Asset) >= len(w.SoundAssets) || e.Radius <= 0 {
			continue
		}
		d := pos.Sub(render.V3(e.Center)).Len()
		offer(w.SoundAssets[e.Asset], .86*smoothAudio(float64(d), max(48, float64(e.Radius)*1.6+16)), false)
	}
	out := make([]audio.LoopTarget, 0, len(wanted))
	for _, t := range wanted {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Music != out[j].Music {
			return out[i].Music
		}
		if out[i].Gain != out[j].Gain {
			return out[i].Gain > out[j].Gain
		}
		return out[i].File < out[j].File
	})
	n := 0
	kept := out[:0]
	for _, t := range out {
		if !t.Music {
			n++
			if n > 24 {
				continue
			}
		}
		kept = append(kept, t)
	}
	return kept
}

func (a *App) updateMapAudio() {
	m := &a.mapAudio
	var w *world.WLD
	if !a.loading.active && a.screen == screenWorld && a.wv != nil && a.wv.world != nil {
		w = a.wv.world.WLD
	}
	m.refresh -= a.frameDT
	if m.world != w || m.refresh <= 0 {
		if m.world != w {
			m.step = 0
		}
		m.world, m.refresh = w, .25
		m.targets = nil
		if w != nil {
			m.targets = mapAudioTargets(w, a.wv.pos)
			for i := range m.targets {
				m.targets[i].File = a.audioPath(m.targets[i].File, m.targets[i].Music)
			}
		}
	}
	a.audioError("map ambience", a.sound.Ambience(m.targets, a.frameDT))
	if w == nil {
		return
	}
	v := a.wv
	if !v.moving || v.jumping || v.evading || a.state == nil || a.state.Self.Dead || a.state.Self.Sitting || a.state.Self.Mounted || v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2]) || math.Abs(float64(v.pos[1]-v.world.Height(v.pos[0], v.pos[2]))) > .2 {
		m.step = 0
		return
	}
	interval := .36
	if v.walking {
		interval = .52
	}
	if v.motion&0x3f >= 2 && v.motion&0x3f <= 4 {
		interval *= 1.35
	}
	m.step += a.frameDT
	if m.step < interval {
		return
	}
	m.step = math.Mod(m.step, interval)
	name := terrainWalkSound(w, v.pos[0], v.pos[2])
	if name != "" {
		file := a.audioPath(name, false)
		a.audioError(file, a.sound.Play(file, .35))
	}
}

func terrainWalkSound(w *world.WLD, x, z float32) string {
	if w == nil || w.Dungeon || x < 0 || z < 0 || x > float32(w.MapSize) || z > float32(w.MapSize) {
		return ""
	}
	side := int(w.MapSize/2) + 1
	ix, iz := min(int(x/2), side-1), min(int(z/2), side-1)
	index := iz*side + ix
	if index < 0 || index >= len(w.TextureMap) {
		return ""
	}
	layer := int(w.TextureMap[index])
	if layer >= len(w.Layers) {
		return ""
	}
	return w.Layers[layer].Sound
}
