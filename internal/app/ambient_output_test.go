package app

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// The harness moves this test's stream to an isolated sink before opening the
// gate. It records ONLY that sink and verifies non-zero ambient output.
func TestAmbientCapturedAtMap19(t *testing.T) {
	gate := os.Getenv("SHAIYA_AUDIO_CAPTURE_GATE")
	if gate == "" {
		t.Skip("requires isolated capture harness")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("world/19.wld")
	if err != nil {
		t.Fatal(err)
	}
	w, err := world.ParseWLD(raw)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := audio.New(arc.ReadFile, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	if err := os.WriteFile(gate+".pid", []byte(fmt.Sprint(os.Getpid())), 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(gate); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("capture sink not ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	engine.SetMix(0, 0, 1)
	engine.SetVolume(.6)
	a := &App{screen: screenWorld, arc: arc, sound: engine, soundErrors: map[string]bool{}, state: game.New(), frameDT: .016, wv: &worldView{world: &scene.World{WLD: w}, pos: render.V3{635.946, 7.56, 1841.597}}}
	end := time.Now().Add(3 * time.Second)
	for time.Now().Before(end) {
		a.updateMapAudio()
		time.Sleep(16 * time.Millisecond)
	}
	if len(a.soundErrors) > 0 {
		t.Fatal(a.soundErrors)
	}
	if len(a.mapAudio.targets) != 1 || a.mapAudio.targets[0].Music {
		t.Fatal("expected field ambience alone", a.mapAudio.targets)
	}
	t.Logf("map19 ambience scheduled through real application update: %+v", a.mapAudio.targets)
}
