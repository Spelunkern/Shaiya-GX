package world

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func TestVisibilityAuthorityAndGMDefaults(t *testing.T) {
	makePlayer := func(id uint32, admin bool, faction uint8) *ConnectedPlayer {
		p := RegisterPlayer(db.Character{ID: id, MapID: 60078}, nil, admin, faction, nil, nil)
		t.Cleanup(p.Unregister)
		return p
	}
	light := makePlayer(990001, false, 0)
	dark := makePlayer(990002, false, 1)
	gm := makePlayer(990003, true, 0)
	otherGM := makePlayer(990004, true, 1)
	if !gm.hidden.Load() || !gm.immune.Load() || !gm.noMobTarget.Load() {
		t.Fatal("GM defaults are not protected")
	}
	if light.canSee(gm) || dark.canSee(gm) || !otherGM.canSee(gm) || !gm.canSee(gm) {
		t.Fatal("hidden GM leaked or became invisible to GMs")
	}
	light.hidden.Store(true)
	if dark.canSee(light) || !gm.canSee(light) || !light.canSee(light) {
		t.Fatal("normal stealth visibility")
	}
	ally := makePlayer(990005, false, 0)
	if !ally.canSee(light) {
		t.Fatal("ally lost stealth ghost")
	}
	if dark.canReceiveWorldPacket(light.MovePacket()) {
		t.Fatal("movement leaked through proximity broadcast")
	}
	if !dark.canReceiveWorldPacket(buildUserLeave(light.CharacterID)) {
		t.Fatal("concealment must still remove known entities")
	}
}

func TestGMProtectionCommandsAndEffectImmunity(t *testing.T) {
	s, packets := gmWireSession(t, db.Character{ID: 990010, MapID: 60078, MaxHP: 100, CurrentHP: 100}, true)
	s.player = RegisterPlayer(*s.char, nil, true, 0, s.out, nil)
	t.Cleanup(s.player.Unregister)
	if !s.invulnerable() {
		t.Fatal("GM is vulnerable by default")
	}
	// Reject damage and friendly effects even if their senders queued them before activation.
	if s.takeUserDamage(1, 100) || s.takeUserSkillDamage(1, [3]uint32{100, 100, 100}) || s.applyIncomingMobDamage(pendingDamage{}) {
		t.Fatal("immune GM died")
	}
	if _, ok := s.addEffect(&db.SkillDef{SkillID: 1, TypeEffect: typeEffectBuff}, 0); ok {
		t.Fatal("immune GM accepted buff")
	}
	s.applyFriendly(pendingFriendly{Source: 1, HP: 100, IsBuff: true})
	if s.char.CurrentHP != 100 || len(s.buffs) != 0 {
		t.Fatal("immune GM was changed")
	}
	s.handleGMCommand("/attack off")
	gmResponse(t, packets)
	if s.invulnerable() || s.player.immune.Load() || !s.player.hidden.Load() {
		t.Fatal("attack off changed wrong state")
	}
	s.handleGMCommand("/char off")
	gmResponse(t, packets)
	if s.player.hidden.Load() || s.invulnerable() {
		t.Fatal("char off changed wrong state")
	}
	s.handleGMCommand("/attack on")
	gmResponse(t, packets)
	if !s.invulnerable() || s.player.hidden.Load() {
		t.Fatal("attack on changed visibility")
	}
	s.handleGMCommand("/char on")
	gmResponse(t, packets)
	if !s.player.hidden.Load() || !s.invulnerable() {
		t.Fatal("char on did not restore invisibility")
	}
	s.breakStealth()
	if !s.player.hidden.Load() || !s.invulnerable() {
		t.Fatal("attack must preserve both GM command modes")
	}
	s.handleGMCommand("/attack invalid")
	gmResponse(t, packets)
	if !s.invulnerable() {
		t.Fatal("malformed command changed protection")
	}
}

func TestStealthDoesNotChangeSpeedAndBreaksOnlyStealth(t *testing.T) {
	s, _ := gmWireSession(t, db.Character{ID: 990011, MapID: 60078, MaxHP: 100, CurrentHP: 100}, false)
	s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, nil)
	t.Cleanup(s.player.Unregister)
	before := s.speeds()
	s.buffs = []ActiveBuff{{ID: 1, SkillID: 60000, Effect: typeEffectBuff, Detail: 88, EndsAt: time.Now().Add(time.Minute)}, {ID: 2, Effect: typeEffectBuff, Detail: 89}}
	s.syncVisibility()
	if !s.player.hidden.Load() || !s.player.noMobTarget.Load() || s.speeds() != before {
		t.Fatal("stealth must conceal without slowing")
	}
	s.breakStealth()
	if s.player.hidden.Load() || !s.player.noMobTarget.Load() || len(s.buffs) != 1 || s.buffs[0].Detail != 89 {
		t.Fatal("stealth break removed unrelated protection")
	}
}

