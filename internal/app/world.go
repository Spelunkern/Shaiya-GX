package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/gm"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// worldView is the in-game 3D view: map, own character and the native
// third-person camera (0x444b10 reset, 0x4397d0 orbit/zoom, 0x43bfa0
// movement; Phoenix tools/reference/ep5_camera_controls.md).
type worldView struct {
	draws                       worldDrawBreakdown
	dashes                      map[uint32]proto.SkillDash
	requestedSkills             []*requestedSkillPresentation
	ladderFinishing             bool
	ladder                      mapgeometry.LadderMotion
	ladderSendAt, ladderRetryAt float64
	ladderSentMode              uint8
	slope                       mapgeometry.SlopeMotion
	collisionView               bool
	groundFall                  float32
	requestedSkill              *requestedSkillPresentation
	releaseEffects              []skillRelease
	death                       deathView
	localActions                [ui.TotalActionSlots]uint8 // zero means empty; otherwise panel index + 1
	localActionDrag             subActionDrag
	shadow                      *render.ShadowMap
	shadowAttempted             bool
	waterReflection             *render.WaterTarget
	waterWidth, waterHeight     int
	waterFailed                 bool
	sunFlare                    sunFlare
	weather                     weatherView
	mapView                     mapView
	portrait                    *render.Texture
	portraitAppearance          scene.Appearance
	portraitLoaded              bool
	dead                        bool
	uiLeft                      int
	uiRight                     bool
	npcs                        map[uint32]*npcActor
	mapID                       uint16
	actors                      map[uint32]*remoteActor
	visiblePlayers              []*scene.Character
	visiblePlayerMounts         []*scene.Vehicle
	visiblePlayerCloth          []*scene.Character
	crowdCharacters             []*scene.Character
	crowdBatch                  scene.CharacterBatch
	mobs                        map[uint32]*mobActor
	transforms                  map[uint32]*transformActor
	world                       *scene.World
	self                        *scene.Character
	selfMount                   *scene.Vehicle
	selfMountShape              uint8
	selfMountRace               uint8
	selfMountLoaded             bool
	pos                         render.V3
	visualCorrectionOffset      render.V3
	visualCorrectionLeft        float32
	movementRemainder           [2]float64
	facing                      render.V3 // logical heading; diagonals must not feed back into this
	visualFacing                render.V3 // rendered diagonal heading, zero means logical heading
	combatHeading               render.V3 // heading to restore after facing an attack target
	diagonalHeading             render.V3 // W+Q/W+E heading latched when the combo began (0x43bfa0, 0x71d08c)
	eye                         render.V3 // desired eye; follows the position delta
	moving                      bool
	walking                     bool
	autoRun                     bool
	travel                      bool
	approachTarget              uint32
	worldPressActed             bool    // the world already acted on this left press
	stepDT                      float64 // duration of the walking step being solved
	engagedID                   uint32  // target of the last basic attack sent by the chase
	engagedAt                   float64 // when it was sent
	approachSkill               []byte
	approachLearned             proto.LearnedSkill
	approachLoot                bool
	approachNPC                 bool
	destination                 render.V3
	travelEffect                *scene.Effect
	firstPerson                 bool
	extendedCameraZoom          bool
	mouselook                   bool
	cameraReady                 bool
	cameraEye, cameraTarget     render.V3
	shake                       float32 // seconds of camera shake left (0x54fe20 / 0x54fe70)
	shakeOffset                 render.V3
	firstDirection              render.V3
	attackMoveUntil             float64
	attackPose                  bool
	attackActionID              uint64
	jumping                     bool
	jumpPending                 bool // jump clip started, impulse not yet applied
	jumpActive, jumpMoved       bool // track the whole jump, including its landing frames
	landingAction               landingAction
	jumpFall                    float32
	jumpFloor                   float32
	partySelection              uint32
	controlTime                 float64
	forwardRelease              float64
	forwardReleased             bool
	backRelease                 float64
	backReleased                bool
	evading                     bool
	evadeAction                 int
	evadeDirection              render.V3
	evadeTime                   float32
	sideRelease                 [2]float64
	sideReleased                [2]bool
	sitPending                  bool
	sitDeadline                 float64
	sitAttack                   uint32
	selfGestureSerial           uint64
	waterDirty                  bool
	waterPresentation           waterPresentation
	waterFallSpeed              float32
	sendT                       float64
	motion                      uint8
	reviveAt                    float64
	effect                      *scene.Effect
	runEffects                  []runTrail
	runEffectAt                 map[uint32]float64
	castEffects                 []castEffect
	effectUpdates               []scene.EffectUpdate
	visibleEffects              []*scene.Effect
	projectiles                 []skillProjectile
	physicalProjectiles         []physicalProjectile
	physicalSkillQueued         bool
	impacts                     []skillImpact
	areaShowers                 []skillAreaShower
	impactCues                  []skillImpactCue
	attackSounds                []attackSound
	effectErrors                map[string]bool
	groundBatches               [4][]render.M4
	groundColors                [4][][4]float32
	groundStart                 map[uint32]float64
}

type worldDrawBreakdown struct {
	WorldGeometry int `json:"main_map_geometry"`
	Reflected     int `json:"reflection_pass"`
	WorldScene    int `json:"total_world_scene"`
	UI            int `json:"ui"`
}

const (
	runSpeed   = 5.0 // native normal speed base
	zoomMin    = 2.1 // distance to the character origin
	zoomMax    = 13.0
	camArrowYw = 150.0 * math.Pi / 180
	camArrowPt = 80.0 * math.Pi / 180
)

