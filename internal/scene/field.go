package scene

import (
	"fmt"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Instance is a placed model.
type Instance struct {
	Model *Model
	World render.M4
	Mani  bool
}

// World is a drawable map.
type World struct {
	dungeonInstances  []render.M4
	occlusion         *render.TerrainOcclusion
	occlusionRevision uint64
	occlusionEye      render.V3
	water             *waterSurface
	elapsed           float64
	stream            *fieldStream
	WLD               *world.WLD
	Sections          []Section
	Objects           []Instance
	rotors            []maniRotor
	vanis             []*vaniAsset
	mapFX             *mapEffects
	frameDT           float32
	Sky               *Sky
	dg                []dgPart
	dgOrder           []int
	dungeonOccluders  []dungeonOccluder
	dgShadow          *dungeonShadowIndex
	dgColliders       []world.CollisionMesh
	cameraSeen        map[int]bool
	FogColor          render.V3
	FogStart          float32
	FogEnd            float32
	heights           []float32
	side              int
	step              float32
}

type TerrainLayer struct {
	Tile     *render.Texture
	Mask     *render.Texture
	TileSize float32
}

// Section is one field section (1x1 below map size 1536, else 2x2; file
// suffix <z><x>), with its own masks and lightmap.
type Section struct {
	Mesh     *render.Mesh
	Layers   []TerrainLayer
	Lightmap *render.Texture
}

// LoadField loads world/<stem>.wld with its field data in
// world/Field/<stem>/ (<stem>_00_A<i>.dds masks, <stem>_00_L.dds lightmap).
func LoadField(a *Assets, stem string) (*World, error) {
	b, err := a.Arc.ReadFile("world/" + stem + ".wld")
	if err != nil {
		return nil, err
	}
	w, err := world.ParseWLD(b)
	if err != nil {
		return nil, err
	}
	if w.Dungeon {
		return nil, fmt.Errorf("%s is a dungeon", stem)
	}
	out := &World{WLD: w, FogColor: render.V3{w.FogColor[0] / 255, w.FogColor[1] / 255, w.FogColor[2] / 255},
		FogStart: w.FogStart, FogEnd: w.FogEnd}
	out.buildTerrain()
	tiles := make([]*render.Texture, len(w.Layers))
	for i, l := range w.Layers {
		if strings.TrimSpace(l.Texture) == "" {
			continue // unused layer slot
		}
		if tiles[i] = a.worldTexture("Terrain/Detail/" + DDSName(path.Base(l.Texture))); tiles[i] != nil {
			tiles[i].SetWrap(true)
		}
	}
	n := out.sectionsPerSide()
	for sz := 0; sz < n; sz++ {
		for sx := 0; sx < n; sx++ {
			sec := &out.Sections[sz*n+sx]
			field := fmt.Sprintf("world/Field/%s/%s_%d%d", stem, stem, sz, sx)
			for i, l := range w.Layers {
				sec.Layers = append(sec.Layers, TerrainLayer{TileSize: l.TileSize, Tile: tiles[i],
					Mask: a.optionalTexture(fmt.Sprintf("%s_A%d.dds", field, i))})
			}
			sec.Lightmap = a.optionalTexture(field + "_L.dds")
		}
	}
	out.addObjects(a)
	out.addManis(a)
	out.addVanis(a)
	out.loadMapEffects(a)
	out.Sky = LoadSky(a, w)
	out.water = loadWater(a, out)
	return out, nil
}

// Height of the terrain at raw x,z (same triangles as the mesh).
func (w *World) Height(x, z float32) float32 {
	if w.side < 2 {
		return 0
	}
	fx, fz := x/w.step, z/w.step
	maxI := float32(w.side - 1)
	fx = min(max(fx, 0), maxI)
	fz = min(max(fz, 0), maxI)
	x0, z0 := int(fx), int(fz)
	x1, z1 := min(x0+1, w.side-1), min(z0+1, w.side-1)
	tx, tz := fx-float32(x0), fz-float32(z0)
	h := func(i, j int) float32 { return w.heights[j*w.side+i] }
	if tx+tz <= 1 {
		return h(x0, z0) + tx*(h(x1, z0)-h(x0, z0)) + tz*(h(x0, z1)-h(x0, z0))
	}
	return h(x1, z1) + (1-tx)*(h(x0, z1)-h(x1, z1)) + (1-tz)*(h(x1, z0)-h(x1, z1))
}

// buildTerrain: samples every MapSize/(side-1) units, height (raw-10000)/50,
// diagonal (x1,z0)-(x0,z1) (Phoenix terrain_builder / wld_data).
func (w *World) heightsOnly() bool {
	side := int(w.WLD.MapSize/2) + 1
	if side < 2 || len(w.WLD.Heights) < side*side {
		return false
	}
	w.side, w.step = side, float32(w.WLD.MapSize)/float32(side-1)
	w.heights = make([]float32, side*side)
	for i, raw := range w.WLD.Heights[:side*side] {
		w.heights[i] = (float32(raw) - 10000) / 50
	}
	return true
}

func (w *World) sectionsPerSide() int {
	if w.WLD.MapSize >= 1536 {
		return 2
	}
	return 1
}

// buildTerrain: samples every MapSize/(side-1) units, height (raw-10000)/50
// (0x5aef30: x0.02 - 200), diagonal (x1,z0)-(x0,z1). One mesh per section;
// UV covers the section (masks, lightmap), UV2 is the raw XZ (tiles).
func (w *World) buildTerrain() {
	if !w.heightsOnly() {
		return
	}
	side := w.side
	n := w.sectionsPerSide()
	cells := (side - 1) / n
	w.Sections = make([]Section, n*n)
	for sz := 0; sz < n; sz++ {
		for sx := 0; sx < n; sx++ {
			x0, z0 := sx*cells, sz*cells
			v, idx := outTerrain(w, x0, z0, cells, cells, sx, sz)
			w.Sections[sz*n+sx].Mesh = render.NewMesh(v, idx)
		}
	}
}

var groupDirs = []struct {
	dir string
	get func(*world.WLD) world.Group
}{
	{"Entity/Building", func(w *world.WLD) world.Group { return w.Buildings }},
	{"Entity/Shape", func(w *world.WLD) world.Group { return w.Shapes }},
	{"Entity/Tree", func(w *world.WLD) world.Group { return w.Trees }},
	{"Entity/Grass", func(w *world.WLD) world.Group { return w.Grass }},
	{"Entity/Object", func(w *world.WLD) world.Group { return w.Objects }},
}

func (w *World) addObjects(a *Assets) {
	for _, g := range groupDirs {
		grp := g.get(w.WLD)
		for _, inst := range grp.Instances {
			if inst.Asset < 0 || int(inst.Asset) >= len(grp.Assets) {
				continue
			}
			name := grp.Assets[inst.Asset]
			if !strings.EqualFold(path.Ext(name), ".smod") {
				continue
			}
			m, err := a.SMOD(g.dir + "/" + name)
			if err != nil {
				a.Missing[g.dir+"/"+name] = true
				continue
			}
			w.Objects = append(w.Objects, Instance{Model: m,
				World: render.Basis(render.V3(inst.Pos), render.V3(inst.Fwd), render.V3(inst.Up))})
		}
	}
}

// Draw renders terrain passes, the lightmap and the objects.
func (w *World) Draw(s *render.Scene3D) {
	if w.stream != nil {
		w.stream.draw(s)
	}
	for _, sec := range w.Sections {
		drawSection(s, sec)
	}
	frustum := render.ViewFrustum(s.View, s.Proj)
	for _, p := range w.dg {
		if !frustum.BoxVisible(p.minimum, p.maximum) || !render.ClipBoxVisible(s.ClipPlane, p.minimum, p.maximum) || s.OccludedBox(p.minimum, p.maximum) {
			continue
		}
		if p.lightmap != nil {
			s.DrawLightmapped(p.mesh, p.tex, p.lightmap)
		} else {
			// No lightmap pages: EP5 material (Phoenix draws it the same way).
			s.DrawModel(p.mesh, p.tex, render.Identity(), render.NativeWorldAlphaRef(true), false)
		}
	}
	for _, rotor := range w.rotors {
		transform := rotor.transform(w.elapsed)
		if !dungeonModelVisible(s, frustum, rotor.model, transform) {
			continue
		}
		for _, p := range rotor.model.Parts {
			if !dungeonMeshVisible(s, frustum, p.Mesh, transform) {
				continue
			}
			s.DrawModel(p.Mesh, p.Tex, transform, render.NativeWorldAlphaRef(true), false)
		}
	}
	// Retain native MAni/SMOD order; consecutive single-material placements
	// can share a draw without reordering overlapping material parts.
	w.drawDungeonObjects(s, frustum)

	for _, vani := range w.vanis {
		vani.drawCulled(s, w.elapsed, frustum)
	}
}

// Update advances only visual world animation; server state is unaffected.
func (w *World) Update(dt float32) {
	w.frameDT = max(dt, 0)
	w.elapsed += float64(max(dt, 0))
	if w.Sky != nil {
		w.Sky.Update(dt)
	}
}

// HasTerrain reports whether the world has a heightmap (fields).
func (w *World) HasTerrain() bool { return w != nil && w.side >= 2 }
