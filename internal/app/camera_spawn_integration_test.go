//go:build integration

package app

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestCameraSpawnOnRealMaps(t *testing.T) {
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
	for index, tc := range []struct {
		mapID uint16
		pos   render.V3
	}{{19, render.V3{641.0693, 3.8071747, 1772.789}}, {1, render.V3{803.7864, .87561035, 1434.4695}}, {1, render.V3{578, 78, 1762}}} {
		a, err := New(Config{}, arc)
		if err != nil {
			t.Fatal(err)
		}
		a.state = game.New()
		a.state.Phase = game.PhaseInWorld
		a.state.MapID = tc.mapID
		a.state.SelfID = 1
		a.state.Self.X, a.state.Self.Y, a.state.Self.Z = tc.pos[0], tc.pos[1], tc.pos[2]
		a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 0, Class: 5, Gender: 0, Height: 2}
		a.screen = screenWorld
		a.enterWorld()
		v := a.wv
		// Native reset + draw instructions, expressed relative to model feet:
		// eye=(0,2.4000001,-5), target=(0,1.3999999,0).
		if v.eye.Sub(tc.pos.Add(render.V3{0, 2.4000001, -5})).Len() > .0001 || v.target().Sub(tc.pos.Add(render.V3{0, 1.3999999, 0})).Len() > .0001 {
			t.Fatal("initial framing differs from native model-relative camera", v.eye, v.target())
		}
		t.Logf("map=%d pos=%v floor=%v initial=%v target=%v", tc.mapID, v.pos, v.world.Height(v.pos[0], v.pos[2]), v.eye, v.target())
		for frame := 0; frame < 120; frame++ {
			v.constrainCamera()
			v.updateRenderedCamera(1.0 / 60)
			gl.Viewport(0, 0, 1024, 768)
			gl.ClearDepth(1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			a.drawWorldView(1024, 768)
			if frame == 0 || frame == 119 {
				hit, ok := v.world.CameraIntersection(v.cameraOrigin(), v.eye)
				t.Logf("frame=%d desired=%v rendered=%v target=%v hit=%v at=%v", frame, v.eye, v.cameraEye, v.cameraTarget, ok, hit)
				if !ok && v.cameraEye.Sub(tc.pos.Add(render.V3{0, 2.4000001, -5})).Len() > .0001 || v.cameraTarget.Sub(tc.pos.Add(render.V3{0, 1.3999999, 0})).Len() > .0001 {
					t.Fatal("camera framing changed while idle", frame, v.cameraEye, v.cameraTarget)
				}
			}
		}
		if out := os.Getenv("SHAIYA_TEST_ARTIFACTS"); out != "" {
			if err := render.SavePNG(fmt.Sprintf("%s/camera-spawn-%d-map%d.png", out, index, tc.mapID), 1024, 768); err != nil {
				t.Fatal(err)
			}
		}
		a.Close()
	}
}
