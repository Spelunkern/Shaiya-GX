package scene

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// LoginScene is the login3 backdrop (0x4bcd60 load, 0x4ba650 render;
// Phoenix ep5_login.json for the placed extras).
type LoginScene struct {
	World      *World
	Camera     *LoginCamera
	sky        *render.Texture
	quad       *render.Mesh
	back       *Model
	smoke      *Model
	rock       *Model
	smokeAngle float64
}

func LoadLoginScene(a *Assets) (*LoginScene, error) {
	w, err := LoadDungeon(a, "world/login3.wld")
	if err != nil {
		return nil, err
	}
	s := &LoginScene{World: w, Camera: NewLoginCamera(), sky: a.Texture("Sky/login_sky.dds")}
	s.back, _ = a.SMOD("Entity/Building/login_effect_back01a.SMOD")
	s.smoke, _ = a.SMOD("Entity/Building/Logineffect_smoke01a.SMOD")
	s.rock, _ = a.SMOD("Entity/Building/Login_floatrock.SMOD")
	// Unit quad in XY; uv (0,1) at (-1,-1) like the native quad helper.
	s.quad = render.NewMesh([]render.Vertex{
		{Pos: render.V3{-1, -1, 0}, UV: [2]float32{0, 1}},
		{Pos: render.V3{-1, 1, 0}, UV: [2]float32{0, 0}},
		{Pos: render.V3{1, 1, 0}, UV: [2]float32{1, 0}},
		{Pos: render.V3{1, -1, 0}, UV: [2]float32{1, 1}},
	}, []uint32{0, 1, 2, 0, 2, 3})
	return s, nil
}

func (s *LoginScene) Update(dt float64) {
	s.World.Update(float32(dt))
	s.Camera.Update(dt)
	s.smokeAngle = math.Mod(s.smokeAngle+dt*0.1, 2*math.Pi)
}

func translate(v render.V3) render.M4 {
	m := render.Identity()
	m[12], m[13], m[14] = v[0], v[1], v[2]
	return m
}

// smokeWorld is RY(angle)*RZ(-30deg) about the target (native samples).
func (s *LoginScene) smokeWorld() render.M4 {
	c, sn := float32(math.Cos(s.smokeAngle)), float32(math.Sin(s.smokeAngle))
	r0 := render.V3{0.8660254, -0.5, 0}
	r2 := render.V3{0, 0, 1}
	row0 := r0.Scale(c).Sub(r2.Scale(sn))
	row2 := r0.Scale(sn).Add(r2.Scale(c))
	t := LoginTarget
	return render.M4{row0[0], row0[1], row0[2], 0, 0.5, 0.8660254, 0, 0, row2[0], row2[1], row2[2], 0, t[0], t[1], t[2], 1}
}

// Draw renders the extended backdrop, then the world and floating rock.
func (s *LoginScene) Draw(r *render.Scene3D, aspect float32) {
	eye := s.Camera.Eye
	r.SetCameraView(eye, LoginTarget, render.V3{0, 1, 0})
	r.Proj = render.PerspectiveFovLH(math.Pi/4, aspect, 1, 400)
	r.SkyProj = render.PerspectiveFovLH(math.Pi/4, aspect, 10, 100000)
	r.FogStart, r.FogEnd = 0, 0 // the login draws without fog
	r.LightDir = render.V3{-1, -1, -1}
	r.Begin()
	// Native 0x4ba919..0x4bad21: sky, smoke and rear effect share the
	// extended projection with depth disabled, before the world pass.
	r.BeginSky()
	// Sky: vertical plane 20000x10000 at 8000 along the view (diffuse 0xbbbbbb).
	if s.sky != nil {
		fwd := LoginTarget.Sub(eye).Norm()
		c := eye.Add(fwd.Scale(8000))
		w := render.M4{10000, 0, 0, 0, 0, 5000, 0, 0, 0, 0, 1, 0, c[0], c[1], c[2], 1}
		r.DrawTinted(s.quad, s.sky, w, [4]float32{0xbb / 255.0, 0xbb / 255.0, 0xbb / 255.0, 1}, false, false)
	}
	if s.smoke != nil {
		for _, p := range s.smoke.Parts {
			r.DrawTinted(p.Mesh, p.Tex, s.smokeWorld(), [4]float32{1, 1, 1, 1}, true, true)
		}
	}
	if s.back != nil {
		back := translate(LoginTarget.Add(render.V3{0, 15, 250}))
		for _, p := range s.back.Parts {
			r.DrawTinted(p.Mesh, p.Tex, back, [4]float32{1, 1, 1, 1}, true, true)
		}
	}
	r.EndSky()
	s.World.Draw(r)
	if s.rock != nil {
		rock := translate(LoginTarget.Add(render.V3{3, -13, 10}))
		for _, p := range s.rock.Parts {
			r.DrawModel(p.Mesh, p.Tex, rock, 0.5, false)
		}
	}
	s.World.DrawMapEffects(r, eye)
	r.End()
}
