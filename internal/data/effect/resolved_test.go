package effect

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"os"
	"testing"
)

func TestNativeResolvedRecipes(t *testing.T) {
	for _, tc := range []struct {
		id             uint16
		flight, impact string
	}{
		{29, "CA_magic.EFT", "DA_target_generic.EFT"},
		{34, "CA_fire.EFT", "DA_target_generic.EFT"},
		{37, "CA_ice.EFT", "DA_target_generic.EFT"},
		{45, "CA_piercing.EFT", "DA_target_generic.EFT"},
		{103, "", "DA_target_dispel.EFT"},
		{30, "", "DA_target_bash.EFT"},
	} {
		flight, impact := ProjectileLibrary(tc.id), ImpactLibrary(tc.id)
		if tc.flight != "" {
			tc.flight = "Effect/" + tc.flight
		}
		if tc.impact != "" {
			tc.impact = "Effect/" + tc.impact
		}
		if flight != tc.flight || impact != tc.impact {
			t.Errorf("skill %d: %q / %q", tc.id, flight, impact)
		}
	}
	for _, id := range []uint16{0, 1, 364, 65535} {
		if ImpactLibrary(id) != "" || ProjectileLibrary(id) != "" {
			t.Errorf("invented recipe for %d", id)
		}
	}
}
func TestRealResolvedRecipes(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	files := map[string]bool{}
	supported := 0
	projectiles, impacts := map[string]bool{}, map[string]bool{}
	for id := uint16(1); id <= 400; id++ {
		for i, file := range []string{ProjectileLibrary(id), ImpactLibrary(id)} {
			if file == "" {
				continue
			}
			if i == 0 {
				projectiles[file] = true
			} else {
				impacts[file] = true
			}
			if files[file] {
				continue
			}
			files[file] = true
			raw, err := a.ReadFile(file)
			if err != nil {
				t.Error(err)
				continue
			}
			l, err := ParseEFT(raw)
			if err != nil {
				t.Error(file, err)
				continue
			}
			if len(l.Sequences) == 0 {
				t.Error("no sequences", file)
				continue
			}
			for _, r := range l.Sequences[0].Records {
				if r.Effect < 0 || int(r.Effect) >= len(l.Effects) {
					t.Error("invalid emitter", file)
					continue
				}
				if l.Effects[r.Effect].LinearSupported() {
					supported++
					break
				}
			}
		}
	}
	t.Logf("%d projectile libraries, %d impact libraries, %d unique; %d have supported particles in sequence 0", len(projectiles), len(impacts), len(files), supported)
}
