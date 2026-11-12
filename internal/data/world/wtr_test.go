package world

import (
	"encoding/binary"
	"math"
	"path"
	"strings"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
)

func TestWTRReadsNativeScaleARGBAndFrameCadence(t *testing.T) {
	b := make([]byte, 16+2*256)
	binary.LittleEndian.PutUint32(b, math.Float32bits(64))
	binary.LittleEndian.PutUint32(b[4:], 0x98ffffff)
	binary.LittleEndian.PutUint32(b[8:], 40)
	binary.LittleEndian.PutUint32(b[12:], 2)
	copy(b[16:], "caust00.jpg")
	copy(b[272:], "caust01.jpg")
	w, err := ParseWTR(b)
	if err != nil || w.TileSize != 64 || w.ARGB != 0x98ffffff || w.FrameMillis != 40 || len(w.Textures) != 2 || w.Textures[1] != "caust01.jpg" {
		t.Fatal("native WTR fields", w, err)
	}
	for seconds, want := range map[float64]int{0: 0, .039: 0, .04: 1, .079: 1, .08: 0, .12: 1, -1: 0} {
		if got := w.FrameAt(seconds); got != want {
			t.Fatal("flipbook cadence", seconds, got, want)
		}
	}
	for n := range len(b) {
		if _, err := ParseWTR(b[:n]); err == nil {
			t.Fatal("accepted truncated WTR", n)
		}
	}
	for _, value := range []float32{0, -1, float32(math.NaN()), float32(math.Inf(1))} {
		binary.LittleEndian.PutUint32(b, math.Float32bits(value))
		if _, err := ParseWTR(b); err == nil {
			t.Fatal("accepted invalid texture scale", value)
		}
	}
}

func TestRealWaterFrameResources(t *testing.T) {
	a := openArchive(t)
	for _, filename := range []string{"Entity/Water/World.wtr", "Entity/Water/B1_Water.wtr", "Entity/Water/A2.wtr"} {
		raw, err := a.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		w, err := ParseWTR(raw)
		if err != nil {
			t.Fatal(err)
		}
		unique := map[string]bool{}
		total := 0
		for _, name := range w.Textures {
			base := path.Base(strings.ReplaceAll(name, `\`, "/"))
			file := "Entity/Water/" + strings.TrimSuffix(base, path.Ext(base)) + ".dds"
			if unique[strings.ToLower(file)] {
				continue
			}
			unique[strings.ToLower(file)] = true
			data, err := a.ReadFile(file)
			if err != nil {
				t.Fatalf("%s references %s: %v", filename, file, err)
			}
			texture, err := img.Decode(data)
			if err != nil {
				t.Fatalf("%s: %v", file, err)
			}
			if texture.DDS != nil {
				for _, level := range texture.DDS.Levels {
					total += len(level)
				}
			} else if texture.RGBA != nil {
				total += len(texture.RGBA.Pix)
			}
		}
		t.Logf("%s: %d authored frames, %d unique textures, %.2f MiB pixel data", filename, len(w.Textures), len(unique), float64(total)/(1<<20))
	}
}

func TestRealWTR(t *testing.T) {
	a := openArchive(t)
	eachFile(t, a, ".wtr", func(name string, b []byte) error {
		w, err := ParseWTR(b)
		if err == nil {
			t.Logf("%s: tile=%g ARGB=%08x frame=%dms frames=%d", name, w.TileSize, w.ARGB, w.FrameMillis, len(w.Textures))
		}
		return err
	})
}

func FuzzWTR(f *testing.F) {
	f.Add(make([]byte, 16))
	f.Fuzz(func(t *testing.T, b []byte) { ParseWTR(b) })
}
