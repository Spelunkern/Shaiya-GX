package scene

import (
	"fmt"
	"path"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
)

// CreatureActor shares the character skinning path, with clips from MON.
type CreatureActor struct {
	boneEffects      []creatureBoneEffect
	boneEffectsReady bool
	attackEffects    []*Effect
	pendingEffects   []string
	*Character
	Name       string
	Level      uint16
	Definition sdata.Monster
	folder     string
	visual     model.Creature
	action     int
	cueTime    float32
}

func (s *Assets) MonsterDefinition(id uint16) (sdata.Monster, error) {
	if s.monsters == nil {
		b, err := s.Arc.ReadFile("Monster/Monster.SData")
		if err != nil {
			return sdata.Monster{}, err
		}
		s.monsters, err = sdata.ParseMonsters(b)
		if err != nil {
			return sdata.Monster{}, err
		}
	}
	if int(id) >= len(s.monsters) {
		return sdata.Monster{}, fmt.Errorf("monster %d out of range", id)
	}
	return s.monsters[id], nil
}

func (s *Assets) BuildMonster(id uint16) (*CreatureActor, error) {
	def, err := s.MonsterDefinition(id)
	if err != nil {
		return nil, err
	}
	if s.creatures == nil {
		b, err := s.Arc.ReadFile("Monster/monster.MON")
		if err != nil {
			return nil, err
		}
		s.creatures, err = model.ParseMON(b)
		if err != nil {
			return nil, err
		}
	}
	if int(def.Model) >= len(s.creatures) {
		return nil, fmt.Errorf("monster model %d out of range", def.Model)
	}
	visual := s.creatures[def.Model]
	actor, err := s.buildCreature("Monster", visual, def.Name)
	if actor != nil {
		actor.Level, actor.Definition = def.Level, def
	}
	return actor, err
}

func (s *Assets) BuildNPC(def sdata.NPC) (*CreatureActor, error) {
	if s.npcVisuals == nil {
		b, err := s.Arc.ReadFile("Npc/npc.MON")
		if err != nil {
			return nil, err
		}
		s.npcVisuals, err = model.ParseMON(b)
		if err != nil {
			return nil, err
		}
	}
	if def.Model < 0 || int(def.Model) >= len(s.npcVisuals) {
		return nil, fmt.Errorf("NPC model %d out of range", def.Model)
	}
	return s.buildCreature("Npc", s.npcVisuals[def.Model], def.Name)
}

func (s *Assets) buildCreature(folder string, visual model.Creature, name string) (*CreatureActor, error) {
	c := &Character{assets: s}
	for _, part := range visual.Parts {
		p, err := s.skinFile(folder + "/3DC/" + path.Base(part[0]))
		if err != nil {
			return nil, err
		}
		cp := *p
		cp.tex = s.modelTexture(folder + "/DDS/" + DDSName(path.Base(part[1])))
		cp.cut = true
		c.parts = append(c.parts, &cp)
	}
	m := &CreatureActor{Character: c, Name: name, folder: folder, visual: visual, action: -1}
	if err := m.SetAction(6); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *CreatureActor) SetAction(action int) error {
	if action == 5 {
		m.cueTime = 0
	}
	if m.cueTime > 0 {
		return nil
	}
	if m.action == action {
		return nil
	}
	if action < 0 || action >= len(m.visual.Animations) {
		return fmt.Errorf("invalid creature action %d", action)
	}
	clip, err := m.assets.Clip(m.folder + "/ANI/" + path.Base(m.visual.Animations[action]))
	if err != nil {
		return err
	}
	m.clip, m.Time, m.action = clip, 0, action
	if action == 5 {
		m.queueModelEffect(action)
	}
	return nil
}

// Death is a one-shot; live movement/breath clips loop in the shared sampler.
func (m *CreatureActor) Update(dt float32) {
	oneShot := m.cueTime > 0
	m.cueTime = max(0, m.cueTime-dt)
	m.Character.Update(dt)
	if (m.action == 5 || oneShot) && m.clip != nil {
		if end := m.clip.Duration(); end > 0 {
			m.Time = min(m.Time, max(0, end-0.001))
		}
	}
}

// PlayOnce presents a confirmed attack/hit; locomotion resumes afterward.
func (m *CreatureActor) PlayOnce(action int) {
	// A damage reaction must not restart or replace a confirmed attack.
	if action == 7 && m.Attacking() {
		return
	}
	m.cueTime = 0
	if err := m.SetAction(action); err != nil {
		m.assets.Missing[err.Error()] = true
		return
	}
	m.Time = 0
	if action != 5 {
		m.queueModelEffect(action)
	}
	if m.clip != nil {
		m.cueTime = m.clip.Duration()
	}
}

// Sound returns the original MON attack1/2/3 or death WAV reference.
func (m *CreatureActor) Sound(index int) string {
	if m == nil || index < 0 || index >= len(m.visual.Sounds) {
		return ""
	}
	return m.visual.Sounds[index]
}

func (m *CreatureActor) Attacking() bool { return m.cueTime > 0 && m.action >= 2 && m.action <= 4 }
func (m *CreatureActor) AttackAction(slot uint8) int {
	if slot > 2 {
		slot = 0
	}
	action := int(m.Definition.AttackAnimations[slot]) - 1
	if action < 0 || action > 2 {
		action = 0
	}
	return action + 2
}
