package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/element"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"math"
	"testing"
	"time"
)

func TestElementalDamageAllPairsAcrossCombatPaths(t *testing.T) {
	// Zero luck/variance and ample damage isolate elemental arithmetic. Critical
	// outcomes are accounted for explicitly instead of depending on RNG seeding.
	att := CharacterStats{Level: 50}
	att.Attack = [3]int32{1000, 1000, 1000}
	def := CharacterStats{Level: 50}
	mob := mobCombat{Level: 50}
	mob.Attack = [3]int32{1000, 1000, 1000}
	for a := uint8(0); a < 9; a++ {
		for d := uint8(0); d < 9; d++ {
			att.AttackAttr, def.DefenseAttr, mob.Attrib = a, d, d
			factor := float64(element.Multiplier(a, d))
			for _, kind := range []int{attackMelee, attackRanged} {
				got, crit := userDamage(&att, &def, kind, false)
				want := float64(int32(1000 * factor))
				if crit {
					want *= 1.5
				}
				if got != int32(want) {
					t.Fatalf("PvP %d/%d got%d want%.0f", a, d, got, want)
				}
			}
			var damage [3]int32
			crit := weaponDamage(&att, &mob, attackMelee, &damage, a, d, false)
			want := float32(990 * factor)
			if crit {
				want *= 1.5
			}
			if damage[0] != int32(want) {
				t.Fatalf("PvE %d/%d got%d want%v", a, d, damage[0], want)
			}
			crit = mobWeaponDamage(&mob, attackMelee, 1000, 0, a, d, &def, &damage)
			want = float32(1010 * factor)
			if crit {
				want *= 1.5
			}
			if damage[0] != int32(want) {
				t.Fatalf("mob skill %d/%d got%d want%v", a, d, damage[0], want)
			}
		}
	}
}

func TestSpellElementOverrideIgnoredDefenseAndFixedDamage(t *testing.T) {
	att := CharacterStats{Level: 50, AttackAttr: 2}
	att.Attack = [3]int32{1000, 1000, 1000}
	mob := mobCombat{Level: 50, Attrib: 4}
	victim := CharacterStats{Level: 50, DefenseAttr: 4}
	for _, detail := range []uint16{0, detailNoElement} {
		sk := &db.SkillDef{DamageType: 1, AttrType: 1, TypeDetail: detail}
		expectedDef := uint8(4)
		if detail == detailNoElement {
			expectedDef = 0
		}
		factor := float64(element.Multiplier(1, expectedDef))
		damage, crit := playerSkillDamage(&att, &mob, sk, skillTarget{}, [3]uint32{}, false)
		want := float32(990 * factor)
		if crit {
			want *= 1.5
		}
		if damage[0] != int32(want) {
			t.Fatal("PvE spell override/no-element", detail, damage, want)
		}
		damage, crit = userSkillDamage(&att, &victim, sk, [3]uint32{}, false)
		v := float64(int32(1000 * factor))
		if crit {
			v *= 1.5
		}
		if damage[0] != int32(v) {
			t.Fatal("PvP spell override/no-element", detail, damage, v)
		}
		md := &db.MobDef{}
		md.Attacks[0].Attrib = 2
		md.Attacks[0].Attack = 1000
		damage, crit = mobSkillDamage(&mob, md, sk, &victim, 1000)
		want = float32(1010 * factor)
		if crit {
			want *= 1.5
		}
		if damage[0] != int32(want) {
			t.Fatal("mob spell override/no-element", detail, damage, want)
		}
	}
	sk := &db.SkillDef{DamageType: 0, DamageHP: 177, AttrType: 1}
	got, _ := playerSkillDamage(&att, &mob, sk, skillTarget{}, [3]uint32{}, false)
	if got[0] != 177 {
		t.Fatal("fixed native damage must bypass element", got)
	}
}

