package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func TestDetailsOnlyTrimDecorationsWhenReduced(t *testing.T) {
	s := &render.Scene3D{View: render.Identity()}
	e := fieldDrawEntry{object: &cellObject{detailClass: 1, detailRank: 3}}
	if !e.detailVisible(s, render.V3{0, 0, 500}, 1) {
		t.Fatal("default detail lost vegetation")
	}
	s.DetailBias = 1
	if !e.detailVisible(s, render.V3{0, 0, 20}, 1) || e.detailVisible(s, render.V3{0, 0, 150}, 1) || e.detailVisible(s, render.V3{0, 0, 40}, 1) {
		t.Fatal("medium detail vegetation budget not applied")
	}
	e.object.detailClass = 0
	if !e.detailVisible(s, render.V3{0, 0, 500}, 1) {
		t.Fatal("structural mesh was culled")
	}
	e.object.detailClass = 2
	s.DetailBias = 2
	if e.detailVisible(s, render.V3{0, 0, 200}, 10) || !e.detailVisible(s, render.V3{0, 0, 40}, 10) {
		t.Fatal("low tree distance budget not applied")
	}
	s.Shadow = true
	if !e.detailVisible(s, render.V3{0, 0, 500}, 1) {
		t.Fatal("shadow pass inherited decoration budget")
	}
}
