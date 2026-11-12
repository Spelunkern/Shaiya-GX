package scene

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestRealGestureAnimations(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("requires original data")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	count := 0
	a.Walk(func(file string, _ *archive.File) {
		if !strings.HasPrefix(strings.ToLower(file), "character/") || !strings.HasSuffix(strings.ToLower(file), "_action.alt") {
			return
		}
		data, err := a.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		actions, err := model.ParseALT(data)
		if err != nil {
			t.Fatal(err)
		}
		for motion := uint8(116); motion <= 125; motion++ {
			action, _ := proto.GestureAction(motion)
			if action >= len(actions) || actions[action].Name == "" {
				t.Fatalf("%s missing action %d", file, action)
			}
			name := path.Join(path.Dir(file), "ANI", path.Base(actions[action].Name))
			if _, err := a.ReadFile(name); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		}
		if len(actions) <= 1 || !strings.HasSuffix(strings.ToLower(actions[1].Name), "_001_walk.ani") {
			t.Fatalf("%s has no standard walk action", file)
		}
		if _, err := a.ReadFile(path.Join(path.Dir(file), "ANI", path.Base(actions[1].Name))); err != nil {
			t.Fatalf("walk animation in %s: %v", file, err)
		}
		count++
	})
	if count == 0 {
		t.Fatal("no character action tables")
	}
	t.Logf("verified all ten gestures in %d action tables", count)
}