func (a *App) enterWorld() {
	a.releaseWorld()
	st := a.state
	v := &worldView{mapID: st.MapID, pos: render.V3{st.Self.X, st.Self.Y, st.Self.Z}}
	w, err := scene.LoadMap(a.assets, st.MapID)
	if err != nil {
		log.Printf("map %d: %v", st.MapID, err)
	}
	v.world = w
	if w != nil && st.PhysicsMapID == st.MapID && !a.physicsClockAt.IsZero() {
		w.SyncPhysicsClock(st.PhysicsTime + time.Since(a.physicsClockAt).Seconds())
	}
	// Facing from the 0x0105 direction (u16 angle); default +Z.
	ang := float64(st.Self.Dir) / 65536 * 2 * math.Pi
	v.facing = render.V3{float32(math.Sin(ang)), 0, float32(math.Cos(ang))}
	a.refreshSelf(v)
	for f := range a.assets.Missing {
		log.Printf("map %d: missing %s", st.MapID, f)
		delete(a.assets.Missing, f)
	}
	v.resetCamera()
	a.wv = v
	a.chatBubbles = nil
	a.loadLocalActions()
}

func (v *worldView) target() render.V3 {
	if v.firstPerson {
		return v.eye.Add(v.firstDirection)
	}
	height := float32(.5)
	if v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2]) {
		height = -.2
	}
	return v.cameraOrigin().Add(render.V3{0, height, 0})
}

func (v *worldView) resetCamera() {
	v.firstPerson = false
	v.eye = v.cameraOrigin().Sub(v.facing.Scale(5)).Add(render.V3{0, 1.5, 0})
	v.snapRenderedCamera()
}

// rotateEye turns the eye around the target: yaw about world up, pitch
// about view x up; pitch is rejected when the eye-to-origin direction gets
// within 0.95 of the vertical.
func (v *worldView) rotateEye(yaw, pitch float64) {
	t := v.target()
	off := v.eye.Sub(t)
	c, s := float32(math.Cos(yaw)), float32(math.Sin(yaw))
	off = render.V3{off[0]*c - off[2]*s, off[1], off[0]*s + off[2]*c}
	yawed := t.Add(off)
	if pitch != 0 {
		axis := off.Cross(render.V3{0, 1, 0}).Norm()
		rotated := rotateAxis(off, axis, float32(pitch))
		dir := t.Add(rotated).Sub(v.cameraOrigin()).Norm()
		if dir[1] >= .95 || dir[1] <= -.95 {
			// Native 0x4397d0 rejects the complete orbit at a pole.
			return
		}
		off = rotated
	}
	v.eye = t.Add(off)
	if v.world.HasTerrain() {
		v.eye = cameraOrbitFloor(yawed, v.eye, v.world.Height(v.eye[0], v.eye[2])+.5)
	}
}

func rotateAxis(p, k render.V3, ang float32) render.V3 {
	c, s := float32(math.Cos(float64(ang))), float32(math.Sin(float64(ang)))
	return p.Scale(c).Add(k.Cross(p).Scale(s)).Add(k.Scale(k.Dot(p) * (1 - c)))
}

func (a *App) updateWorld(in *platform.Input, dt float64) {
	// Phoenix limits simulation steps to 1/60s and stalled frames to 250ms.
	// Edge events and physical mouse deltas belong to the first step only.
	if a.wv != nil {
		a.wv.controlTime += max(dt, 0)
	}
	remaining := min(max(dt, 0), .25)
	controls := *in
	for {
		step := min(remaining, 1.0/60)
		remaining -= step
		a.stepWorld(&controls, step, remaining < 1e-8)
		if remaining < 1e-8 || a.wv == nil || a.state == nil {
			a.checkNPCWindowRange()
			return
		}
		controls.Chars, controls.Keys, controls.Actions = nil, nil, nil
		controls.Released = ""
		controls.Pressed, controls.Clicked, controls.RightPressed, controls.MiddlePressed = false, false, false, false
		controls.Wheel, controls.MouseDX, controls.MouseDY = 0, 0, 0
		controls.MouseLook = true
	}
}

func (a *App) updateWorldStep(in *platform.Input, dt float64) {
	if a.wv != nil {
		a.wv.controlTime += max(dt, 0)
	}
	a.stepWorld(in, dt, true)
}

