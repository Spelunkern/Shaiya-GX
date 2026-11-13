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
type MusicZone struct {
	Box        Box
	Radius     float32
	Asset, Unk int32
}

// SoundInstance is a point emitter with an authored radius and sound asset index.
type SoundInstance struct {
	Asset  int32
	Center Vec3
	Radius float32
}

// NPCPlacement retains original WLD positions and additional patrol points.
type NPCPlacement struct {
	Type, TypeID int32
	Position     Vec3
	Orientation  float32
	Patrol       []Vec3
}

type NamedArea struct {
	// TextID is the original WLD record index, including bitmap/other modes.
	// IDs in world/<map>.txt are sparse and must not be compacted.
	TextID        int
	Box           Box
	Radius        float32
	Text, Comment string
	Mode, Unknown int32
}

type WLD struct {
	NamedAreas   []NamedArea
	SoundAssets  []string
	Sounds       []SoundInstance
	MusicAssets  []string
	MusicZones   []MusicZone
	NPCs         []NPCPlacement
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
	// Audio and static map metadata precede the sky and fog.
	w.MusicAssets = r.names()
	for i, n := 0, r.count(36); i < n; i++ {
		w.MusicZones = append(w.MusicZones, MusicZone{r.box(), r.f32(), r.i32(), r.i32()})
	}
	w.SoundAssets = r.names()
	nz := r.count(28) // zones
	for i := 0; i < nz && r.err == nil; i++ {
		r.box()
		r.take(4 * r.count(4))
	}
	for i, n := 0, r.count(20); i < n; i++ {
		w.Sounds = append(w.Sounds, SoundInstance{Asset: r.i32(), Center: r.vec3(), Radius: r.f32()})
	}
	r.skipList(24 + 4)                        // monster-restricted zones
	r.skipList(24 + 4 + 512 + 1 + 2 + 1 + 12) // portals
	r.skipList(4 + 24 + 4 + 4 + 4)            // spawns
	for i, n := 0, r.count(24+4+512+4+4); i < n; i++ {
		area := NamedArea{Box: r.box(), Radius: r.f32(), Text: r.fixed(256), Comment: r.fixed(256), Mode: r.i32(), Unknown: r.i32(), TextID: i}
		w.NamedAreas = append(w.NamedAreas, area)
	}
	npcs := r.i32() // NPCs + patrol points
	if npcs < 0 {
		return nil, fmt.Errorf("world: negative NPC count %d", npcs)
	}
	for npcs > 0 && r.err == nil {
		npc := NPCPlacement{Type: r.i32(), TypeID: r.i32(), Position: r.vec3(), Orientation: r.f32()}
		p := r.count(12)
		if int32(p) >= npcs {
			return nil, fmt.Errorf("world: NPC patrol count %d exceeds remaining records %d", p, npcs)
		}
		npc.Patrol = make([]Vec3, p)
		for i := range npc.Patrol {
			npc.Patrol[i] = r.vec3()
		}
		w.NPCs = append(w.NPCs, npc)
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
