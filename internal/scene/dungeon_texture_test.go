package scene

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
)

func TestLooseDungeonTexturesKeepCanonicalPriority(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"world/Map/dungeon/fort/light.dds",
		"world/dungeon/fort/light.dds",
		"world/dungeon/fort/entity.dds",
		"Entity/Texture/entity.dds",
	} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	a, err := archive.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	s := &Assets{Arc: a, Missing: map[string]bool{}}
	s.FindTexture("absent.dds") // Build the path index without uploading a texture.
	if got := s.byStem["light"]; got != "world/dungeon/fort/light.dds" {
		t.Fatal(got)
	}
	if got := s.byStem["entity"]; got != "Entity/Texture/entity.dds" {
		t.Fatal(got)
	}
}
