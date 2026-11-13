package image

import (
	"encoding/binary"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"os"
	"testing"
)

func TestCursorHotspotBottomUpAndMask(t *testing.T) {
	b := make([]byte, 22+40+16+8)
	put16 := func(i int, v uint16) { binary.LittleEndian.PutUint16(b[i:], v) }
	put32 := func(i int, v uint32) { binary.LittleEndian.PutUint32(b[i:], v) }
	put16(2, 2)
	put16(4, 1)
	b[6], b[7] = 2, 2
	put16(10, 1)
	put32(14, uint32(len(b)-22))
	put32(18, 22)
	put32(22, 40)
	put32(26, 2)
	put32(30, 4)
	put16(34, 1)
	put16(36, 24)
	// Bottom row blue, top row red; top-right is transparent in AND mask.
	b[62], b[65] = 255, 255
	b[72], b[75] = 255, 255
	b[82] = 0x40
	c, err := DecodeCUR(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.HotX != 1 || c.HotY != 0 || c.Image.W != 2 || c.Image.Pix[0] != 255 || c.Image.Pix[7] != 0 || c.Image.Pix[10] != 255 || c.Image.Pix[11] != 255 {
		t.Fatal("cursor orientation/mask/hotspot", c, c.Image.Pix)
	}
	for _, bad := range [][]byte{nil, b[:21], b[:len(b)-1]} {
		if _, err := DecodeCUR(bad); err == nil {
			t.Fatal("truncated cursor accepted")
		}
	}
	put16(10, 2)
	if _, err := DecodeCUR(b); err == nil {
		t.Fatal("hotspot outside cursor accepted")
	}
}

func TestConvertedPNGCursors(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original assets")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	for i := 1; i <= 7; i++ {
		b, err := arc.ReadFile(fmt.Sprintf("Cursor/%02d.png", i))
		if err != nil {
			t.Fatal(err)
		}
		c, err := DecodePNG(b)
		if err != nil {
			t.Fatal(i, err)
		}
		opaque, clear := 0, 0
		for j := 3; j < len(c.RGBA.Pix); j += 4 {
			if c.RGBA.Pix[j] > 0 {
				opaque++
			} else {
				clear++
			}
		}
		if opaque == 0 || clear == 0 {
			t.Fatal("lost cursor artwork/transparency", i)
		}
		t.Logf("cursor %d: %dx%d hotspot %d,%d", i, c.RGBA.W, c.RGBA.H, 0, 0)
	}
}
