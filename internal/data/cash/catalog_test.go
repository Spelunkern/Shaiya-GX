package cash

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
)

func fixture() []Product {
	p := Product{Index: 1, Bag: 1, Icon: 2, Cost: 45, Code: "PACK_1", Name: "Sword, \"GX\" 🌟", Description: "Line one\nLine two"}
	p.Items[0] = Item{1001, 2}
	p.Items[23] = Item{25004, 255}
	return []Product{p}
}
func TestCSVRoundTripPreservesTextAndAllSlots(t *testing.T) {
	source := fixture()
	raw, err := Encode(source)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(source, got) {
		t.Fatal("CSV lost fields", got)
	}
	hash, _ := Hash(source)
	crlf := bytes.ReplaceAll(raw, []byte("\n"), []byte("\r\n"))
	got, err = Parse(append([]byte{0xef, 0xbb, 0xbf}, crlf...))
	if err != nil {
		t.Fatal(err)
	}
	normalized, _ := Hash(got)
	if normalized != hash {
		t.Fatal("BOM/CRLF changed semantic hash")
	}
	source[0].Cost++
	changed, _ := Hash(source)
	if changed == hash {
		t.Fatal("price change did not change hash")
	}
}
func TestCatalogRejectsInvalidBusinessData(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func([]Product) []Product
	}{
		{"duplicate code", func(p []Product) []Product { return append(p, p[0]) }},
		{"wire code too long", func(p []Product) []Product { p[0].Code = strings.Repeat("A", 21); return p }},
		{"category", func(p []Product) []Product { p[0].Bag = 5; return p }},
		{"missing quantity", func(p []Product) []Product { p[0].Items[0].Count = 0; return p }},
		{"invalid packed ID", func(p []Product) []Product { p[0].Items[0].ItemID = 1256; return p }},
		{"priced empty", func(p []Product) []Product { p[0].Items = [24]Item{}; return p }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Encode(tc.change(fixture())); err == nil {
				t.Fatal("invalid catalog accepted")
			}
		})
	}
}
func TestCSVRejectsWrongColumnsAndNumericOverflow(t *testing.T) {
	raw, _ := Encode(fixture())
	for _, col := range []int{3, 8} {
		rows, _ := csv.NewReader(bytes.NewReader(raw)).ReadAll()
		rows[1][col] = "-1"
		if col == 8 {
			rows[1][col] = "1000"
		}
		var b bytes.Buffer
		w := csv.NewWriter(&b)
		w.WriteAll(rows)
		if _, err := Parse(b.Bytes()); err == nil {
			t.Fatal("invalid unsigned number accepted", col)
		}
	}
	if _, err := Parse(bytes.Replace(raw, []byte("price"), []byte("cost"), 1)); err == nil {
		t.Fatal("unknown schema accepted")
	}
}
