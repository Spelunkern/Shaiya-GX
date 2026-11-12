//go:build integration

package platform

import (
	"math"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/go-gl/glfw/v3.4/glfw"
)

func TestMouseLookCaptureAndFocus(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_INPUT") != "1" {
		t.Skip("requires isolated desktop for pointer capture")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := Open(Config{Title: "mouse capture probe", Width: 640, Height: 480})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	win.Focus()
	waitFocus := func(focused bool) {
		t.Helper()
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			glfw.PollEvents()
			if (win.GetAttrib(glfw.Focused) != 0) == focused {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("focus transition timed out")
	}
	waitFocus(true)
	if win.GetAttrib(glfw.Focused) == 0 {
		t.Fatal("test window did not receive focus")
	}
	q := win.InstallInput()
	win.SetCursorPos(200, 150)
	q.Frame()
	q.SetMouseLook(true)
	if !q.mouselook {
		t.Fatal("mouse capture failed")
	}
	x, y := win.GetCursorPos()
	win.SetCursorPos(x+100, y-20)
	q.cur.RightDown = true
	in := q.Frame()
	ww, _ := win.GetSize()
	fw, _ := win.GetFramebufferSize()
	scale := float64(fw) / float64(ww)
	if !in.MouseLook || in.MouseX != int(200*scale) || in.MouseY != int(150*scale) || math.Abs(in.MouseDX-100*scale) > .001 || math.Abs(in.MouseDY+20*scale) > .001 {
		t.Fatal("captured movement changed UI coordinates or lost displacement", in)
	}
	in = q.Frame()
	if in.MouseDX != 0 || in.MouseDY != 0 {
		t.Fatal("stationary cursor repeated movement", in)
	}
	q.SetMouseLook(false)
	x, y = win.GetCursorPos()
	if math.Abs(x-200) > .001 || math.Abs(y-150) > .001 || win.GetInputMode(glfw.CursorMode) != glfw.CursorNormal {
		t.Fatal("release did not restore cursor", x, y)
	}
	q.SetMouseLook(true)
	other, err := glfw.CreateWindow(100, 100, "focus probe", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Destroy()
	other.Focus()
	waitFocus(false)
	q.cur.RightDown = true
	in = q.Frame()
	if in.RightDown || in.MouseDX != 0 || in.MouseDY != 0 {
		t.Fatal("unfocused window retained camera input", in)
	}
	q.SetMouseLook(true)
	if q.mouselook || win.GetInputMode(glfw.CursorMode) != glfw.CursorNormal {
		t.Fatal("focus loss retained pointer capture")
	}
}
