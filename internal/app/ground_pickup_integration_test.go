//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestGroundLabelPickupAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PICKUP") != "1" {
		t.Skip("requires disposable item and GL fixture")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "ground pickup", Width: 256, Height: 256, Hidden: true, Debug: true})
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
	st, w, send, wait := testworld.Enter(t)
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world = st, w
	a.wv = &worldView{pos: render.V3{st.Self.X, st.Self.Y, st.Self.Z}}
	var key [2]uint8
	var item proto.InventoryItem
	for k, it := range st.Inventory {
		if k[0] >= 1 && k[0] <= 5 && it.Type == 25 && it.TypeID == 4 && it.Count >= 5 {
			key, item = k, it
			break
		}
	}
	if item.Count < 5 {
		t.Fatal("disposable stack missing")
	}
	send(proto.PickupPacket(0xfffffffe))
	deadline := time.Now().Add(10 * time.Second)
	for {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("missing-item rejection did not arrive")
		}
		found := false
		for _, line := range st.Chat {
			found = found || line.Text == "That item is no longer available."
		}
		if found {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if st.Inventory[key].Count != item.Count {
		t.Fatal("rejected pickup changed inventory")
	}
	before := map[uint32]bool{}
	for id := range st.GroundItems {
		before[id] = true
	}
	drop := proto.NewWriter(proto.RemoveItem)
	drop.U8(key[0])
	drop.U8(key[1])
	drop.U8(1)
	send(drop.Bytes())
	var id uint32
	wait(func() bool {
		for k, it := range st.GroundItems {
			if !before[k] && it.Type == 25 && it.TypeID == 4 {
				id = k
				return true
			}
		}
		return false
	})
	if st.Inventory[key].Count != item.Count-1 {
		t.Fatal("drop was not confirmed")
	}
	ground := st.GroundItems[id]
	pos := groundItemLabelPosition(nil, ground)
	a.scene3d.View = render.LookAtLH(pos.Add(render.V3{0, 0, -5}), pos, render.V3{0, 1, 0})
	a.scene3d.Proj = render.PerspectiveFovLH(0.7853981634, 1, .1, 100)
	gl.Viewport(0, 0, 256, 256)
	gl.ClearDepth(1)
	gl.DepthMask(true)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(256, 256)
	a.drawPlayerNames(256, 256)
	a.sprites.End()
	x, y, _, ok := projectLabel(pos, a.scene3d.View.Mul(a.scene3d.Proj), 256, 256)
	if !ok {
		t.Fatal("drop label not projected")
	}
	click := &platform.Input{Clicked: true, MouseX: x, MouseY: y - a.font.Height/2}
	if got := a.pickGroundLabel(click, 256, 256); got != id {
		t.Fatalf("clicked drop %d, got %d", id, got)
	}
	a.updateCombat(click, 256, 256)
	if _, ok := st.GroundItems[id]; !ok {
		t.Fatal("item removed before confirmation")
	}
	deadline = time.Now().Add(10 * time.Second)
	for {
		a.pumpWorld()
		if a.state == nil {
			t.Fatal("disconnected")
		}
		if _, ok := st.GroundItems[id]; !ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("pickup confirmation timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if st.Inventory[key].Count != item.Count {
		t.Fatalf("pickup did not restore stack: %d", st.Inventory[key].Count)
	}
	// Drop another item and pick up the rendered mesh, outside its label.
	send(drop.Bytes())
	var modelID uint32
	wait(func() bool {
		for k, it := range st.GroundItems {
			if !before[k] && it.Type == 25 && it.TypeID == 4 {
				modelID = k
				return true
			}
		}
		return false
	})
	ground = st.GroundItems[modelID]
	base := groundItemPosition(nil, ground)
	a.scene3d.View = render.LookAtLH(base.Add(render.V3{0, 1, -3}), base.Add(render.V3{0, .15, 0}), render.V3{0, 1, 0})
	gl.DepthMask(true)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.scene3d.Begin()
	a.drawGroundItems()
	a.scene3d.End()
	a.sprites.Begin(256, 256)
	a.drawPlayerNames(256, 256)
	a.sprites.End()
	click = nil
	for y := 75; y < 200 && click == nil; y++ {
		for x := 75; x < 180; x++ {
			c := &platform.Input{Clicked: true, MouseX: x, MouseY: y}
			if a.pickGroundModel(c, 256, 256) == modelID && a.pickGroundLabel(c, 256, 256) == 0 {
				click = c
				break
			}
		}
	}
	if click == nil {
		t.Fatal("could not click dropped item mesh outside its label")
	}
	a.updateCombat(click, 256, 256)
	if _, ok := st.GroundItems[modelID]; !ok {
		t.Fatal("model pickup removed item before confirmation")
	}
	deadline = time.Now().Add(10 * time.Second)
	for {
		a.pumpWorld()
		if a.state == nil {
			t.Fatal("disconnected after model pickup")
		}
		if _, ok := st.GroundItems[modelID]; !ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("model pickup confirmation timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if st.Inventory[key].Count != item.Count {
		t.Fatalf("model pickup did not restore stack: %d", st.Inventory[key].Count)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("pickup GL error %#x", code)
	}
}