func (a *App) stepWorld(in *platform.Input, dt float64, lastStep bool) {
	v := a.wv
	if v == nil {
		return
	}
	startPos := v.pos
	a.publishPartySelection()
	if a.wv != v {
		return
	}
	if v.extendedCameraZoom && !a.state.Self.IsGM {
		v.restoreCameraZoomLimit()
	}
	dashing := a.updateSkillDashes()
	a.updateQueuedSkill(in)
	if a.wv != v {
		return
	}
	a.updateTargetEffects()
	if a.wv != v {
		return
	}
	settings := a.clientSettings()
	pointer := *in
	if pointer.RightDown {
		dx, dy := float64(pointer.MouseX-a.lastMouseX), float64(pointer.MouseY-a.lastMouseY)
		if pointer.MouseLook {
			dx, dy = pointer.MouseDX, pointer.MouseDY
		}
		sensitivity := float64(settings.MouseSensitivity)
		yawFactor, pitchFactor := (2*sensitivity-1)/5, (2*sensitivity-1)/5
		if v.firstPerson {
			yawFactor, pitchFactor = sensitivity/3, (sensitivity+2)/5
		}
		pointer.MouseDX, pointer.MouseDY = dx*yawFactor, dy*pitchFactor
		if settings.InvertY {
			pointer.MouseDY = -pointer.MouseDY
		}
		pointer.MouseLook = true
	}
	in = &pointer
	v.mouselook = in.RightDown
	yaw, pitch := v.cameraInput(in, dt, a.lastMouseX, a.lastMouseY)
	a.lastMouseX, a.lastMouseY = in.MouseX, in.MouseY
	backEvade := false
	evadeMotion := uint8(0)
	if in.SuppressAutoRun {
		v.forwardReleased = false
		v.backReleased = false
		v.sideReleased = [2]bool{}
	}
	for _, key := range in.Released {
		if key == 'Q' || key == 'E' {
			side := 0
			if key == 'E' {
				side = 1
			}
			if v.sideReleased[side] && v.controlTime-v.sideRelease[side] < .301 {
				evadeMotion = proto.MotionEvadeLeft
				if side == 1 {
					evadeMotion = proto.MotionEvadeRight
				}
				v.sideReleased[side] = false
			} else {
				v.sideRelease[side], v.sideReleased[side] = v.controlTime, true
			}
		}
		if key == 'S' {
			if v.backReleased && v.controlTime-v.backRelease < .301 {
				backEvade = true
				v.backReleased = false
			} else {
				v.backRelease, v.backReleased = v.controlTime, true
			}
		}
		if key == 'W' {
			if v.forwardReleased && v.controlTime-v.forwardRelease < .301 {
				v.autoRun = true
				v.forwardReleased = false
			} else {
				v.forwardRelease, v.forwardReleased = v.controlTime, true
			}
		}
	}
	if in.HasKey(platform.KeyTab) {
		v.walking = !v.walking
	}
	toggleSit := false
	for _, c := range in.Chars {
		if c == 'c' || c == 'C' {
			toggleSit = true
		}
	}
	if a.state.Self.Dead || toggleSit || in.HasKey(platform.KeySpace) || in.Held('W') || in.Held('S') || in.Held('A') || in.Held('D') || in.Held('Q') || in.Held('E') || in.HasKey(platform.KeyNumLock) || in.RightDown && in.MouseDown {
		v.travel = false
	}
	if a.state.Self.Dead || toggleSit || in.Held('S') || backEvade {
		v.autoRun = false
	} else if in.HasKey(platform.KeyNumLock) {
		v.autoRun = !v.autoRun
	}
	intent := characterControls(in, v.autoRun)
	if !a.updateActionApproach(in) {
		return
	}
	// Orbiting the camera alone must not interrupt a target-facing attack or cast.
	combatFacing := v.combatHeading.Len() > 0 && (a.clock < v.attackMoveUntil || v.self.ActionActive())
	mouseSteer := in.RightDown && !combatFacing
	if !in.SuppressAutoRun && (intent.forward || intent.back || intent.side != 0 || intent.turn != 0 || mouseSteer) {
		v.restoreMovementHeading()
	}
	var dir render.V3
	back := intent.back && !intent.forward
	strafe := intent.side
	forward := intent.forward
	speed := playerMovementSpeed(a.state.Self.Speed, a.state.Self.Mounted)
	if a.state.Self.Vehicle.State == proto.VehicleDriver && a.state.Self.Vehicle.Speed == 5 && speed > 0 {
		speed = 8
	}
	if a.state.Self.Vehicle.State == proto.VehiclePassenger {
		forward, back, strafe = false, false, 0
		intent.turn = 0
		v.autoRun = false
		v.travel = false
		speed = 0
		if e := a.state.Entities[a.state.Self.Vehicle.Driver]; e != nil {
			v.pos = render.V3{e.X, e.Y, e.Z}
		}
		if in.HasKey(platform.KeySpace) {
			if err := a.world.Send(proto.VehicleLeavePacket()); err != nil {
				a.backToLogin("Unable to dismount.")
				return
			}
		}
	}
	if a.state.Self.Dead || a.state.Self.Mounted {
		v.sitAttack = 0
	}
	swimming := v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2])
	remaining := float32(math.MaxFloat32)
	if v.travel && !in.SuppressAutoRun {
		delta := v.destination.Sub(v.pos)
		delta[1] = 0
		remaining = delta.Len()
		if remaining < .005 {
			v.travel = false
		}
		forward = v.travel
	}
	wantsMove := forward || back || strafe != 0 || intent.turn != 0 || in.HasKey(platform.KeySpace)
	if !v.jumpActive && !v.sitPending && !a.state.Self.Dead && !a.state.Self.Mounted && !swimming {
		motion := uint8(0)
		request := false
		if a.state.Self.Sitting && (toggleSit || wantsMove || v.sitAttack != 0) {
			motion = proto.MotionStandUp
			request = true
		} else if !a.state.Self.Sitting && toggleSit && !wantsMove {
			motion = proto.MotionSitDown
			request = true
		}
		if request {
			if !a.requestPosture(motion) {
				return
			}
		}
	}
	blocked := dashing || a.state.Self.Sitting || v.sitPending || v.self != nil && v.self.PlayingAction(11)
	if blocked {
		dir = render.V3{}
	} else {
		turnSpeed := speed
		if v.walking {
			turnSpeed /= 2
		}
		if a.state.Self.IsGM && a.state.GMTurbo {
			turnSpeed *= gm.TurboMultiplier
		}
		// EP5.4 0x43f4f5 turns 15 degrees per unit of this step's travel.
		turn := float64(intent.turn) * float64(turnSpeed) * dt * (math.Pi / 12)
		yaw += turn
		v.turnFacing(turn)
	}
	if yaw != 0 || pitch != 0 {
		v.orbitCamera(yaw, pitch)
	}
	if v.facing.Len() == 0 {
		v.facing = render.V3{0, 0, 1}
	}
	fwd := cameraHeading(v.target().Sub(v.eye), v.facing)
	// Original 0x43bfa0: W+Q / W+E build the diagonal from the character's own
	// facing, latched on the first frame and kept while both keys stay down, so
	// the camera can orbit (look behind) without bending the run.
	diagonal := !blocked && !v.travel && !in.SuppressAutoRun && in.Held('W') && intent.side != 0 && !intent.back
	if !diagonal {
		v.diagonalHeading = render.V3{}
	} else if v.diagonalHeading.Len() == 0 {
		v.diagonalHeading = v.facing
	}
	if !diagonal && !blocked && speed > 0 && !a.state.Self.Dead && (mouseSteer || !v.travel && (intent.forward || intent.back)) {
		v.facing = fwd
	}
	right := render.V3{0, 1, 0}.Cross(v.facing)
	if !blocked {
		if diagonal {
			dir = controlVector(v.diagonalHeading, render.V3{0, 1, 0}.Cross(v.diagonalHeading), intent)
		} else {
			dir = controlVector(fwd, right, intent)
		}
		if v.travel && !in.SuppressAutoRun {
			dir = v.destination.Sub(v.pos)
			dir[1] = 0
			remaining = dir.Len()
			if remaining > .005 {
				v.facing = dir.Norm()
			}
		}
	}
	v.visualFacing = render.V3{}
	if v.evading && (v.self == nil || !v.self.PlayingAction(v.evadeAction) || a.state.Self.Dead || blocked || swimming) {
		v.evading = false
	}
	if backEvade {
		evadeMotion = proto.MotionBackstep
	}
	if evadeMotion != 0 && !in.HasKey(platform.KeySpace) && !v.jumpActive && !v.ladder.Active && !v.evading && !blocked && !swimming && !v.jumping && speed > 0 && !a.state.Self.Dead && !a.state.Self.Mounted && v.self != nil {
		action, _ := proto.LocomotionAction(evadeMotion)
		v.evading = v.self.PlayAction(action, false)
		if v.evading {
			v.autoRun, v.travel = false, false
			v.evadeAction, v.evadeTime = action, 0
			v.evadeDirection = fwd.Scale(-1)
			if action == 14 {
				v.evadeDirection = right.Scale(-1)
			} else if action == 15 {
				v.evadeDirection = right
			}
			if err := a.world.Send(proto.MotionPacket(evadeMotion)); err != nil {
				a.backToLogin("Unable to send evasive movement.")
				return
			}
		}
	}
	if v.evading {
		dir = v.evadeDirection
		back, forward = v.evadeAction == 13, false
		if v.evadeAction == 14 {
			strafe = -1
		} else if v.evadeAction == 15 {
			strafe = 1
		}
	}
	climbFrame, climbMoving := a.updateLadderMovement(in, forward, back, strafe, blocked, speed, dt)
	if a.wv != v {
		return
	}
	jumpPressed := in.HasKey(platform.KeySpace)
	canJump := !climbFrame && !blocked && !swimming && !v.jumping && !v.evading && speed > 0 && !a.state.Self.Dead && a.state.Self.Vehicle.State != proto.VehiclePassenger && v.world.PlayerSupported(v.pos)
	if v.jumpPending {
		// One frame after the key: the impulse, and the jump request, begin.
		if !canJump {
			v.jumpPending = false
		} else {
			// MotionJump has no position. Flush unsent walking/running progress
			// first, so the server grants the impulse at the same support surface.
			if v.moving || dir.Len() > 0 {
				if err := a.world.Send(v.movePacket(v.motion &^ proto.MotionFlag80)); err != nil {
					a.backToLogin("Unable to send jump position.")
					return
				}
			}
			v.beginJump()
			if err := a.world.Send(proto.MotionPacket(proto.MotionJump)); err != nil {
				a.backToLogin("Unable to send jump.")
				return
			}
		}
	} else if jumpPressed && canJump && !v.jumpActive {
		v.startJumpAnimation()
		v.approachTarget, v.approachSkill, v.engagedID = 0, nil, 0
		a.skills.queuedSkill = nil
		a.stopCastEffects(a.state.SelfID)
	}
	if a.clock < v.attackMoveUntil {
		dir = render.V3{}
	}
	moving := dir.Len() > 0 && speed > 0 && !a.state.Self.Dead
	if climbFrame {
		moving = climbMoving
	}
	diving := false
	if moving && !climbFrame {
		horizontal := dir.Norm()
		if forward && strafe != 0 || swimming || a.state.Self.Mounted && !back {
			v.visualFacing = horizontal
		}
		dir = horizontal
		if swimming {
			dir = swimDirection(dir, v.target().Sub(v.eye), forward, in.Held('S'))
			diving = dir[1] < -0.02
		}
		if v.evading && v.evadeAction != 13 {
			speed = 3
			if v.evadeTime >= .5 {
				speed = 0
			}
		} else if !forward && strafe != 0 && !a.state.Self.Mounted {
			speed = backwardMovementSpeed(a.state.Self.Speed, a.state.Self.Mounted)
		} else if back {
			if v.evading {
				speed = 2.5
			} else {
				speed = backwardMovementSpeed(a.state.Self.Speed, a.state.Self.Mounted)
			}
		} else if v.walking {
			speed /= 2
		}
		if a.state.Self.IsGM && a.state.GMTurbo {
			speed *= gm.TurboMultiplier
		}
		v.stepDT = min(float64(dt), float64(remaining)/float64(max(speed, .0001)))
		moving = v.step(dir.Scale(min(speed*float32(dt), remaining)))
	}
	if v.evading {
		v.evadeTime += float32(dt)
	}
	if dashing {
		// Server-authored dash already placed the actor.
	} else if climbFrame {
		// Ladder movement already integrated using the shared guide.
	} else if v.jumping {
		if v.advanceJump(float32(dt)) {
			v.waterDirty = true
		}
	} else if v.settleIntoWater(float32(dt)) || v.settleOnGround(float32(dt)) {
		v.waterDirty = true
	}
	swimming = v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2])
	if swimming && in.Space && speed > 0 && !a.state.Self.Dead && v.riseInWater(float32(dt)) {
		v.waterDirty = true
	} else if !diving && v.floatAtSurface(float32(dt)) {
		v.waterDirty = true
	}
	swimming = v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2])
	if swimming && !v.waterPresentation.swimming && v.self != nil && !a.state.Self.Dead && !a.state.Self.Mounted {
		v.self.ReleaseAction()
	}
	a.updateWaterPresentation(float32(dt), moving, swimming)
	airborne := v.jumping || !v.world.PlayerSupported(v.pos)
	if v.visualCorrectionLeft > 0 {
		if airborne || climbFrame || dashing || a.state.Self.Dead {
			v.clearVisualCorrection()
		} else {
			v.advanceVisualCorrection(float32(dt))
		}
	}
	if v.self != nil {
		if v.attackPose && (!v.self.ActionActive() || v.self.ActionID() != v.attackActionID) {
			v.attackPose = false
		}
		if moving && v.attackPose {
			v.self.ReleaseAction()
			v.attackPose = false
		}
		if moving {
			v.self.CancelCast()
			a.stopCastEffects(a.state.SelfID)
		}
		if moving || v.jumping {
			cancelGesture(v.self)
		}
		if a.state.Self.Dead && !v.dead {
			v.self.PlayAction(9, true)
		}
		if !a.state.Self.Dead && v.dead {
			v.self.ReleaseAction()
		}
		v.dead = a.state.Self.Dead
		if v.selfGestureSerial != a.state.Self.GestureSerial {
			v.selfGestureSerial = a.state.Self.GestureSerial
			if action, ok := proto.GestureAction(a.state.Self.Gesture); ok && !a.state.Self.Dead && !a.state.Self.Mounted && !moving && !v.self.PlayingAction(8) {
				v.self.PlayAction(action, false)
			}
		}
		v.self.Mounted = a.state.Self.Mounted
		clip := "000_normal"
		switch {
		case climbFrame && !a.state.Self.Dead:
			clip = "018_ladder"
		case a.state.Self.Mounted:
			v.self.SetMountedClip(moving, scene.VehicleAlternate(a.state.Self.MountShape))
			clip = ""
		case swimming && moving:
			clip = "007_swim"
		case swimming:
			clip = "006_swnormal"
		case a.state.Self.Sitting:
			clip = "012_sit"
		case airborne && moving && v.walking:
			clip = "001_walk"
		case airborne && moving:
			clip = "002_run"
		case moving && back:
			clip = "003_bstep"
		case moving && !forward && strafe < 0:
			clip = "004_lstep"
		case moving && !forward && strafe > 0:
			clip = "005_rstep"
		case moving && v.walking:
			clip = "001_walk"
		case moving:
			clip = "002_run"
		}
		if clip != "" {
			v.self.SetClip(clip)
		}
		jumpAction := v.self.PlayingAction(8)
		previousPoseTime := v.self.Time
		a.setSelfAnimationRates(v.self)
		v.self.Update(float32(dt))
		if climbFrame && !moving && !v.self.ActionActive() {
			v.self.Time = previousPoseTime
		}
		// Drop the finished one-shot before drawing, including after landing.
		if jumpAction && !v.self.ActionActive() && clip != "" {
			v.self.SetClip(clip)
		}
	}
	if v.sitAttack != 0 && !a.state.Self.Sitting && !v.sitPending && (v.self == nil || !v.self.PlayingAction(11)) {
		target := v.sitAttack
		v.sitAttack = 0
		if !wantsMove && a.state.Target == target && !a.attackTarget() {
			return
		}
	}
	// GX sends prediction samples at 10 Hz; the server owns collision and Y.
	v.sendT -= dt
	motion := uint8(0)
	if moving {
		switch {
		case back:
			motion = 2
		case !forward && strafe < 0:
			motion = 3
		case !forward && strafe > 0:
			motion = 4
		case !v.walking || swimming:
			motion = 1
		}
	}
	if climbFrame {
		motion = 0
		if climbMoving {
			motion = 1
			if v.ladder.Mode == mapgeometry.LadderDown {
				motion = 2
			}
		}
	}
	if v.jumping {
		motion = 4 | proto.MotionFlag80
	}
	v.constrainCamera()
	v.shakeOffset = render.V3{}
	if v.shake > 0 {
		// 0x54fe70 offsets the desired camera (one random value in [-0.2, 0.2] per
		// axis, eye and target together), not the drawn one: the follow step 0x550090
		// then eases toward it, which is what keeps the shake subtle. Shake Screen off
		// keeps the timer but adds no offset.
		v.shake = max(0, v.shake-float32(dt))
		if a.clientSettings().ShakeScreen {
			v.shakeOffset = render.V3{rand.Float32()*.4 - .2, rand.Float32()*.4 - .2, rand.Float32()*.4 - .2}
		}
	}
	v.updateRenderedCamera(float32(dt))
	// A slow frame can run several physics steps at once. Periodic packets
	// must represent the position at the end of that frame, when they are sent.
	periodic := lastStep && v.sendT <= 0
	if moving != v.moving || motion != v.motion || ((moving || v.jumping) && periodic) || (v.waterDirty && (periodic || (swimming && v.pos[1] >= scene.SwimSurfaceY))) {
		if err := a.world.Send(v.movePacket(motion)); err != nil {
			a.backToLogin("Unable to send movement.")
			return
		}
		nextSample := 0.1
		if periodic && moving == v.moving && motion == v.motion && (moving || v.jumping) {
			nextSample = max(0, nextSample+v.sendT)
		}
		v.sendT, v.moving, v.motion = nextSample, moving, motion
		v.waterDirty = false
	}
	a.state.Self.X, a.state.Self.Y, a.state.Self.Z = v.pos[0], v.pos[1], v.pos[2]
	if v.jumpActive {
		v.jumpMoved = v.jumpMoved || startPos[0] != v.pos[0] || startPos[2] != v.pos[2]
		if (!v.jumpPending && !airborne && !v.self.PlayingAction(8)) || swimming {
			v.jumpActive = false
		}
	}
	a.updateLandingAction(in)
}

