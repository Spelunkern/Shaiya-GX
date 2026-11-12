package world

import (
	"encoding/binary"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"math"
	"os"
	"strings"
	"testing"
)

func TestSMODBoundsRejectTruncatedAndInvalidSphere(t *testing.T) {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint32(b[12:], math.Float32bits(12))
	_, r, err := SMODBounds(b)
	if err != nil || r != 12 {
		t.Fatal("valid sphere", r, err)
	}
	if _, _, err := SMODBounds(b[:15]); err == nil {
		t.Fatal("truncated sphere")
	}
	for _, value := range []float32{-1, float32(math.Inf(1)), float32(math.NaN())} {
		binary.LittleEndian.PutUint32(b[12:], math.Float32bits(value))
		if _, _, err := SMODBounds(b); err == nil {
			t.Fatal("invalid radius accepted", value)
		}
	}
}
func TestRealSMODBounds(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original assets")
	}
	a, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	count := 0
	a.Walk(func(name string, f *archive.File) {
		if !strings.HasSuffix(strings.ToLower(name), ".smod") {
			return
		}
		b, err := a.ReadPrefix(name, 16)
		if err == nil {
			_, _, err = SMODBounds(b)
		}
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
		count++
	})
	t.Logf("validated %d SMOD sphere headers", count)
}
