package app

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestRealSkillActionSounds(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	raw, err := a.ReadFile("Character/Skill.SData")
	if err != nil {
		t.Fatal(err)
	}
	defs, err := sdata.ParseSkills(raw)
	if err != nil {
		t.Fatal(err)
	}
	// Source skill IDs retain the original SData, independently of Phoenix's
	// Azeroth IDs. Physical Double Damage must not acquire a magic release cue.
	for _, tc := range []struct {
		id             uint16
		ready, release string
	}{
		{30, "ch_buready01.wav", "ch_buattack01.wav"},
		{35, "ch_buready01.wav", "ef_in_firing01.wav"},
		{36, "ch_maready01.wav", "ef_in_firing01.wav"},
		{37, "ch_buready01.wav", "ef_in_firing01.wav"},
		{57, "ch_maready01.wav", ""},
	} {
		d, ok := defs[sdata.SkillKey{ID: tc.id, Level: 1}]
		if !ok || scene.SkillActionSound(scene.SkillCastAction(d.Animation, d.Effect)) != tc.ready ||
			scene.SkillActionSound(scene.SkillAction(d.Animation, d.Attack, d.Effect, 0, 0)) != tc.release {
			t.Fatalf("skill %d: original action/sound mismatch", tc.id)
		}
	}
	sounds := map[string]bool{}
	for _, d := range defs {
		if d.Effect == 0 {
			continue
		}
		for _, action := range []int{scene.SkillCastAction(d.Animation, d.Effect), scene.SkillAction(d.Animation, d.Attack, d.Effect, 0, 0)} {
			name := scene.SkillActionSound(action)
			if name == "" || sounds[name] {
				continue
			}
			sounds[name] = true
			b, err := a.ReadFile("Sound/" + name)
			if err != nil {
				t.Fatal(err)
			}
			pcm, err := audio.DecodeWAV(b)
			if err != nil || len(pcm) == 0 || len(pcm) > 4<<20 {
				t.Fatalf("%s: invalid skill PCM (%d bytes): %v", name, len(pcm), err)
			}
			t.Logf("%s: %.2f s", name, float64(len(pcm))/4/audio.SampleRate)
		}
	}
	if len(sounds) != 6 {
		t.Fatalf("common skill actions resolved %d sounds, want six", len(sounds))
	}
}

func TestMusicUsesMapZone(t *testing.T) {
	w := &world.WLD{MusicAssets: []string{"track.wav"}, MusicZones: []world.MusicZone{{Box: world.Box{Min: world.Vec3{0, 0, 0}, Max: world.Vec3{10, 10, 10}}, Asset: 0}}}
	if mapMusic(w, 5, 5) != "track.wav" || mapMusic(w, 20, 5) != "" {
		t.Fatal("zone selection mismatch")
	}
	w.MusicZones[0].Asset = 4
	if mapMusic(w, 5, 5) != "" {
		t.Fatal("invalid music index accepted")
	}
}

func TestRealWeaponSounds(t *testing.T) {
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
	for weapon := uint8(1); weapon <= 15; weapon++ {
		for variant := uint8(0); variant < 3; variant++ {
			swing, hit := scene.WeaponSounds(weapon, variant == 2, variant)
			for _, name := range []string{swing, hit} {
				if name == "" || files[name] {
					continue
				}
				files[name] = true
				raw, err := a.ReadFile("Sound/" + name)
				if err != nil {
					t.Fatal(err)
				}
				pcm, err := audio.DecodeWAV(raw)
				if err != nil || len(pcm) == 0 || len(pcm) > 4<<20 {
					t.Fatalf("%s: invalid weapon PCM (%d bytes): %v", name, len(pcm), err)
				}
			}
		}
	}
	if len(files) != 87 {
		t.Fatalf("native weapon catalog resolved %d WAVs, want 87", len(files))
	}
	t.Log("15 weapon categories resolved 87 original WAVs, all decoded within the voice budget")
}