func TestEquippedElementSlotsAndLiveDetection(t *testing.T) {
	s, _ := forgeRuleSession(t)
	defs := db.MustDefs()
	defs.Item(1, 1).Attrib = 1
	defs.Item(16, 1).Attrib = 2
	defs.Items[db.ItemKey{Type: 17, TypeID: 1}] = &db.ItemDef{Type: 17, TypeID: 1, Attrib: 3}
	s.items = []db.Item{{Bag: 0, Slot: 5, Type: 1, TypeID: 1, Count: 1}, {Bag: 0, Slot: 0, Type: 16, TypeID: 1, Count: 1}, {Bag: 0, Slot: 1, Type: 17, TypeID: 1, Count: 1}}
	e := GetEquipmentStats(s.items)
	if e.AttackAttr != 1 || e.DefenseAttr != 3 {
		t.Fatal("only weapon/upper armor set elements", e)
	}
	p := &ConnectedPlayer{}
	p.UpdateItems(s.items)
	s.player = p
	s.addEffect(&db.SkillDef{SkillID: 60001, SkillLevel: 1, TypeEffect: typeEffectBuff, TypeDetail: 95, AttrType: 8, KeepSeconds: 60}, 0)
	if p.equipmentDefenseAttr() != 8 {
		t.Fatal("detection ignored armor-element buff")
	}
	s.addEffect(&db.SkillDef{SkillID: 60002, SkillLevel: 1, TypeEffect: typeEffectDebuff, TypeDetail: 68, KeepSeconds: 60}, 0)
	if p.equipmentDefenseAttr() != 0 {
		t.Fatal("detection ignored element removal")
	}
	s.removeEffects(func(b *ActiveBuff) bool { return b.Detail == 68 }, "test")
	if p.equipmentDefenseAttr() != 3 {
		t.Fatal("native removal must restore armor rather than replay remaining buff")
	}
	s.removeEffects(func(b *ActiveBuff) bool { return true }, "test")
	if p.equipmentDefenseAttr() != 3 {
		t.Fatal("expired buff did not restore armor")
	}
}

func TestAttackElementReplacementDoesNotResurrectOldBuff(t *testing.T) {
	s, _ := forgeRuleSession(t)
	a := &db.SkillDef{SkillID: 60003, SkillLevel: 1, TypeEffect: typeEffectBuff, TypeDetail: 94, AttrType: 1, KeepSeconds: 60}
	b := *a
	b.SkillID = 60004
	b.AttrType = 5
	s.addEffect(a, 0)
	s.addEffect(&b, 0)
	if len(s.buffs) != 1 || s.stats().AttackAttr != 5 {
		t.Fatal("different elemental buff remained active", s.buffs)
	}
	s.buffs[0].EndsAt = time.Now().Add(-time.Second)
	s.processBuffs()
	if len(s.buffs) != 0 || s.stats().AttackAttr != 0 {
		t.Fatal("older element reappeared after expiry")
	}
	// Normal monster attacks deliberately use neutral row, not mob.Attrib.
	m := mobCombat{Level: 50, Attrib: 1}
	m.Attack[0] = 1000
	target := CharacterStats{Level: 50, DefenseAttr: 4}
	got, crit := mobDamage(&m, &db.MobAttack{}, &target)
	want := float32(1010 * float64(float32(.8)))
	if crit {
		want *= 1.5
	}
	if math.Abs(float64(got)-float64(int32(want))) > 0 {
		t.Fatal("normal mob attack inherited base element", got, want)
	}
}

