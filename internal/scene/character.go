package scene

import (
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Appearance selects a character's parts (FUN_00417220): race, gender,
// job, face, hair, height and the Model byte of the equipped helmet,
// upper, lower, hands and feet (0 = unequipped).
type Appearance struct {
	Race, Gender, Job                                uint8
	Face, Hair                                       uint8
	Height                                           uint8
	Helmet                                           int // -1 = none, else Model
	Upper, Lower                                     uint8
	Hand, Foot                                       uint8
	ShowHelmet                                       bool
	WeaponType, WeaponModel, ShieldType, ShieldModel uint8
	WeaponEnchant, OffhandEnchant                    uint8
	CloakType, CloakModel, CloakFit                  uint8
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
	vertices []render.SkinVertex // retained for posed shadow bounds
	bounds   []boneBounds
	bounded  bool
	top      render.SkinVertex
	min, max render.V3
	mesh     *render.SkinMesh
	tex      *render.Texture
	bones    []render.M4 // B (inverse bind) per bone
	cut      bool        // MLT Visibility mode: alpha test
	pose     []render.M4
}

// Character is a posed, animated actor.
type Character struct {
	enchantTime          float64
	cloak                *cloakInstance
	attachments          []attachment
	actions              []model.Action
	actionsLoaded        bool
	locked, hold         bool
	actionID             uint64
	actionSlot           int
	castLeft             float32
	castTotal            float32
	castLoop             int
	parts                []*skinPart
	head                 *skinPart
	clip                 *Clip
	clipName             string
	folder               string
	prefix               string
	assets               *Assets
	Mounted              bool
	lastFinals           []render.M4
	lastFinalsClip       *Clip
	lastFinalsTime       float32
	poseClip             *Clip
	poseTime             float32
	poseReady            bool
	boundClip            *Clip
	boundTime            float32
	boundWorld           render.M4
	boundMounted         bool
	boundMin             render.V3
	boundMax             render.V3
	boundReady           bool
	boundValid           bool
	shadowScratch        []render.M4 // reused while fitting this character's shadow
	shadowFit            characterShadowFit
	shadowFitClip        *Clip
	shadowFitTime        float32
	shadowFitWorld       render.M4
	shadowFitReady       bool
	shadowVertices       []render.Vertex
	shadowIndices        []uint32
	shadowSeen           map[int]bool
	shadowReceiverWorld  *World
	shadowReceiverBounds [5]float32
	shadowReceiverReady  bool
	Time                 float32
	World                render.M4
}

// SetClip switches to <prefix>_<name>.ANI (e.g. "000_normal", "002_run"),
// restarting it; unknown clips keep the current one.
func (c *Character) SetClip(name string) {
	if c == nil || c.locked || c.clipName == name {
		return
	}
	fileName := name
	// The four original Vile idle assets spell "nomal" (also in their ALT).
	if c.folder == "Vile" && name == "000_normal" {
		fileName = "000_nomal"
	}
	// Several original swim-idle files contain race/sex-specific typos.
	if name == "006_swnormal" {
		if c.folder == "Elf" && strings.HasPrefix(c.prefix, "elm") {
			fileName = "006_swmormal"
		} else if c.folder == "Vile" && strings.HasPrefix(c.prefix, "viw") {
			fileName = "006_swnomal"
		}
	}
	cl, err := c.assets.Clip(fmt.Sprintf("Character/%s/ANI/%s_%s.ANI", c.folder, c.prefix, fileName))
	if err != nil {
		return
	}
	c.clip, c.clipName, c.Time = cl, name, 0
	c.actionID++
}

// SetMountedClip uses the standing-board animations when present. Some
// character variants do not ship them, so retain the regular riding clips.
func (c *Character) SetMountedClip(moving, alternate bool) {
	name := "021_veh_br"
	if moving {
		name = "020_veh_run"
	}
	if alternate {
		alt := "097_veh_br"
		if moving {
			alt = "022_veh_run"
		}
		if _, ok := c.assets.Arc.Lookup(fmt.Sprintf("Character/%s/ANI/%s_%s.ANI", c.folder, c.prefix, alt)); ok {
			name = alt
		}
	}
	c.SetClip(name)
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
			cp.tex = s.modelTexture(fmt.Sprintf("Character/%s/DDS/%s", folder, DDSName(path.Base(m.Textures[r.Texture]))))
		}
		cp.cut = r.Alpha == 0
		c.parts = append(c.parts, &cp)
		if part == "face" {
			c.head = &cp
		}
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
	c.attachCloak(a)
	c.attachItem(a.WeaponType, a.WeaponModel, false, a.WeaponEnchant)
	c.attachItem(a.ShieldType, a.ShieldModel, true, a.OffhandEnchant)
	c.SetClip(clip)
	if c.clip == nil {
		return c, fmt.Errorf("clip %s_%s not found", prefix, clip)
	}
	return c, nil
}

