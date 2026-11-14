//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestQuestIndicatorsAndNativeForgeModesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
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
	for _, tc := range []struct {
		file string
		seq  int
	}{{"Effect/Quest.EFT", 0}, {"Effect/Quest.EFT", 1}, {"Effect/Quest.EFT", 2}, {"Effect/Quest.EFT", 3}, {"Effect/up.EFT", 1}} {
		fx, err := scene.LoadEffect(a.assets, tc.file, tc.seq)
		if err != nil {
			t.Fatal(tc, err)
		}
		if fx.Skipped != 0 {
			t.Fatalf("%v omitted %d emitters", tc, fx.Skipped)
		}
		for i := 0; i < 30; i++ {
			fx.Update(1.0 / 60)
		}
		if fx.Done() {
			t.Fatalf("%v expired before its display interval", tc)
		}
	}
	if err = a.Preview("world-npcquest"); err != nil {
		t.Fatal(err)
	}
	a.loadSkills()
	auraFound := false
	a.state.SelfID = 1
	a.state.Target = 0
	for key, d := range a.skillDefs {
		if key.Level == 1 && strings.Contains(strings.ToLower(d.Name), "peace") {
			t.Logf("peace skill %d %q target=%d effect=%d", key.ID, d.Name, d.Target, d.Effect)
			if strings.Contains(strings.ToLower(d.Name), "aura") {
				auraFound = true
				if a.resolveSkillTarget(d) != 1 {
					t.Fatal("Aura of Peace did not self-target")
				}
			}
		}
	}
	if !auraFound {
		t.Fatal("Aura of Peace source fixture absent")
	}
	e := a.state.Entities[42]
	e.X, e.Z = a.wv.pos[0]+3, a.wv.pos[2]+3
	e.Y = a.wv.world.Height(e.X, e.Z)
	key := sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}
	def := a.npcDefs[key]
	def.StartQuests = []uint16{1}
	def.EndQuests = []uint16{1}
	a.npcDefs[key] = def
	a.questDefs[0] = sdata.Quest{Name: "Indicator fixture", MinLevel: 0, MaxLevel: 65535, Faction: 6, Jobs: [6]uint8{1, 1, 1, 1, 1, 1}, StartNPCType: e.NpcType, StartNPCID: e.TypeID, EndNPCType: e.NpcType, EndNPCID: e.TypeID}
	a.state.Quests = map[uint16]proto.ActiveQuest{}
	a.state.FinishedQuests = nil
	a.prepareNPCs()
	npc := a.wv.npcs[42]
	if npc == nil || npc.questEffect == nil || npc.questKind != ui.MapQuestStart {
		t.Fatalf("available quest has no indicator: %+v", npc)
	}
	first := npc.questEffect
	a.state.Quests[1] = proto.ActiveQuest{ID: 1}
	a.clock += .3
	a.prepareNPCs()
	if npc.questEffect == nil || npc.questEffect == first || npc.questKind != ui.MapQuestEnd {
		t.Fatal("quest indicator did not change to ready")
	}
	for id := uint32(43); id < 67; id++ {
		copy := *e
		copy.ID = id
		a.state.Entities[id] = &copy
	}
	for frame := 0; frame < 16; frame++ {
		a.clock += 1.0 / 60
		a.prepareNPCs()
	}
	indicators := 0
	for _, actor := range a.wv.npcs {
		if actor.questEffect != nil {
			indicators++
		}
	}
	if indicators != 16 {
		t.Fatalf("nearby quest effect cap: got %d, want 16", indicators)
	}
	marks := a.questMapMarkers(float32(a.wv.world.WLD.MapSize), nil)
	if len(marks) == 0 {
		t.Fatal("ready quest has no map mark")
	}
	out := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if out == "" {
		out = t.TempDir()
	}
	for _, preview := range []string{"world-npcquest", "world-npcquest-detail", "world-npcquest-reward", "world-quest"} {
		if err = a.Preview(preview); err != nil {
			t.Fatal(err)
		}
		a.Draw(1024, 768)
		if err = render.SavePNG(filepath.Join(out, preview+".png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	a.state = game.New()
	a.blacksmith.selected = [4]bool{}
	for mode, name := range []string{"lapis", "enhance", "recreate"} {
		a.blacksmithUI.Mode = mode
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.drawBlacksmith(1024, 768, "Blacksmith")
		a.sprites.End()
		if err = render.SavePNG(filepath.Join(out, "forge-"+name+".png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	if e := gl.GetError(); e != gl.NO_ERROR {
		t.Fatalf("GL error %x", e)
	}
	t.Log("all four Quest.EFT sequences and completion emitters supported; available/ready transition and map marker verified; native NPC, journal and three forge modes rendered")
}
