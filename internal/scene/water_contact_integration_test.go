//go:build integration

package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"os"
	"testing"
)

func TestWaterContactResource(t *testing.T) {
	root := os.Getenv("SHAIYA_DATA")
	if root == "" {
		t.Skip("requires assets")
	}
	for _, name := range []string{"Cwater.EFT", "Water.EFT"} {
		b, err := os.ReadFile(root + "/Effect/" + name)
		if err != nil {
			t.Fatal(err)
		}
		lib, err := effect.ParseEFT(b)
		if err != nil {
			t.Fatal(err)
		}
		if len(lib.Sequences) != 1 || len(lib.Effects) != 2 {
			t.Fatal(name, "unexpected contact library")
		}
		for _, e := range lib.Effects {
			if !e.LinearSupported() || e.BaseAxis != 1 {
				t.Fatal(name, "unsupported horizontal contact emitter")
			}
		}

	}
}
