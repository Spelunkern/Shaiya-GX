package scene

import "testing"

func TestSceneObjectsAreNotShadowReceivers(t *testing.T) {
	for _, w := range []*World{
		{Objects: []Instance{{Model: &Model{}}}},
		{rotors: []maniRotor{{}}},
		{vanis: []*vaniAsset{{}}},
	} {
		if w.HasShadowReceivers() {
			t.Fatal("object-only scene still enables projected shadows")
		}
	}
	if !(&World{side: 2}).HasShadowReceivers() {
		t.Fatal("terrain receiver disabled")
	}
	if !(&World{dgShadow: &dungeonShadowIndex{triangles: []shadowTriangle{{}}}}).HasShadowReceivers() {
		t.Fatal("dungeon receiver disabled")
	}
}
