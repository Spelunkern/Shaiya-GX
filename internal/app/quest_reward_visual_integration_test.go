//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestQuestRewardChoiceOriginalAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "quest rewards", Width: 1024, Height: 734, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Preview("world-npcquest-reward"); err != nil {
		t.Fatal(err)
	}
	q := a.questDefs[225]
	if q.UserSelect < 2 {
		t.Fatalf("quest 226 offers %d choices", q.UserSelect)
	}
	for i := 0; i < int(q.UserSelect) && i < 3; i++ {
		if label := a.questRewardLabel(q, i); !strings.Contains(label, " XP, ") || !strings.Contains(label, " gold") {
			t.Fatalf("incomplete reward label: %q", label)
		}
	}
	for i := 0; i < 240; i++ {
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 734)
		if a.SceneReady() {
			break
		}
	}
	if !a.SceneReady() {
		t.Fatal("field did not finish streaming")
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "quest-reward-preview.png"), 1024, 734); err != nil {
			t.Fatal(err)
		}
	}
	// Exercise the maximum three protocol choices with the same original UI.
	a.questDefs[225].UserSelect = 3
	a.questDefs[225].Results[2] = a.questDefs[225].Results[0]
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.Draw(1024, 734)
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "quest-reward-three-preview.png"), 1024, 734); err != nil {
			t.Fatal(err)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("reward GL error %#x", code)
	}
}
