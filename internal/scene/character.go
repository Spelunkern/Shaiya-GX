package scene

import (
	"fmt"
	"path"
	"strings"

	"github.com/Spelunkern/shaiya-go-client/internal/data/model"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
)

// Appearance selects a character's parts (FUN_00417220): race, gender,
// job, face, hair, height and the Model byte of the equipped helmet,
// upper, lower, hands and feet (0 = unequipped).
type Appearance struct {
	Race, Gender, Job uint8
	Face, Hair        uint8
	Height            uint8
	Helmet            int // -1 = none, else Model
	Upper, Lower      uint8
	Hand, Foot        uint8
	ShowHelmet        bool
}

var raceFolders = [...]string{"Human", "Elf", "Vile", "DeathEater"}
var racePrefix = [...]string{"hu", "el", "vi", "de"}

// Prefix is race + m/w + body type: f for jobs 0-1, r for 2-3, m for 4-5
// (Phoenix ep5_character_creation.json; DAT_0220f068 sets in 0x417220).
func (a Appearance) Prefix() string {
	body := "fmr"[0:1]
	switch a.Job {
	case 2, 3:
		body = "r"
	case 4, 5:
		body = "m"
	}
	g := "m"
	if a.Gender == 1 {
		g = "w"
	}
	return racePrefix[a.Race%4] + g + body
}

// HeightScale per the creation table (0.96 .. 1.04).
func (a Appearance) HeightScale() float32 {
	s := [...]float32{0.96, 0.98, 1.0, 1.02, 1.04}
	return s[min(int(a.Height), len(s)-1)]
}

type skinPart struct {
	mesh  *render.SkinMesh
	tex   *render.Texture
	bones []render.M4 // B (inverse bind) per bone
	cut   bool        // MLT Visibility mode: alpha test
	pose  []render.M4
}

// Character is a posed, animated actor.
type Character struct {
	parts    []*skinPart
	clip     *Clip
	clipName string
	folder   string
	prefix   string
	assets   *Assets
	Time     float32
	World    render.M4
}

// SetClip switches to <prefix>_<name>.ANI (e.g. "000_normal", "002_run"),
// restarting it; unknown clips keep the current one.
func (c *Character) SetClip(name string) {
	if c == nil || c.clipName == name {
		return
	}
	cl, err := c.assets.Clip(fmt.Sprintf("Character/%s/ANI/%s_%s.ANI", c.folder, c.prefix, name))
	if err != nil {
		return
	}
	c.clip, c.clipName, c.Time = cl, name, 0
}

func (s *Assets) mlt(folder, prefix, part string) (*model.MLT, error) {
	b, err := s.Arc.ReadFile(fmt.Sprintf("Character/%s/%s_%s.MLT", folder, prefix, part))
	if err != nil {
		return nil, err
	}
	return model.ParseMLT(b)
}

// BuildCharacter loads the parts for an appearance with the given clip
// suffix (e.g. "019_select").
func BuildCharacter(s *Assets, a Appearance, clip string) (*Character, error) {
	folder := raceFolders[a.Race%4]
	prefix := a.Prefix()
	c := &Character{folder: folder, prefix: prefix, assets: s}
	add := func(part string, index int) {
		m, err := s.mlt(folder, prefix, part)
		if err != nil || index < 0 || index >= len(m.Records) {
			if err == nil {
				err = fmt.Errorf("record %d of %d", index, len(m.Records))
			}
			s.Missing[fmt.Sprintf("%s_%s[%d]: %v", prefix, part, index, err)] = true
			return
		}
		r := m.Records[index]
		if int(r.Mesh) >= len(m.Meshes) || r.Mesh < 0 {
			return
		}
		p, err := s.skinPart(folder, m.Meshes[r.Mesh])
		if err != nil {
			s.Missing[err.Error()] = true
			return
		}
		cp := *p
		if r.Texture >= 0 && int(r.Texture) < len(m.Textures) {
			cp.tex = s.Texture(fmt.Sprintf("Character/%s/DDS/%s", folder, DDSName(path.Base(m.Textures[r.Texture]))))
		}
		cp.cut = r.Alpha == 0
		c.parts = append(c.parts, &cp)
	}
	add("upper", int(a.Upper))
	add("lower", int(a.Lower))
	add("hand", int(a.Hand))
	add("foot", int(a.Foot))
	add("face", int(a.Face))
	if a.Helmet >= 0 && a.ShowHelmet {
		add("helmet", a.Helmet)
	} else {
		add("hair", int(a.Hair))
	}
	c.SetClip(clip)
	if c.clip == nil {
		return c, fmt.Errorf("clip %s_%s not found", prefix, clip)
	}
	return c, nil
}

