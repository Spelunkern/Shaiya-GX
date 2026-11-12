package world

import "testing"

func TestRealMapOneFog(t *testing.T) {
	a := openArchive(t)
	b, err := a.ReadFile("world/1.wld")
	if err != nil {
		t.Fatal(err)
	}
	w, err := ParseWLD(b)
	if err != nil {
		t.Fatal(err)
	}
	if w.FogColor != [3]float32{108, 130, 147} || w.FogStart != 300 || w.FogEnd != 600 {
		t.Fatal("unexpected original map-one RGB/min/max", w.FogColor, w.FogStart, w.FogEnd)
	}
	if len(w.NPCs) == 0 {
		t.Fatal("map one lost its original NPC placements")
	}
	first := w.NPCs[0]
	if len(w.NPCs) != 286 || first.Type != 1 || first.TypeID != 1 || first.Position != (Vec3{1534.2275, 75.999916, 1788.6565}) {
		t.Fatalf("unexpected map one NPC placements: %d, first %+v", len(w.NPCs), first)
	}
}
