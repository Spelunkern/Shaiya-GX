package app

import (
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestCraftValuesDecodeAndRejectInvalid(t *testing.T) {
	v, ok := parseCraftValues("01020304050607080953  ")
	if !ok || v.stats != [9]uint16{1, 2, 3, 4, 5, 6, 700, 800, 900} || v.enchant != 3 {
		t.Fatalf("unexpected values: %+v %t", v, ok)
	}
	for _, bad := range []string{"", "0102030405060708095", "010203040506070809x3"} {
		if _, ok := parseCraftValues(bad); ok {
			t.Fatalf("invalid craft accepted: %q", bad)
		}
	}
	if got := craftSummary("01000000000001000000"); got != "STR +1, HP +100" {
		t.Fatal(got)
	}
}

func TestItemDetailUsesInstanceAndGemDefinitions(t *testing.T) {
	def := &sdata.Item{Name: "Sword", Quality: 1200, Attack: 12}
	item := &proto.InventoryItem{Count: 1, Quality: 17, Craft: "01000000000000000002", Gems: [6]uint8{5, 9}}
	rows := itemDetailLines(def, item, 1, "Sell: 10 gold", func(id uint8) string {
		if id == 5 {
			return "Power Lapis"
		}
		return ""
	})
	var texts []string
	for _, row := range rows {
		texts = append(texts, row.text)
	}
	joined := strings.Join(texts, "|")
	for _, want := range []string{"Sword [2]", "endurance : 17", "STR +1", "Lapis: Power Lapis", "Lapis: Lapis 9"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	shop := itemDetailLines(def, nil, 1, "Buy: 100 gold", nil)
	for _, row := range shop {
		if strings.Contains(row.text, "endurance") || strings.Contains(row.text, "Lapis:") {
			t.Fatal("shop definition invented mutable instance state")
		}
	}
}

func TestItemDescriptionRemovesColorControls(t *testing.T) {
	got := stripItemColors("{c5}Enhancement{/c}: {custom} +2")
	if got != "Enhancement: {custom} +2" {
		t.Fatal(got)
	}
}

func TestTooltipPreservesInlineColorsAndCompleteDescription(t *testing.T) {
	glyphs := tooltipGlyphs("Plain {c3}gold{/c} tail", white)
	var plain strings.Builder
	yellow := 0
	for _, g := range glyphs {
		plain.WriteRune(g.char)
		if g.color == ([4]float32{1, 1, 0, 1}) {
			yellow++
		}
	}
	if plain.String() != "Plain gold tail" || yellow != 4 {
		t.Fatal("inline description colours lost", plain.String(), yellow)
	}
	rows := itemDetailLines(&sdata.Item{Name: "Item", Description: "first\\n{c3}second{/c}\nthird"}, nil, 1, "", nil)
	if len(rows) != 4 || rows[2].text != "{c3}second{/c}" || rows[3].text != "third" {
		t.Fatal("item description was truncated", rows)
	}
}
