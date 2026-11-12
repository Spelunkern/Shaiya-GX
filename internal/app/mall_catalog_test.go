package app

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
)

func TestMallCatalogUsesOriginalData(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("requires original assets")
	}
	arc, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.arc = arc
	if !a.mallCommand("/mall catalog 1") || len(a.mallCatalog) != 61 || len(a.state.Chat) != 6 {
		t.Fatal("first catalog page missing")
	}
	if a.mallCatalog[0].Code != "Basicos_1" || a.mallCatalog[0].Name == "" {
		t.Fatal("original product missing")
	}
	if !a.mallCommand("/mall catalog 13") || len(a.state.Chat) != 8 {
		t.Fatal("last catalog page missing")
	}
	if !a.mallCommand("/mall catalog 14") || len(a.state.Chat) != 9 {
		t.Fatal("out of range catalog page")
	}
}
