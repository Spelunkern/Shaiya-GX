//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHUDPolishOriginalAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original data")
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
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = a.Preview("world-inventory"); err != nil {
		t.Fatal(err)
	}
	a.loadSkills()
	for id := uint16(1); id < 1000 && len(a.buffs.rows) < 6; id++ {
		for key, def := range a.skillDefs {
			if key.ID == id && key.Level == 1 && def.Icon > 0 {
				if len(a.buffs.rows)%2 == 0 {
					def.Effect = 4
				} else {
					def.Effect = 3
				}
				a.buffs.rows = append(a.buffs.rows, buffBadge{buff: proto.ActiveBuff{Skill: id, Instance: uint32(id)}, def: def, until: 60 + float64(len(a.buffs.rows))*30, duration: 300, timed: true})
			}
		}
	}
	a.chatOpen = true
	a.chatInput.Text = "Chat: original geometry, colour and font"
	a.state.Chat = append(a.state.Chat, []game.ChatLine{{Op: proto.ChatNormal, Sender: "Player", SenderID: 1, Text: "Hello world"}, {Op: proto.ChatParty, Sender: "Friend", SenderID: 2, Text: "Party chat in Phoenix colours"}, {Text: "System message"}}...)
	a.Draw(1024, 768)
	if err = render.SavePNG(filepath.Join(dir, "ui-polish-inventory-chat-buffs.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	a.inventory.open = false
	a.wv.mapView.full = true
	a.Draw(1024, 768)
	if err = render.SavePNG(filepath.Join(dir, "ui-polish-map.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	a.wv.mapView.full = false
	if err = a.Preview("world-status"); err != nil {
		t.Fatal(err)
	}
	a.state.Pools = proto.Pools{HP: 617, MP: 140, SP: 667}
	a.Draw(1024, 768)
	if err = render.SavePNG(filepath.Join(dir, "ui-polish-status.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
}
