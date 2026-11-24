package scene

import (
	"fmt"
	"math"
	"path"
	"slices"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/skilltiming"
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
	CloakEmblem                                      [6]uint8
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

// ScaleMatrix applies the height factor to all three axes: Game.exe 0x415248
// pushes the same value three times into the model's scaling matrix.
func (a Appearance) ScaleMatrix() render.M4 {
	m, s := render.Identity(), a.HeightScale()
	m[0], m[5], m[10] = s, s, s
	return m
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
	unlit    bool        // native cloak shoulder disables lighting
	twoSided bool        // draw thin shells and cloak shoulders from either side
	shoulder bool        // the cloak's skinned shoulder; hidden with the cloak
	pose     []render.M4
}

// Character is a posed, animated actor.
type Character struct {
	weaponType           uint8
	combatPoseLeft       float32
	clipFile             string
	Concealed            bool
	FadeOpacity          float32 // replaces the stealth fade when set (spirits)
	Hidden               bool    // shown as another creature instead (Degeneration)
	CorpseFade           float32 // opaque screen-door coverage, 0 visible, 1 gone
	enchantTime          float64
	cloak                *cloakInstance
	rate                 playbackRate
	attachments          []attachment
	actions              []model.Action
	actionsLoaded        bool
	spinLeft             float32
	externalTime         bool
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
	blendFrom            []bonePose
	blendLeft            float32
	blendToFirstFrame    bool
	locomotionBlend      bool
	oneShot              bool
	folder               string
	prefix               string
	assets               *Assets
	Mounted              bool
	hiddenWeaponAction   uint64
	lastFinals           []render.M4
	lastFinalsClip       *Clip
	lastFinalsTime       float32
	poseClip             *Clip
	poseTime             float32
	poseReady            bool
	poseParts            []*skinPart
	poseSources          []int
	poseBones            [][]int
	boundPalettes        []boundPalette
	envelope             characterEnvelope
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
	if c == nil || c.locked || c.locomotionBlend && c.blendLeft > 0 && isLocomotionClip(name) {
		return
	}
	c.setClip(name, false)
}

func isLocomotionClip(name string) bool {
	switch name {
	case "000_normal", "001_walk", "002_run", "003_bstep", "004_lstep", "005_rstep", "006_swnormal", "007_swim":
		return true
	}
	return false
}

func (c *Character) setClip(name string, normalIdle bool) {
	locomotion := isLocomotionClip(name)
	blend := blendSharedBones
	if locomotion && name != "000_normal" {
		blend = blendToFirstKey
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
	switch name {
	case "001_walk", "002_run", "003_bstep", "004_lstep", "005_rstep":
		c.rate.next = rateMove
		defer func() { c.rate.next = rateNone }()
	}
	var file string
	if !normalIdle {
		file = c.locomotionFile(name)
	}
	if file == "" {
		file = fmt.Sprintf("Character/%s/ANI/%s_%s.ANI", c.folder, c.prefix, fileName)
	}
	if c.clipFile == file && !c.oneShot {
		return
	}
	cl, err := c.assets.Clip(file)
	if err != nil {
		c.assets.Missing[err.Error()] = true
		return
	}
	c.clipFile = file
	if name == "018_ladder" {
		c.startClipImmediately(cl, false)
	} else {
		c.startClipWithBlend(cl, false, blend)
		c.locomotionBlend = locomotion
	}
	c.clipName = name
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

// StyleCounts returns how many hair and face styles the body's material lists
// (<prefix>_hair.MLT, <prefix>_face.MLT) define; they differ by race, sex and
// job body and bound what the appearance window may pick.
func (s *Assets) StyleCounts(a Appearance) (hair, face int) {
	folder, prefix := raceFolders[a.Race%4], a.Prefix()
	count := func(part string) int {
		if m, err := s.mlt(folder, prefix, part); err == nil {
			return len(m.Records)
		}
		return 0
	}
	return max(1, count("hair")), max(1, count("face"))
}

// BuildCharacter loads the parts for an appearance with the given clip
// suffix (e.g. "019_select").
func BuildCharacter(s *Assets, a Appearance, clip string) (*Character, error) {
	folder := raceFolders[a.Race%4]
	prefix := a.Prefix()
	c := &Character{folder: folder, prefix: prefix, assets: s}
	add := func(part string, index int) {
		m, err := s.mlt(folder, prefix, part)
		// A hair or face index the body's MLT does not define (or whose mesh
		// is missing) shows style 1 instead of nothing; style 0 if only one.
		if err == nil && (part == "hair" || part == "face") {
			valid := func(i int) bool {
				return i >= 0 && i < len(m.Records) && m.Records[i].Mesh >= 0 && int(m.Records[i].Mesh) < len(m.Meshes)
			}
			if !valid(index) {
				index = min(1, len(m.Records)-1)
			}
		}
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
		// The native character pass draws both sides (0x41520c). A helmet
		// can be an open shell, so culling its reverse faces exposes holes.
		cp.twoSided = part == "helmet"
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
	c.weaponType = a.WeaponType
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
	ani             *model.ANI
	restLocal       []render.M4
	restRotation    [][4]float32
	restTranslation [][3]float32
	rotSorted       []bool
	transSorted     []bool
	reach           []boneReach
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
	c := &Clip{
		ani:         a,
		rotSorted:   make([]bool, len(a.Bones)),
		transSorted: make([]bool, len(a.Bones)),
	}
	// local_i = rest_i * inv(rest_parent) (row vectors).
	for i, bone := range a.Bones {
		l := render.M4(bone.Rest)
		if p := bone.Parent; p >= 0 && int(p) < len(a.Bones) {
			l = l.Mul(render.M4(a.Bones[p].Rest).Inverse())
		}
		c.restLocal = append(c.restLocal, l)
		q, t := render.MatrixQuat(l)
		c.restRotation = append(c.restRotation, q)
		c.restTranslation = append(c.restTranslation, t)
		c.rotSorted[i] = rotKeysSorted(bone.Rot)
		c.transSorted[i] = transKeysSorted(bone.Trans)
	}
	c.reach = c.makeReach()
	s.clips[key] = c
	return c, nil
}

// Duration in seconds (30 fps).
func (c *Clip) Duration() float32 {
	return float32(c.ani.End-c.ani.Start) / 30
}

// Finals returns each bone's animated global matrix at time t (looping).
func (c *Clip) Finals(t float32) []render.M4 {
	return c.finalsInto(t, nil)
}

type bonePose struct {
	rotation [4]float32
	position [3]float32
}

func (c *Clip) frame(t float32, oneShot bool) float32 {
	a := c.ani
	if d := c.Duration(); d > 0 {
		if oneShot && t >= d {
			return float32(a.End)
		}
		for t >= d {
			t -= d
		}
	}
	return float32(a.Start) + t*30
}

func (c *Clip) restPose(i int) bonePose {
	if len(c.restRotation) == len(c.ani.Bones) && len(c.restTranslation) == len(c.ani.Bones) {
		return bonePose{c.restRotation[i], c.restTranslation[i]}
	}
	q, t := render.MatrixQuat(c.restLocal[i])
	return bonePose{q, t}
}

func (c *Clip) localPose(i int, frame float32) bonePose {
	pose := c.restPose(i)
	bone := c.ani.Bones[i]
	if len(bone.Rot) > 0 {
		if i < len(c.rotSorted) && c.rotSorted[i] {
			pose.rotation = sampleRotSorted(bone.Rot, frame)
		} else {
			pose.rotation = sampleRot(bone.Rot, frame)
		}
	}
	if len(bone.Trans) > 0 {
		if i < len(c.transSorted) && c.transSorted[i] {
			pose.position = sampleTransSorted(bone.Trans, frame)
		} else {
			pose.position = sampleTrans(bone.Trans, frame)
		}
	}
	return pose
}

func poseBuffer(out []render.M4, count int) []render.M4 {
	if cap(out) < count {
		return make([]render.M4, count)
	}
	return out[:count]
}

func composeBone(out []render.M4, i int, parent int32, pose bonePose) {
	transform := render.QuatMatrix(pose.rotation, pose.position)
	if parent >= 0 && int(parent) < i {
		transform = transform.MulAffine(out[parent])
	}
	out[i] = transform
}

// finalsInto writes into actor-owned storage; shared clips retain no mutable pose.
func (c *Clip) finalsInto(t float32, out []render.M4) []render.M4 {
	frame := c.frame(t, false)
	out = poseBuffer(out, len(c.ani.Bones))
	for i, bone := range c.ani.Bones {
		composeBone(out, i, bone.Parent, c.localPose(i, frame))
	}
	return out
}

func rotKeysSorted(k []model.RotKey) bool {
	if len(k) <= 8 {
		return false
	}
	for i := 1; i < len(k); i++ {
		if k[i].Frame < k[i-1].Frame || float32(k[i].Frame) < float32(k[i-1].Frame) {
			return false
		}
	}
	return true
}

func transKeysSorted(k []model.TransKey) bool {
	if len(k) <= 8 {
		return false
	}
	for i := 1; i < len(k); i++ {
		if k[i].Frame < k[i-1].Frame || float32(k[i].Frame) < float32(k[i-1].Frame) {
			return false
		}
	}
	return true
}

func sampleRot(k []model.RotKey, f float32) [4]float32 {
	if f <= float32(k[0].Frame) {
		return k[0].Q
	}
	if f >= float32(k[len(k)-1].Frame) {
		return k[len(k)-1].Q
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

func sampleRotSorted(k []model.RotKey, f float32) [4]float32 {
	if f <= float32(k[0].Frame) {
		return k[0].Q
	}
	if f >= float32(k[len(k)-1].Frame) {
		return k[len(k)-1].Q
	}
	lo, hi := 1, len(k)-1
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if f <= float32(k[mid].Frame) {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	a, b := k[lo-1], k[lo]
	t := (f - float32(a.Frame)) / float32(b.Frame-a.Frame)
	return render.Slerp(a.Q, b.Q, t)
}

func sampleTrans(k []model.TransKey, f float32) [3]float32 {
	if f <= float32(k[0].Frame) {
		return k[0].T
	}
	if f >= float32(k[len(k)-1].Frame) {
		return k[len(k)-1].T
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

func sampleTransSorted(k []model.TransKey, f float32) [3]float32 {
	if f <= float32(k[0].Frame) {
		return k[0].T
	}
	if f >= float32(k[len(k)-1].Frame) {
		return k[len(k)-1].T
	}
	lo, hi := 1, len(k)-1
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if f <= float32(k[mid].Frame) {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	a, b := k[lo-1], k[lo]
	t := (f - float32(a.Frame)) / float32(b.Frame-a.Frame)
	return [3]float32{a.T[0] + (b.T[0]-a.T[0])*t, a.T[1] + (b.T[1]-a.T[1])*t, a.T[2] + (b.T[2]-a.T[2])*t}
}

// Update advances the animation.
func (c *Character) Update(dt float32) {
	c.combatPoseLeft = max(0, c.combatPoseLeft-dt)
	c.enchantTime += float64(max(0, dt))
	playback := c.advanceBlend(dt)
	if c.externalTime {
		c.externalTime = false
	} else {
		c.Time += playback * c.rate.factor()
	}
	if c.spinLeft > 0 {
		c.spinLeft = max(0, c.spinLeft-dt)
		if c.spinLeft == 0 {
			c.ReleaseAction()
		} else if c.clip != nil && c.clip.Duration() > 0 {
			c.Time = float32(math.Mod(float64(c.Time), float64(c.clip.Duration())))
			return
		}
	}
	if c.cloak != nil {
		c.cloak.prepared = false
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
			c.Time = d
			if !c.hold {
				c.locked = false
				switch {
				case c.actionSlot == 8:
					// ALT 8 always ends in normal idle before locomotion.
					c.setClip("000_normal", true)
				case c.actionSlot == 10 || c.actionSlot == 11 ||
					c.actionSlot >= 13 && c.actionSlot <= 15 ||
					c.actionSlot >= 116 && c.actionSlot <= 125:
					if action := c.actions[c.actionSlot]; uint8(action.Mode) == 2 {
						switch action.Next {
						case 0:
							c.setClip("000_normal", true)
						case 12:
							c.setClip("012_sit", false)
						}
					}
				}
			}
		}
	}
}

// PlayBasicAttack alone refreshes the native weapon-ready posture. Spells
// can share an attack ALT slot without changing this basic-attack timer.
func (c *Character) PlayBasicAttack(weapon, variant uint8) bool {
	c.rate.next = rateAttack
	defer func() { c.rate.next = rateNone }()
	if !c.PlayAction(AttackAction(weapon, variant), false) {
		return false
	}
	c.combatPoseLeft = 10
	return true
}

// PlayAction uses the original ALT index. Death holds the final frame;
// other confirmed one-shots return to locomotion after their duration.
// A running jump keeps its clip until it finishes or the actor dies.
func (c *Character) PlayAction(slot int, hold bool) bool {
	if c == nil || slot != 9 && (c.hold || c.PlayingAction(8)) {
		return false
	}
	c.loadActions()
	if slot < 0 || slot >= len(c.actions) || c.actions[slot].Name == "" {
		return false
	}
	file := fmt.Sprintf("Character/%s/ANI/%s", c.folder, path.Base(c.actions[slot].Name))
	cl, err := c.assets.Clip(file)
	if err != nil {
		c.assets.Missing[err.Error()] = true
		return false
	}
	c.spinLeft, c.externalTime = 0, false
	c.clipFile = file
	if slot == 8 {
		// PS107 and EP5.4 activate the jump directly and cancel any blend.
		c.startClipImmediately(cl, true)
	} else {
		c.startClipWithBlend(cl, true, blendSharedBones)
	}
	c.clipName = file
	c.actionID++
	c.actionSlot = slot
	c.castLeft = 0
	c.locked, c.hold = true, hold
	// A one-shot whose ALT Next is a weapon ready stance (every melee skill and
	// swing) ends by blending into the equipped weapon's stance, not into the
	// relaxed idle. The ALT names one stance per class, whatever weapon is held.
	if a := c.actions[slot]; a.Mode&0xff == 2 && isReadySlot(int(a.Next)) {
		c.combatPoseLeft = 10
	}
	return true
}

// ActionSeconds is the length of an ALT slot's clip at its authored 30 fps,
// or zero when the slot or its file is missing.
func (c *Character) ActionSeconds(slot int) float32 {
	if c == nil {
		return 0
	}
	c.loadActions()
	if slot < 0 || slot >= len(c.actions) || c.actions[slot].Name == "" {
		return 0
	}
	cl, err := c.assets.Clip(fmt.Sprintf("Character/%s/ANI/%s", c.folder, path.Base(c.actions[slot].Name)))
	if err != nil || cl == nil {
		return 0
	}
	return cl.Duration()
}

func (c *Character) ReleaseAction() {
	if c != nil {
		c.spinLeft, c.externalTime = 0, false
		c.actionID++
		c.locked, c.hold = false, false
		c.castLeft = 0
	}
}

// ReleaseActionExceptJump releases a pose that a late network result made
// obsolete. The jump clip (ALT slot 8) always runs to its end.
func (c *Character) ReleaseActionExceptJump() {
	if c != nil && !c.PlayingAction(8) {
		c.ReleaseAction()
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

// ActionFinished reports that the current one-shot clip reached its last frame.
func (c *Character) ActionFinished() bool {
	return c != nil && c.locked && c.clip != nil && c.Time >= c.clip.Duration()
}

func (c *Character) fadeOpacity() float32 {
	if c.FadeOpacity > 0 {
		return c.FadeOpacity
	}
	return render.StealthOpacity
}

func (c *Character) PlayingAction(slot int) bool {
	return c != nil && c.locked && c.actionSlot == slot
}

// ActionPlaybackStarted reports when the one-shot's entry blend has finished.
func (c *Character) ActionPlaybackStarted(slot int) bool {
	return c.PlayingAction(slot) && c.blendLeft <= 0
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

// ContactTimes returns the native first and second contact markers
// (Metadata[2], Metadata[3], 30 phases/s, 0 when absent) of the playing action.
func (c *Character) ContactTimes() (first, second float32) {
	if c.actionSlot < 0 || c.actionSlot >= len(c.actions) || c.clip == nil || len(c.actions[c.actionSlot].Metadata) < 4 {
		return 0, 0
	}
	m := c.actions[c.actionSlot].Metadata
	d := c.clip.Duration()
	if m[2] > 0 {
		first = min(m[2]/30, d)
	}
	if m[3] > 0 {
		second = min(m[3]/30, d)
	}
	return
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
		if c.blendLeft > 0 || c.oneShot {
			frame := c.clip.frame(c.Time, c.oneShot)
			c.lastFinals = poseBuffer(c.lastFinals, len(c.clip.ani.Bones))
			for i, bone := range c.clip.ani.Bones {
				composeBone(c.lastFinals, i, bone.Parent, c.localPose(i, frame))
			}
		} else {
			c.lastFinals = c.clip.finalsInto(c.Time, c.lastFinals)
		}
		c.lastFinalsClip, c.lastFinalsTime = c.clip, c.Time
	}
	return c.lastFinals
}

func (c *Character) preparePose() []render.M4 {
	if c == nil || c.clip == nil {
		return nil
	}
	finals := c.finals()
	// Parts have immutable inverse binds. Reuse a palette only when every bind
	// matrix matches exactly; different equipment/skeletons keep their own pose.
	if !slices.Equal(c.poseParts, c.parts) {
		c.poseParts = append(c.poseParts[:0], c.parts...)
		c.poseSources = make([]int, len(c.parts))
		c.poseBones = make([][]int, len(c.parts))
		c.boundPalettes = make([]boundPalette, len(c.parts))
		for i, p := range c.parts {
			c.poseSources[i] = i
			for j := 0; j < i; j++ {
				if slices.Equal(p.bones, c.parts[j].bones) {
					c.poseSources[i] = c.poseSources[j]
					c.boundPalettes[c.poseSources[i]].shared = true
					break
				}
			}
		}
		c.findUsedPoseBones()
		c.poseReady = false
	}
	if !c.poseReady || c.poseClip != c.clip || c.poseTime != c.Time {
		for partIndex, p := range c.parts {
			if source := c.poseSources[partIndex]; source != partIndex {
				p.pose = c.parts[source].pose
				continue
			}
			if cap(p.pose) < len(p.bones) {
				p.pose = make([]render.M4, len(p.bones))
			}
			p.pose = p.pose[:len(p.bones)]
			if used := c.poseBones[partIndex]; len(used) != 0 {
				for _, i := range used {
					f := render.Identity()
					if i < len(finals) {
						f = finals[i]
					}
					p.pose[i] = p.bones[i].MulAffine(f)
				}
				continue
			}
			for i, b := range p.bones {
				f := render.Identity()
				if i < len(finals) {
					f = finals[i]
				}
				p.pose[i] = b.MulAffine(f)
			}
		}
		c.poseClip, c.poseTime, c.poseReady = c.clip, c.Time, true
	}
	return finals
}

// A palette can serve several pieces with identical inverse binds. Keep the
// union of their weighted bones and fall back if any piece has unknown bounds.
// The skeleton still computes every animated joint for hierarchy and equipment.
func (c *Character) findUsedPoseBones() {
	type paletteUsage struct {
		used    [render.MaxBones]bool
		unknown bool
	}
	usage := make([]paletteUsage, len(c.parts))
	for i, part := range c.parts {
		palette := &usage[c.poseSources[i]]
		if !part.bounded || len(part.vertices) == 0 {
			palette.unknown = true
			continue
		}
		for bone, box := range part.bounds {
			palette.used[bone] = palette.used[bone] || box.used
		}
	}
	for i, part := range c.parts {
		palette := &usage[i]
		if i != c.poseSources[i] || palette.unknown {
			continue
		}
		for bone := range min(len(part.bones), render.MaxBones) {
			if palette.used[bone] {
				c.poseBones[i] = append(c.poseBones[i], bone)
			}
		}
		if len(c.poseBones[i]) == len(part.bones) {
			c.poseBones[i] = nil
		}
	}
}

func (c *Character) prepareDraw(s *render.Scene3D) ([]render.M4, bool) {
	finals, visible := c.prepareDrawPose(s.Shadow)
	if !visible {
		return nil, false
	}
	if lo, hi, valid := c.drawOcclusionBounds(); valid && s.OccludedBox(lo, hi) {
		return nil, false
	}
	return finals, true
}

func (c *Character) prepareDrawPose(shadow bool) ([]render.M4, bool) {
	if c == nil || c.clip == nil || c.Hidden || c.CorpseFade >= 1 || (c.Concealed && shadow) {
		return nil, false
	}
	finals := c.preparePose()
	if c.cloak != nil {
		if !c.cloak.prepared {
			c.cloak.update(c.World, finals)
		}
		c.cloak.prepared = false
	}
	return finals, true
}

func (c *Character) drawOcclusionBounds() (render.V3, render.V3, bool) {
	lo, hi, valid := c.WorldBounds()
	if valid {
		if c.cloak != nil && (c.cloak.reused || c.cloak.usingGrid) {
			unionTransformedBox(&lo, &hi, &valid, c.cloak.min, c.cloak.max, c.World)
		} else if c.cloak != nil {
			for _, p := range c.cloak.physics.positions {
				for axis := 0; axis < 3; axis++ {
					lo[axis] = min(lo[axis], p[axis])
					hi[axis] = max(hi[axis], p[axis])
				}
			}
		}
	}
	return lo, hi, valid
}

func (c *Character) Draw(k *render.Skinner, s *render.Scene3D) {
	finals, visible := c.prepareDraw(s)
	if !visible {
		return
	}
	ambient := float32(.8) // Phoenix account_ep5_render
	if s.CharacterAmbient > 0 {
		ambient = s.CharacterAmbient
	}
	k.Begin(s, ambient)
	k.SetWorld(c.World)
	if c.CorpseFade != 0 {
		k.SetCorpseFade(c.CorpseFade)
	}
	if c.Concealed {
		k.SetOpacity(c.fadeOpacity())
	}
	paletteSource := -1
	for i, p := range c.parts {
		if p.shoulder && c.Concealed {
			continue // concealed characters show no cloak, emblem or shoulder
		}
		if source := c.poseSources[i]; source != paletteSource {
			k.SetPalette(p.pose)
			paletteSource = source
		}
		ref := float32(-1)
		if p.cut {
			ref = 0.5
		}
		if p.unlit {
			k.SetAmbient(1)
		}
		if p.twoSided {
			k.DrawTwoSided(p.mesh, p.tex, ref)
		} else {
			k.Draw(p.mesh, p.tex, ref)
		}
		if p.unlit {
			k.SetAmbient(ambient)
		}
	}
	// A concealed character (stealth, GM char mode) shows neither its cloak nor
	// the emblem printed on it; only the faded body remains.
	if c.cloak != nil && !c.Concealed {
		c.cloak.draw(s, c.World, false)
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

// PlaySkillAction preserves the native two-second, repeating spin state.
// Any new action replaces it immediately; it is not an uninterruptible hold.
func (c *Character) PlaySkillAction(id uint16, slot int) bool {
	if !c.PlayAction(slot, false) {
		return false
	}
	if id == 38 || id == 156 {
		c.spinLeft = float32(skilltiming.SpinDuration.Seconds())
	}
	return true
}
func (c *Character) SetActionFrame(frame float32) {
	if c != nil && c.locked {
		c.Time = frame / 30
		c.externalTime = true
	}
}
