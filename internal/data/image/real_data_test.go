package image

import (
	"os"
	"strings"
	"testing"

	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
)

// N1: every TGA and DDS of the real archive decodes (SHAIYA_DATA=dir).
func TestRealArchiveTextures(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	var ok, bad int
	fails := map[string]int{}
	a.Walk(func(p string, f *archive.File) {
		lp := strings.ToLower(p)
		isTGA, isDDS := strings.HasSuffix(lp, ".tga"), strings.HasSuffix(lp, ".dds")
		if !isTGA && !isDDS {
			return
		}
		b, err := a.Read(f)
		if err == nil {
			_, err = Decode(b)
		}
		if err != nil {
			bad++
			fails[err.Error()]++
			if bad <= 10 {
				t.Logf("%s: %v", p, err)
			}
			return
		}
		ok++
	})
	t.Logf("decoded %d, failed %d", ok, bad)
	for e, n := range fails {
		t.Logf("  %4d  %s", n, e)
	}
	if bad > 0 {
		t.Errorf("%d textures failed", bad)
	}
}
