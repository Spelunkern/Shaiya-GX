package scene

import (
	"log"
	"math"
	"path"
	"strings"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Phoenix WATER_SURFACE_Y and the original water controllers share raw Y=0.
const waterSurfaceY = float32(0)

const SwimEntryY = waterSurfaceY - 1.6
const SwimSurfaceY = waterSurfaceY - 1.5

// WaterFloor reports terrain deep enough for the map's water to support swimming.
func (w *World) WaterFloor(x, z float32) (float32, bool) {
	if w == nil || w.water == nil || !w.HasTerrain() || x < 0 || z < 0 || x > float32(w.WLD.MapSize) || z > float32(w.WLD.MapSize) {
		return 0, false
	}
	floor := w.Height(x, z)
	return floor, floor < SwimEntryY
}

// SwimmingAt uses the authored map-wide WTR surface and the original
// character thresholds documented in Phoenix (before its +0.25 adjustment).
// It is presentation state; positions remain server-confirmed.
func (w *World) SwimmingAt(x, y, z float32) bool {
	_, deep := w.WaterFloor(x, z)
	return deep && y <= SwimSurfaceY
}

// UnderwaterFogAt follows the camera, including in shallow water. The fog
// colour blends Phoenix's underwater reference with the map-authored WTR tint;
// the precise original-client underwater shader remains unverified.
func (w *World) UnderwaterFogAt(eye render.V3) (render.V3, bool) {
	if w == nil || w.water == nil || !w.HasTerrain() || eye[0] < 0 || eye[2] < 0 || eye[0] > float32(w.WLD.MapSize) || eye[2] > float32(w.WLD.MapSize) || eye[1] >= waterSurfaceY || eye[1] < w.Height(eye[0], eye[2]) {
		return render.V3{}, false
	}
	t := w.water.tint
	return render.V3{.8*.128 + .2*t[0], .8*.295 + .2*t[1], .8*.382 + .2*t[2]}, true
}

type waterSurface struct {
	mesh       *render.Mesh
	frames     []*render.Texture
	definition *world.WTR
	tint       [4]float32
	normals    [2]*render.Texture
	wetCells   []cellKey
	indexed    bool
}

func loadWater(a *Assets, w *World) *waterSurface {
	if !w.HasTerrain() || !strings.EqualFold(path.Ext(w.WLD.Layout), ".wtr") {
		return nil
	}
	size := float32(w.WLD.MapSize)
	water := &waterSurface{tint: [4]float32{.12, .32, .42, 1}, mesh: render.NewMesh([]render.Vertex{
		{Pos: render.V3{0, waterSurfaceY, 0}, UV2: [2]float32{0, 0}},
		{Pos: render.V3{size, waterSurfaceY, 0}, UV2: [2]float32{size, 0}},
		{Pos: render.V3{size, waterSurfaceY, size}, UV2: [2]float32{size, -size}},
		{Pos: render.V3{0, waterSurfaceY, size}, UV2: [2]float32{0, -size}},
	}, []uint32{0, 1, 2, 0, 2, 3})}
	water.indexTerrain(w)
	file := "Entity/Water/" + path.Base(strings.ReplaceAll(w.WLD.Layout, `\`, "/"))
	b, err := a.Arc.ReadFile(file)
	if err == nil {
		water.definition, err = world.ParseWTR(b)
	}
	if err != nil {
		log.Printf("water %s: %v", file, err)
		return water
	}
	d := water.definition
	water.tint = [4]float32{float32(d.ARGB>>16&255) / 255, float32(d.ARGB>>8&255) / 255, float32(d.ARGB&255) / 255, float32(d.ARGB>>24) / 255}
	for i, name := range []string{"WaterBumps1.tga", "WaterBumps2.tga"} {
		if data, err := a.Arc.ReadFile(name); err == nil {
			if height, err := img.DecodeTGA(data); err == nil {
				if normal := img.WaterNormals(height); normal != nil {
					water.normals[i] = render.Upload(img.Texture{RGBA: normal})
					water.normals[i].SetWrap(true)
				}
			}
		}
	}
	for _, name := range d.Textures {
		name = "Entity/Water/" + DDSName(path.Base(strings.ReplaceAll(name, `\`, "/")))
		var t *render.Texture
		b, err := a.Arc.ReadFile(name)
		if err == nil {
			var decoded img.Texture
			decoded, err = img.Decode(b)
			if err == nil {
				t = render.Upload(decoded)
				t.SetWrap(true)
			}
		}
		if err != nil {
			log.Printf("water frame %s: %v", name, err)
		}
		water.frames = append(water.frames, t)
	}
	return water
}

// A triangle can meet the plane only if at least one of its vertices is at or
// below it. Index whole cells (including shared edge vertices) conservatively.
func (water *waterSurface) indexTerrain(w *World) {
	water.wetCells, water.indexed = nil, false
	if w.stream == nil || w.side < 2 || len(w.heights) < w.side*w.side {
		return
	}
	for z := 0; z < w.side-1; z += terrainCellSamples {
		for x := 0; x < w.side-1; x += terrainCellSamples {
			wet := false
			for iz := z; iz <= min(z+terrainCellSamples, w.side-1) && !wet; iz++ {
				for ix := x; ix <= min(x+terrainCellSamples, w.side-1); ix++ {
					if w.heights[iz*w.side+ix] <= waterSurfaceY {
						wet = true
						break
					}
				}
			}
			if wet {
				water.wetCells = append(water.wetCells, cellKey{x / terrainCellSamples, z / terrainCellSamples})
			}
		}
	}
	water.indexed = true
}

// DrawWater follows the original WTR flipbook. Transparent surfaces draw after
// opaque actors/geometry, so submerged models retain the water's attenuation.
func (w *World) DrawWater(s *render.Scene3D) {
	water := w.water
	if !w.WaterVisible(s.View, s.Proj) {
		return
	}
	s.Begin()
	if len(water.frames) > 0 {
		index := water.definition.FrameAt(w.elapsed)
		if frame := water.frames[index]; frame != nil {
			s.WaterNormals = water.normals
			// All four authored scroll rates wrap together after 200/3 units.
			s.WaterTime = float32(math.Mod(w.elapsed*1.5, 200.0/3))
			s.DrawWater(water.mesh, frame, water.definition.TileSize, water.tint)
			s.WaterNormals = [2]*render.Texture{}
			return
		}
	}
	// Keep the previous simple surface as a visible fallback for missing assets.
	s.DrawSolid(water.mesh, [4]float32{.12, .32, .42, 1})
}

// WaterVisible is conservative for the map-wide flat WTR surface.
func (w *World) WaterVisible(view, projection render.M4) bool {
	if w == nil || w.water == nil || w.WLD == nil {
		return false
	}
	size := float32(w.WLD.MapSize)
	frustum := render.ViewFrustum(view, projection)
	if !frustum.BoxVisible(render.V3{0, waterSurfaceY, 0}, render.V3{size, waterSurfaceY, size}) {
		return false
	}
	water, stream := w.water, w.stream
	if !water.indexed || stream == nil || !w.Ready() || projection[0] == 0 || projection[5] == 0 || projection[10] <= 1 || projection[14] >= 0 {
		return true
	}
	for _, v := range view {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return true
		}
	}
	// Inverse() returns identity for a singular matrix; it must not turn an
	// unknown camera into an apparent in-map viewpoint.
	det := view[0]*(view[5]*view[10]-view[6]*view[9]) - view[4]*(view[1]*view[10]-view[2]*view[9]) + view[8]*(view[1]*view[6]-view[2]*view[5])
	if math.Abs(float64(det)) < 1e-6 || view[3] != 0 || view[7] != 0 || view[11] != 0 || view[15] != 1 {
		return true
	}
	far := projection[14] / (1 - projection[10])
	near := -projection[14] / projection[10]
	if far <= 0 || near <= 0 || math.IsNaN(float64(far)) || math.IsInf(float64(far), 0) {
		return true
	}
	// Only trust occlusion when the full view is resident and all loaded
	// terrain sections have at least one opaque layer.
	rx, ry := float64(1/projection[0]), float64(1/projection[5])
	required := far * float32(math.Sqrt(1+rx*rx+ry*ry))
	if math.IsNaN(float64(required)) || math.IsInf(float64(required), 0) || stream.radius+stream.size()*.25 < required {
		return true
	}
	for _, mat := range stream.materials {
		opaque := false
		for _, layer := range mat.Layers {
			if layer.Mask != nil && layer.Tile != nil {
				opaque = true
				break
			}
		}
		if !opaque {
			return true
		}
	}
	if len(stream.materials) == 0 {
		return true
	}
	inverse := view.Inverse()
	eye := render.V3{inverse[12], inverse[13], inverse[14]}
	if math.IsNaN(float64(eye[0])) || math.IsNaN(float64(eye[1])) || math.IsNaN(float64(eye[2])) || math.IsInf(float64(eye[0]), 0) || math.IsInf(float64(eye[1]), 0) || math.IsInf(float64(eye[2]), 0) || eye[0] < 0 || eye[2] < 0 || eye[0] > size || eye[2] > size || eye[1] <= w.Height(eye[0], eye[2])+near*2 {
		return true
	}
	cellSize := stream.size()
	if stream.center != (cellKey{int(math.Floor(float64(eye[0] / cellSize))), int(math.Floor(float64(eye[2] / cellSize)))}) {
		return true
	}
	for _, cell := range water.wetCells {
		x, z := float32(cell.x)*cellSize, float32(cell.z)*cellSize
		if frustum.BoxVisible(render.V3{x - .01, waterSurfaceY - .01, z - .01}, render.V3{min(x+cellSize, size) + .01, waterSurfaceY + .01, min(z+cellSize, size) + .01}) {
			return true
		}
	}
	return false
}

func (w *World) HasWaterNormals() bool {
	return w != nil && w.water != nil && w.water.normals[0] != nil && w.water.normals[1] != nil
}

func (w *waterSurface) Delete() {
	w.mesh.Delete()
	for _, t := range w.normals {
		if t != nil {
			t.Delete()
		}
	}
	for _, t := range w.frames {
		if t != nil {
			t.Delete()
		}
	}
}
