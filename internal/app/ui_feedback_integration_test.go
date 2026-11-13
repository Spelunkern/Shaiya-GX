//go:build integration

package app

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestNativeUIFeedbackGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
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
	cursors := a.LoadCursors(win)
	defer cursors.Close()
	for i := 0; i < 7; i++ {
		cursors.Set(i)
	}
	if err = a.Preview("world-inventory"); err != nil {
		t.Fatal(err)
	}
	a.clock = 100
	a.frameDT = 0
	own := a.state.Slots[0]
	own.Level, own.Mode = 60, 1
	a.state.Slots[0] = own
	a.state.Details.SkillPoint = 10
	a.skills.open = true
	a.skills.category = 1
	a.skillBook.Position.Move(350, 80, 1024, 768, 333, 489)
	a.loadSkills()
	skill := a.skillDefs[sdata.SkillKey{ID: 57, Level: 1}]
	if skill.ResetTime == 0 {
		t.Fatal("expected original Double Damage reuse", skill)
	}
	a.state.Skills[0] = proto.LearnedSkill{ID: 57, Level: 1, Number: 0}
	a.state.SkillBarReady = true
	a.state.QuickSlots = []proto.QuickSlot{{Slot: 0, Bag: 100, Number: 0}, {Slot: 1, Bag: 1, Number: 1}}
	a.skills.cooldowns = map[uint16]float64{57: 100 + float64(skill.ResetTime)/2}
	potion := a.state.Inventory[[2]uint8{1, 1}]
	def := a.itemsItem(potion.Type, potion.TypeID)
	a.inventory.cooldowns = map[uint8]float64{def.ReqIg: 100 + itemReuseSeconds(def.ReqIg)/2}
	a.skills.pressedUntil[0] = 100.15
	if a.skillDefs[sdata.SkillKey{ID: 31, Level: 3}].Name == "" {
		t.Fatal("missing original buff level")
	}
	a.state.Buffs = []proto.ActiveBuff{{Instance: 1, Skill: 31, Level: 3}}
	a.confirmedBuffs(proto.CharacterActiveBuffs)
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	capture := func(name string) {
		t.Helper()
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.2, .3, .4, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.drawInventory(1024, 768)
		a.drawSkills(1024, 768)
		a.drawBuffs(1024, 768)
		a.sprites.End()
		if e := gl.GetError(); e != gl.NO_ERROR {
			t.Fatalf("UI feedback GL error %#x", e)
		}
		if err := render.SavePNG(filepath.Join(dir, name), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	capture("native-ui-cooldowns.png")
	a.clock = a.skills.cooldowns[57] + .1
	capture("native-ui-ready-flash.png")
	// A half sweep must reveal the right side and still shade the left side.
	gl.ClearColor(1, 1, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.sprites.DrawCooldown(a.white, 100, 100, 32, .5, [4]float32{0, 0, 0, 1})
	a.sprites.End()
	pixels := make([]byte, 1024*768*4)
	gl.ReadPixels(0, 0, 1024, 768, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
	sample := func(x, y int) byte { return pixels[((767-y)*1024+x)*4] }
	if sample(105, 110) > 5 || sample(126, 110) < 250 {
		t.Fatal("native cooldown sweep did not uncover clockwise", sample(105, 110), sample(126, 110))
	}

	a.wv.self = a.inventory.preview
	a.prepareSelfForDraw()
	d := a.skillDefs[sdata.SkillKey{ID: 31, Level: 1}]
	if !a.wv.self.PlayAction(scene.SkillAction(d.Animation, d.Attack, d.Effect, a.playerWeapon(a.state.SelfID), 0), false) {
		t.Fatal("missing real release animation")
	}
	a.wv.pos = render.V3{}
	target := a.state.Entities[11]
	target.X, target.Y, target.Z = 10, 0, 0
	result := proto.SkillResult{ID: 37, Level: 1, Source: a.state.SelfID, Target: 11}
	a.wv.self.Time = .5
	a.startSkillProjectile(result, "Effect/CA_ice.EFT", "Effect/DA_target_generic.EFT")
	if len(a.wv.projectiles) != 1 || math.Abs(float64(a.wv.projectiles[0].pos[0]-5)) > .01 || a.wv.projectiles[0].instance.Time < .08 {
		t.Fatal("late packet replayed launch/flight instead of catching up")
	}
	a.wv.self.Time = .7
	a.startSkillProjectile(result, "Effect/CA_ice.EFT", "Effect/DA_target_generic.EFT")
	if len(a.wv.projectiles) != 1 || len(a.wv.impacts) != 1 || a.wv.impacts[0].instance.Time < .08 {
		t.Fatal("elapsed flight did not proceed directly to original impact")
	}
	t.Log("real EFT late launch at .5s resumes halfway; at .7s starts the elapsed impact without another projectile")
	t.Logf("single-column K; original %ds skill / %.0fs potion reuse; press, ready flash, level-three buff and all seven OS cursors", skill.ResetTime, itemReuseSeconds(def.ReqIg))
}
