//go:build integration

package scene

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestRealStationaryZeroLifeEffectGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "static effects", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := NewAssets(arc)
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View = render.LookAtLH(render.V3{0, 1, -5}, render.V3{}, render.V3{0, 1, 0})
	s.Proj = render.PerspectiveFovLH(1, 1, .1, 1000)
	var selected string
	arc.Walk(func(file string, _ *archive.File) {
		if selected != "" || !strings.HasSuffix(strings.ToLower(file), ".eft") {
			return
		}
		raw, err := arc.ReadFile(file)
		if err != nil {
			return
		}
		lib, err := effect.ParseEFT(raw)
		if err != nil {
			return
		}
		for seq, sequence := range lib.Sequences {
			found := false
			for _, r := range sequence.Records {
				if r.Effect >= 0 && int(r.Effect) < len(lib.Effects) {
					d := &lib.Effects[r.Effect]
					if d.InitialDelay == 0 && d.Unk13 == 0 && len(d.Colors) == 0 && d.LinearSupported() {
						found = true
						break
					}
				}
			}
			if !found {
				continue
			}
			x, err := loadEffectLibrary(assets, file, seq, lib)
			if err != nil {
				continue
			}
			for i := 0; i < 120; i++ {
				x.Update(1.0/60 + .000001)
			}
			for _, e := range x.emitters {
				if e.definition.InitialDelay == 0 && e.definition.Unk13 == 0 && len(e.definition.Colors) == 0 && len(e.particles) > 0 {
					x.emitters = []effectEmitter{e}
					render.ResetDrawCalls()
					x.Draw(s)
					if render.DrawCalls() == 0 {
						t.Fatal("no effect draw")
					}
					selected = file
					x.Stop()
					x.Update(.25)
					if !x.Done() {
						t.Fatal("zero-life emitter was retained after stop")
					}
					break
				}
			}
			if selected != "" {
				break
			}
		}
	})
	if selected == "" {
		t.Fatal("no usable zero-life effect in original assets")
	}
	t.Logf("loaded and rendered zero-life emitter from %s", selected)
}
