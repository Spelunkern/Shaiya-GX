package scene

import (
	"fmt"
	"path"
	"strings"

	"github.com/Spelunkern/shaiya-go-client/internal/data/world"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
)

// Instance is a placed model.
type Instance struct {
	Model *Model
	World render.M4
}

// World is a drawable map.
type World struct {
	WLD      *world.WLD
	Sections []Section
	Objects  []Instance
	Sky      *Sky
	dg       []dgPart
	FogColor render.V3
	FogStart float32
	FogEnd   float32
	heights  []float32
	side     int
	step     float32
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
		if tiles[i] = a.Texture("Terrain/Detail/" + DDSName(path.Base(l.Texture))); tiles[i] != nil {
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
	out.Sky = LoadSky(a, w)
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
			cs := cells + 1
			v := make([]render.Vertex, 0, cs*cs)
			inv := 1 / float32(cells)
			for z := 0; z < cs; z++ {
				for x := 0; x < cs; x++ {
					gx, gz := x0+x, z0+z
					px, pz := float32(gx)*w.step, float32(gz)*w.step
					v = append(v, render.Vertex{
						Pos: render.V3{px, w.heights[gz*side+gx], pz}, Normal: render.V3{0, 1, 0},
						UV: [2]float32{float32(x) * inv, float32(z) * inv}, UV2: [2]float32{px, pz}})
				}
			}
			idx := make([]uint32, 0, cells*cells*6)
			for z := 0; z < cells; z++ {
				for x := 0; x < cells; x++ {
					a, b := uint32(z*cs+x), uint32(z*cs+x+1)
					c, d := uint32((z+1)*cs+x), uint32((z+1)*cs+x+1)
					idx = append(idx, a, b, c, b, d, c)
				}
			}
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
	for _, sec := range w.Sections {
		first := true
		for _, l := range sec.Layers {
			if l.Mask == nil || l.Tile == nil || sec.Mesh == nil {
				continue
			}
			s.DrawTerrainLayer(sec.Mesh, l.Mask, l.Tile, l.TileSize, first)
			first = false
		}
		if sec.Lightmap != nil && sec.Mesh != nil && !first {
			s.DrawLightmap(sec.Mesh, sec.Lightmap)
		}
	}
	for _, p := range w.dg {
		if p.lightmap != nil {
			s.DrawLightmapped(p.mesh, p.tex, p.lightmap)
		} else {
			// No lightmap pages: EP5 material (Phoenix draws it the same way).
			s.DrawModel(p.mesh, p.tex, render.Identity(), 0.5, false)
		}
	}
	for _, o := range w.Objects {
		for _, p := range o.Model.Parts {
			s.DrawModel(p.Mesh, p.Tex, o.World, 0.5, false)
		}
	}
}

// HasTerrain reports whether the world has a heightmap (fields).
func (w *World) HasTerrain() bool { return w != nil && w.side >= 2 }