func TestMobRetargetsOnlyVisibleCombatParticipants(t *testing.T) {
	m := MapMob{AggroTarget: 1, aggro: map[uint32]int64{1: 100, 2: 10}, def: &db.MobDef{Country: 2}, AI: 1, ChaseRange: 100}
	players := []PlayerCombatSnapshot{{CharacterID: 1, Alive: true, Invisible: true}, {CharacterID: 2, Alive: true}, {CharacterID: 3, Alive: true}}
	if m.liveTarget(players) != nil {
		t.Fatal("mob retained invisible target")
	}
	m.AggroTarget = 0
	m.retargetByAggro(players)
	if m.AggroTarget != 2 {
		t.Fatal("mob did not choose participating visible attacker")
	}
	players[1].Invisible = true
	m.AggroTarget = 0
	m.retargetByAggro(players)
	if m.AggroTarget != 0 {
		t.Fatal("mob chose an uninvolved bystander")
	}
	if m.targetWeakest(players) {
		t.Fatal("weakest targeting bypassed concealment")
	}
}

func TestConcealmentRemovesAndReintroducesStationaryUsers(t *testing.T) {
	makeSession := func(id uint32, admin bool) *Session {
		s, _ := gmWireSession(t, db.Character{ID: id, MapID: 60079, PosX: 100, PosZ: 100, CurrentHP: 100}, admin)
		s.player = RegisterPlayer(*s.char, nil, admin, 0, s.out, nil)
		t.Cleanup(s.player.Unregister)
		s.placeView()
		return s
	}
	normal := makeSession(990021, false)
	gm := makeSession(990022, true)
	observer := makeSession(990023, true)
	if normal.player.knows(viewUser, gm.char.ID) || !observer.player.knows(viewUser, gm.char.ID) {
		t.Fatal("GM default view filtering")
	}
	gm.gmRevealed = true
	gm.syncVisibility()
	if !normal.player.knows(viewUser, gm.char.ID) {
		t.Fatal("stationary GM did not reappear")
	}
	gm.gmRevealed = false
	gm.syncVisibility()
	if normal.player.knows(viewUser, gm.char.ID) || !observer.player.knows(viewUser, gm.char.ID) {
		t.Fatal("stationary concealment did not reconcile viewers")
	}
	gm.gmVulnerable = true
	gm.buffs = []ActiveBuff{{ID: 1, Effect: typeEffectBuff, Detail: 88}}
	gm.gmRevealed = true
	gm.syncVisibility()
	if !normal.player.knows(viewUser, gm.char.ID) {
		t.Fatal("allied ordinary stealth ghost disappeared")
	}
	gm.gmRevealed = false
	gm.syncVisibility()
	if normal.player.knows(viewUser, gm.char.ID) {
		t.Fatal("GM concealment over existing stealth leaked to ally")
	}

}

func TestMobLosingHiddenTargetReturnsHomeWithoutAggroingBystander(t *testing.T) {
	const mapID = 60080
	sh := shard(mapID)
	m := NewStaticMob(&db.MobDef{MaxHP: 100, Country: 2, AI: 1, ChaseRange: 100}, 1, 100, 100, nil, 0)
	m.PosX = 140
	m.AggroTarget = 1
	m.aggro = map[uint32]int64{1: 100}
	m.noTargetUntil = time.Time{}
	sh.mu.Lock()
	old := sh.entities
	sh.entities = MapEntities{Mobs: []MapMob{m}}
	sh.mu.Unlock()
	t.Cleanup(func() { sh.mu.Lock(); sh.entities = old; sh.mu.Unlock() })
	TickMobCombat([]PlayerCombatSnapshot{{CharacterID: 1, MapID: mapID, PosX: 141, PosZ: 100, Alive: true, Invisible: true}, {CharacterID: 2, MapID: mapID, PosX: 142, PosZ: 100, Alive: true}}, 64)
	sh.mu.Lock()
	got := sh.entities.Mobs[0]
	sh.mu.Unlock()
	if got.AggroTarget != 0 || !got.returningHome {
		t.Fatalf("mob did not disengage: target=%d returning=%t", got.AggroTarget, got.returningHome)
	}
}

