//go:build integration

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestAuctionPanelsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires a current OpenGL runtime and installed data")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "GX auction test", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
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
	a.state = game.New()
	a.state.Details.Money = 151_852_352
	ends := time.Now().Add(5 * time.Hour).Unix()
	rows := []proto.AuctionRow{
		{ID: 1, Name: "Long Sword", Seller: "Merchant", Type: 1, TypeID: 1, Level: 1, Count: 1, Price: 55_000, Buyout: 1_200_000, Ends: ends, Active: true},
		{ID: 2, Name: "Rune Gaia Sword", Seller: "Merchant", Type: 1, TypeID: 100, Level: 56, Count: 1, Price: 5_500_000, Buyout: 12_000_000, Ends: ends + 40*3600, Active: true, Mine: true},
		{ID: 3, Name: "Apple", Seller: "Merchant", Type: 25, TypeID: 1, Count: 40, Price: 550, Buyout: 1200, Ends: ends - 4*3600, Active: true, HighBidder: true},
	}
	a.auction = auctionView{npc: 1, selected: 0, focus: -1, grade: 7, group: 1, sub: 2, rows: rows}
	a.auction.other = auctionList{selected: 0, rows: []proto.AuctionRow{
		{ID: 4, Name: "Gold settlement", Gold: 15380, Price: 16000, Reason: "Bought", Ends: ends},
		{ID: 5, Name: "Refund", Gold: 99000, Reason: "Bid refund", Ends: ends},
	}}
	out := os.Getenv("SHAIYA_CAPTURE_DIR")
	for view := proto.AuctionBrowse; view <= proto.AuctionWatch; view++ {
		a.auction.view = view
		for _, modal := range []string{"", "setup", "bid", "details"} {
			a.auction.modal = modal
			for _, viewport := range [][2]int{{404, 480}, {1024, 768}} {
				width, height := viewport[0], viewport[1]
				b := auctionDialogBox(modal, width, height)
				if modal == "setup" {
					a.auction.duration = 0
					a.updateAuctionDialog(&platform.Input{MouseX: b.x + 0xb7, MouseY: b.y + 0x8d, Pressed: true}, width, height)
					if !a.auction.durationOpen {
						t.Fatalf("%dx%d view %d could not open duration", width, height, view)
					}
				} else if modal == "" && view == proto.AuctionBrowse {
					a.updateAuctionSearchBar(&platform.Input{Pressed: true}, 0x1b8-90, 0x42+4)
				}
				gl.Viewport(0, 0, int32(width), int32(height))
				gl.ClearColor(.1, .1, .12, 1)
				gl.Clear(gl.COLOR_BUFFER_BIT)
				a.sprites.Begin(width, height)
				a.drawAuction(width, height)
				a.sprites.End()
				if code := gl.GetError(); code != gl.NO_ERROR {
					t.Fatalf("%dx%d view %d modal %q GL error %#x", width, height, view, modal, code)
				}
				if modal == "setup" {
					a.updateAuctionDialog(&platform.Input{MouseX: b.x + 0x70, MouseY: b.y + 0xa4 + 3*0xf + 2, Pressed: true}, width, height)
					if a.auction.duration != 3 || a.auction.durationOpen {
						t.Fatalf("%dx%d view %d could not select the longest duration", width, height, view)
					}
				}
				a.auction.dropdown = 0
				if out != "" && width == 1024 {
					_ = os.MkdirAll(out, 0o700)
					if err := render.SavePNG(filepath.Join(out, fmt.Sprintf("auction-v%d-%s.png", view, modal)), width, height); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}