func TestNativeWeaponRemovalClearsElementBuffs(t *testing.T) {
	s, _ := forgeRuleSession(t)
	defs := db.MustDefs()
	defs.Item(1, 1).Attrib = 1
	s.items = []db.Item{{UID: 1001, Bag: 0, Slot: 5, Type: 1, TypeID: 1, Count: 1}}
	s.addEffect(&db.SkillDef{SkillID: 60007, SkillLevel: 1, TypeEffect: typeEffectBuff, TypeDetail: 94, AttrType: 2, KeepSeconds: 60}, 0)
	if s.stats().AttackAttr != 2 {
		t.Fatal("weapon element buff not applied")
	}
	s.items[0].UID = 1002
	if got := s.stats().AttackAttr; got != 1 || len(s.buffs) != 0 {
		t.Fatal("weapon swap retained element buff", got, s.buffs)
	}
}

func TestNativeArmorElementRemovalDoesNotReplayAnotherBuff(t *testing.T) {
	s, _ := forgeRuleSession(t)
	defs := db.MustDefs()
	defs.Items[db.ItemKey{Type: 17, TypeID: 1}] = &db.ItemDef{Type: 17, TypeID: 1, Attrib: 3}
	s.items = []db.Item{{UID: 2001, Bag: 0, Slot: 1, Type: 17, TypeID: 1, Count: 1}}
	a := &db.SkillDef{SkillID: 60008, SkillLevel: 1, TypeEffect: typeEffectBuff, TypeDetail: 95, AttrType: 1, KeepSeconds: 60}
	b := *a
	b.SkillID = 60009
	b.AttrType = 2
	s.addEffect(a, 0)
	s.addEffect(&b, 0)
	if s.stats().DefenseAttr != 2 {
		t.Fatal("latest armor element not applied")
	}
	s.removeEffects(func(b *ActiveBuff) bool { return b.SkillID == 60009 }, "test")
	if s.stats().DefenseAttr != 3 || len(s.buffs) != 1 {
		t.Fatal("native expiry must restore armor, not earlier spell")
	}
}

func TestNativeMagicElementAllPairs(t *testing.T) {
	att := CharacterStats{Level: 50, ItemAttack: 1000, Int: 100}
	att.Attack[attackMagic] = 1000
	victim := CharacterStats{Level: 50}
	for a := uint8(0); a < 9; a++ {
		for d := uint8(0); d < 9; d++ {
			var damage [3]int32
			crit := userWeaponDamage(&att, &victim, attackMagic, &damage, a, d, false)
			attack := float32(float64(element.Multiplier(a, d)) * 1000)
			want := float32(float64(attack) - float64(attack)/100*2)
			if crit {
				want = float32(float64(want) * 1.5)
			}
			if damage[0] != int32(want) {
				t.Fatal("magic element path", a, d, damage, want)
			}
		}
	}
}

func TestMobElementRemovalExpiryUsesNativeEventOrder(t *testing.T) {
	const mapID = 62031
	const id = uint32(0x71234567)
	world := shard(mapID)
	world.mu.Lock()
	old := world.entities
	world.entities = MapEntities{Mobs: []MapMob{{ID: id, MobID: 1, Level: 10, CurrentHP: 10000, MaxHP: 10000, def: &db.MobDef{Level: 10, Attrib: 3}}}}
	world.mu.Unlock()
	t.Cleanup(func() { world.mu.Lock(); world.entities = old; world.mu.Unlock() })
	a := &db.SkillDef{SkillID: 60011, SkillLevel: 1, TypeEffect: typeEffectDebuff, TypeDetail: 68, KeepSeconds: 60}
	b := *a
	b.SkillID = 60012
	ApplyMobEffect(mapID, id, 1, a)
	ApplyMobEffect(mapID, id, 1, &b)
	if m, ok := MobCombatByID(mapID, id); !ok || m.Attrib != 0 {
		t.Fatal("mob element not removed")
	}
	world.mu.Lock()
	world.entities.Mobs[0].Effects[0].endsAt = time.Now().Add(-time.Second)
	world.mu.Unlock()
	TickMobEffects()
	if m, _ := MobCombatByID(mapID, id); m.Attrib != 3 {
		t.Fatal("expiry must restore mob base even with another removal active")
	}
	c := b
	c.SkillID = 60013
	ApplyMobEffect(mapID, id, 1, &c)
	if m, _ := MobCombatByID(mapID, id); m.Attrib != 0 {
		t.Fatal("new removal did not neutralize restored mob")
	}
}

