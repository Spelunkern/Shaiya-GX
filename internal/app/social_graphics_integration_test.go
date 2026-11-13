//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOriginalSocialAndGraphicsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true})
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
	if err = a.Preview("world-inventory"); err != nil {
		t.Fatal(err)
	}
	a.inventory.open = false
	a.guild.open = true
	a.state.Social.Friends = map[uint32]proto.SocialContact{123: {ID: 123, Name: "Buddy", Class: 1, Online: true, Memo: "Tank"}}
	a.state.Social.Blocked = map[uint32]proto.SocialContact{124: {ID: 124, Name: "Blocked", Memo: "Memo"}}
	a.state.Social.Search = []proto.PartySearchEntry{{Level: 40, Class: 1, Name: "Defender"}}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	capture := func(name string, draw func()) {
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.05, .08, .1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		draw()
		a.sprites.End()
		if err := render.SavePNG(filepath.Join(dir, name), 1024, 768); err != nil {
			t.Fatal(err)
		}
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("GL error %#x", code)
		}
	}
	for page, name := range map[int]string{1: "social-buddy.png", 2: "social-block.png", 3: "social-party-search.png"} {
		a.guildUI.Page = page
		capture(name, func() { a.drawGuild(1024, 768) })
	}
	a.gameMenu.Open(ui.MenuGraphics)
	capture("classic-graphics-sliders.png", func() { a.gameMenu.Draw(a.sprites, a.font, 1024, 768) })
	// Gamma works on the existing completed frame and preserves monotonic levels.
	target, err := render.NewWaterTarget(1024, 768)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	previous := uint8(0)
	for level := 0; level <= 4; level++ {
		target.Capture(render.V3{.25, .25, .25}, func() {})
		target.PresentBrightness(level)
		var pixel [4]uint8
		gl.ReadPixels(512, 384, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
		if pixel[0] <= previous {
			t.Fatal("brightness did not increase", level, pixel, previous)
		}
		previous = pixel[0]
	}
	a.prepareMonsters()
	a.loadSkills()
	old := len(a.wv.impacts)
	a.startMobSkillEffects(proto.SkillResult{Source: a.state.SelfID, Target: a.state.SelfID, ID: 33, Level: 100})
	if len(a.wv.impacts) <= old {
		t.Fatal("monster level-100 result did not produce original impact")
	}
	a.state.Entities[900] = &game.Entity{ID: 900, Kind: game.KindMob, TypeID: 2009, X: a.wv.pos[0] + 1, Z: a.wv.pos[2]}
	a.prepareMonsters()
	mob := a.wv.mobs[900]
	if mob == nil || mob.actor == nil {
		t.Fatal("missing original monster")
	}
	r := proto.SkillResult{Source: 900, Target: a.state.SelfID, ID: 33, Level: 100}
	a.startMobSkillEffects(r)
	mob.actor.Update(.1)
	clock := mob.actor.Time
	a.startMobSkillEffects(r)
	if mob.actor.Time != clock {
		t.Fatal("same area attack restarted monster animation")
	}

}
