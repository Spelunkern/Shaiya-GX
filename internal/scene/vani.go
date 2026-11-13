package scene

import (
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

type vaniPart struct {
	mesh     *render.Mesh
	texture  *render.Texture
	frames   [][]world.Vertex
	faces    []world.Face
	vertices []render.Vertex
}

func loadVani(a *Assets, file string) (*vaniAsset, error) {
	data, err := a.Arc.ReadFile(file)
	if err != nil {
		return nil, err
	}
	parsed, err := world.ParseVANI(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return uploadVani(a, file, parsed)
}
func uploadVani(a *Assets, file string, parsed *world.VANI) (*vaniAsset, error) {
	asset := &vaniAsset{frame: -1, count: parsed.FrameCount, frameMilliseconds: parsed.FrameMilliseconds, center: render.V3(parsed.Center), radius: parsed.Radius}
	asset.cascade = strings.EqualFold(path.Base(file), "Login_effect_water01a.VANI")
	for _, m := range parsed.Meshes {
		part := vaniPart{texture: a.FindTexture(m.Texture), frames: m.Frames, faces: m.Faces, vertices: make([]render.Vertex, len(m.Frames[0]))}
		if part.texture != nil {
			part.texture.SetWrap(true)
		}
		indices := make([]uint32, 0, len(m.Faces)*3)
		for _, face := range m.Faces {
			indices = append(indices, uint32(face[0]), uint32(face[1]), uint32(face[2]))
		}
		part.mesh = render.NewAnimatedMesh(len(part.vertices), indices)
		asset.parts = append(asset.parts, part)
	}
	return asset, nil
}

type vaniAsset struct {
	visibleWorlds     []render.M4
	parts             []vaniPart
	frameMilliseconds int32
	cascade           bool
	worlds            []render.M4
	frame             int
	count             int
	center            render.V3
	radius            float32
}

func (v *vaniAsset) draw(s *render.Scene3D, elapsed float64) {
	if len(v.worlds) == 0 || v.count == 0 {
		return
	}
	frame := v.frameAt(elapsed)
	if frame != v.frame {
		for i := range v.parts {
			p := &v.parts[i]
			for j, src := range p.frames[frame] {
				p.vertices[j] = render.Vertex{Pos: render.V3(src.Pos), Normal: render.V3(src.Normal), UV: src.UV}
			}
			p.mesh.UpdateVertices(p.vertices)
		}
		v.frame = frame
	}
	if v.cascade {
		// Phoenix's login cascade reference: preserve smooth texture alpha,
		// emit unlit water and leave the backdrop's depth unchanged.
		if !s.Shadow {
			for _, matrix := range v.worlds {
				for _, part := range v.parts {
					s.DrawEffect(part.mesh, part.texture, matrix, [4]float32{1, 1, 1, 1}, 4, 5)
				}
			}
			gl.DepthMask(true)
		}
		return
	}
	if len(v.worlds) == 1 {
		for _, part := range v.parts {
			s.DrawModel(part.mesh, part.texture, v.worlds[0], render.NativeWorldAlphaRef(false), false)
		}
		return
	}
	s.SetInstances(v.worlds)
	for _, part := range v.parts {
		s.DrawInstanced(part.mesh, part.texture, render.NativeWorldAlphaRef(false))
	}
}

func (v *vaniAsset) close() {
	for _, part := range v.parts {
		part.mesh.Delete()
	}
}

// addVanis shares one dynamic vertex stream per asset, irrespective of the
// number of WLD placements. Each asset retains its authored frame interval.
func (w *World) addVanis(a *Assets) {
	assets := make(map[string]*vaniAsset)
	for _, group := range []world.Group{w.WLD.VaniA, w.WLD.VaniB} {
		for _, inst := range group.Instances {
			if inst.Asset < 0 || int(inst.Asset) >= len(group.Assets) {
				continue
			}
			name := path.Base(strings.ReplaceAll(group.Assets[inst.Asset], `\`, "/"))
			file := "Entity/VAni/" + name
			key := strings.ToLower(file)
			asset, exists := assets[key]
			if !exists {
				asset, _ = loadVani(a, file)
				if asset != nil {
					w.vanis = append(w.vanis, asset)
				}
				assets[key] = asset
			}
			if asset != nil {
				asset.worlds = append(asset.worlds, render.Basis(render.V3(inst.Pos), render.V3(inst.Fwd), render.V3(inst.Up)))
			}
		}
	}
}

func (v *vaniAsset) frameAt(elapsed float64) int {
	if v.count <= 0 {
		return 0
	}
	interval := v.frameMilliseconds
	if interval <= 0 {
		interval = 33
	}
	return int(math.Mod(math.Floor(max(0, elapsed)*1000/float64(interval)), float64(v.count)))
}

// Eager dungeon placements keep all authored transforms and filter into scratch
// storage; animation bounds encompass every frame, not just the current pose.
func (v *vaniAsset) drawCulled(s *render.Scene3D, elapsed float64, f render.Frustum) {
	if v.cascade || v.radius <= 0 {
		v.draw(s, elapsed)
		return
	}
	all := v.worlds
	v.visibleWorlds = v.visibleWorlds[:0]
	for _, matrix := range all {
		c, r := transformPoint(v.center, matrix), v.radius*sphereScale(matrix)
		if f.SphereVisible(c, r) && render.ClipSphereVisible(s.ClipPlane, c, r) && !s.OccludedSphere(c, r) {
			v.visibleWorlds = append(v.visibleWorlds, matrix)
		}
	}
	v.worlds = v.visibleWorlds
	v.draw(s, elapsed)
	v.worlds = all
}
