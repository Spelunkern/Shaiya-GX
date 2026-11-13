package audio

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/ebitengine/oto/v3"
)

type entry struct {
	pcm  []byte
	used uint64
}
type voice struct {
	player *oto.Player
	gain   float64
}
type Engine struct {
	context              *oto.Context
	ready                <-chan struct{}
	read                 func(string) ([]byte, error)
	cache                map[string]entry
	failed               map[string]error
	cacheBytes           int
	clock                uint64
	voices               []voice
	music                *oto.Player
	musicName            string
	volume               float64
	musicMix, effectsMix float64
}

// New creates the process's single audio context. Asset loading stays lazy.
func New(read func(string) ([]byte, error), volume float64) (*Engine, error) {
	c, ready, err := oto.NewContext(&oto.NewContextOptions{SampleRate: SampleRate, ChannelCount: 2, Format: oto.FormatSignedInt16LE, BufferSize: 100 * time.Millisecond})
	if err != nil {
		return nil, err
	}
	e := &Engine{context: c, ready: ready, read: read, cache: map[string]entry{}, failed: map[string]error{}, musicMix: 1, effectsMix: 1}
	e.SetVolume(volume)
	return e, nil
}
func (e *Engine) SetVolume(volume float64) {
	if math.IsNaN(volume) {
		volume = 0
	}
	e.volume = math.Max(0, math.Min(1, volume))
	if e.music == nil && e.volume > 0 {
		e.musicName = ""
	}
	for _, v := range e.voices {
		v.player.SetVolume(e.volume * e.effectsMix * v.gain)
	}
	if e.music != nil {
		e.music.SetVolume(e.volume * e.musicMix * 0.4)
	}
}

// SetMix changes the native music/effect channels, including active voices.
func (e *Engine) SetMix(music, effects float64) {
	if math.IsNaN(music) || math.IsNaN(effects) {
		return
	}
	e.musicMix, e.effectsMix = math.Max(0, math.Min(1, music)), math.Max(0, math.Min(1, effects))
	e.SetVolume(e.volume)
}
func (e *Engine) readyNow() bool {
	select {
	case <-e.ready:
		return true
	default:
		return false
	}
}
func (e *Engine) pcm(file string) ([]byte, error) {
	key := strings.ToLower(file)
	if err := e.failed[key]; err != nil {
		return nil, err
	}
	e.clock++
	if hit, ok := e.cache[key]; ok {
		hit.used = e.clock
		e.cache[key] = hit
		return hit.pcm, nil
	}
	raw, err := e.read(file)
	if err != nil {
		e.failed[key] = err
		return nil, err
	}
	pcm, err := DecodeOGG(raw)
	if err != nil {
		err = fmt.Errorf("%s: %w", file, err)
		e.failed[key] = err
		return nil, err
	}
	const limit = 32 << 20
	if len(pcm) > limit {
		return pcm, nil
	}
	for e.cacheBytes+len(pcm) > limit {
		oldest := ""
		stamp := ^uint64(0)
		for k, v := range e.cache {
			if v.used < stamp {
				oldest, stamp = k, v.used
			}
		}
		e.cacheBytes -= len(e.cache[oldest].pcm)
		delete(e.cache, oldest)
	}
	e.cache[key] = entry{pcm: pcm, used: e.clock}
	e.cacheBytes += len(pcm)
	return pcm, nil
}
func (e *Engine) Play(file string, gain float64) error {
	if !e.readyNow() || e.volume == 0 || e.effectsMix == 0 {
		return nil
	}
	if math.IsNaN(gain) {
		return fmt.Errorf("audio: invalid gain")
	}
	if err := e.Update(); err != nil {
		return err
	}
	if len(e.voices) >= 24 {
		return nil
	}
	pcm, err := e.pcm(file)
	if err != nil {
		return err
	}
	if len(pcm) > 4<<20 {
		return fmt.Errorf("audio: effect exceeds 4 MiB")
	}
	gain = math.Max(0, math.Min(1, gain))
	p := e.context.NewPlayer(bytes.NewReader(pcm))
	p.SetVolume(e.volume * e.effectsMix * gain)
	p.Play()
	e.voices = append(e.voices, voice{p, gain})
	return nil
}
func (e *Engine) Music(file string) error {
	if !e.readyNow() || e.musicName == file {
		return nil
	}
	e.musicName = file
	if e.music != nil {
		p := e.music
		p.PauseAndStopReading()
		e.music = nil
		if err := p.Err(); err != nil {
			return err
		}
	}
	if file == "" || e.volume == 0 {
		return nil
	}
	pcm, err := e.pcm(file)
	if err != nil {
		return err
	}
	e.music = e.context.NewPlayer(&loopReader{pcm: pcm})
	e.music.SetVolume(e.volume * e.musicMix * 0.4)
	e.music.Play()
	return nil
}
func (e *Engine) Update() error {
	if err := e.context.Err(); err != nil {
		return err
	}
	var first error
	kept := e.voices[:0]
	for _, v := range e.voices {
		if v.player.IsPlaying() {
			kept = append(kept, v)
		} else {
			if err := v.player.Err(); err != nil && first == nil {
				first = err
			}
		}
	}
	clear(e.voices[len(kept):])
	e.voices = kept
	return first
}
func (e *Engine) Close() error {
	var first error
	for _, v := range e.voices {
		v.player.PauseAndStopReading()
		if err := v.player.Err(); err != nil && first == nil {
			first = err
		}
	}
	if e.music != nil {
		e.music.PauseAndStopReading()
		if err := e.music.Err(); err != nil && first == nil {
			first = err
		}
	}
	e.voices = nil
	e.music = nil
	e.cache = nil
	return first
}

type loopReader struct {
	pcm    []byte
	offset int
}

func (r *loopReader) Read(dst []byte) (int, error) {
	if len(r.pcm) == 0 {
		return 0, io.EOF
	}
	total := 0
	for total < len(dst) {
		n := copy(dst[total:], r.pcm[r.offset:])
		total += n
		r.offset = (r.offset + n) % len(r.pcm)
	}
	return total, nil
}
