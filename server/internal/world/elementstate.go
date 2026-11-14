package world

import "github.com/Spelunkern/Shaiya-GX/server/internal/db"

// Native elemental fields are event state, not an unordered sum of buffs.
// RemApplySkillBuff 0x485130 and RemApplySkillDebuff 0x485720 restore equipment
// even if a different elemental effect remains in the applied-skill list.
type elementSource struct {
	UID          int64
	Type, TypeID uint8
	Gems         [6]uint8
	Present      bool
}
type elementRuntime struct {
	ready           bool
	weapon, armor   elementSource
	attack, defense uint8
	attackFlags     [9]bool
	defenseSkill    bool
}

func (s *Session) elementSource(slot uint8) elementSource {
	if it := equippedItem(s.items, slot); it != nil {
		return elementSource{it.UID, it.Type, it.TypeID, it.Gems, true}
	}
	return elementSource{}
}
func (s *Session) applyElement(b ActiveBuff) {
	e := &s.elements
	switch {
	case b.Effect == typeEffectBuff && b.Detail == 94 && b.Attr < 9:
		if !e.attackFlags[b.Attr] {
			e.attack = b.Attr
			e.attackFlags[b.Attr] = true
		}
	case b.Effect == typeEffectBuff && b.Detail == 95:
		e.defense = b.Attr
		e.defenseSkill = true
	case b.Effect == typeEffectDebuff && b.Detail == 68:
		e.defense = 0
	}
}
func (s *Session) removeElement(b ActiveBuff) {
	base := GetEquipmentStats(s.items)
	e := &s.elements
	switch {
	case b.Effect == typeEffectBuff && b.Detail == 94:
		if b.Attr < 9 {
			e.attackFlags[b.Attr] = false
		}
		e.attack = base.AttackAttr
	case b.Effect == typeEffectBuff && b.Detail == 95:
		if e.armor.Present {
			e.defense = base.DefenseAttr
		}
		e.defenseSkill = false
		for _, remaining := range s.buffs {
			if remaining.Effect == typeEffectBuff && remaining.Detail == 95 {
				e.defenseSkill = true
			}
		}
	case b.Effect == typeEffectDebuff && b.Detail == 68:
		if e.armor.Present {
			e.defense = base.DefenseAttr
		}
	}
}
func (s *Session) syncElements(base EquipmentStats) {
	e := &s.elements
	weapon, armor := s.elementSource(5), s.elementSource(1)
	if !e.ready {
		*e = elementRuntime{ready: true, weapon: weapon, armor: armor, attack: base.AttackAttr, defense: base.DefenseAttr}
		for _, b := range s.buffs {
			s.applyElement(b)
		}
		return
	}
	if e.weapon != weapon {
		old := e.weapon
		e.weapon = weapon // Publish before removal's stats callbacks.
		// ItemEquipmentRem removes weapon element skills on an actual unequip.
		if old.Present && (old.UID != weapon.UID || old.Type != weapon.Type || old.TypeID != weapon.TypeID || !weapon.Present) {
			s.removeEffects(func(b *ActiveBuff) bool { return b.Effect == typeEffectBuff && b.Detail == 94 }, "(weapon removed)")
			e.attackFlags = [9]bool{}
		}
		protected := false
		for _, flag := range e.attackFlags[1:] {
			protected = protected || flag
		}
		if !protected {
			e.attack = base.AttackAttr
		}
	}
	if e.armor != armor {
		old := e.armor
		e.armor = armor
		if old.Present && (old.UID != armor.UID || old.Type != armor.Type || old.TypeID != armor.TypeID || !armor.Present) {
			e.defense = 0
		}
		if !e.defenseSkill {
			e.defense = base.DefenseAttr
		}
	}
}
func (s *Session) replaceAttackElement(def *db.SkillDef) {
	e := &s.elements
	if def.TypeEffect != typeEffectBuff || def.TypeDetail != 94 || def.AttrType > 8 || e.attackFlags[def.AttrType] || e.attack > 8 || !e.attackFlags[e.attack] {
		return
	}
	s.removeEffects(func(b *ActiveBuff) bool {
		return b.Effect == typeEffectBuff && b.Detail == 94 && b.Attr != def.AttrType
	}, "(attack element replaced)")
}
