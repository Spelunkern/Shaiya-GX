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

func TestRealAnimatedEffectTexturesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "animated effect", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := NewAssets(arc)
	var chosen string
	arc.Walk(func(path string, f *archive.File) {
		if chosen != "" || !strings.HasSuffix(strings.ToLower(path), ".eft") {
			return
		}
		b, err := arc.Read(f)
		if err != nil {
			return
		}
		lib, err := effect.ParseEFT(b)
		if err != nil || lib.Format != "EFT" {
			return
		}
		for seq, sequence := range lib.Sequences {
			for _, record := range sequence.Records {
				if record.Effect < 0 || int(record.Effect) >= len(lib.Effects) {
					continue
				}
				def := &lib.Effects[record.Effect]
				if len(def.Textures) <= 1 || !def.LinearSupported() {
					continue
				}
				instance, err := LoadEffect(assets, path, seq)
				if err != nil {
					continue
				}
				for i := range instance.emitters {
					e := &instance.emitters[i]
					if len(e.textures) > 1 {
						e.advanceTexture(max(e.definition.FrameDelay, 0) + 1)
						if e.textureFrame != 1 {
							t.Fatal("real texture sequence did not advance")
						}
						chosen = path
						return
					}
				}
			}
		}
	})
	if chosen == "" {
		t.Fatal("no supported effect with multiple original textures")
	}
	t.Logf("loaded and advanced original animated effect %s", chosen)
}