func (a *App) drawWorldView(w, h int) {
	v := a.wv
	if v == nil || v.world == nil || w <= 0 || h <= 0 {
		return
	}
	v.draws = worldDrawBreakdown{}
	drawStart := render.DrawCalls()
	eye, target := v.eye, v.target()
	if v.cameraReady {
		eye, target = v.cameraEye, v.cameraTarget
	}
	s := a.scene3d
	s.StaticObjectLOD = true
	s.ViewportW, s.ViewportH = w, h
	settings := a.clientSettings()
	s.HideWeaponEffects = !settings.SkillEffects
	s.GeometryReduction = 2 - settings.Details
	s.SetCameraView(eye, target, render.V3{0, 1, 0})
	aspect := projectionAspect(w, h)
	distanceScale := []float32{.5, .75, 1}[settings.VisualDistance]
	far := max(float32(200), max(float32(400), v.world.FogEnd)*distanceScale)
	if v.world.WLD != nil && v.world.WLD.Dungeon {
		far = max(far, v.world.WLD.FogEnd*distanceScale)
	}
	// Enclose every frustum corner, including wide viewports and pitched cameras.
	tangent := float32(math.Tan(nativeWorldFOV / 2))
	radius := far * float32(math.Sqrt(float64(1+tangent*tangent*(1+aspect*aspect))))
	if a.world != nil {
		v.world.EnableAsyncStreaming()
	}
	v.world.UpdateVisible(eye, radius)
	switch s.GeometryReduction {
	case 1:
		far = min(far*.75, 260)
	case 2:
		far = min(far*.55, 120)
	}
	s.Proj = render.PerspectiveFovLH(nativeWorldFOV, aspect, 0.1, far)
	s.SkyProj = render.PerspectiveFovLH(nativeWorldFOV, aspect, 10, 100000)
	v.world.PrepareTerrainOcclusion(s, eye)
	forward := target.Sub(eye)
	forward[1] = 0
	if forward.Len() > 0 {
		forward = forward.Scale(20 / forward.Len())
	}
	v.weather.update(a.weatherForView(), float32(a.frameDT), eye.Add(forward).Add(render.V3{0, 15, 0}))
	weatherColor, weatherNear, weatherFar := v.weather.fog(v.world.FogColor, v.world.FogStart*distanceScale, v.world.FogEnd*distanceScale)
	if s.GeometryReduction > 0 {
		weatherFar = min(weatherFar, far)
		weatherNear = min(weatherNear, weatherFar*.65)
	}
	s.FogColor, s.FogStart, s.FogEnd = weatherColor, weatherNear, weatherFar
	if v.weather.blend > 0 {
		render.ClearBackground(weatherColor)
	}
	underwaterColor, underwater := v.world.UnderwaterFogAt(eye)
	if underwater {
		// Phoenix's underwater range; the original's fog curve is not yet known.
		s.FogColor, s.FogStart, s.FogEnd = underwaterColor, 5, 65
		render.ClearBackground(underwaterColor)
	}
	s.LightDir = render.V3{-1, -1, -1}
	s.Begin()
	v.world.Update(float32(a.frameDT))
	a.vehicle.effectBudget = 2
	a.prepareSelfForDraw()
	a.prepareRemotePlayers()
	a.attachVehiclePassengers()
	a.prepareMonsters()
	a.prepareTransforms()
	a.prepareNPCs()
	a.updateRunEffects()
	a.updateCastEffects()
	a.updateTravelEffect(float32(a.frameDT))
	a.updateResolvedEffects()
	if v.effect != nil {
		v.effect.Update(float32(a.frameDT))
	}
	s.WaterReflection = nil
	waterEffect := settings.Water && s.GeometryReduction < 2 && (underwater || eye[1] > 0)
	surfaceVisible, reflectionRegion := v.world.WaterCoverage(s.View, s.Proj, waterEffect && !underwater)
	waterVisible := waterEffect && surfaceVisible
	if waterVisible {
		const reflectionDivisor = 2
		width, height := max(1, w/reflectionDivisor), max(1, h/reflectionDivisor)
		if v.waterReflection != nil && (v.waterWidth != width || v.waterHeight != height) {
			v.waterReflection.Delete()
			v.waterReflection, v.waterFailed = nil, false
		}
		if v.waterReflection == nil && !v.waterFailed {
			var err error
			v.waterReflection, err = render.NewWaterTarget(width, height)
			v.waterWidth, v.waterHeight = width, height
			if err != nil {
				v.waterFailed = true
				log.Printf("water reflection: %v", err)
			}
		}
	}
	beforeReflection := render.DrawCalls()
	if v.waterReflection != nil && waterVisible && !underwater {
		view := s.View
		mirror := render.Identity()
		mirror[5] = -1
		reflectedView := mirror.Mul(view)
		reflectedEye := eye
		reflectedEye[1] = -reflectedEye[1]
		// Widen by two target texels for bilinear filtering and scissor rounding.
		reflectionRegion = reflectionRegion.Expand(2/float32(v.waterWidth), 2/float32(v.waterHeight))
		v.world.PrepareMapEffects(s, eye, scene.EffectCamera{Eye: reflectedEye, View: reflectedView, Projection: s.Proj})
		v.waterReflection.CaptureRegion(s.FogColor, reflectionRegion, func() {
			s.View, s.ClipPlane = reflectedView, [4]float32{0, 1, 0, 0}
			s.CullProjection = reflectionRegion.CropProjection(s.Proj)
			s.Begin()
			a.drawWorldContents(reflectedEye, false)
			v.world.DrawTranslucentVANI(s)
			v.world.RenderMapEffects(s)
		})
		s.CullProjection = render.M4{}
		s.View, s.ClipPlane = view, [4]float32{}
		s.Begin()
		s.WaterReflection = v.waterReflection
	} else if v.waterReflection != nil && waterVisible && underwater {
		v.world.PrepareMapEffects(s, eye)
		fogColor, fogStart, fogEnd := s.FogColor, s.FogStart, s.FogEnd
		s.FogColor, s.FogStart, s.FogEnd = weatherColor, weatherNear, weatherFar
		v.waterReflection.Capture(s.FogColor, func() {
			s.ClipPlane = [4]float32{0, 1, 0, 0}
			s.Begin()
			a.drawWorldContents(eye, false)
			v.world.DrawTranslucentVANI(s)
			v.world.RenderMapEffects(s)
		})
		s.ClipPlane = [4]float32{}
		s.FogColor, s.FogStart, s.FogEnd = fogColor, fogStart, fogEnd
		s.Begin()
		s.WaterReflection, s.WaterBelow = v.waterReflection, true
	} else {
		v.world.PrepareMapEffects(s, eye)
	}
	v.draws.Reflected = render.DrawCalls() - beforeReflection
	a.drawWorldContents(eye, underwater)
	if s.WaterReflection != nil && !s.WaterBelow && v.world.HasWaterNormals() {
		s.CaptureWaterBackground()
	}
	// Water Surface Effect controls reflection/refraction, not water visibility.
	if surfaceVisible {
		v.world.DrawVisibleWater(s)
	}
	beforeVANI := render.DrawCalls()
	v.world.DrawTranslucentVANI(s)
	v.draws.WorldGeometry += render.DrawCalls() - beforeVANI
	for _, e := range v.waterPresentation.effects {
		e.Draw(s)
	}
	v.world.RenderMapEffects(s)
	if !underwater {
		v.weather.draw(s, a.assets, eye)
	}
	s.WaterReflection = nil
	s.WaterBelow = false
	s.End()
	v.sunFlare.update(v.world, eye, s.View, s.SkyProj, far, float32(a.frameDT))
	a.drawCollisionView()
	v.draws.WorldScene = render.DrawCalls() - drawStart
}