func (s *Assets) skinPart(folder, mesh string) (*skinPart, error) {
	file := fmt.Sprintf("Character/%s/3DC/%s", folder, path.Base(strings.ReplaceAll(mesh, `\`, "/")))
	return s.skinFile(file)
}

func (s *Assets) skinFile(file string) (*skinPart, error) {
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
	p := &skinPart{mesh: render.NewSkinMesh(v, idx), vertices: v}
	if len(v) > 0 {
		p.min, p.max = v[0].Pos, v[0].Pos
		p.top = v[0]
		for _, vertex := range v {
			if vertex.Pos[1] > p.top.Pos[1] {
				p.top = vertex
			}
			for axis := 0; axis < 3; axis++ {
				p.min[axis] = min(p.min[axis], vertex.Pos[axis])
				p.max[axis] = max(p.max[axis], vertex.Pos[axis])
			}
		}
	}

	for _, b := range m.Bones {
		p.bones = append(p.bones, render.M4(b))
	}
	p.bounds, p.bounded = makeBoneBounds(v, len(p.bones))
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
func (c *Character) Update(dt float32) {
	c.enchantTime += float64(max(0, dt))
	c.Time += dt
	if c.cloak != nil {
		c.cloak.elapsed += max(0, dt)
	}
	if c.castLeft > 0 {
		c.castLeft = max(0, c.castLeft-dt)
		if c.castLeft == 0 {
			c.ReleaseAction()
		} else if c.clip != nil && c.Time >= c.clip.Duration() {
			left, loop := c.castLeft, c.castLoop
			if c.PlayAction(loop, false) {
				c.castLeft = left
			}
		}
	}
	if c.locked && c.clip != nil {
		if d := c.clip.Duration(); d > 0 && c.Time >= d {
			if c.hold {
				c.Time = max(0, d-0.001)
			} else {
				c.locked = false
			}
		}
	}
}

// PlayAction uses the original ALT index. Death holds the final frame;
// other confirmed one-shots return to locomotion after their duration.
func (c *Character) PlayAction(slot int, hold bool) bool {
	if c == nil || c.hold && slot != 9 {
		return false
	}
	if !c.actionsLoaded {
		c.actionsLoaded = true
		file := fmt.Sprintf("Character/%s/%s_action.alt", c.folder, c.prefix)
		b, err := c.assets.Arc.ReadFile(file)
		if err == nil {
			c.actions, err = model.ParseALT(b)
		}
		if err != nil {
			c.assets.Missing[err.Error()] = true
		}
	}
	if slot < 0 || slot >= len(c.actions) || c.actions[slot].Name == "" {
		return false
	}
	file := fmt.Sprintf("Character/%s/ANI/%s", c.folder, path.Base(c.actions[slot].Name))
	cl, err := c.assets.Clip(file)
	if err != nil {
		c.assets.Missing[err.Error()] = true
		return false
	}
	c.clip, c.clipName, c.Time = cl, file, 0
	c.actionID++
	c.actionSlot = slot
	c.castLeft = 0
	c.locked, c.hold = true, hold
	return true
}

func (c *Character) ReleaseAction() {
	if c != nil {
		c.actionID++
		c.locked, c.hold = false, false
		c.castLeft = 0
	}
}

func (c *Character) PlayCast(ready int, seconds float32) bool {
	if seconds < 0 || math.IsNaN(float64(seconds)) || !c.PlayAction(ready, false) {
		return false
	}
	// A server cast-start has no duration on the wire. If local data gives
	// none, keep preparing until its result/cancellation instead of reusing
	// an instant release that was predicted before the announcement.
	if seconds == 0 {
		seconds = float32(math.Inf(1))
	}
	c.castLeft, c.castTotal, c.castLoop = seconds, seconds, ready+1
	return true
}

func (c *Character) CancelCast() {
	if c != nil && c.castLeft > 0 {
		c.ReleaseAction()
	}
}

func (c *Character) ActionActive() bool { return c != nil && c.locked }

func (c *Character) PlayingAction(slot int) bool {
	return c != nil && c.locked && c.actionSlot == slot
}

func (c *Character) PlayingClip(name string) bool {
	return c != nil && !c.locked && c.clipName == name
}

// The original social gesture clips occupy consecutive ALT slots 116..125.
func (c *Character) PlayingGesture() bool {
	return c != nil && c.locked && c.actionSlot >= 116 && c.actionSlot <= 125
}

// ActionID changes whenever an action is replaced or explicitly released.
func (c *Character) ActionID() uint64 { return c.actionID }

// ContactTime is the first positive ALT contact marker, in seconds.
// Native 0x419107/0x419d26/0x41b79c/0x41ba26 consume Metadata[2:4].
// An absent marker stays absent instead of borrowing another weapon's timing.
func (c *Character) ContactTime() float32 {
	if c.actionSlot < 0 || c.actionSlot >= len(c.actions) || c.clip == nil {
		return 0
	}
	first := float32(0)
	for _, phase := range c.actions[c.actionSlot].Metadata[2:] {
		if phase > 0 && (first == 0 || phase < first) {
			first = phase
		}
	}
	return min(first/30, c.clip.Duration())
}

// CastHand uses the same native hand bindings as weapon attachments. Births
// follow the current ANI pose; existing particles retain their world position.
func (c *Character) CastHand(hand int) (render.M4, bool) {
	if c == nil || c.clip == nil || hand < 0 || hand > 1 {
		return render.M4{}, false
	}
	bones, ok := itemBones[c.prefix]
	if !ok {
		return render.M4{}, false
	}
	finals := c.finals()
	bone := bones[hand]
	if bone < 0 || bone >= len(finals) {
		return render.M4{}, false
	}
	return finals[bone].Mul(c.World), true
}

// Draw poses every part (B_part * F) and draws it.
func (c *Character) finals() []render.M4 {
	if c == nil || c.clip == nil {
		return nil
	}
	if c.lastFinalsClip != c.clip || c.lastFinalsTime != c.Time || c.lastFinals == nil {
		c.lastFinals = c.clip.Finals(c.Time)
		c.lastFinalsClip, c.lastFinalsTime = c.clip, c.Time
	}
	return c.lastFinals
}

func (c *Character) preparePose() []render.M4 {
	if c == nil || c.clip == nil {
		return nil
	}
	finals := c.finals()
	if !c.poseReady || c.poseClip != c.clip || c.poseTime != c.Time {
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
		}
		c.poseClip, c.poseTime, c.poseReady = c.clip, c.Time, true
	}
	return finals
}

func (c *Character) Draw(k *render.Skinner, s *render.Scene3D) {
	if c == nil || c.clip == nil {
		return
	}
	finals := c.preparePose()
	if c.cloak != nil {
		c.cloak.update(c.World, finals)
	}
	if lo, hi, valid := c.WorldBounds(); valid {
		if c.cloak != nil {
			for _, p := range c.cloak.physics.positions {
				for axis := 0; axis < 3; axis++ {
					lo[axis] = min(lo[axis], p[axis])
					hi[axis] = max(hi[axis], p[axis])
				}
			}
		}
		if s.OccludedBox(lo, hi) {
			return
		}
	}
	k.Begin(s, 0.8) // character ambient (Phoenix account_ep5_render)
	for _, p := range c.parts {
		k.SetPose(c.World, p.pose)
		ref := float32(-1)
		if p.cut {
			ref = 0.5
		}
		k.Draw(p.mesh, p.tex, ref)
	}
	if c.cloak != nil {
		c.cloak.draw(s, c.World)
	}
	if !c.Mounted {
		c.drawAttachments(s, finals)
	}
}

// PortraitFrame fits the cached body bounds, including tall character families.
// Attachments and animated excursions are not included; callers keep a minimum distance.
func (c *Character) PortraitFrame(aspect, fov, heightScale float32) (height, distance float32) {
	if c == nil || len(c.parts) == 0 {
		return 1, 0
	}
	lo, hi := c.parts[0].min, c.parts[0].max
	for _, p := range c.parts {
		for axis := 0; axis < 3; axis++ {
			lo[axis] = min(lo[axis], p.min[axis])
			hi[axis] = max(hi[axis], p.max[axis])
		}
	}
	height = max(1, (lo[1]+hi[1])*heightScale/2)
	halfY := (hi[1] - lo[1]) * heightScale / 2
	halfX := (hi[0] - lo[0]) / 2
	depth := (hi[2] - lo[2]) / 2
	tangent := float32(math.Tan(float64(fov) / 2))
	return height, max(halfY/tangent, halfX/(tangent*aspect))*1.1 + depth
}

// InventoryFrame fills the full-body opening vertically. Bind-pose arm span
// must not shrink the idle actor, and equipment must not change camera zoom.
func (c *Character) InventoryFrame(fov, heightScale float32) (center, distance float32) {
	if c == nil || len(c.parts) == 0 {
		return 1, 3
	}
	lo, hi := c.parts[0].min[1], c.parts[0].max[1]
	for _, p := range c.parts {
		lo = min(lo, p.min[1])
		hi = max(hi, p.max[1])
	}
	center = (lo + hi) * heightScale / 2
	distance = (hi - lo) * heightScale / 2 / float32(math.Tan(float64(fov)/2)) * 1.03
	return
}

// CastTiming shares the actor's server-started preparation clock with the UI.
func (c *Character) CastTiming() (remaining, total float32, active bool) {
	if c == nil || c.castLeft <= 0 {
		return 0, 0, false
	}
	return c.castLeft, c.castTotal, true
}
