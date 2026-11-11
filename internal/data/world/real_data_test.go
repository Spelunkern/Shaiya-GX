package world

import (
	"os"
	"strings"
	"testing"

	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
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
