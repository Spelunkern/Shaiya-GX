package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"os"
	"path/filepath"
	"testing"
)

func TestMallCatalogUsesNativeData(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("requires loose assets")
	}
	arc, err := archive.OpenDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("Cash.SData")
	if err != nil {
		t.Fatal(err)
	}
	products, err := sdata.ParseCash(raw)
	if err != nil {
		t.Fatal(err)
	}
	visible := 0
	for _, p := range products {
		if p.Bag != 0 {
			visible++
		}
	}
	if visible == 0 {
		t.Skip("catalog deliberately has no public products")
	}
	a := inputApp()
	a.arc = arc
	if !a.mallCommand("/mall catalog 1") || len(a.mallCatalog) != len(products) || len(a.state.Chat) != 1+min(5, visible) {
		t.Fatal("first CSV catalog page missing")
	}
	if a.mallCatalog[0].Code != products[0].Code {
		t.Fatal("product identity changed")
	}
	pages := (visible + 4) / 5
	before := len(a.state.Chat)
	if !a.mallCommand(fmt.Sprintf("/mall catalog %d", pages)) || len(a.state.Chat) != before+1+visible-(pages-1)*5 {
		t.Fatal("last catalog page missing")
	}
	before = len(a.state.Chat)
	if !a.mallCommand(fmt.Sprintf("/mall catalog %d", pages+1)) || len(a.state.Chat) != before+1 {
		t.Fatal("out of range page")
	}
}

func TestMallLoadsNativeBinaryAndIgnoresCSV(t *testing.T) {
	dir := t.TempDir()
	p := cash.Product{Code: "CSV", Name: "CSV only", Bag: 1}
	p.Items[0] = cash.Item{ItemID: 1001, Count: 1}
	raw, err := sdata.EncodeCash([]cash.Product{p})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "Cash.SData"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "cash.csv"), []byte("invalid ignored CSV"), 0600)
	arc, err := archive.OpenDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.arc = arc
	if err = a.loadMallCatalog(); err != nil {
		t.Fatal(err)
	}
	if len(a.mallCatalog) != 1 || a.mallCatalog[0].Code != "CSV" {
		t.Fatal("native client failed")
	}
}
