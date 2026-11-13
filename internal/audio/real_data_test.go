package audio

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
)

func TestRealSkillImpactOGG(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	d, ok := a.LookupDir("Effect")
	if !ok {
		t.Fatal("missing original effect directory")
	}
	libraries, sounds := 0, map[string]bool{}
	for _, f := range d.Files {
		name := effect.ImpactSound("Effect/" + f.Name)
		if name == "" {
			continue
		}
		libraries++
		if sounds[name] {
			continue
		}
		sounds[name] = true
		raw, err := a.ReadFile("Sound/" + name)
		if err != nil {
			t.Fatal(err)
		}
		pcm, err := DecodeOGG(raw)
		if err != nil || len(pcm) == 0 || len(pcm) > 4<<20 {
			t.Fatalf("%s: invalid effect PCM (%d bytes): %v", name, len(pcm), err)
		}
		t.Logf("%s: %.2f s", name, float64(len(pcm))/4/SampleRate)
	}
	if libraries != 26 || len(sounds) != 11 {
		t.Fatalf("incomplete native sound catalog: %d libraries, %d sounds", libraries, len(sounds))
	}
}

func TestRealOGG(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	for _, file := range []string{"Sound/ui_button001.ogg", "Sound/mob_ape_att01.ogg", "Sound/Music/bgm_title.ogg"} {
		raw, err := a.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		pcm, err := DecodeOGG(raw)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		t.Logf("%s: %.2f s", file, float64(len(pcm))/4/SampleRate)
	}
}

// Explicit opt-in: checks the real device at volume zero, not audible fidelity.
func TestAudioDevice(t *testing.T) {
	if os.Getenv("SHAIYA_AUDIO_TEST") != "1" {
		t.Skip("requires SHAIYA_AUDIO_TEST=1")
	}
	e, err := New(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	select {
	case <-e.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("audio device did not become ready")
	}
	p := e.context.NewPlayer(bytes.NewReader(make([]byte, SampleRate*4/2)))
	p.SetVolume(0)
	p.Play()
	if !p.IsPlaying() {
		t.Fatal("player did not start")
	}
	deadline := time.Now().Add(2 * time.Second)
	for p.IsPlaying() {
		if time.Now().After(deadline) {
			t.Fatal("audio did not drain")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := p.Err(); err != nil {
		t.Fatal(err)
	}
	// Retain references so garbage collection cannot hide a missing stop.
	music := e.context.NewPlayer(&loopReader{pcm: make([]byte, 1024)})
	music.SetVolume(0)
	music.Play()
	e.music, e.musicName = music, "fixture"
	if err := e.Music(""); err != nil {
		t.Fatal(err)
	}
	if music.IsPlaying() {
		t.Fatal("old music kept playing after switching tracks")
	}
	// Map loops reuse voices, honor both mixers, fade out, and stop on close.
	e.SetVolume(1)
	e.SetMix(0, 0, 0) // Keep the physical device silent.
	e.cache["ambient"] = entry{pcm: make([]byte, 4096)}
	e.cache["map-music"] = entry{pcm: make([]byte, 4096)}
	e.effectsMix, e.musicMix, e.ambientMix = 1, 1, 1
	targets := []LoopTarget{{File: "ambient", Gain: .5}, {File: "map-music", Gain: .72, Music: true}}
	if err := e.Ambience(targets, .1); err != nil {
		t.Fatal(err)
	}
	e.SetVolume(0)
	ambient := e.loops["ambient"].player
	if err := e.Ambience(targets, .1); err != nil || e.loops["ambient"].player != ambient {
		t.Fatal("unchanged map loop restarted", err)
	}
	e.SetMix(.3, .6, .8)
	for range 100 {
		if err := e.Ambience(nil, .1); err != nil {
			t.Fatal(err)
		}
	}
	if len(e.loops) != 0 || ambient.IsPlaying() {
		t.Fatal("departed map loops did not stop")
	}
	effect := e.context.NewPlayer(&loopReader{pcm: make([]byte, 1024)})
	effect.SetVolume(0)
	effect.Play()
	e.voices = []voice{{player: effect}}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if effect.IsPlaying() {
		t.Fatal("effect kept playing after engine shutdown")
	}
}

func TestRealMapLoopPlayback(t *testing.T) {
	if os.Getenv("SHAIYA_AUDIO_TEST") != "1" {
		t.Skip("requires device")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	e, err := New(arc.ReadFile, 1e-9)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	targets := []LoopTarget{{File: "Sound/Music/bgm_town01.ogg", Gain: .72, Music: true}, {File: "Sound/back/bg_stream02.ogg", Gain: .5}}
	deadline := time.Now().Add(5 * time.Second)
	for len(e.loops) < 2 {
		if err := e.Ambience(targets, .016); err != nil {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("real tracks never started: loops%d pending%d", len(e.loops), len(e.pending))
		}
		time.Sleep(10 * time.Millisecond)
	}
	for range 10 {
		if err := e.Ambience(targets, .016); err != nil {
			t.Fatal(err)
		}
	}
	for file, v := range e.loops {
		if !v.player.IsPlaying() || v.player.Volume() <= 0 || v.player.Err() != nil {
			t.Fatalf("%s not playing: gain%v error%v", file, v.player.Volume(), v.player.Err())
		}
		t.Logf("%s playing, gain%v", file, v.gain)
	}
	e.SetMix(1, 1, 0)
	if e.loops[targets[1].File].player.Volume() != 0 || e.loops[targets[0].File].player.Volume() == 0 {
		t.Fatal("ambient channel did not isolate its volume")
	}
	e.SetMix(0, 1, 1)
	if e.loops[targets[0].File].player.Volume() != 0 || e.loops[targets[1].File].player.Volume() == 0 {
		t.Fatal("music channel did not isolate its volume")
	}
}
