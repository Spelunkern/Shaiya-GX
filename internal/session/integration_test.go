//go:build integration

package session_test

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func enterDisposableWorld(t *testing.T) (*game.State, func([]byte), func(func() bool)) {
	st, _, send, wait := testworld.Enter(t)
	return st, send, wait
}

func TestShutdownAgainstRapture(t *testing.T) {
	st, w, send, _ := testworld.Enter(t)
	x, y, z := st.Self.X+1.25, st.Self.Y, st.Self.Z
	send(proto.MovePacket(0, 0, x, y, z))
	if err := w.Shutdown(3 * time.Second); err != nil {
		t.Fatal(err)
	}
	// No sleep or logout request: FIN must let the server save the final move
	// and release the account before a fresh login can succeed.
	next, world, _, _ := testworld.Enter(t)
	if next.Self.X != x || next.Self.Y != y || next.Self.Z != z {
		t.Fatal("shutdown lost the final position", next.Self, x, y, z)
	}
	if err := world.Shutdown(3 * time.Second); err != nil {
		t.Fatal(err)
	}
	t.Log("graceful close saved the final move and immediately released the account for re-entry")
}

func TestGameplayAgainstRapture(t *testing.T) {
	st, send, wait := enterDisposableWorld(t)
	var src [2]uint8
	found := false
	for bag := uint8(0); bag <= 5 && !found; bag++ {
		for slot := uint8(0); slot < 24; slot++ {
			key := [2]uint8{bag, slot}
			if _, ok := st.Inventory[key]; ok {
				src, found = key, true
				break
			}
		}
	}
	if !found {
		t.Fatal("starter inventory is empty")
	}
	dst := [2]uint8{1, 23}
	if _, occupied := st.Inventory[dst]; occupied {
		t.Fatal("disposable destination occupied")
	}
	item := st.Inventory[src]
	send(proto.ItemMovePacket(src[0], src[1], dst[0], dst[1]))
	wait(func() bool { return st.Inventory[dst].Type == item.Type && st.Inventory[src].Type == 0 })
	send(proto.ItemMovePacket(dst[0], dst[1], src[0], src[1]))
	wait(func() bool { return st.Inventory[src].Type == item.Type && st.Inventory[dst].Type == 0 })
	if st.Inventory[src] != item {
		t.Fatal("restored item differs from original")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("Character/Skill.SData")
	if err != nil {
		t.Fatal(err)
	}
	defs, err := sdata.ParseSkills(raw)
	if err != nil {
		t.Fatal(err)
	}
	var candidate sdata.SkillKey
	found = false
	for key, d := range defs {
		if key.Level != 1 || d.RequiredLevel > st.Slots[0].Level || d.Classes[0] == 0 || d.Show == 0 || d.Grow > 1 || uint16(d.Points) > st.Details.SkillPoint || d.Effect == 0 || d.Target != 2 {
			continue
		}
		if d.Country != 0 && d.Country != 2 && d.Country <= 5 {
			continue
		}
		if !found || key.ID < candidate.ID {
			candidate, found = key, true
		}
	}
	if !found {
		t.Fatal("no eligible active level-one skill in client data")
	}
	send(proto.LearnSkillPacket(candidate.ID, candidate.Level))
	wait(func() bool { return st.LastLearn.ID == candidate.ID && st.LastLearn.Level == candidate.Level })
	if st.LastLearn.Result != 0 {
		t.Fatalf("server rejected candidate %d: %d", candidate.ID, st.LastLearn.Result)
	}
	number := st.LastLearn.Number
	target := uint32(0)
	if defs[candidate].Target == 2 {
		target = st.SelfID
	}
	send(proto.UseSkillPacket(number, target, defs[candidate].Target == 2))
	wait(func() bool { return st.LastSkill.ID == candidate.ID && st.LastSkill.Source == st.SelfID })
	if st.LastSkill.Result != 0 {
		t.Fatalf("skill use rejected: %d", st.LastSkill.Result)
	}
	t.Logf("skill %d learned as number %d; use returned result %d", candidate.ID, number, st.LastSkill.Result)
	assigned := []proto.QuickSlot{{Bar: 0, Slot: 9, Bag: 100, Number: uint16(number)}}
	packet, err := proto.SkillBarPacket(assigned)
	if err != nil {
		t.Fatal(err)
	}
	characterID := st.SelfID
	send(packet)
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	send(proto.SelectCharacterPacket(characterID))
	wait(func() bool { return st.Phase == game.PhaseInWorld && st.SkillBarReady && st.InventoryReady })
	if len(st.QuickSlots) != 1 || st.QuickSlots[0] != assigned[0] {
		t.Fatalf("bar not persisted: %+v", st.QuickSlots)
	}
	if defs[candidate].ResetTime > 5 && st.Skills[number].Delay <= 0 {
		t.Fatal("remaining cooldown not restored")
	}
	t.Logf("bar restored after re-entry; skill delay=%d seconds", st.Skills[number].Delay)
	npcRaw, err := arc.ReadFile("Npc/NpcQuest.SData")
	if err != nil {
		t.Fatal(err)
	}
	npcs, err := sdata.ParseNPCs(npcRaw)
	if err != nil {
		t.Fatal(err)
	}
	itemRaw, err := arc.ReadFile("Item/Item.SData")
	if err != nil {
		t.Fatal(err)
	}
	itemDefs, err := sdata.ParseItems(itemRaw)
	if err != nil {
		t.Fatal(err)
	}
	merchantID := uint32(0)
	itemIndex := uint8(0)
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind != game.KindNpc || e.NpcType != 1 {
				continue
			}
			if def, ok := npcs[sdata.NPCKey{Type: 1, ID: e.TypeID}]; ok {
				for i, it := range def.Items {
					if i < 256 && itemDefs[it] != nil && itemDefs[it].RecoveryHP > 0 && itemDefs[it].Special == 0 && (it.Type == 25 || it.Type == 100 || it.Type == 101 || it.Type == 102) && (merchantID == 0 || id < merchantID) {
						merchantID, itemIndex = id, uint8(i)
					}
				}
			}
		}
		return merchantID != 0
	})
	send(proto.NpcTalkPacket(merchantID))
	beforeMoney := st.Details.Money
	send(proto.ShopBuyPacket(merchantID, itemIndex, 2))
	wait(func() bool { return st.LastShop.Type != 0 })
	if st.LastShop.Result != 0 || st.Details.Money >= beforeMoney {
		t.Fatalf("purchase rejected or did not debit gold: %+v", st.LastShop)
	}
	purchased := st.LastShop
	itemKey := [2]uint8{purchased.Bag, purchased.Slot}
	if st.Inventory[itemKey].Type != purchased.Type {
		t.Fatal("purchased item not applied")
	}
	beforeHP := st.Pools.HP
	send(proto.UseItemPacket(purchased.Bag, purchased.Slot))
	wait(func() bool { return st.LastItemUse.ID == st.SelfID })
	if st.LastItemUse.Type != purchased.Type || st.LastItemUse.Count != purchased.Count-1 {
		t.Fatalf("consumable not confirmed: %+v", st.LastItemUse)
	}
	wait(func() bool { return st.Pools.HP > beforeHP || st.Pools.HP == st.Details.MaxHP })
	t.Logf("consumable confirmed: remaining=%d HP=%d", st.LastItemUse.Count, st.Pools.HP)
	send(proto.StopAttackPacket())
	st.LastShop = proto.ShopResult{}
	beforeMoney = st.Details.Money
	send(proto.ShopSellPacket(purchased.Bag, purchased.Slot, 1))
	wait(func() bool { return st.LastShop.Type != 0 })
	if st.LastShop.Result != 0 || st.Details.Money < beforeMoney {
		t.Fatalf("sale rejected or debited gold: %+v", st.LastShop)
	}
	t.Logf("bought and sold item %d/%d via NPC %d", purchased.Type, purchased.TypeID, merchantID)
	send(proto.NpcTalkPacket(0))
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
}