func (s *Assets) skinPart(folder, mesh string) (*skinPart, error) {
	file := fmt.Sprintf("Character/%s/3DC/%s", folder, path.Base(strings.ReplaceAll(mesh, `\`, "/")))
	key := "3dc:" + strings.ToLower(file)
	if p, ok := s.parts[key]; ok {
		return p, nil
	}
	b, err := s.Arc.ReadFile(file)
	if err != nil {
		return nil, err
	}
	m, err := model.Parse3DC(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	v := make([]render.SkinVertex, len(m.Vertices))
	for i, x := range m.Vertices {
		v[i] = render.SkinVertex{Pos: render.V3(x.Pos), Normal: render.V3(x.Normal), UV: x.UV,
			Weight: x.Weight, Bone0: float32(x.Bones[0]), Bone1: float32(x.Bones[1])}
	}
	idx := make([]uint32, 0, len(m.Faces)*3)
	for _, f := range m.Faces {
		if int(f[0]) < len(v) && int(f[1]) < len(v) && int(f[2]) < len(v) {
			idx = append(idx, uint32(f[0]), uint32(f[1]), uint32(f[2]))
		}
	}
	p := &skinPart{mesh: render.NewSkinMesh(v, idx)}
	for _, b := range m.Bones {
		p.bones = append(p.bones, render.M4(b))
	}
	s.parts[key] = p
	return p, nil
}

// Clip is an ANI prepared for sampling.
type Clip struct {
	ani       *model.ANI
	restLocal []render.M4
}

func (s *Assets) Clip(file string) (*Clip, error) {
	key := "ani:" + strings.ToLower(file)
	if c, ok := s.clips[key]; ok {
		return c, nil
	}
	b, err := s.Arc.ReadFile(file)
	if err != nil {
		return nil, err
	}
	a, err := model.ParseANI(b)
	if err != nil {
		return nil, err
	}
	c := &Clip{ani: a}
	// local_i = rest_i * inv(rest_parent) (row vectors).
	for _, bone := range a.Bones {
		l := render.M4(bone.Rest)
		if p := bone.Parent; p >= 0 && int(p) < len(a.Bones) {
			l = l.Mul(render.M4(a.Bones[p].Rest).Inverse())
		}
		c.restLocal = append(c.restLocal, l)
	}
	s.clips[key] = c
	return c, nil
}

// Duration in seconds (30 fps).
func (c *Clip) Duration() float32 {
	return float32(c.ani.End-c.ani.Start) / 30
}

// Finals returns each bone's animated global matrix at time t (looping).
func (c *Clip) Finals(t float32) []render.M4 {
	a := c.ani
	frame := float32(a.Start)
	if d := c.Duration(); d > 0 {
		for t >= d {
			t -= d
		}
		frame += t * 30
	}
	out := make([]render.M4, len(a.Bones))
	for i, bone := range a.Bones {
		rq, rt := render.MatrixQuat(c.restLocal[i])
		if len(bone.Rot) > 0 {
			rq = sampleRot(bone.Rot, frame)
		}
		if len(bone.Trans) > 0 {
			rt = sampleTrans(bone.Trans, frame)
		}
		l := render.QuatMatrix(rq, rt)
		if p := bone.Parent; p >= 0 && int(p) < i {
			l = l.Mul(out[p])
		}
		out[i] = l
	}
	return out
}

func sampleRot(k []model.RotKey, f float32) [4]float32 {
	if f <= float32(k[0].Frame) {
		return k[0].Q
	}
	for i := 1; i < len(k); i++ {
		if f <= float32(k[i].Frame) {
			a, b := k[i-1], k[i]
			t := (f - float32(a.Frame)) / float32(b.Frame-a.Frame)
			return render.Slerp(a.Q, b.Q, t)
		}
	}
	return k[len(k)-1].Q
}

func sampleTrans(k []model.TransKey, f float32) [3]float32 {
	if f <= float32(k[0].Frame) {
		return k[0].T
	}
	for i := 1; i < len(k); i++ {
		if f <= float32(k[i].Frame) {
			a, b := k[i-1], k[i]
			t := (f - float32(a.Frame)) / float32(b.Frame-a.Frame)
			return [3]float32{a.T[0] + (b.T[0]-a.T[0])*t, a.T[1] + (b.T[1]-a.T[1])*t, a.T[2] + (b.T[2]-a.T[2])*t}
		}
	}
	return k[len(k)-1].T
}

// Update advances the animation.
func (c *Character) Update(dt float32) { c.Time += dt }

// Draw poses every part (B_part * F) and draws it.
func (c *Character) Draw(k *render.Skinner, s *render.Scene3D) {
	if c == nil || c.clip == nil {
		return
	}
	finals := c.clip.Finals(c.Time)
	k.Begin(s, 0.8) // character ambient (Phoenix account_ep5_render)
	for _, p := range c.parts {
		if cap(p.pose) < len(p.bones) {
			p.pose = make([]render.M4, len(p.bones))
		}
		p.pose = p.pose[:len(p.bones)]
		for i, b := range p.bones {
			f := render.Identity()
			if i < len(finals) {
				f = finals[i]
			}
			p.pose[i] = b.Mul(f)
		}
		k.SetPose(c.World, p.pose)
		ref := float32(-1)
		if p.cut {
			ref = 0.5
		}
		k.Draw(p.mesh, p.tex, ref)
	}
}