func (a *App) drawWorldContents(eye render.V3, underwater bool) {
	v, s := a.wv, a.scene3d
	if !underwater && v.world.Sky != nil {
		v.world.Sky.DrawWeather(s, eye, v.weather.blend)
	}
	beforeMap := render.DrawCalls()
	v.world.DrawOpaque(s)
	if s.ClipPlane == [4]float32{} {
		v.draws.WorldGeometry = render.DrawCalls() - beforeMap
	}
	a.drawGroundItems()
	a.drawSelf()
	a.drawRemotePlayers()
	a.drawMonsters()
	a.drawTransforms()
	a.drawNPCs()
	a.drawRunEffects()
	a.drawCastEffects()
	a.drawResolvedEffects()
	if v.travel && v.travelEffect != nil {
		v.travelEffect.Draw(s)
	}
	if v.effect != nil {
		v.effect.Draw(s)
	}
}

// Prepare the mount and rider exactly once; camera passes can draw this pose
// repeatedly without advancing the mount animation or changing the seat.
func (a *App) prepareSelfForDraw() {
	v := a.wv
	if v.self != nil {
		v.self.Mounted = a.state.Self.Mounted
		if v.self.Mounted {
			v.self.SetMountedClip(v.moving, scene.VehicleAlternate(a.state.Self.MountShape))
		}
		visualPos := v.visualPosition()
		if v.ladder.Active && !a.state.Self.Mounted {
			visualPos = v.world.LadderVisual(visualPos, v.ladder.Index)
		}
		riderPos := visualPos
		heading := v.facing
		if v.visualFacing.Len() > 0 {
			heading = v.visualFacing
		}
		if a.state.Self.Mounted && a.state.Self.Vehicle.State != proto.VehiclePassenger {
			if slot, ok := a.selfSlot(); ok {
				if !v.selfMountLoaded || v.selfMountShape != a.state.Self.MountShape || v.selfMountRace != slot.Race {
					v.selfMountShape, v.selfMountRace, v.selfMountLoaded = a.state.Self.MountShape, slot.Race, true
					var err error
					v.selfMount, err = a.assets.BuildVehicle(slot.Race, v.selfMountShape)
					if err != nil {
						log.Printf("mount: %v", err)
					}
				}
				if v.selfMount != nil {
					v.selfMount.SetMotion(v.moving, v.walking || v.motion&0x3f == 0, v.motion&0x3f == 2, v.jumping)
					v.selfMount.Update(float32(a.frameDT))
					v.selfMount.World = render.Basis(visualPos, heading.Scale(-1), render.V3{0, 1, 0})
					riderPos = v.selfMount.Seat()
					v.selfMount.PrepareEffects(float32(a.frameDT), a.clientSettings().SkillEffects, true, &a.vehicle.effectBudget)
				}
			}
		} else {
			v.selfMount, v.selfMountLoaded = nil, false
		}
		v.self.Concealed = a.state.Self.Concealed
		v.self.World = render.Basis(riderPos, heading.Scale(-1), render.V3{0, 1, 0})
	}
}

