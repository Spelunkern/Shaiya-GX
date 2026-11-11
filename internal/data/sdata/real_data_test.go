package sdata

import (
	"os"
	"testing"

	"github.com/Spelunkern/shaiya-go-client/internal/data/archive"
)

// N1: Item.SData decrypts and parses to its last byte.
func TestRealItems(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := a.ReadFile("Item/Item.SData")
	if err != nil {
		t.Fatal(err)
	}
	items, err := ParseItems(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d items", len(items))
	if it := items[ItemKey{1, 1}]; it != nil {
		t.Logf("1/1 = %q model %d level %d", it.Name, it.Model, it.MinLevel)
	}
}