func TestElementalLapisCanLinkAndNextSocketActivatesAfterExtraction(t *testing.T) {
	for _, intrinsic := range []uint8{0, 3} {
		t.Run(fmt.Sprint(intrinsic), func(t *testing.T) {
			s, add := forgeRuleSession(t)
			defs := db.MustDefs()
			defs.Item(1, 1).Attrib = intrinsic
			defs.Item(30, 1).Attrib = 1
			defs.Item(30, 2).Attrib = 2
			target := add(db.Item{Bag: 0, Slot: 5, Type: 1, TypeID: 1, Count: 1, Gems: [6]uint8{1}})
			add(db.Item{Bag: 1, Slot: 0, Type: 100, TypeID: 140, Count: 2})
			add(db.Item{Bag: 1, Slot: 1, Type: 30, TypeID: 2, Count: 1})
			add(db.Item{Bag: 1, Slot: 2, Type: 100, TypeID: 49, Count: 1})
			want := uint8(1)
			if intrinsic != 0 {
				want = intrinsic
			}
			if s.stats().AttackAttr != want {
				t.Fatal("initial priority")
			}
			s.handleGemAdd(proto.GemAddHammerPacket(1, 1, 0, 5, 1, 2))
			current := s.items[findItemByUID(s.items, target.UID)]
			if current.Gems[0] != 1 || current.Gems[1] != 2 || s.stats().AttackAttr != want {
				t.Fatal("second element must link but remain inactive", current)
			}
			s.handleGemRemove(proto.GemRemoveHammerPacket([2]uint8{0, 5}, [2]uint8{1, 0}, 0))
			want = 2
			if intrinsic != 0 {
				want = intrinsic
			}
			if s.stats().AttackAttr != want {
				t.Fatal("extraction did not promote next socket or preserve intrinsic", s.stats().AttackAttr, want)
			}
		})
	}
}

func TestInactiveElementalLapisGiveNoOtherBonusOrTriggeredSkill(t *testing.T) {
	_, _ = forgeRuleSession(t)
	defs := db.MustDefs()
	first, ordinary := defs.Item(30, 1), defs.Item(30, 2)
	first.Attrib, first.ConstStr, first.Range, first.Special = 2, 7, 5, 1
	ordinary.Attrib, ordinary.ConstStr, ordinary.Range = 0, 3, 2
	defs.Items[db.ItemKey{Type: 30, TypeID: 3}] = &db.ItemDef{Type: 30, TypeID: 3, Attrib: 3, ConstStr: 11, Range: 9, Special: 2}
	defs.Skills = map[db.SkillKey]*db.SkillDef{
		{SkillID: 1, Level: 100}: {SkillID: 1, SkillLevel: 100, TypeEffect: typeEffectAttack},
		{SkillID: 2, Level: 100}: {SkillID: 2, SkillLevel: 100, TypeEffect: typeEffectAttack},
	}
	it := db.Item{Bag: 0, Slot: 5, Type: 1, TypeID: 1, Count: 1, Gems: [6]uint8{1, 2, 3}}
	defs.Item(1, 1).Attrib = 1
	e := GetEquipmentStats([]db.Item{it})
	if e.Str != 3 || e.Range != 2 || len(gemSkillsOf(&it)) != 0 {
		t.Fatal("pre-element gear accepted inactive lapis bonus", e)
	}
	defs.Item(1, 1).Attrib = 0
	e = GetEquipmentStats([]db.Item{it})
	skills := gemSkillsOf(&it)
	if e.Str != 10 || e.Range != 7 || len(skills) != 1 || skills[0].SkillID != 1 {
		t.Fatal("first element / ordinary lapis bonus priority", e, skills)
	}
}
