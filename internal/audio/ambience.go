package audio

import (
	"fmt"
	"math"
	"strings"
)

// LoopTarget is a non-positional map track. Phoenix map_audio.gd mixes by
// distance, keeps the strongest emitter per asset, and fades at 2.6/s.
type LoopTarget struct {
	File  string
	Gain  float64
	Music bool
}
type ambientVoice struct {
	voice
	target float64
	music  bool
}

// Ambience replaces the desired map mix without restarting unchanged loops.
// Retiring tracks fade out before their readers and PCM are released.
func (e *Engine) Ambience(targets []LoopTarget, dt float64) error {
	if !e.readyNow() {
		return nil
	}
	if e.loops == nil {
		e.loops = map[string]*ambientVoice{}
	}
	for _, v := range e.loops {
		v.target = 0
	}
	var first error
	// Release completed obsolete decodes after map changes. At most two jobs
	// can be decoding, and they never access Oto or mutate the owner's cache.
	wanted := map[string]bool{}
	for _, t := range targets {
		wanted[strings.ToLower(t.File)] = true
	}
	for file, ch := range e.pending {
		if !wanted[file] {
			select {
			case <-ch:
				delete(e.pending, file)
			default:
			}
		}
	}
	for _, t := range targets {
		if t.File == "" || math.IsNaN(t.Gain) || t.Gain <= .001 {
			continue
		}
		if v := e.loops[t.File]; v != nil {
			v.target = max(v.target, min(1, t.Gain))
			continue
		}
		mix := e.ambientMix
		if t.Music {
			mix = e.musicMix
		}
		if len(e.loops) >= 32 || e.volume*mix == 0 {
			continue
		}
		pcm, ready, err := e.mapPCM(t.File)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		if !ready {
			continue
		}
		p := e.context.NewPlayer(&loopReader{pcm: pcm})
		p.SetVolume(0)
		p.Play()
		e.loops[t.File] = &ambientVoice{voice: voice{player: p}, target: min(1, t.Gain), music: t.Music}
	}
	rate := min(1, max(0, dt)*2.6)
	for file, v := range e.loops {
		v.gain += (v.target - v.gain) * rate
		if v.target == 0 && v.gain <= .001 {
			v.player.PauseAndStopReading()
			if err := v.player.Err(); err != nil && first == nil {
				first = err
			}
			delete(e.loops, file)
			continue
		}
		mix := e.ambientMix
		if v.music {
			mix = e.musicMix
		}
		v.player.SetVolume(e.volume * mix * v.gain)
	}
	return first
}

// Decode map tracks off the render thread, with bounded concurrent work.
type decodedAudio struct {
	pcm []byte
	err error
}

func (e *Engine) mapPCM(file string) ([]byte, bool, error) {
	key := strings.ToLower(file)
	if err := e.failed[key]; err != nil {
		return nil, false, err
	}
	if ch := e.pending[key]; ch != nil {
		select {
		case result := <-ch:
			delete(e.pending, key)
			if result.err != nil {
				e.failed[key] = result.err
				return nil, false, result.err
			}
			e.rememberPCM(key, result.pcm)
			return result.pcm, true, nil
		default:
			return nil, false, nil
		}
	}
	if hit, ok := e.cache[key]; ok {
		e.clock++
		hit.used = e.clock
		e.cache[key] = hit
		return hit.pcm, true, nil
	}
	if len(e.pending) >= 2 {
		return nil, false, nil
	}
	if e.pending == nil {
		e.pending = map[string]chan decodedAudio{}
	}
	ch := make(chan decodedAudio, 1)
	e.pending[key] = ch
	read := e.read
	go func() {
		raw, err := read(file)
		var pcm []byte
		if err == nil {
			pcm, err = DecodeOGG(raw)
		}
		if err != nil {
			err = fmt.Errorf("%s: %w", file, err)
		}
		ch <- decodedAudio{pcm, err}
	}()
	return nil, false, nil
}