func TestHiddenBroadcastsAndPartyCoordinates(t *testing.T) {
	hidden := RegisterPlayer(db.Character{ID: 990030, MapID: 60081, PosX: 123, PosY: 45, PosZ: 678}, nil, true, 0, nil, nil)
	viewer := RegisterPlayer(db.Character{ID: 990031, MapID: 60081}, nil, false, 0, nil, nil)
	t.Cleanup(hidden.Unregister)
	t.Cleanup(viewer.Unregister)
	equipment := packet.New(packet.CharacterEquipment)
	equipment.U32(hidden.CharacterID)
	equipment.U8(5)
	equipment.U8(1)
	equipment.U8(1)
	for _, frame := range [][]byte{hidden.MovePacket(), hidden.ShapePacket(), equipment.MustFinish(), buildSkillUseResult(packet.SkillUse, 0, hidden.CharacterID, 999, 1, 1, 0, 0, 0), buildMobAttackCharacter(0, 999, hidden.CharacterID, 1)} {
		if viewer.canReceiveWorldPacket(frame) {
			t.Fatalf("hidden identity leaked in opcode %x", binary.LittleEndian.Uint16(frame[2:4]))
		}
	}
	roster := packet.New(packet.PartyRoster)
	roster.U8(0)
	roster.U8(1)
	writePartyMember(roster, hidden)
	original := roster.MustFinish()
	redacted := viewer.redactPartyPositions(original)
	if binary.LittleEndian.Uint32(original[60:64]) == 0 {
		t.Fatal("fixture lost coordinates")
	}
	for _, v := range redacted[60:72] {
		if v != 0 {
			t.Fatal("hidden party position leaked")
		}
	}
	if binary.LittleEndian.Uint32(original[60:64]) == 0 {
		t.Fatal("redaction mutated shared broadcast frame")
	}
}

func TestProtectedGMSelfBuffKeepsConfirmationAndExternalImmunity(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	def := &db.SkillDef{SkillID: 60010, SkillLevel: 1, TypeEffect: typeEffectBuff, KeepSeconds: 60}
	db.SetDefsForTest(&db.Defs{Skills: map[db.SkillKey]*db.SkillDef{{SkillID: 60010, Level: 1}: def}})
	s, packets := gmWireSession(t, db.Character{ID: 990012, MapID: 60080, MaxHP: 100, CurrentHP: 100}, true)
	s.player = RegisterPlayer(*s.char, nil, true, 0, s.out, nil)
	t.Cleanup(s.player.Unregister)
	if !s.teamUsable(s.player) {
		t.Fatal("own immune character must be friendly-usable")
	}
	s.applyFriendly(pendingFriendly{Source: s.char.ID, Op: packet.SkillUseNoTarget, SkillID: def.SkillID, Level: 1, IsBuff: true})
	if len(s.buffs) != 1 || s.buffs[0].CasterUser != s.char.ID {
		t.Fatal("own buff was not applied with provenance")
	}
	// The confirmed native result drives the client's release sound and effects.
	timeout := time.After(3 * time.Second)
	confirmed := false
	for !confirmed {
		select {
		case frame := <-packets:
			confirmed = packet.Opcode(binary.LittleEndian.Uint16(frame)) == packet.SkillUseNoTarget
		case <-timeout:
			t.Fatal("missing self-buff confirmation/audio trigger")
		}
	}
	s.applyFriendly(pendingFriendly{Source: 123, SkillID: def.SkillID, Level: 1, IsBuff: true})
	if len(s.buffs) != 1 || s.buffs[0].CasterUser != s.char.ID {
		t.Fatal("external buff replaced own buff")
	}
	s.breakStealth()
	s.handleGMCommand("/attack on")
	gmResponse(t, packets)
	if len(s.buffs) != 1 || !s.player.hidden.Load() || !s.invulnerable() {
		t.Fatal("GM commands or actions removed own buff or command protection")
	}
	if _, ok := s.addEffect(&db.SkillDef{SkillID: 60011, TypeEffect: typeEffectDebuff}, 0, s.char.ID); ok {
		t.Fatal("self provenance allowed a debuff through immunity")
	}
}