func (a *App) drawSelf() {
	v := a.wv
	if v.self == nil || v.firstPerson && !a.scene3d.Shadow && a.scene3d.ClipPlane == ([4]float32{}) {
		return
	}
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
	if a.state.Self.Mounted && v.selfMount != nil {
		v.selfMount.Concealed = a.state.Self.Concealed
		if v.selfMount.Visible(frustum, a.scene3d.ClipPlane) {
			v.selfMount.Draw(a.skinner, a.scene3d)
			v.selfMount.DrawEffects(a.scene3d)
		}
	}
	if v.self.Visible(frustum, a.scene3d.ClipPlane) {
		v.self.Draw(a.skinner, a.scene3d)
	}
	a.drawSelfShadow()
}

var _ = game.PhaseInWorld

// selfSlot combines roster identity with server-confirmed equipment. Both the
// world actor and the inventory preview consume the same appearance source.
func (a *App) selfSlot() (proto.CharacterSlot, bool) {
	for _, slot := range a.state.Slots {
		if slot.ID != a.state.SelfID {
			continue
		}
		if a.state.InventoryReady {
			for i := range slot.EquipType {
				it := a.state.Inventory[[2]uint8{0, uint8(i)}]
				slot.EquipType[i], slot.EquipTypeID[i] = it.Type, it.TypeID
			}
			slot.Cloak = a.state.Inventory[[2]uint8{0, 7}].Gems
		}
		return slot, true
	}
	return proto.CharacterSlot{}, false
}

