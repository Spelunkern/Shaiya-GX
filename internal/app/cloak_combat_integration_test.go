//go:build integration

package app

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func verifyNativeCloakAndHealing(t *testing.T, a, b *App, pump func(func() bool)) {
	t.Helper()
	send := func(app *App, p []byte) {
		t.Helper()
		if err := app.world.Send(p); err != nil {
			t.Fatal(err)
		}
	}
	if a.state.Inventory[[2]uint8{1, 22}].Type != 24 {
		t.Fatal("missing disposable original cloak")
	}
	send(a, proto.ItemMovePacket(1, 22, 0, 7))
	pump(func() bool {
		return a.state.Inventory[[2]uint8{0, 7}].Type == 24 && b.state.Entities[a.state.SelfID].Shape.Equipment[7][0] == 24
	})
	b.prepareRemotePlayers()
	if !a.wv.self.HasCloak() || b.wv.actors[a.state.SelfID] == nil || !b.wv.actors[a.state.SelfID].character.HasCloak() {
		t.Fatal("server-equipped model-zero cloak absent on self or observer")
	}
	send(a, proto.ItemMovePacket(0, 7, 1, 22))
	pump(func() bool {
		return a.state.Inventory[[2]uint8{0, 7}].Type == 0 && b.state.Entities[a.state.SelfID].Shape.Equipment[7][0] == 0
	})
	b.prepareRemotePlayers()
	if a.wv.self.HasCloak() || b.wv.actors[a.state.SelfID].character.HasCloak() {
		t.Fatal("unequipped cloak remained visible")
	}
	send(b, proto.LearnSkillPacket(148, 1))
	pump(func() bool { return b.state.LastLearn.ID == 148 })
	if b.state.LastLearn.Result != 0 {
		t.Fatal("priest could not learn original Instant Heal", b.state.LastLearn)
	}
	number := b.state.LastLearn.Number
	b.state.Target = a.state.SelfID
	before := len(b.combatNumbers)
	packet := b.learnedSkillPacket(b.state.Skills[number])
	if b.approachSkillPacket(packet) {
		t.Fatal("nearby heal unexpectedly required approach")
	}
	if len(b.combatNumbers) != before {
		t.Fatal("healing predicted before confirmation")
	}
	send(b, packet)
	pump(func() bool { return b.state.LastSkill.ID == 148 && b.state.LastSkill.Target == a.state.SelfID })
	if b.state.LastSkill.Result > 1 || len(b.combatNumbers) <= before || !b.combatNumbers[len(b.combatNumbers)-1].Healing {
		t.Fatal("server Instant Heal did not produce green combat number", b.state.LastSkill, b.combatNumbers)
	}
	t.Log("native cloak equipped/removed on self and observer; original Instant Heal number appears only after server confirmation")
}

func TestPhoenixCombatNumbersGL(t *testing.T) {
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
	if err = a.Preview("world-inventory"); err != nil {
		t.Fatal(err)
	}
	a.scene3d.View = render.LookAtLH(render.V3{0, 2, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0})
	a.scene3d.Proj = render.PerspectiveFovLH(math.Pi/4, 1024.0/768, .1, 50)
	a.combatNumbers = []combatNumber{
		{Point: render.V3{-1.7, 1, 0}, Amount: 1234},
		{Point: render.V3{-.5, 1, 0}, Amount: 567, Critical: true},
		{Point: render.V3{.6, 1, 0}, Missed: true},
		{Point: render.V3{1.7, 1, 0}, Amount: 890, Healing: true},
	}
	a.clock = .1
	gl.Viewport(0, 0, 1024, 768)
	gl.ClearColor(.15, .18, .22, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawCombatNumbers(1024, 768)
	a.sprites.End()
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("combat number GL error %#x", code)
	}
	pixels := make([]byte, 1024*768*4)
	gl.ReadPixels(0, 0, 1024, 768, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
	green, red, light := 0, 0, 0
	for i := 0; i < len(pixels); i += 4 {
		r, g, b := int(pixels[i]), int(pixels[i+1]), int(pixels[i+2])
		if g > 90 && g > r*2 && g > b {
			green++
		}
		if r > 100 && r > g*2 && r > b*2 {
			red++
		}
		if r > 150 && g > 150 && b > 150 {
			light++
		}
	}
	if green < 10 || red < 10 || light < 10 {
		t.Fatal("atlas digits/critical/healing were not rendered", green, red, light)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = render.SavePNG(filepath.Join(dir, "combat-numbers-phoenix.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
}
