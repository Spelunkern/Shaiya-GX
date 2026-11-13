package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestNativeSkillReleasePhases(t *testing.T) {
	for _, tc := range []struct {
		id                            uint16
		attack, effect, weapon, level uint8
		file                          string
		sequence                      int
	}{
		{31, 0, 4, 1, 1, "IN_mc_yellow.EFT", 0},
		{85, 3, 3, 0, 1, "IN_mc_water.EFT", 0},
		{37, 3, 3, 0, 2, "IN_ma_water.EFT", 0},
		{57, 1, 1, 1, 3, "IN_sa_ddamage.EFT", 2},
		{57, 1, 1, 11, 3, "IN_mc_yellow.EFT", 0},
		{38, 1, 1, 1, 2, "IN_sd_wind.EFT", 1},
	} {
		r := proto.SkillResult{Source: 1, Target: 2, ID: tc.id, Level: tc.level}
		d := sdata.Skill{Attack: tc.attack, Effect: tc.effect}
		file, sequence := skillReleaseLibrary(r, d, tc.weapon)
		if file != "Effect/"+tc.file || sequence != tc.sequence {
			t.Fatalf("skill %d: %s:%d; want %s:%d", tc.id, file, sequence, tc.file, tc.sequence)
		}
		for _, invalid := range []proto.SkillResult{
			{Source: 1, ID: tc.id},
			{Source: 1, ID: tc.id, Level: 100},
			{Source: 1, ID: tc.id, Level: tc.level, Result: 5},
			{Source: 1, ID: tc.id, Level: tc.level, Result: 3},
		} {
			if file, _ := skillReleaseLibrary(invalid, d, tc.weapon); file != "" {
				t.Fatal("refused or invalid cast created a release", invalid)
			}
		}
		d.Effect = 0
		if file, _ := skillReleaseLibrary(r, d, tc.weapon); file != "" {
			t.Fatal("passive created a release")
		}
	}
}
