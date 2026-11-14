//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNPCDefinitionsTargetPredictionAndExpandedMapGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-options"); err != nil {
		t.Fatal(err)
	}
	d, ok := a.skillDefinition(33, 100)
	if !ok || d.Name != "Feeble" || d.Icon != 119 || d.VisualCode != 98 {
		t.Fatalf("NPC definition mismatch: %+v", d)
	}
	a.wv.self, err = scene.BuildCharacter(a.assets, scene.Appearance{Race: 0, Job: 5, Gender: 0, Height: 2, Face: 1, Hair: 1, Helmet: -1, Upper: 20, Lower: 20, Hand: 20, Foot: 20}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	a.prepareSelfForDraw()
	if a.wv.self == nil {
		t.Fatal("preview character unavailable")
	}
	a.loadSkills()
	a.state.Target = 0
	a.state.Pools.MP, a.state.Pools.SP = 65535, 65535
	found := false
	for key, d := range a.skillDefs {
		if key.Level == 1 && strings.Contains(strings.ToLower(d.Name), "magic") && strings.Contains(strings.ToLower(d.Name), "arrow") {
			found = true
			k := proto.LearnedSkill{ID: key.ID, Level: 1, Number: 7}
			a.state.Skills[7] = k
			if a.learnedSkillPacket(k) == nil || a.state.Target == 0 {
				t.Fatal("Magic Arrows did not select a living enemy")
			}
			before := a.wv.self.ActionID()
			for i := 0; i < 10; i++ {
				a.requestedSkillPresentation(proto.UseSkillPacket(7, 0, false))
			}
			if a.wv.self.ActionID() != before || a.wv.requestedSkill != nil {
				t.Fatal("invalid target replayed animation")
			}
		}
	}
	if !found {
		for key, d := range a.skillDefs {
			if key.Level == 1 && d.Attack == 3 && d.Target == 1 {
				t.Logf("candidate %d %q", key.ID, d.Name)
			}
		}
		t.Fatal("Magic Arrow fixture absent")
	}
	a.menuOpen = false
	a.wv.mapView.full = true
	x, y, scale := a.minimap.FullRect(1024, 768)
	a.lastMouseX = x + int(180*scale)
	a.lastMouseY = y + int(210*scale)
	a.Draw(1024, 768)
	out := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if out != "" {
		if err = os.MkdirAll(out, 0755); err != nil {
			t.Fatal(err)
		}
		if err = render.SavePNG(filepath.Join(out, "expanded-map-coordinates.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	if e := gl.GetError(); e != gl.NO_ERROR {
		t.Fatalf("GL error %x", e)
	}
	// Marker 100 retains NPC metadata for debuff icons/timers.
	if _, ok := a.npcSkillDefs[sdata.SkillKey{ID: 33, Level: 1}]; !ok {
		t.Fatal("NPC cache empty")
	}
	t.Log("NpcSkill identity/icon/visual verified; automatic enemy selection verified; ten forced targetless Magic Arrows produced no predicted animation; expanded map rendered")
}
