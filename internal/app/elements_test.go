package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"strings"
	"testing"
)

func TestElementNamesUseOriginalMessagesAndRomanTiers(t *testing.T) {
	for id := uint8(1); id <= 8; id++ {
		name := elementName(id, text.SysMsg{5204: "Fuego", 5205: "Agua", 5206: "Tierra", 5207: "Viento"})
		want := [4]string{"Fuego", "Agua", "Tierra", "Viento"}[(id-1)%4] + " " + [2]string{"I", "II"}[(id-1)/4]
		if name != want {
			t.Fatal(id, name, want)
		}
	}
	if elementName(255) != "non-elemental" {
		t.Fatal("invalid element named")
	}
}

func TestEquipmentTooltipUsesLinkedElementAndPreservesDefinition(t *testing.T) {
	a := continuousForgeApp()
	a.items[sdata.ItemKey{Type: 30, TypeID: 1}].Attribute = 5
	a.items[sdata.ItemKey{Type: 30, TypeID: 2}].Attribute = 2
	def := a.itemsItem(1, 1)
	it := proto.InventoryItem{Type: 1, TypeID: 1, Gems: [6]uint8{1}}
	a.drawItemDetails(def, &it, 1, 0, 0, 1024, 768, "")
	if !strings.Contains(tooltipRowsForElementTest(a.tooltip), "Fire II") || def.Attribute != 0 {
		t.Fatal("linked element missing or cached definition mutated")
	}
	it.Gems[1] = 2
	a.drawItemDetails(def, &it, 1, 0, 0, 1024, 768, "")
	if !strings.Contains(tooltipRowsForElementTest(a.tooltip), "Fire II") || strings.Contains(tooltipRowsForElementTest(a.tooltip), "Water I") {
		t.Fatal("first elemental lapis must retain priority")
	}
	it.Gems = [6]uint8{}
	a.drawItemDetails(def, &it, 1, 0, 0, 1024, 768, "")
	if strings.Contains(tooltipRowsForElementTest(a.tooltip), "Fire") {
		t.Fatal("removed lapis still changes tooltip")
	}
}

func TestElementalSkillTooltipShowsAttributeAndOverrideTier(t *testing.T) {
	a := inputApp()
	key := sdata.SkillKey{ID: 1, Level: 1}
	a.skillDefs[key] = sdata.Skill{Name: "Flame", Attribute: 5}
	a.queueSkillTooltip(key)
	if !strings.Contains(tooltipRowsForElementTest(a.tooltip), "Fire II") {
		t.Fatal("spell element not visible")
	}
	a.skillDefs[key] = sdata.Skill{Name: "Neutral weapon", Detail: 94}
	a.queueSkillTooltip(key)
	if !strings.Contains(tooltipRowsForElementTest(a.tooltip), "non-elemental") {
		t.Fatal("neutral override not visible")
	}
}

func tooltipRowsForElementTest(rows []itemDetailLine) string {
	var out strings.Builder
	for _, row := range rows {
		out.WriteString(row.text)
		out.WriteByte('\n')
	}
	return out.String()
}

func TestSocketElementPriorityAndGrayMarkers(t *testing.T) {
	a := continuousForgeApp()
	a.items[sdata.ItemKey{Type: 30, TypeID: 1}].Attribute = 5
	a.items[sdata.ItemKey{Type: 30, TypeID: 2}].Attribute = 2
	a.items[sdata.ItemKey{Type: 30, TypeID: 3}] = &sdata.Item{Type: 30, TypeID: 3, Attribute: 5}
	a.items[sdata.ItemKey{Type: 30, TypeID: 4}] = &sdata.Item{Type: 30, TypeID: 4, Stats: [6]uint16{7}}
	d := a.itemsItem(1, 1)
	d.Slot = 6
	it := proto.InventoryItem{Type: 1, TypeID: 1, Gems: [6]uint8{4, 1, 2, 3}}
	check := func(want uint8, mask [6]bool) {
		t.Helper()
		value, inactive := a.itemElementState(d, &it)
		if value != want || inactive != mask {
			t.Fatal(value, inactive, want, mask)
		}
		a.drawItemDetails(d, &it, 1, 0, 0, 1024, 768, "")
		for i, socket := range a.tooltipSockets {
			if socket.inactive != mask[i] {
				t.Fatal("tooltip markers diverge", i)
			}
		}
	}
	check(5, [6]bool{false, false, true, true})
	it.Gems[1] = 0
	check(2, [6]bool{false, false, false, true})
	d.Attribute = 3
	check(3, [6]bool{false, false, true, true})
}
