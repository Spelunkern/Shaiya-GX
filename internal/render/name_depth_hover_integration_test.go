//go:build integration

package render_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

// The cursor probe must answer from an asynchronous sample: never before the
// GPU delivered it, and the same as the synchronous click test once it has.
func TestNameVisibleAtHoverIsAsynchronousAndMatchesClicks(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w, err := platform.Open(platform.Config{Title: "hover depth probe", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	s, err := render.NewSprites(16)
	if err != nil {
		t.Fatal(err)
	}
	gl.Viewport(0, 0, 64, 64)
	gl.ClearDepth(.25)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	s.Begin(64, 64)
	s.CaptureNameDepth(64, 64)
	if s.NameVisibleAtHover(44, 12, 64, 64, .125) {
		t.Fatal("hover answered before any sample completed")
	}
	visible := false
	for i := 0; i < 100 && !visible; i++ {
		gl.Finish()
		visible = s.NameVisibleAtHover(44, 12, 64, 64, .125)
	}
	if !visible || s.NameVisibleAtHover(44, 12, 64, 64, .75) != s.NameVisibleAt(44, 12, 64, 64, .75) {
		t.Fatal("hover sample disagrees with the click test")
	}
	if s.NameVisibleAtHover(10, 10, 64, 64, .125) {
		t.Fatal("hover reused a sample from another pixel")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
