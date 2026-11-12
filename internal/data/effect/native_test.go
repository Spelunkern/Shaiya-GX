package effect

import (
	"math"
	"testing"
)

func TestNativeCastLibraries(t *testing.T) {
	for _, tc := range []struct {
		id   uint16
		file string
	}{
		{37, "Effect/IN_ma_water.EFT"}, {53, "Effect/IN_ma_water.EFT"},
		{30, "Effect/IN_mc_yellow.EFT"}, {42, "Effect/IN_mc_black.EFT"},
		{103, "Effect/IN_ma_generic.EFT"}, {48, "Effect/IN_ma_fire.EFT"},
	} {
		if got := CastLibrary(tc.id); got != tc.file {
			t.Fatalf("skill %d: %s", tc.id, got)
		}
	}
}

func TestVelocityCurveSupportRejectsSingularSegments(t *testing.T) {
	e := Emitter{Mesh: -1, InitialDelay: 1, Unk13: 1, Textures: []int32{0}, Opacity: []OpacityFrame{{Opacity: 1}, {Opacity: 2, Time: 1}}}
	if !e.LinearSupported() {
		t.Fatal("valid velocity curve rejected")
	}
	for _, keys := range [][]OpacityFrame{
		{{Time: 1}, {Time: 1}}, {{Time: 1}, {Time: 0}},
		{{Opacity: -math.MaxFloat32}, {Opacity: math.MaxFloat32, Time: math.SmallestNonzeroFloat32}},
	} {
		e.Opacity = keys
		if e.LinearSupported() {
			t.Fatal("singular/overflowing velocity segment accepted", keys)
		}
	}
}

func TestNativeFieldMappingDiffersFromParsecNames(t *testing.T) {
	l, err := ParseEFT(fixture("EFT"))
	if err != nil {
		t.Fatal(err)
	}
	p := l.Effects[0].Parameters()
	if p.SourceBlend != 2 || p.DestinationBlend != 5 || p.LifeMin != .1 || p.RateMax != 0 || p.RateMin != 0 {
		t.Fatalf("native offsets: %+v", p)
	}
	d := Emitter{Mesh: -1, InitialDelay: 1, Unk13: 1, Textures: []int32{0}}
	if !d.LinearSupported() {
		t.Fatal("linear billboard rejected")
	}
	d.Unk21 = 1
	if !d.LinearSupported() {
		t.Fatal("attractor must be supported by the particle runtime")
	}
}

func TestNativeMeshBranch(t *testing.T) {
	e := Emitter{Mesh: 0}
	if !e.UsesMesh() {
		t.Fatal("mesh index 0 must be drawable")
	}
	e.Unk10 = 1
	if e.UsesMesh() {
		t.Fatal("Unknown10 must force the billboard branch")
	}
	e.Mesh = -1
	e.Unk10 = 0
	if e.UsesMesh() {
		t.Fatal("negative mesh index")
	}
}

func TestStationaryZeroLifeSupportIsBounded(t *testing.T) {
	d := Emitter{Textures: []int32{0}, Colors: []ColorFrame{{}, {Time: 1}}}
	if !d.LinearSupported() {
		t.Fatal("bounded static emitter rejected")
	}
	d.Spread1[0] = 1
	if !d.LinearSupported() {
		t.Fatal("force-free zero-life motion rejected")
	}
	d.Opacity = []OpacityFrame{{}, {Time: 1, Opacity: 1}}
	if d.LinearSupported() {
		t.Fatal("unbounded velocity curve accepted")
	}
	d.Opacity = nil
	d.Spread1[0] = 0
	d.Unk20 = 1
	if d.LinearSupported() {
		t.Fatal("unbounded static force accepted")
	}
}
