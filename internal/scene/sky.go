package scene

import (
	"strings"

	"github.com/Spelunkern/shaiya-go-client/internal/data/world"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
)

// Sky is the field sky (0x54c8a0 / 0x54c9b0 / 0x54dd00): the data/sky/sky.3do
// dome with the map's gradient texture and two cloud planes.
//
// Clouds: quads of half size 10000 at +1000 and +1100 (0x70baa8..0x70babc),
// UV 0..4 (0x70bac0/0x70bac4), vertex colour 0x50ffffff, scrolling 0.01 and
// 0.005 UV/s (0x70ba98/0x70ba9c). NO VERIFICADO: pass order and states.
type Sky struct {
	Dome    *render.Mesh
	DomeTex *render.Texture
	Clouds  [2]*render.Texture
	cloudUV [2]float32
	quad    *render.Mesh
}

func LoadSky(a *Assets, w *world.WLD) *Sky {
	s := &Sky{}
	if b, err := a.Arc.ReadFile("Sky/sky.3DO"); err == nil {
		if o, err := world.Parse3DO(b); err == nil {
			v := make([]render.Vertex, len(o.Vertices))
			for i, x := range o.Vertices {
				v[i] = render.Vertex{Pos: render.V3(x.Pos), Normal: render.V3(x.Normal), UV: x.UV}
			}
			idx := make([]uint32, 0, len(o.Faces)*3)
			for _, f := range o.Faces {
				idx = append(idx, uint32(f[0]), uint32(f[1]), uint32(f[2]))
			}
			s.Dome = render.NewMesh(v, idx)
		}
	}
	tex := func(name string) *render.Texture {
		if strings.TrimSpace(name) == "" {
			return nil
		}
		return a.Texture("Sky/" + name)
	}
	s.DomeTex = tex(w.Sky)
	s.Clouds[0], s.Clouds[1] = tex(w.CloudA), tex(w.CloudB)
	for _, c := range s.Clouds {
		if c != nil {
			c.SetWrap(true)
		}
	}
	const half, uv = 10000, 4
	v := []render.Vertex{
		{Pos: render.V3{-half, 0, -half}, UV: [2]float32{0, 0}},
		{Pos: render.V3{half, 0, -half}, UV: [2]float32{uv, 0}},
		{Pos: render.V3{half, 0, half}, UV: [2]float32{uv, uv}},
		{Pos: render.V3{-half, 0, half}, UV: [2]float32{0, uv}},
	}
	s.quad = render.NewMesh(v, []uint32{0, 1, 2, 0, 2, 3})
	return s
}

// Update scrolls the clouds.
func (s *Sky) Update(dt float32) {
	for i, speed := range [2]float32{0.01, 0.005} {
		s.cloudUV[i] += speed * dt
		if s.cloudUV[i] > 4 {
			s.cloudUV[i] -= 4
		}
	}
}

// Draw renders behind everything, centred on the eye, with its own
// (10, 100000) projection like the original sky pass.
func (s *Sky) Draw(r *render.Scene3D, eye render.V3) {
	if s == nil {
		return
	}
	r.BeginSky()
	if s.Dome != nil && s.DomeTex != nil {
		w := render.Identity()
		w[12], w[13], w[14] = eye[0], eye[1], eye[2]
		r.DrawUnlit(s.Dome, s.DomeTex, w, 1, [2]float32{})
	}
	for i, h := range [2]float32{1000, 1100} {
		if s.Clouds[i] == nil {
			continue
		}
		w := render.Identity()
		w[12], w[13], w[14] = eye[0], eye[1]+h, eye[2]
		r.DrawUnlit(s.quad, s.Clouds[i], w, 0x50/255.0, [2]float32{s.cloudUV[i], s.cloudUV[i]})
	}
	r.EndSky()
}
