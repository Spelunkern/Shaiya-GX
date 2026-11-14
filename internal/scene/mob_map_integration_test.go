//go:build integration

package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"math"
	"os"
	"runtime"
	"testing"
)

func TestFieldDungeonAndMonsterEffectsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 640, Height: 360, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := NewAssets(arc)
	w, err := loadStreamedField(assets, "1")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if len(w.dg) == 0 || len(w.dgColliders) == 0 || w.dgShadow == nil {
		t.Fatalf("field dungeon incomplete: %d parts, %d colliders", len(w.dg), len(w.dgColliders))
	}
	lo, hi := w.dg[0].minimum, w.dg[0].maximum
	lightmaps := 0
	for _, part := range w.dg {
		for axis := range lo {
			lo[axis] = min(lo[axis], part.minimum[axis])
			hi[axis] = max(hi[axis], part.maximum[axis])
		}
		if part.lightmap != nil {
			lightmaps++
		}
	}
	if lo[0] < 1000 || hi[0] > 1800 || lo[2] < 1400 || hi[2] > 2000 || lightmaps == 0 {
		t.Fatalf("incorrect field placement or lightmaps: %v..%v, %d", lo, hi, lightmaps)
	}
	w.stream.trim()
	for _, part := range w.dg {
		for _, tex := range []*render.Texture{part.tex, part.lightmap} {
			if tex != nil && !gl.IsTexture(tex.ID) {
				t.Fatal("field streaming deleted dungeon texture")
			}
		}
	}
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	eye := render.V3{1425, 85, 1640}
	r.View = render.LookAtLH(eye, render.V3{1425, 80, 1705}, render.V3{0, 1, 0})
	r.Proj = render.PerspectiveFovLH(math.Pi/4, 640.0/360, .1, 500)
	r.ViewportW, r.ViewportH = 640, 360
	r.Begin()
	w.Draw(r)
	r.End()
	gl.Finish()
	t.Logf("map 1: %d dungeon parts, %d colliders, %d lightmapped parts, bounds %v..%v", len(w.dg), len(w.dgColliders), lightmaps, lo, hi)
	raw, err := arc.ReadFile("Monster/monster.MON")
	if err != nil {
		t.Fatal(err)
	}
	visuals, err := model.ParseMON(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{17, 18, 19, 24, 26, 28} {
		m, err := assets.buildCreature("Monster", visuals[index], "test")
		if err != nil {
			t.Fatal(index, err)
		}
		m.World = render.Basis(render.V3{1425, 70, 1705}, render.V3{0, 0, 1}, render.V3{0, 1, 0})
		if index == 28 {
			m.PlayOnce(2)
		}
		for frame := 0; frame < 12; frame++ {
			m.Update(1.0 / 60)
			budget := 2
			m.PrepareEffects(render.V3{1425, 75, 1710}, 1.0/60, true, true, &budget)
		}
		attached := 0
		for _, fx := range m.boneEffects {
			if fx.instance != nil {
				attached++
			}
		}
		if index != 28 && attached == 0 {
			t.Fatalf("model %d: missing bone effects: %v", index, assets.Missing)
		}
		if index == 28 && len(m.attackEffects) == 0 {
			t.Fatalf("attack effect missing: %v", assets.Missing)
		}
		r.Begin()
		m.DrawEffects(r)
		r.End()
		t.Logf("monster model %d: %d attached, %d attack effects", index, attached, len(m.attackEffects))
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %x", code)
	}
}
