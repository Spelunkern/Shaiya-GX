package sdata

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRealLegacyCashConvertsWithoutLosingAnyField(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("requires legacy/catalog fixtures")
	}
	legacy := os.Getenv("SHAIYA_LEGACY_CASH")
	if legacy == "" {
		legacy = filepath.Join(dir, "Cash.SData")
	}
	raw, err := os.ReadFile(legacy)
	if os.IsNotExist(err) {
		t.Skip("converted client has no legacy binary")
	}
	if err != nil {
		t.Fatal(err)
	}
	original, err := ParseCash(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := cash.Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	converted, err := cash.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, converted) {
		t.Fatal("binary-to-CSV conversion lost a field or slot")
	}
	t.Logf("all fields and 24 slots preserved for %d products", len(original))
}
