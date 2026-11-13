//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNativeControlsAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_NATIVE_CONTROLS") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and two disposable characters")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "native controls probe", Width: 1024, Height: 734, Hidden: true})
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
	st, wa, _, waitA := testworld.Enter(t)
	other, wb, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitA(func() bool { return st.Entities[other.SelfID] != nil })
	waitB(func() bool { e := other.Entities[st.SelfID]; return e != nil && e.Shape != nil })
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, wa, screenWorld
	a.enterWorld()
	b, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	b.state, b.world, b.screen = other, wb, screenWorld
	b.enterWorld()
	pump := func(condition func() bool) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for !condition() {
			a.pumpWorld()
			b.pumpWorld()
			if a.state == nil || b.state == nil || time.Now().After(deadline) {
				t.Fatal("native controls confirmation timeout")
			}
			time.Sleep(time.Millisecond)
		}
	}
	verifyNativeChat(t, a, b, pump)
	verifyNativeWeaponEnchants(t, a, b, pump)
	verifyNativeSocial(t, a, b, pump)
	v := a.wv
	// Land on the map's real terrain before measuring the jump.
	v.step(render.V3{})
	floor := v.pos[1]
	a.updateWorld(&platform.Input{Space: true, Keys: []platform.Key{platform.KeySpace}}, 1.0/60)
	if !v.jumping || v.pos[1] <= floor || !v.self.PlayingAction(8) {
		t.Fatal("local jump did not take off", v.pos, floor)
	}
	pump(func() bool {
		return other.Entities[st.SelfID].Gesture == proto.MotionJump && other.Entities[st.SelfID].Motion == 4|proto.MotionFlag80
	})
	b.prepareRemotePlayers()
	if actor := b.wv.actors[st.SelfID]; actor == nil || !actor.character.PlayingAction(8) {
		t.Fatal("observer did not animate the jump")
	}
	maxY := v.pos[1]
	for range 180 {
		a.updateWorld(&platform.Input{Space: true}, 1.0/60)
		maxY = max(maxY, v.pos[1])
	}
	if v.jumping || math.Abs(float64(v.pos[1]-floor)) > .001 || maxY <= floor+.5 {
		t.Fatal("jump failed to land or repeated while held", floor, maxY, v.pos)
	}
	pump(func() bool {
		return other.Entities[st.SelfID].Motion == 0 && math.Abs(float64(other.Entities[st.SelfID].Y-floor)) < .001
	})
	if st.Self.Sitting || other.Entities[st.SelfID].Sitting {
		t.Fatal("jump changed posture")
	}
	// Walk/run, both strafes, backwards and autorun use normal movement snapshots.
	a.updateWorld(&platform.Input{Keys: []platform.Key{platform.KeyTab}}, .01)
	if !v.walking {
		t.Fatal("Tab did not enable walking")
	}
	a.updateWorld(&platform.Input{Down: "W"}, .1)
	pump(func() bool { return other.Entities[st.SelfID].Motion == 0 })
	a.updateWorld(&platform.Input{Down: "Q"}, .1)
	pump(func() bool { return other.Entities[st.SelfID].Motion == 3 })
	b.prepareRemotePlayers()
	if clip := remoteLocomotion(b.wv.world, other.Entities[st.SelfID]); clip != "004_lstep" {
		t.Fatal("left strafe clip", clip)
	}
	a.updateWorld(&platform.Input{Down: "E"}, .1)
	pump(func() bool { return other.Entities[st.SelfID].Motion == 4 })
	a.updateWorld(&platform.Input{Down: "S"}, .1)
	pump(func() bool { return other.Entities[st.SelfID].Motion == 2 })
	a.updateWorld(&platform.Input{Released: "W"}, .1)
	a.updateWorld(&platform.Input{Released: "W"}, .1)
	if !v.autoRun {
		t.Fatal("double W did not enable autorun")
	}
	before := v.pos
	a.updateWorld(&platform.Input{}, .1)
	if v.pos.Sub(before).Len() == 0 {
		t.Fatal("autorun did not travel")
	}
	a.updateWorld(&platform.Input{Down: "S"}, .1)
	if v.autoRun {
		t.Fatal("S did not cancel autorun")
	}
	a.updateWorld(&platform.Input{}, .01)
	// First person and reset do not send gameplay actions or change posture.
	a.updateWorld(&platform.Input{Keys: []platform.Key{platform.KeyF10}}, .01)
	if !v.firstPerson {
		t.Fatal("F10 did not enable first person")
	}
	a.drawWorldView(1024, 734)
	a.updateWorld(&platform.Input{Keys: []platform.Key{platform.KeyF9}}, .01)
	if v.firstPerson {
		t.Fatal("F9 did not reset the camera")
	}
	// Project a nearby point onto the real map and exercise ground picking,
	// travel completion and keyboard cancellation with the same controller.
	a.drawWorldView(1024, 734)
	goal := v.pos.Add(v.facing.Scale(3))
	goal[1] = v.world.Height(goal[0], goal[2])
	x, y, _, visible := projectLabel(goal, a.scene3d.View.Mul(a.scene3d.Proj), 1024, 734)
	if !visible {
		t.Fatal("ground travel fixture is outside the camera")
	}
	a.moveToGround(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 734)
	if !v.travel || v.destination.Sub(goal).Len() > .1 {
		t.Fatal("terrain click did not resolve its destination", goal, v.destination)
	}
	if v.travelEffect == nil {
		t.Fatal("point.EFT was not attached to the destination")
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		a.updateTravelEffect(.2)
		a.Draw(1024, 734)
		if err := render.SavePNG(filepath.Join(dir, "point-marker-sqlite.png"), 1024, 734); err != nil {
			t.Fatal(err)
		}
	}
	for range 120 {
		a.updateTravelEffect(.05)
	}
	if v.travelEffect.Done() {
		t.Fatal("point.EFT stopped before reaching the destination")
	}
	for range 240 {
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	if v.travel || v.pos.Sub(goal).Len() > .1 {
		t.Fatal("ground travel did not finish", goal, v.pos)
	}
	a.updateTravelEffect(.016)
	if v.travelEffect != nil {
		t.Fatal("point.EFT remained after arrival")
	}
	v.travel, v.destination = true, goal.Add(v.facing.Scale(3))
	a.updateWorld(&platform.Input{Down: "S"}, .01)
	if v.travel {
		t.Fatal("keyboard did not cancel ground travel")
	}
	a.updateWorld(&platform.Input{}, .01)
	a.updateWorld(&platform.Input{Released: "S"}, .1)
	a.updateWorld(&platform.Input{Released: "S"}, .1)
	if !v.evading || !v.self.PlayingAction(13) {
		t.Fatal("double S did not start the backward evasion")
	}
	pump(func() bool { return other.Entities[st.SelfID].Gesture == proto.MotionBackstep })
	b.prepareRemotePlayers()
	if !b.wv.actors[st.SelfID].character.PlayingAction(13) {
		t.Fatal("observer did not animate backward evasion")
	}
	for range 180 {
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	if v.evading {
		t.Fatal("backward evasion did not finish")
	}
	for _, tc := range []struct {
		key    string
		action int
		motion uint8
	}{{"Q", 14, proto.MotionEvadeLeft}, {"E", 15, proto.MotionEvadeRight}} {
		a.updateWorld(&platform.Input{Released: tc.key}, .1)
		a.updateWorld(&platform.Input{Released: tc.key}, .1)
		if !v.evading || !v.self.PlayingAction(tc.action) {
			t.Fatal("lateral evasion did not start", tc.key)
		}
		pump(func() bool { return other.Entities[st.SelfID].Gesture == tc.motion })
		b.prepareRemotePlayers()
		if !b.wv.actors[st.SelfID].character.PlayingAction(tc.action) {
			t.Fatal("observer missed lateral evasion", tc.key)
		}
		if st.Self.Sitting || other.Entities[st.SelfID].Sitting {
			t.Fatal("lateral evasion changed posture")
		}
		for range 180 {
			a.updateWorld(&platform.Input{}, 1.0/60)
		}
	}
	// Phoenix controls must preserve logical heading on a diagonal, steer from
	// right drag and keep travel stable at different rendering frame rates.
	v.walking, v.autoRun, v.travel = false, false, false
	start := v.pos
	type movementProbe struct {
		name  string
		input platform.Input
		want  render.V3
	}
	for _, tc := range []movementProbe{
		{"diagonal", platform.Input{Down: "WE"}, render.V3{1, 0, 1}.Norm()},
		{"right mouse strafe", platform.Input{Down: "D", RightDown: true, MouseLook: true}, render.V3{1, 0, 0}},
		{"mouse chord", platform.Input{MouseDown: true, RightDown: true, MouseLook: true}, render.V3{0, 0, 1}},
	} {
		v.place(start)
		v.facing = render.V3{0, 0, 1}
		v.resetCamera()
		for range 12 {
			a.updateWorld(&tc.input, 1.0/60)
		}
		delta := v.pos.Sub(start)
		delta[1] = 0
		if delta.Len() < .01 || delta.Norm().Sub(tc.want).Len() > .001 || v.facing.Sub(render.V3{0, 0, 1}).Len() > .001 {
			t.Fatal("inconsistent control", tc.name, delta, v.facing)
		}
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	var endpoints []render.V3
	for _, fps := range []int{30, 60, 144} {
		v.place(start)
		v.facing = render.V3{0, 0, 1}
		v.resetCamera()
		for i := 0; i < fps; i++ {
			a.updateWorld(&platform.Input{Down: "WE"}, 1.0/float64(fps))
		}
		endpoints = append(endpoints, v.pos)
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	for _, end := range endpoints[1:] {
		if end.Sub(endpoints[0]).Len() > .005 {
			t.Fatal("movement depends on rendering FPS", endpoints)
		}
	}
	v.place(start)
	if os.Getenv("SHAIYA_TEST_ACTIONS") == "1" {
		verifyNativeCloakAndHealing(t, a, b, pump)
		verifyInstantSkillTiming(t, a, pump)
		verifyNativeApproach(t, a, pump)
		verifyPotionReuse(t, a, pump)
	}
	a.chatOpen = true
	a.chatUI.Channel = proto.ChannelNormal
	a.chatInput = ui.TextField{Text: "GX UTF-8: 你好 مرحبا नमस्ते 👨‍👩‍👧‍👦 🇦🇷", Max: 120}
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	pump(func() bool {
		for _, line := range other.Chat {
			if line.Text == "GX UTF-8: 你好 مرحبا नमस्ते 👨‍👩‍👧‍👦 🇦🇷" {
				return true
			}
		}
		return false
	})
	a.updateWorld(&platform.Input{Chars: []rune{'c'}}, .01)
	pump(func() bool { return st.Self.Sitting && !v.sitPending })
	a.updateWorld(&platform.Input{Chars: []rune{'c'}}, .01)
	pump(func() bool { return !st.Self.Sitting && !v.sitPending })
	// The preceding posture test acknowledges stand-up before its animation ends.
	for range 120 {
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	a.openGameMenu()
	beforeOptions := v.pos
	a.updateCharacters(&platform.Input{Down: "W"}, 1024, 734, .1)
	if v.pos.Sub(beforeOptions).Len() < .01 || !a.menuOpen {
		t.Fatal("options blocked character controls")
	}
	a.updateCharacters(&platform.Input{}, 1024, 734, .01)
	v.autoRun = true
	beforeOptions = v.pos
	a.updateCharacters(&platform.Input{}, 1024, 734, .1)
	if v.pos.Sub(beforeOptions).Len() < .01 || !v.autoRun {
		t.Fatal("options blocked automatic movement")
	}
	v.autoRun = false
	a.updateCharacters(&platform.Input{}, 1024, 734, .01)
	// Mode selection is pending until OK; dropdown navigation must not open chat.
	a.gameMenu.SetDisplayModes([]platform.DisplayMode{{Width: 800, Height: 600}, {Width: 1024, Height: 768}})
	a.gameMenu.Open(ui.MenuGraphics)
	mx, my := a.gameMenu.Origin(1024, 734)
	for _, in := range []platform.Input{{MouseX: mx + 145, MouseY: my + 90, Pressed: true, MouseDown: true}, {MouseX: mx + 145, MouseY: my + 90, Clicked: true}, {Keys: []platform.Key{platform.KeyDown}}, {Keys: []platform.Key{platform.KeyEnter}}} {
		a.updateCharacters(&in, 1024, 734, .01)
	}
	if a.chatOpen || a.gameMenu.DisplayPopupOpen() || a.gameMenu.Settings.DisplayMode != (platform.DisplayMode{Width: 800, Height: 600}) {
		t.Fatal("resolution dropdown leaked keyboard input")
	}
	for _, in := range []platform.Input{{MouseX: mx + 230, MouseY: my + 500, Pressed: true, MouseDown: true}, {MouseX: mx + 230, MouseY: my + 500, Clicked: true}} {
		a.updateCharacters(&in, 1024, 734, .01)
	}
	if a.gameMenu.Page != ui.MenuMain || a.gameMenu.Settings.DisplayMode != (platform.DisplayMode{}) {
		t.Fatal("display selection did not cancel")
	}
	x, y = a.gameMenu.Origin(1024, 734)
	a.updateGameMenu(&platform.Input{MouseX: x + 70, MouseY: y + 80 + 6*32, Pressed: true, MouseDown: true}, 1024, 734)
	a.updateGameMenu(&platform.Input{MouseX: x + 70, MouseY: y + 80 + 6*32, Clicked: true}, 1024, 734)
	if !a.menuConfirm.Visible || a.menuIntent != ui.MenuCharacters || a.menuPending {
		t.Fatal("character screen action bypassed confirmation")
	}
	a.updateGameMenu(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 734)
	if !a.menuPending {
		t.Fatal("logout did not wait for server acknowledgement")
	}
	pump(func() bool { return !a.menuPending && !a.menuOpen && a.wv == nil })
	t.Logf("jump %.3f above ground, landing and remote animation confirmed; walk, strafe, backward, autorun and first person passed", maxY-floor)
	t.Log("Phoenix mouse chord/right strafe/straight diagonals, 30/60/144 FPS and menu logout acknowledgement passed")
}
