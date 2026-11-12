//go:build integration

package scene

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestSwimAnimationAliasesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "swim clips", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := NewAssets(arc)
	for _, appearance := range []Appearance{{Race: 0, Job: 0}, {Race: 1, Gender: 0, Job: 2}, {Race: 2, Gender: 1, Job: 2}, {Race: 3, Job: 0}} {
		appearance.Helmet = -1
		c, err := BuildCharacter(a, appearance, "000_normal")
		if err != nil {
			t.Fatal(err)
		}
		for _, clip := range []string{"006_swnormal", "007_swim"} {
			c.SetClip(clip)
			if c.clipName != clip {
				t.Fatalf("%s could not load %s", appearance.Prefix(), clip)
			}
		}
	}
}