// Combat uses a second login after the fixture is placed near an existing spawn.
func TestCombatAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_COMBAT") != "1" {
		t.Skip("requires combat fixture")
	}
	st, send, wait := enterDisposableWorld(t)
	killed := 0
	var drops []proto.GroundItem
	for killed < 24 && len(drops) == 0 {
		target := nearestLiveMob(t, st, send, wait)
		initialExp, initialLevel := st.Details.Exp, st.Slots[0].Level
		send(proto.MovePacket(0, 0, target.X+1, target.Y, target.Z))
		send(proto.TargetMobPacket(target.ID))
		wait(func() bool { return target.HealthKnown })
		previousDrops := make(map[uint32]bool, len(st.GroundItems))
		for id := range st.GroundItems {
			previousDrops[id] = true
		}
		send(proto.MobAttackPacket(target.ID))
		retryAt := time.Now().Add(time.Second)
		wait(func() bool {
			e := st.Entities[target.ID]
			if st.Self.Dead || e != nil && e.Dead {
				return true
			}
			if e != nil && time.Now().After(retryAt) {
				send(proto.MovePacket(0, 0, e.X+1, e.Y, e.Z))
				send(proto.MobAttackPacket(e.ID))
				retryAt = time.Now().Add(time.Second)
			}
			return false
		})
		if st.Self.Dead {
			t.Fatal("combat fixture died before defeating mob")
		}
		killed++
		wait(func() bool { return st.Details.Exp > initialExp || st.Slots[0].Level > initialLevel })
		for id, item := range st.GroundItems {
			if !previousDrops[id] {
				drops = append(drops, item)
			}
		}
		t.Logf("defeated mob %d (%d), exp=%d", target.ID, target.TypeID, st.Details.Exp)
	}
	send(proto.StopAttackPacket())
	if len(drops) == 0 {
		t.Fatal("no drop after 24 kills; loot not verified")
	}
	item := drops[0]
	id := item.ID
	beforeMoney := st.Details.Money
	beforeCount := 0
	for _, it := range st.Inventory {
		beforeCount += int(it.Count)
	}
	send(proto.MovePacket(0, 0, item.X, item.Y, item.Z))
	send(proto.PickupPacket(id))
	wait(func() bool {
		count := 0
		for _, it := range st.Inventory {
			count += int(it.Count)
		}
		_, remains := st.GroundItems[id]
		return !remains && (st.Details.Money > beforeMoney || count > beforeCount)
	})
	for _, it := range st.Inventory {
		if it.Type == 26 && it.TypeID == 0 {
			t.Fatal("gold inserted into inventory")
		}
	}
	t.Logf("picked up drop %d type=%d/%d", id, item.Type, item.TypeID)
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
}

func TestDeathAndReviveAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_DEATH") != "1" {
		t.Skip("requires low-HP disposable fixture")
	}
	st, send, wait := enterDisposableWorld(t)
	target := nearestLiveMob(t, st, send, wait)
	send(proto.MovePacket(0, 0, target.X+1, target.Y, target.Z))
	send(proto.MobAttackPacket(target.ID))
	wait(func() bool { return st.LastAttack.Source == st.SelfID || st.Self.Dead })
	send(proto.StopAttackPacket())
	wait(func() bool { return st.Self.Dead })
	if st.Self.Speed == nil || st.Self.Speed.Attack != 7 || st.Self.Speed.Move != 2 {
		t.Fatal("Rapture's 0x051C state was not decoded before death", st.Self.Speed)
	}
	t.Logf("death confirmed; HP=%d", st.Pools.HP)
	x, z := st.Self.X, st.Self.Z
	send(proto.RevivePacket())
	wait(func() bool { return !st.Self.Dead && st.Pools.HP > 0 && (st.Self.X != x || st.Self.Z != z) })
	t.Logf("revived at %.2f %.2f %.2f; HP=%d", st.Self.X, st.Self.Y, st.Self.Z, st.Pools.HP)
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
}

// Querying the target also drains the initial entity snapshot before choosing again.
func nearestLiveMob(t *testing.T, st *game.State, send func([]byte), wait func(func() bool)) *game.Entity {
	t.Helper()
	for {
		var target *game.Entity
		wait(func() bool {
			var nearest float32 = 1e9
			for _, e := range st.Entities {
				if e.Kind != game.KindMob || e.Dead || e.TypeID != 2009 || e.HealthKnown && e.HP <= 0 {
					continue
				}
				dx, dz := e.X-st.Self.X, e.Z-st.Self.Z
				if d := dx*dx + dz*dz; d < nearest {
					nearest, target = d, e
				}
			}
			return target != nil
		})
		known := target.HealthKnown
		target.HealthKnown = false
		send(proto.TargetMobPacket(target.ID))
		wait(func() bool { return target.HealthKnown })
		if known && target.HP > 0 {
			return target
		}
	}
}

func TestCastAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_CAST") != "1" {
		t.Skip("requires caster fixture")
	}
	st, send, wait := enterDisposableWorld(t)
	send(proto.LearnSkillPacket(37, 1))
	wait(func() bool { return st.LastLearn.ID == 37 })
	if st.LastLearn.Result != 0 {
		t.Fatalf("learn rejected: %+v", st.LastLearn)
	}
	target := nearestLiveMob(t, st, send, wait)
	send(proto.MovePacket(0, 0, target.X+3, target.Y, target.Z))
	send(proto.UseSkillPacket(st.LastLearn.Number, target.ID, false))
	wait(func() bool { return st.LastCast.Source == st.SelfID && st.LastCast.ID == 37 })
	started := time.Now()
	wait(func() bool { return st.LastSkill.Source == st.SelfID && st.LastSkill.ID == 37 })
	if elapsed := time.Since(started); elapsed < time.Second {
		t.Fatalf("cast resolved too soon: %s", elapsed)
	}
	t.Logf("cast skill %d → result %d after %s", st.LastCast.ID, st.LastSkill.Result, time.Since(started))
	bar, err := proto.SkillBarPacket([]proto.QuickSlot{{Slot: 0, Bag: 100, Number: uint16(st.LastLearn.Number)}})
	if err != nil {
		t.Fatal(err)
	}
	send(bar)
	send(proto.StopAttackPacket())
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
}
