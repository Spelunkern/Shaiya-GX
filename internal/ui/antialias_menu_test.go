package ui

import (
	"encoding/json"
	"testing"
)

func TestAntialiasDefaultsAndExactSampleChoices(t *testing.T) {
	if DefaultClientSettings().Antialias != AntialiasOff {
		t.Fatal("AA must default to Off")
	}
	value := DefaultClientSettings()
	if err := json.Unmarshal([]byte(`{"Glow":1,"Music":0.4}`), &value); err != nil || value.Antialias != AntialiasOff || value.Glow != 1 {
		t.Fatal("legacy settings migration", err)
	}
	menu := &GameMenu{}
	menu.SetAntialiasSamples([]int{4, 8})
	want := []int{AntialiasOff, AntialiasFXAA, AntialiasTAA, AntialiasMSAA4, AntialiasMSAA8}
	if len(menu.antialiasRows()) != len(want) {
		t.Fatal(menu.antialiasRows())
	}
	for i, v := range menu.antialiasRows() {
		if v != want[i] {
			t.Fatal("unsupported sample level offered")
		}
	}
	if NormalizeAntialias(AntialiasMSAA16, []int{2, 4}) != AntialiasMSAA4 || NormalizeAntialias(AntialiasMSAA2, nil) != AntialiasOff {
		t.Fatal("unsupported saved AA did not fall back")
	}
	value.Antialias = AntialiasTAA
	b, e := json.Marshal(value)
	if e != nil {
		t.Fatal(e)
	}
	var restored ClientSettings
	if e = json.Unmarshal(b, &restored); e != nil || restored.Antialias != AntialiasTAA {
		t.Fatal("TAA settings did not persist", e)
	}
}
