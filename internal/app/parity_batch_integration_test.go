//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestOriginalNPCPotionsAndMapLabelsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	win, err := platform.Open(platform.Config{Width: 1280, Height: 800, Hidden: true, Debug: true})
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
	if err = a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 500 && !a.SceneReady(); i++ {
		a.Draw(1280, 800)
	}
	if !a.SceneReady() {
		t.Fatal("world did not finish loading")
	}
	a.loadNPCs()
	a.ensureQuickPotions()
	a.quickPotions.bindings = [2]sdata.ItemKey{{Type: 25, TypeID: 4}, {Type: 25, TypeID: 5}}
	a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Type: 25, TypeID: 4, Count: 25}
	a.state.Inventory[[2]uint8{1, 1}] = proto.InventoryItem{Type: 25, TypeID: 5, Count: 12}
	if a.assets.Texture("interface/QuickPotion.png") == nil {
		t.Fatal("original potion frame missing")
	}
	out := os.Getenv("SHAIYA_CAPTURE_DIR")
	if out == "" {
		out = t.TempDir()
	}
	if err = os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []uint8{1, 2, 3, 4, 6, 11} {
		def, ok := a.npcDefs[sdata.NPCKey{Type: kind, ID: 1}]
		if !ok {
			t.Fatal("missing NPC fixture", kind)
		}
		if a.npcPortrait(def) == nil {
			t.Fatal("missing portrait", kind, def.Model)
		}
		a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: kind, TypeID: 1, X: a.wv.pos[0] + 3, Y: a.wv.pos[1], Z: a.wv.pos[2]}
		a.shop.npcID = 42
		a.Draw(1280, 800)
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("NPC %d GL error %x", kind, code)
		}
		if err = render.SavePNG(filepath.Join(out, []string{"", "merchant", "gatekeeper", "blacksmith", "dialogue", "", "warehouse", "", "", "", "", "guild"}[kind]+".png"), 1280, 800); err != nil {
			t.Fatal(err)
		}
	}
	a.shop.npcID = 0
	for _, area := range a.wv.world.WLD.NamedAreas {
		if area.Mode != 2 {
			continue
		}
		lo, hi := render.V3(area.Box.Min), render.V3(area.Box.Max)
		a.wv.pos = lo.Add(hi).Scale(.5)
		a.wv.pos[1] = a.wv.world.Height(a.wv.pos[0], a.wv.pos[2])
		a.wv.resetCamera()
		a.clock = 20
		a.mapLabels = mapLabelView{}
		a.Draw(1280, 800)
		a.clock = 21
		a.Draw(1280, 800)
		if a.mapLabels.banner == "" {
			t.Fatal("entering authored region did not show its banner")
		}
		if err = render.SavePNG(filepath.Join(out, "zone-banner-clock-potions.png"), 1280, 800); err != nil {
			t.Fatal(err)
		}
		break
	}

	// Mode zero is an authored ray-hover region with a localized TXT ID.
	for _, area := range a.wv.world.WLD.NamedAreas {
		if area.Mode != 0 {
			continue
		}
		center := render.V3(area.Box.Min).Add(render.V3(area.Box.Max)).Scale(.5)
		a.scene3d.View = render.LookAtLH(center.Add(render.V3{0, 0, -5}), center, render.V3{0, 1, 0})
		a.scene3d.Proj = render.PerspectiveFovLH(.7853981634, 1, .1, 100)
		a.lastMouseX, a.lastMouseY = 256, 256
		a.mapLabels.banner = ""
		a.mapLabels.nextCheck = a.clock + 1
		gl.Viewport(0, 0, 512, 512)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		a.sprites.Begin(512, 512)
		a.drawMapLabels(512, 512)
		a.sprites.End()
		pixels := make([]byte, 512*512*4)
		gl.ReadPixels(0, 0, 512, 512, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
		lit := 0
		for i := 0; i < len(pixels); i += 4 {
			if pixels[i] > 100 && pixels[i+1] > 100 && pixels[i+2] > 100 {
				lit++
			}
		}
		if lit < 20 {
			t.Fatal("hover label was not rendered", lit)
		}
		if err = render.SavePNG(filepath.Join(out, "hover-label.png"), 512, 512); err != nil {
			t.Fatal(err)
		}
		break
	}
	// A ground NPC must use interpolated terrain height rather than stale patrol Y.
	e := a.state.Entities[42]
	e.NpcType, e.TypeID = 1, 1
	e.X, e.Z = a.wv.pos[0], a.wv.pos[2]
	e.Y = 500
	a.prepareNPCs()
	if got := a.wv.npcs[42].position.pos[1]; got != 500 {
		t.Fatal("stationary NPC lost authoritative height", got)
	}
	e.X += 3
	e.MoveSerial++
	a.prepareNPCs()
	a.clock += .25
	a.prepareNPCs()
	pos := a.wv.npcs[42].position.pos
	if pos[1] != a.wv.world.Height(pos[0], pos[2]) {
		t.Fatal("patrol did not follow ground", pos)
	}
	if x, y := ui.QuickPotionOrigin(a.quickPotions.Position, 1280, 800); x < 0 || y < 0 {
		t.Fatal("potion anchor outside viewport")
	}
}

func TestClickApproachAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("GX_PARITY_ONLINE") != "1" {
		t.Skip("requires isolated GX fixture")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	win, err := platform.Open(platform.Config{Width: 512, Height: 512, Hidden: true})
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
	st, conn, send, pump := testworld.Enter(t)
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, conn, screenWorld
	a.enterWorld()
	for i := 0; !a.SceneReady(); i++ {
		if i > 500 {
			t.Fatal("warmup")
		}
		a.Draw(512, 512)
	}
	var npc *game.Entity
	pump(func() bool {
		best := float32(1e30)
		for _, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 1 {
				delta := render.V3{e.X, 0, e.Z}.Sub(render.V3{a.wv.pos[0], 0, a.wv.pos[2]})
				if d := delta.Len(); d < best {
					npc, best = e, d
				}
			}
		}
		return npc != nil
	})
	v := a.wv
	v.place(render.V3{npc.X + 8, v.world.Height(npc.X+8, npc.Z), npc.Z})
	send(v.movePacket(0))
	a.openNPC(npc.ID)
	if !v.approachNPC || !v.travel || a.shop.npcID != 0 {
		t.Fatal("merchant opened without approaching")
	}
	for i := 0; i < 600 && v.approachTarget != 0; i++ {
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	if a.shop.npcID != npc.ID || (render.V3{v.pos[0] - npc.X, 0, v.pos[2] - npc.Z}).Len() > 2.01 {
		t.Fatal("merchant approach failed", v.pos, npc.X, npc.Z)
	}
	send(proto.NpcTalkPacket(0))
	a.shop.npcID = 0
	key := [2]uint8{1, 23}
	if st.Inventory[key].Count != 5 {
		t.Fatal("disposable potion stack absent")
	}
	drop := proto.NewWriter(proto.RemoveItem)
	drop.U8(1)
	drop.U8(23)
	drop.U8(1)
	send(drop.Bytes())
	var loot uint32
	pump(func() bool {
		for id, it := range st.GroundItems {
			if it.Type == 25 && it.TypeID == 4 {
				loot = id
				return true
			}
		}
		return false
	})
	item := st.GroundItems[loot]
	v.place(render.V3{item.X + 15, v.world.Height(item.X+15, item.Z), item.Z})
	send(v.movePacket(0))

	base := groundItemPosition(v.world, item)
	a.scene3d.View = render.LookAtLH(base.Add(render.V3{0, 1, -3}), base.Add(render.V3{0, .15, 0}), render.V3{0, 1, 0})
	a.scene3d.Proj = render.PerspectiveFovLH(.7853981634, 1, .1, 100)
	a.scene3d.SetReverseDepth(false)
	gl.Viewport(0, 0, 512, 512)
	gl.ClearDepth(1)
	gl.DepthMask(true)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.scene3d.Begin()
	a.drawGroundItems()
	a.scene3d.End()
	a.sprites.Begin(512, 512)
	a.drawPlayerNames(512, 512)
	a.sprites.End()
	var click *platform.Input
	for y := 150; y < 370 && click == nil; y += 2 {
		for x := 150; x < 370; x += 2 {
			c := &platform.Input{Clicked: true, MouseX: x, MouseY: y}
			if a.pickGroundModel(c, 512, 512) == loot {
				click = c
				break
			}
		}
	}
	if click == nil {
		t.Fatal("real dropped mesh was not clickable")
	}
	a.updateCombat(click, 512, 512)
	a.updateWorld(click, 1.0/60)
	if v.approachTarget != loot || !v.travel {
		t.Fatal("issuing click canceled pickup approach")
	}
	for i := 0; i < 600 && v.approachTarget != 0; i++ {
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	if v.approachTarget != 0 {
		t.Fatal("pickup never arrived")
	}
	pump(func() bool { _, present := st.GroundItems[loot]; return !present && st.Inventory[key].Count == 5 })

	send(proto.ChatPacket(proto.ChannelNormal, false, "Chat común 😀"))
	deadline := time.Now().Add(5 * time.Second)
	for len(a.chatBubbles) == 0 {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("ordinary server echo did not create speech bubble")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if a.chatBubbles[0].text != "Chat común 😀" || a.chatBubbles[0].speaker != st.SelfID {
		t.Fatal("chat echo lost UTF-8 or speaker", a.chatBubbles)
	}
	t.Log("clicked loot pickup and NPC approach confirmed; ordinary UTF-8 server echo created speech bubble")
}
