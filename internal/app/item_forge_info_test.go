package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"strings"
	"testing"
)

func TestClientForgeInformationUsesDefinitionValues(t *testing.T) {
	for _, tc := range []struct {
		d    sdata.Item
		want string
	}{
		{sdata.Item{Type: 30, ReqIg: 31}, "Base link chance: 41%"},
		{sdata.Item{Type: 30, ReqIg: 255}, "Base link chance: 1%"},
		{sdata.Item{Type: 30, ReqIg: 0}, "Base link chance: 100%"},
		{sdata.Item{Type: 100, Special: 36, ReqVG: 20}, "Linking bonus: +20%"},
		{sdata.Item{Type: 44, Special: 58, ReqVG: 90}, "Extraction chance: 90%"},
		{sdata.Item{Type: 100, Special: 58, ReqVG: 150}, "Extraction chance: 100%"},
		{sdata.Item{Type: 101, Special: 62, ReqVG: 201}, ""},
		{sdata.Item{Type: 95, ReqIg: 31}, ""},
	} {
		if got := itemForgeInfo(&tc.d); got != tc.want {
			t.Fatal(got, tc.want)
		}
	}
	a := continuousForgeApp()
	d := &sdata.Item{Type: 30, ReqIg: 31, Description: "Native description"}
	a.drawItemDetails(d, nil, 1, 0, 0, 1024, 768, "")
	if a.tooltipDescription != d.Description || !strings.Contains(a.tooltipExtra.forgeInfo, "41%") {
		t.Fatal("forge info must follow original description in the shared tooltip")
	}
}

func TestSkillElementLabelUsesRequestedFamilyColor(t *testing.T) {
	a := inputApp()
	key := sdata.SkillKey{ID: 1, Level: 1}
	for id := uint8(1); id <= 8; id++ {
		a.skillDefs[key] = sdata.Skill{Name: "Spell", Attribute: id}
		a.queueSkillTooltip(key)
		found := false
		for _, row := range a.tooltip {
			if row.text == elementName(id) {
				found = true
				if row.color != skillElementColor(id) {
					t.Fatal("wrong element label color", id)
				}
			}
		}
		if !found || skillElementColor(id) != skillElementColor((id-1)%4+1) {
			t.Fatal("tier colors must match", id)
		}
	}
}
