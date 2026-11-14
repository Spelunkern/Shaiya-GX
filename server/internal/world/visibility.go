package world

import (
	"encoding/binary"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// GX visibility describes only entities the viewer is permitted to receive.
// Forbidden viewers receive a native leave and no subsequent movement/shape.
const gxVisibility packet.Opcode = 0x7F02

func (p *ConnectedPlayer) canSee(subject *ConnectedPlayer) bool {
	if p == nil || subject == nil {
		return false
	}
	if p == subject || p.IsAdmin {
		return true
	}
	if !canSeeCharacter(p.IsAdmin, subject.IsAdmin, subject.GMVisible()) {
		return false
	}
	return !subject.hidden.Load() || p.Faction() == subject.Faction()
}

func (p *ConnectedPlayer) visibilityPacket() []byte {
	w := packet.New(gxVisibility)
	w.U32(p.CharacterID)
	hidden := uint8(0)
	if p.hidden.Load() {
		hidden = 1
	}
	w.U8(hidden)
	return w.MustFinish()
}

func (s *Session) invulnerable() bool { return s.isAdmin && !s.gmVulnerable }

func (s *Session) syncVisibility() {
	p := s.player
	if p == nil {
		return
	}
	hidden, noTarget := s.isAdmin && !s.gmRevealed, false
	for _, b := range s.buffs {
		if b.Effect != typeEffectBuff {
			continue
		}
		if b.Detail == 88 {
			hidden = true
		}
		if b.Detail == 89 {
			noTarget = true
		}
	}
	p.mu.Lock()
	gmChanged := p.gmVisible != (!s.isAdmin || s.gmRevealed)
	p.gmVisible = !s.isAdmin || s.gmRevealed
	p.mu.Unlock()
	changed := p.hidden.Swap(hidden) != hidden || gmChanged
	p.noMobTarget.Store(hidden || noTarget)
	p.immune.Store(s.invulnerable())
	if changed {
		if hidden {
			s.autoAttack, s.cast = false, nil
			s.send(packet.New(packet.AutoAttackStop).MustFinish(), "stealth attack stop")
		}
		s.refreshView()
		frame := p.visibilityPacket()
		s.send(frame, "GX visibility")
		s.sendToViewers(frame, "GX visibility")
	}
}

// An accepted attack (including a miss), or actual damage, ends stealth.
// GM visibility and invulnerability are independent command modes.
func (s *Session) breakStealth() {
	if s.char == nil {
		return
	}
	s.removeEffects(func(b *ActiveBuff) bool { return b.Effect == typeEffectBuff && b.Detail == 88 }, "stealth broken")
	s.syncVisibility()
}

// Central outbound guard also covers combat, speed, cast and effect broadcasts
// that use proximity delivery rather than the character view list.
func (p *ConnectedPlayer) canReceiveWorldPacket(frame []byte) bool {
	if len(frame) < 4 {
		return true
	}
	var offsets []int
	switch binary.LittleEndian.Uint16(frame[2:4]) {
	case 0x0201, 0x0226, 0x0303, 0x0406, 0x0501, 0x0504, 0x0505, 0x0506, 0x0507, 0x050F, 0x051C, 0x051D, 0x051E, 0x0520, 0x0551, 0x7F02:
		offsets = []int{4}
	case 0x0510, 0x0514, 0x0515:
		offsets = []int{4, 8}
	case 0x0502, 0x0511, 0x0512, 0x0513:
		offsets = []int{5, 9}
	case 0x0503, 0x0517, 0x0518, 0x0519:
		offsets = []int{5}
	case 0x051A:
		offsets = []int{4}
	case 0x060E:
		offsets = []int{8}
	case 0x0605, 0x060B, 0x060C, 0x060D:
		offsets = []int{9}
	}
	for _, at := range offsets {
		if len(frame) < at+4 {
			continue
		}
		id := binary.LittleEndian.Uint32(frame[at : at+4])
		subject := FindPlayerByID(id)
		if subject != nil && !p.canSee(subject) {
			return false
		}
	}
	return true
}

// Party membership remains visible, but a hidden member's coordinates are not
// disclosed to an unauthorized viewer by roster/join packets.
func (p *ConnectedPlayer) redactPartyPositions(frame []byte) []byte {
	if len(frame) < 6 {
		return frame
	}
	start, step, count := 0, 67, 0
	switch binary.LittleEndian.Uint16(frame[2:4]) {
	case 0x0B01:
		start, count = 6, int(frame[5])
	case 0x0B04:
		start, count = 4, 1
	case 0x0B0B:
		if len(frame) < 14 {
			return frame
		}
		start, step, count = 14, 69, int(frame[11])
	case 0x0B0C:
		start, count = 6, 1
	}
	out := frame
	for i := 0; i < count; i++ {
		at := start + i*step
		if at+67 > len(frame) {
			break
		}
		subject := FindPlayerByID(binary.LittleEndian.Uint32(frame[at : at+4]))
		if subject != nil && !p.canSee(subject) {
			if &out[0] == &frame[0] {
				out = append([]byte(nil), frame...)
			}
			clear(out[at+54 : at+66])
		}
	}
	return out
}
