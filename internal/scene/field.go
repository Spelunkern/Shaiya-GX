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
	Terrain  *render.Mesh
	Layers   []TerrainLayer
	Lightmap *render.Texture
	Objects  []Instance
	Sky      *Sky
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
	field := "world/Field/" + stem + "/" + stem + "_00"
	for i, l := range w.Layers {
		tl := TerrainLayer{TileSize: l.TileSize,
			Tile: a.Texture("Terrain/Detail/" + DDSName(path.Base(l.Texture))),
			Mask: a.Texture(fmt.Sprintf("%s_A%d.dds", field, i))}
		if tl.Tile != nil {
			tl.Tile.SetWrap(true)
		}
		out.Layers = append(out.Layers, tl)
	}
	out.Lightmap = a.Texture(field + "_L.dds")
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
func (w *World) buildTerrain() {
	side := int(w.WLD.MapSize/2) + 1
	if side < 2 || len(w.WLD.Heights) < side*side {
		return
	}
	w.side, w.step = side, float32(w.WLD.MapSize)/float32(side-1)
	w.heights = make([]float32, side*side)
	for i, raw := range w.WLD.Heights[:side*side] {
		w.heights[i] = (float32(raw) - 10000) / 50
	}
	v := make([]render.Vertex, side*side)
	inv := 1 / float32(side-1)
	for z := 0; z < side; z++ {
		for x := 0; x < side; x++ {
			px, pz := float32(x)*w.step, float32(z)*w.step
			v[z*side+x] = render.Vertex{
				Pos: render.V3{px, w.heights[z*side+x], pz}, Normal: render.V3{0, 1, 0},
				UV: [2]float32{float32(x) * inv, float32(z) * inv}, UV2: [2]float32{px, pz}}
		}
	}
	idx := make([]uint32, 0, (side-1)*(side-1)*6)
	for z := 0; z < side-1; z++ {
		for x := 0; x < side-1; x++ {
			a, b := uint32(z*side+x), uint32(z*side+x+1)
			c, d := uint32((z+1)*side+x), uint32((z+1)*side+x+1)
			idx = append(idx, a, b, c, b, d, c)
		}
	}
	w.Terrain = render.NewMesh(v, idx)
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
	first := true
	for _, l := range w.Layers {
		if l.Mask == nil || l.Tile == nil || w.Terrain == nil {
			continue
		}
		s.DrawTerrainLayer(w.Terrain, l.Mask, l.Tile, l.TileSize, first)
		first = false
	}
	if w.Lightmap != nil && w.Terrain != nil && !first {
		s.DrawLightmap(w.Terrain, w.Lightmap)
	}
	for _, o := range w.Objects {
		for _, p := range o.Model.Parts {
			s.DrawModel(p.Mesh, p.Tex, o.World, 0.5, false)
		}
	}
}