// Rebuild equipment only from the confirmed inventory, never the local drag.
func (a *App) refreshSelf(v *worldView) {
	slot, ok := a.selfSlot()
	if !ok {
		return
	}
	form := a.state.PlayerForms[a.state.SelfID]
	if form.Kind == proto.FormDisguise {
		sh := form.Shape
		slot.Race, slot.Gender, slot.Class, slot.Face, slot.Hair, slot.Height = sh.Race, sh.Gender, sh.Class, sh.Face, sh.Hair, sh.Height
		for i, item := range sh.Equipment {
			slot.EquipType[i], slot.EquipTypeID[i] = item[0], item[1]
		}
	}
	appearance := a.appearance(&slot)
	if form.Kind == proto.FormDisguise {
		appearance.WeaponEnchant, appearance.OffhandEnchant = form.Shape.Equipment[5][2], form.Shape.Equipment[6][2]
	}
	ch, err := scene.BuildCharacter(a.assets, appearance, "000_normal")
	if err != nil {
		log.Printf("self character: %v", err)
	}
	if ch != nil && a.state.Self.Dead {
		ch.PlayAction(9, true)
	}
	v.self = ch
}

func (a *App) releaseWorld() {
	a.sprites.ResetHoverDepth()
	a.combatNumbers = nil
	a.appearanceEdit = nil
	a.mall = mallView{}
	if a.mallConfirm != nil {
		a.mallConfirm.Visible = false
	}
	a.finishWindowDrag()
	if a.state == nil || a.state.Phase != game.PhaseInWorld && a.state.Phase != game.PhaseLoading {
		a.screenNotices = nil
		a.subActionsOpen = false
		a.playerContext = playerContext{}
		a.questLog = questLogView{}
		a.questItemOffer = itemQuestOffer{}
		if a.questConfirm != nil {
			a.questConfirm.Visible = false
		}
		a.buffs = buffsView{}
		a.party = partyView{}
		a.guild = guildView{}
		a.social = socialView{}
		if a.socialInviteUI != nil {
			a.socialInviteUI.Visible = false
		}
		if a.socialUI != nil {
			a.socialUI.Focus = false
			a.socialUI.EditingMemo = 0
			a.socialUI.Name = ui.TextField{Max: 20}
		}
		a.guildDuel = guildDuelView{}
		a.pvpReward = pvpRewardView{}
		if a.pvpRewardMessage != nil {
			a.pvpRewardMessage.Visible = false
		}
		if a.guildDuelMessage != nil {
			a.guildDuelMessage.Visible = false
		}
		if a.guildMessage != nil {
			a.guildMessage.Visible = false
		}
		if a.unionConfirm != nil {
			a.unionConfirm.Visible = false
		}
		if a.guildConfirm != nil {
			a.guildConfirm.Visible = false
		}
		if a.guildVoteMessage != nil {
			a.guildVoteMessage.Visible = false
		}
		if a.guildCreateUI != nil {
			a.guildCreateUI.Visible = false
		}
		if a.partyMessage != nil {
			a.partyMessage.Visible = false
		}
	}
	if a.wv != nil {
		a.wv.crowdBatch.Delete()
		a.wv.shadow.Delete()
		a.wv.waterReflection.Delete()
		a.scene3d.ReleaseWaterBackground()
		a.scene3d.ReleasePackedEffects()
		if a.wv.mapView.image != nil {
			a.wv.mapView.image.Delete()
		}
		if a.wv.portrait != nil {
			a.wv.portrait.Delete()
		}
		a.wv.world.Close()
	}
	a.wv = nil
}

// SceneReady is used by offline captures, not by the player's movement/input.
func (a *App) SceneReady() bool {
	if a.loading.active {
		return false
	}
	if a.screen == screenWorld {
		return a.wv != nil && a.wv.world != nil && a.wv.world.Ready()
	}
	return true
}

// cameraShake is 0x54fe20: the timer only ever grows to the longer request.
func (a *App) cameraShake(seconds float32) {
	if a.wv != nil && seconds > a.wv.shake {
		a.wv.shake = seconds
	}
}
