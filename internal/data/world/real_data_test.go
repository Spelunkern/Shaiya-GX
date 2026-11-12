package world

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
)

func openArchive(t *testing.T) *archive.Archive {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return a
}

// eachFile runs fn over every archive file with the extension.
func eachFile(t *testing.T, a *archive.Archive, ext string, fn func(path string, b []byte) error) {
	ok, bad := 0, 0
	a.Walk(func(p string, f *archive.File) {
		if !strings.HasSuffix(strings.ToLower(p), ext) {
			return
		}
		b, err := a.Read(f)
		if err == nil {
			err = fn(p, b)
		}
		if err != nil {
			bad++
			if bad <= 10 {
				t.Errorf("%s: %v", p, err)
			}
			return
		}
		ok++
	})
	t.Logf("%s: %d ok, %d failed", ext, ok, bad)
}

// N1: every WLD of the archive parses to the end.
func TestRealWLD(t *testing.T) {
	a := openArchive(t)
	eachFile(t, a, ".wld", func(_ string, b []byte) error { _, err := ParseWLD(b); return err })
}

func TestRealMani(t *testing.T) {
	a := openArchive(t)
	eachFile(t, a, ".mani", func(_ string, b []byte) error { _, err := ParseMani(b); return err })
}

func TestRealVANI(t *testing.T) {
	a := openArchive(t)
	eachFile(t, a, ".vani", func(_ string, b []byte) error { _, err := ParseVANI(b); return err })
}

func TestRealVANIHeaderBounds(t *testing.T) {
	a := openArchive(t)
	count, outside := 0, 0
	eachFile(t, a, ".vani", func(name string, b []byte) error {
		v, err := ParseVANI(b)
		if err != nil {
			return err
		}
		count++
		maximum := float64(0)
		for _, mesh := range v.Meshes {
			for _, frame := range mesh.Frames {
				for _, vertex := range frame {
					dx, dy, dz := float64(vertex.Pos[0]-v.Center[0]), float64(vertex.Pos[1]-v.Center[1]), float64(vertex.Pos[2]-v.Center[2])
					maximum = math.Max(maximum, math.Sqrt(dx*dx+dy*dy+dz*dz))
				}
			}
		}
		if maximum > float64(v.Radius)+.001 {
			outside++
			if outside <= 8 {
				t.Logf("%s: header radius %.2f, vertex radius %.2f", name, v.Radius, maximum)
			}
		}
		return nil
	})
	if count == 0 {
		t.Fatal("no original VANI assets")
	}
	if outside != 0 {
		t.Errorf("%d VANI assets exceed their header sphere", outside)
	}
	t.Logf("%d/%d VANI assets have frames outside the header sphere", outside, count)
}

// brokenSMOD are truncated files no WLD references (leftovers in the
// archive); the original never loads them.
var brokenSMOD = map[string]bool{
	"Entity/Tree/Gate_iii_S.SMOD":          true,
	"world/dungeon/R1_Trade/R1_Trade.SMOD": true,
}

func TestRealSMOD(t *testing.T) {
	a := openArchive(t)
	eachFile(t, a, ".smod", func(p string, b []byte) error {
		_, err := ParseSMOD(b)
		if brokenSMOD[p] {
			return nil
		}
		return err
	})
}

func TestRealDG(t *testing.T) {
	a := openArchive(t)
	eachFile(t, a, ".dg", func(_ string, b []byte) error {
		d, err := ParseDG(b)
		if err == nil && d.Root == nil {
			t.Logf("empty dg")
		}
		return err
	})
}

func TestReal3DO(t *testing.T) {
	a := openArchive(t)
	eachFile(t, a, ".3do", func(_ string, b []byte) error { _, err := Parse3DO(b); return err })
}
