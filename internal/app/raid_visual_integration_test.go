//go:build integration

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestRaidPanelOriginalAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "raid panel", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
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
	a.state = game.New()
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Player 01", Level: 60}
	a.state.Pools.HP, a.state.Pools.MP, a.state.Pools.SP = 80, 60, 40
	a.state.Details.MaxHP, a.state.Details.MaxMP, a.state.Details.MaxSP = 100, 100, 100
	a.state.Party.Raid = true
	a.state.Party.RaidDivision = 1
	a.state.Party.RaidSlots = map[uint16]uint32{}
	a.state.Party.Leader = 1
	for slot := uint16(0); slot < 30; slot++ {
		id := uint32(slot) + 1
		a.state.Party.RaidSlots[slot] = id
		a.state.Party.Members = append(a.state.Party.Members, proto.PartyMember{ID: id, Name: fmt.Sprintf("Player %02d", id), Level: 60, Class: uint8(slot % 6), HP: 80, MaxHP: 100, MP: 60, MaxMP: 100, SP: 40, MaxSP: 100})
	}
	gl.Viewport(0, 0, 1024, 768)
	gl.ClearColor(.08, .08, .08, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawParty(1024, 768)
	a.sprites.End()
	x, y := ui.RaidSlotPosition(1024, 768, 23)
	a.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 5, MouseY: y + 5}, 1024, 768)
	if !a.updatePartyPointer(&platform.Input{Clicked: true, MouseX: x + 5, MouseY: y + 5}, 1024, 768) || a.state.Target != 24 {
		t.Fatal("raid selection failed", a.state.Target)
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "raid-panel-preview.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	a.party.raidHidden = true
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawParty(1024, 768)
	a.sprites.End()
	cx, cy := a.partyUI.RaidCompactOrigin(1024, 768)
	a.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: cx + 20 + 2*24 + 5, MouseY: cy + 12}, 1024, 768)
	a.updatePartyPointer(&platform.Input{Clicked: true, MouseX: cx + 20 + 2*24 + 5, MouseY: cy + 12}, 1024, 768)
	a.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: cx + 10, MouseY: cy + 24 + ui.PartyRowHeight + 4}, 1024, 768)
	a.updatePartyPointer(&platform.Input{Clicked: true, MouseX: cx + 10, MouseY: cy + 24 + ui.PartyRowHeight + 4}, 1024, 768)
	if a.state.Target != 14 {
		t.Fatalf("compact raid selection got %d", a.state.Target)
	}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawParty(1024, 768)
	a.sprites.End()
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "raid-compact-preview.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	pixels := make([]byte, 1024*768*4)
	measure := func() uint64 {
		gl.ReadPixels(0, 0, 1024, 768, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
		var sum uint64
		for i := 0; i < len(pixels); i += 4 {
			sum += uint64(pixels[i]) + uint64(pixels[i+1]) + uint64(pixels[i+2])
		}
		return sum
	}
	before := measure()
	a.party.raidHidden = false
	rx, ry := a.partyUI.RaidOrigin(1024, 768)
	a.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: rx + 308, MouseY: ry + 370}, 1024, 768)
	a.updatePartyPointer(&platform.Input{Clicked: true, MouseX: rx + 429, MouseY: ry + 370}, 1024, 768)
	a.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: rx + 390, MouseY: ry + 410}, 1024, 768)
	a.updatePartyPointer(&platform.Input{Clicked: true, MouseX: rx + 390, MouseY: ry + 410}, 1024, 768)
	if !a.party.raidHidden {
		t.Fatal("raid settings did not close")
	}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawParty(1024, 768)
	a.sprites.End()
	if after := measure(); after >= before {
		t.Fatalf("opacity slider did not dim compact panel: %d -> %d", before, after)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("raid GL error %#x", code)
	}
}
