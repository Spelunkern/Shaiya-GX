//go:build integration

package render

import (
	"os"
	"runtime"
	"testing"
	"unsafe"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestOpaqueSnapshotIgnoresMaterialAlphaGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 32, Height: 32, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := Init(true); err != nil {
		t.Fatal(err)
	}
	gl.ColorMask(true, false, true, true)
	gl.ClearDepth(0)
	snapshot, err := CaptureTexture(32, 32, func() {
		// An opaque material's RGB is visible in an inventory viewport even
		// when its authored texture alpha is zero. The snapshot must be opaque.
		gl.ClearColor(.25, .75, .5, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Delete()
	pixels := make([]byte, 32*32*4)
	gl.GetTextureImage(snapshot.ID, 0, gl.RGBA, gl.UNSIGNED_BYTE, int32(len(pixels)), unsafe.Pointer(&pixels[0]))
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i] < 63 || pixels[i] > 65 || pixels[i+1] < 190 || pixels[i+1] > 192 || pixels[i+2] < 127 || pixels[i+2] > 129 || pixels[i+3] != 255 {
			t.Fatal("material alpha leaked into opaque snapshot", pixels[i:i+4])
		}
	}
	var mask [4]bool
	var depth float64
	gl.GetBooleanv(gl.COLOR_WRITEMASK, &mask[0])
	gl.GetDoublev(gl.DEPTH_CLEAR_VALUE, &depth)
	if mask != [4]bool{true, false, true, true} || depth != 0 {
		t.Fatal("snapshot changed enclosing color/depth state", mask, depth)
	}
}
