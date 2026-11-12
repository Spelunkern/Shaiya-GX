package scene

import (
	"fmt"
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
)

func TestRealSwimAnimationsExist(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original assets")
	}
	a, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	jobs := [4][]uint8{{0, 4}, {2, 4}, {2, 4}, {0, 2}}
	for race, folder := range raceFolders {
		for gender := uint8(0); gender < 2; gender++ {
			for _, job := range jobs[race] {
				prefix := (Appearance{Race: uint8(race), Gender: gender, Job: job}).Prefix()
				idle := "006_swnormal"
				if race == 1 && gender == 0 {
					idle = "006_swmormal"
				}
				if race == 2 && gender == 1 {
					idle = "006_swnomal"
				}
				for _, clip := range []string{idle, "007_swim"} {
					name := fmt.Sprintf("Character/%s/ANI/%s_%s.ANI", folder, prefix, clip)
					if _, ok := a.Lookup(name); !ok {
						t.Errorf("missing %s", name)
					}
				}
			}
		}
	}
}
