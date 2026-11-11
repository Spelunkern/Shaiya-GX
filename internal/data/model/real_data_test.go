package model

import (
	"os"
	"strings"
	"testing"

	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
)

func each(t *testing.T, ext string, fn func(string, []byte) error) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ok, bad := 0, 0
	a.Walk(func(p string, f *archive.File) {
		if !strings.HasSuffix(strings.ToLower(p), ext) || !strings.HasPrefix(p, "Character/") {
			return
		}
		b, err := a.Read(f)
		if err == nil {
			err = fn(p, b)
		}
		if err != nil {
			bad++
			if bad <= 8 {
				t.Errorf("%s: %v", p, err)
			}
			return
		}
		ok++
	})
	t.Logf("%s: %d ok, %d failed", ext, ok, bad)
}

func TestReal3DC(t *testing.T) {
	each(t, ".3dc", func(_ string, b []byte) error { _, err := Parse3DC(b); return err })
}
func TestRealANI(t *testing.T) {
	each(t, ".ani", func(_ string, b []byte) error { _, err := ParseANI(b); return err })
}
func TestRealMLT(t *testing.T) {
	each(t, ".mlt", func(_ string, b []byte) error { _, err := ParseMLT(b); return err })
}
