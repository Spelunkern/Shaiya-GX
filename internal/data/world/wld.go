// Package world reads the world formats: WLD (field and dungeon layouts),
// DG (dungeon geometry with lightmaps) and SMOD (static models).
// Layouts follow Parsec (Shaiya/Wld, Dg, Smod).
package world

import "fmt"

type Instance struct {
	Asset        int32
	Pos, Fwd, Up Vec3
}

type ManiInstance struct {
	Building, Mani int32
	Pos, Fwd, Up   Vec3
}

type EffectInstance struct {
	Pos, Fwd, Up Vec3
	Effect       int32
}

type Group struct {
	Assets    []string
	Instances []Instance
}

// WLD is a world layout. Field-only parts (terrain, sky) are empty for
// dungeons.
type WLD struct {
	Dungeon      bool
	MapSize      uint32
	Heights      []uint16 // (MapSize/2+1)^2 samples
	TextureMap   []byte
	Layers       []TerrainLayer
	Layout       string
	Buildings    Group
	Shapes       Group
	Trees        Group
	Grass        Group
	VaniA, VaniB Group
	Dungeons     Group
	ManiAssets   []string
	Manis        []ManiInstance
	EffectFile   string
	Effects      []EffectInstance
	Objects      Group
	Sky, CloudA  string
	CloudB       string
	FogColor     [3]float32
	FogStart     float32
	FogEnd       float32
}

type TerrainLayer struct {
	Texture  string
	TileSize float32
	Sound    string
}

func (r *reader) names() []string {
	n := r.count(256)
	out := make([]string, n)
	for i := range out {
		out[i] = r.fixed(256)
	}
	return out
}

func (r *reader) group() Group {
	g := Group{Assets: r.names()}
	n := r.count(40)
	g.Instances = make([]Instance, n)
	for i := range g.Instances {
		g.Instances[i] = Instance{Asset: r.i32(), Pos: r.vec3(), Fwd: r.vec3(), Up: r.vec3()}
	}
	return g
}

func ParseWLD(b []byte) (*WLD, error) {
	r := &reader{b: b}
	w := &WLD{}
	kind := r.fixed(4)
	w.Dungeon = kind == "DUN"
	if !w.Dungeon {
		if kind != "FLD" {
			return nil, fmt.Errorf("world: unknown wld kind %q", kind)
		}
		w.MapSize = r.u32()
		side := int(w.MapSize/2) + 1
		if w.MapSize > 8192 {
			return nil, fmt.Errorf("world: map size %d", w.MapSize)
		}
		w.Heights = make([]uint16, side*side)
		for i := range w.Heights {
			w.Heights[i] = r.u16()
		}
		w.TextureMap = append([]byte(nil), r.take(side*side)...)
		n := r.count(516)
		for i := 0; i < n; i++ {
			w.Layers = append(w.Layers, TerrainLayer{Texture: r.fixed(256), TileSize: r.f32(), Sound: r.fixed(256)})
		}
	}
	w.Layout = r.fixed(256)
	w.Buildings, w.Shapes, w.Trees, w.Grass = r.group(), r.group(), r.group(), r.group()
	w.VaniA, w.VaniB, w.Dungeons = r.group(), r.group(), r.group()
	w.ManiAssets = r.names()
	n := r.count(44)
	for i := 0; i < n; i++ {
		w.Manis = append(w.Manis, ManiInstance{Building: r.i32(), Mani: r.i32(), Pos: r.vec3(), Fwd: r.vec3(), Up: r.vec3()})
	}
	w.EffectFile = r.fixed(256)
	n = r.count(40)
	for i := 0; i < n; i++ {
		w.Effects = append(w.Effects, EffectInstance{Pos: r.vec3(), Fwd: r.vec3(), Up: r.vec3(), Effect: r.i32()})
	}
	r.take(12) // three unknown i32
	w.Objects = r.group()
	if r.err != nil {
		return nil, r.err
	}
	// The rest (music, zones, sounds, portals, spawns, areas, NPCs) is
	// server-side or audio data; skip to sky and fog.
	r.names()                  // music assets
	r.skipList(24 + 4 + 4 + 4) // music zones
	r.names()                  // sound assets
	nz := r.count(28)          // zones
	for i := 0; i < nz && r.err == nil; i++ {
		r.box()
		r.take(4 * r.count(4))
	}
	r.skipList(4 + 12 + 4)                    // sound instances
	r.skipList(24 + 4)                        // monster-restricted zones
	r.skipList(24 + 4 + 512 + 1 + 2 + 1 + 12) // portals
	r.skipList(4 + 24 + 4 + 4 + 4)            // spawns
	r.skipList(24 + 4 + 512 + 4 + 4)          // named areas
	npcs := r.i32()                           // NPCs + patrol points
	for npcs > 0 && r.err == nil {
		r.take(4 + 4 + 12 + 4)
		p := r.count(12)
		r.take(12 * p)
		npcs -= int32(p) + 1
	}
	if !w.Dungeon {
		w.Sky, w.CloudA, w.CloudB = r.fixed(256), r.fixed(256), r.fixed(256)
	}
	r.take(24) // two unused colours
	w.FogColor = [3]float32{r.f32(), r.f32(), r.f32()}
	w.FogStart, w.FogEnd = r.f32(), r.f32()
	return w, r.err
}

func (r *reader) skipList(size int) {
	n := r.count(size)
	r.take(n * size)
}
