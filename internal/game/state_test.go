package game

import (
	"encoding/binary"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestNpcPatrolMovementUpdatesAuthoritativePosition(t *testing.T) {
	s := New()
	s.Entities[42] = &Entity{ID: 42, Kind: KindNpc, X: 1, Y: 2, Z: 3}
	w := proto.NewWriter(proto.MapNpcMove)
	w.U32(42)
	w.U8(0)
	w.F32(4)
	w.F32(5)
	w.F32(6)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	e := s.Entities[42]
	if e.X != 4 || e.Y != 5 || e.Z != 6 || e.MoveSerial != 1 {
		t.Fatalf("bad npc movement: %+v", e)
	}
	if _, err := s.Apply(w.Bytes()[:10]); err == nil || e.MoveSerial != 1 || e.X != 4 {
		t.Fatal("truncated packet changed NPC")
	}
	s.Entities[42] = &Entity{ID: 42, Kind: KindMob, X: 1}
	if _, err := s.Apply(w.Bytes()); err != nil || s.Entities[42].X != 1 {
		t.Fatal("NPC movement changed another entity kind")
	}
}

func TestPeriodicEffectsDoNotPredictHealth(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Pools.HP = 100
	s.Entities[2] = &Entity{ID: 2, Kind: KindMob, HP: 80, HealthKnown: true}
	for _, tc := range []struct {
		op     proto.Opcode
		target uint32
	}{{proto.SkillKeep, 1}, {proto.MobPeriodicDamage, 2}} {
		w := proto.NewWriter(tc.op)
		w.U32(tc.target)
		w.U16(17)
		w.U8(2)
		w.U16(12)
		w.U16(3)
		w.U16(4)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		if s.LastPeriodicEffect != (proto.PeriodicEffect{Target: tc.target, Skill: 17, Level: 2, HP: 12, Resource1: 3, Resource2: 4}) {
			t.Fatal(s.LastPeriodicEffect)
		}
		if s.Pools.HP != 100 || s.Entities[2].HP != 80 {
			t.Fatal("periodic visual event predicted health")
		}
		before := s.LastPeriodicEffect
		if _, err := s.Apply(w.Bytes()[:14]); err == nil || s.LastPeriodicEffect != before {
			t.Fatal("truncated pulse changed state")
		}
	}
}

func TestDetectInfoIsAtomicAndDoesNotAssignTarget(t *testing.T) {
	s := New()
	s.Target = 42
	w := proto.NewWriter(proto.SkillDetectInfo)
	w.U32(4)
	w.U16(57)
	w.U8(3)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	want := proto.DetectInfo{DefenseElement: 4, Level: 57, Mode: 3}
	if s.LastDetectInfo != want || s.Target != 42 {
		t.Fatal("detection changed target or decoded incorrectly")
	}
	if _, err := s.Apply(w.Bytes()[:8]); err == nil || s.LastDetectInfo != want {
		t.Fatal("truncated detection mutated state")
	}
}

func TestAppearanceConfirmationUpdatesSelfAndVisiblePlayers(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Slots[0] = proto.CharacterSlot{ID: 1, Race: 3, Hair: 1, Face: 2, Height: 3, Gender: 0}
	s.Self.Shape = &proto.UserShape{ID: 1, Name: "Self", Hair: 1}
	s.Entities[2] = &Entity{ID: 2, Kind: KindUser, Shape: &proto.UserShape{ID: 2, Name: "Remote", Hair: 1}}
	packet := func(id uint32) []byte {
		w := proto.NewWriter(proto.ChangeAppearance)
		w.U32(id)
		w.U8(4)
		w.U8(5)
		w.U8(6)
		w.U8(1)
		return w.Bytes()
	}
	old := s.Entities[2].Shape
	if _, err := s.Apply(packet(2)); err != nil {
		t.Fatal(err)
	}
	if got := s.Entities[2].Shape; got == old || got.Name != "Remote" || got.Hair != 4 || got.Face != 5 || got.Height != 6 || got.Gender != 1 {
		t.Fatalf("remote shape: %+v", got)
	}
	if s.Slots[0].Hair != 1 || old.Hair != 1 {
		t.Fatal("remote update changed self or old shape")
	}
	if _, err := s.Apply(packet(1)[:9]); err == nil || s.Slots[0].Hair != 1 {
		t.Fatal("truncated appearance changed state")
	}
	if _, err := s.Apply(packet(1)); err != nil {
		t.Fatal(err)
	}
	if got := s.Slots[0]; got.Race != 3 || got.Hair != 4 || got.Face != 5 || got.Height != 6 || got.Gender != 1 {
		t.Fatalf("self slot: %+v", got)
	}
	if s.Self.Shape.Hair != 4 || s.Self.Shape.Name != "Self" {
		t.Fatal("self shape not updated")
	}
}

func TestRemoteShapeSurvivesEnterAndMove(t *testing.T) {
	s := New()
	s.SelfID = 1
	shape := make([]byte, 91)
	binary.LittleEndian.PutUint16(shape, uint16(proto.CharacterShape))
	binary.LittleEndian.PutUint32(shape[2:], 42)
	copy(shape[45:], "Remote")
	if _, err := s.Apply(shape); err != nil {
		t.Fatal(err)
	}
	before := s.Entities[42].Shape
	enter := proto.NewWriter(proto.CharacterEnteredMap)
	enter.U32(42)
	enter.U8(1)
	enter.U16(0)
	enter.F32(10)
	enter.F32(20)
	enter.F32(30)
	enter.U32(0)
	enter.U32(0)
	if _, err := s.Apply(enter.Bytes()); err != nil {
		t.Fatal(err)
	}
	if e := s.Entities[42]; e.Shape != before || e.X != 10 {
		t.Fatal("enter lost appearance or position")
	}
	move := proto.NewWriter(proto.CharacterMove)
	move.U32(42)
	move.U8(1)
	move.U16(123)
	move.F32(11)
	move.F32(21)
	move.F32(31)
	if _, err := s.Apply(move.Bytes()); err != nil {
		t.Fatal(err)
	}
	if e := s.Entities[42]; e.Shape != before || e.X != 11 || e.Dir != 123 {
		t.Fatal("move lost appearance or position")
	}
	if _, err := s.Apply(shape[:20]); err == nil {
		t.Fatal("accepted truncated shape")
	}
	if s.Entities[42].Shape != before {
		t.Fatal("malformed shape mutated entity")
	}
	leave := proto.NewWriter(proto.UserLeave)
	leave.U32(42)
	if _, err := s.Apply(leave.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(s.Entities) != 0 {
		t.Fatal("departed player retained")
	}
}

func TestEmptySlotClearsRoster(t *testing.T) {
	s := New()
	s.Slots[3] = proto.CharacterSlot{Slot: 3, ID: 99}
	w := proto.NewWriter(proto.CharacterList)
	w.U8(3)
	w.U32(0)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(s.Slots) != 0 {
		t.Fatal("empty server slot did not clear roster")
	}
}

func TestSpeedSurvivesPositionUpdatesAndMalformedUpdateIsAtomic(t *testing.T) {
	s := New()
	s.SelfID = 10
	w := proto.NewWriter(proto.CharacterDeathState)
	w.U32(20)
	w.U8(7)
	w.U8(255)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Entities[20].Speed.Move != 255 || s.Self.Speed != nil {
		t.Fatal("remote speed changed the local character")
	}
	before := s.Entities[20].Speed
	if _, err := s.Apply(w.Bytes()[:7]); err == nil || s.Entities[20].Speed != before {
		t.Fatal("truncated speed changed the entity")
	}
	w = proto.NewWriter(proto.CharacterMove)
	w.U32(20)
	w.U8(1)
	w.U16(0)
	w.F32(10)
	w.F32(20)
	w.F32(30)
	if _, err := s.Apply(w.Bytes()); err != nil || s.Entities[20].Speed != before {
		t.Fatal("position update lost the speed", err)
	}
}

func TestTeleportClearsOldMapEntitiesButKeepsSameMapView(t *testing.T) {
	s := New()
	s.MapID, s.SelfID, s.Target = 1, 10, 20
	s.Entities[20] = &Entity{ID: 20}
	s.GroundItems[30] = proto.GroundItem{ID: 30}
	teleport := func(mapID uint16) {
		t.Helper()
		w := proto.NewWriter(proto.CharacterMapTeleport)
		w.U32(10)
		w.U16(mapID)
		w.F32(100)
		w.F32(50)
		w.F32(200)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	teleport(1)
	if len(s.Entities) != 1 || len(s.GroundItems) != 1 || s.Target != 20 {
		t.Fatal("same-map teleport erased the view before leave packets")
	}
	teleport(2)
	if len(s.Entities) != 0 || len(s.GroundItems) != 0 || s.Target != 0 || s.MapID != 2 {
		t.Fatal("old-map entities survived the map change")
	}
}

func TestDeletionAndRestorationWaitForSuccessfulServerResult(t *testing.T) {
	s := New()
	s.Slots[0] = proto.CharacterSlot{ID: 10}
	s.Slots[1] = proto.CharacterSlot{ID: 20}
	apply := func(op proto.Opcode, result uint8, id uint32) {
		t.Helper()
		w := proto.NewWriter(op)
		w.U8(result)
		w.U32(id)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	apply(proto.DeleteCharacter, 1, 10)
	if s.Slots[0].PendingDelete {
		t.Fatal("rejected deletion changed the roster")
	}
	apply(proto.DeleteCharacter, 0, 10)
	if !s.Slots[0].PendingDelete || s.Slots[1].PendingDelete || len(s.Slots) != 2 {
		t.Fatal("successful soft deletion removed a slot or changed another character")
	}
	apply(proto.RestoreCharacter, 1, 10)
	if !s.Slots[0].PendingDelete {
		t.Fatal("rejected restore changed the roster")
	}
	apply(proto.RestoreCharacter, 0, 10)
	if s.Slots[0].PendingDelete {
		t.Fatal("successful restoration did not clear deletion")
	}
}

func TestCombatUsesAuthoritativeHealth(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Target = 55
	s.Entities[55] = &Entity{Kind: KindMob, ID: 55}
	hp := proto.NewWriter(proto.TargetMobHpUpdate)
	hp.U32(55)
	hp.U32(100)
	if _, err := s.Apply(hp.Bytes()); err != nil {
		t.Fatal(err)
	}
	attack := proto.NewWriter(proto.CharacterMobAutoAttack)
	attack.U8(0)
	attack.U32(1)
	attack.U32(55)
	attack.U16(20)
	attack.U16(0)
	attack.U16(0)
	if _, err := s.Apply(attack.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.LastAttack.HP != 20 || s.Entities[55].HP != 100 {
		t.Fatal("attack predicted damage instead of awaiting server health")
	}
	pools := proto.NewWriter(proto.CharacterCurrentHitpoints)
	pools.U32(90)
	pools.U32(50)
	pools.U32(80)
	if _, err := s.Apply(pools.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Pools.HP != 90 || s.Pools.SP != 80 {
		t.Fatal("incorrect pools")
	}
	if _, err := s.Apply(pools.Bytes()[:8]); err == nil {
		t.Fatal("accepted truncated pools")
	}
	if s.Pools.HP != 90 {
		t.Fatal("partial pools update")
	}
	death := proto.NewWriter(proto.MobDeath)
	death.U32(55)
	death.U8(1)
	death.U32(1)
	if _, err := s.Apply(death.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !s.Entities[55].Dead || s.Entities[55].HP != 0 || s.Target != 0 {
		t.Fatal("death retained target or live state")
	}
}

func TestResurrectionUsesServerPositionAndPools(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Self.Dead = true
	s.Duel.Outcome = 2
	w := proto.NewWriter(proto.CharacterResurrect)
	w.U32(1)
	w.U8(4)
	w.U32(123)
	w.F32(100)
	w.F32(2)
	w.F32(200)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Self.Dead || s.Self.X != 100 || s.Self.Z != 200 || s.Details.Exp != 123 {
		t.Fatal("resurrection state mismatch")
	}
	if s.Duel.Outcome != 0 {
		t.Fatal("duel death status remained after resurrection")
	}
	pools := proto.NewWriter(proto.CharacterReviveHitpoints)
	pools.U32(1)
	pools.U32(100)
	pools.U32(20)
	pools.U32(30)
	if _, err := s.Apply(pools.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Pools.HP != 100 || s.Pools.MP != 20 || s.Pools.SP != 30 {
		t.Fatal("resurrection pools mismatch")
	}
}

func TestInventoryMovesAreAtomicAndClearEmptySlots(t *testing.T) {
	s := New()
	s.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1}
	w := proto.NewWriter(proto.ItemBagToBag)
	for _, v := range [][5]uint8{{1, 0, 0, 0, 0}, {0, 5, 1, 1, 1}} {
		w.Raw(v[:])
		w.U16(10)
		w.Raw(make([]byte, 6))
		w.Fixed("", 21, true)
	}
	if _, err := s.Apply(w.Bytes()[:len(w.Bytes())-1]); err == nil {
		t.Fatal("accepted partial item move")
	}
	if len(s.Inventory) != 1 || s.Inventory[[2]uint8{1, 0}].Type != 1 {
		t.Fatal("partial move mutated inventory")
	}
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(s.Inventory) != 1 || s.Inventory[[2]uint8{0, 5}].Type != 1 {
		t.Fatal("move did not apply server slots")
	}
}

func TestGroundItemsAwaitServerRemoval(t *testing.T) {
	s := New()
	w := proto.NewWriter(proto.MapAddItem)
	w.U32(77)
	w.Raw([]byte{1, 1, 1, 2})
	w.F32(1)
	w.F32(2)
	w.F32(3)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	proto.PickupPacket(77)
	if len(s.GroundItems) != 1 {
		t.Fatal("pickup request removed item optimistically")
	}
	remove := proto.NewWriter(proto.MapRemoveItem)
	remove.U32(77)
	if _, err := s.Apply(remove.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(s.GroundItems) != 0 {
		t.Fatal("server removal did not clear loot")
	}
}

func TestSkillsAndResourcesFollowServer(t *testing.T) {
	s := New()
	s.Pools = proto.Pools{HP: 100, MP: 30, SP: 20}
	learned := proto.NewWriter(proto.CharacterSkills)
	learned.U8(1)
	learned.U16(42)
	learned.U8(1)
	learned.U8(7)
	learned.U32(0)
	if _, err := s.Apply(learned.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Skills[7].ID != 42 {
		t.Fatal("lost learned number")
	}
	cost := proto.NewWriter(proto.SkillUsedSpMp)
	cost.U32(4)
	cost.U32(100)
	if _, err := s.Apply(cost.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Pools.SP != 16 || s.Pools.MP != 0 || s.Pools.HP != 100 {
		t.Fatal("resource debit underflow or wrong pool")
	}
	reject := proto.NewWriter(proto.LearnNewSkill)
	reject.U8(1)
	reject.U8(8)
	reject.U16(50)
	reject.U8(1)
	if _, err := s.Apply(reject.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(s.Skills) != 1 {
		t.Fatal("rejected learning changed skills")
	}
	reject.Bytes()[2] = 0
	if _, err := s.Apply(reject.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Skills[8].ID != 50 {
		t.Fatal("confirmed learning missing")
	}
	if _, err := s.Apply(learned.Bytes()[:6]); err == nil {
		t.Fatal("accepted truncated skill list")
	}
	if len(s.Skills) != 2 {
		t.Fatal("truncated list replaced skills")
	}
}

func TestEmptySkillBarSignalsReadiness(t *testing.T) {
	s := New()
	w := proto.NewWriter(proto.CharacterSkillBar)
	w.U8(0)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !s.SkillBarReady || s.Unknown[proto.CharacterSkillBar] != 0 {
		t.Fatal("decoded empty bar did not signal readiness")
	}
}

func TestShopUpdatesOnlyOnSuccess(t *testing.T) {
	s := New()
	s.Details.Money = 500
	w := proto.NewWriter(proto.ShopBuy)
	w.Raw([]byte{1, 1, 2, 1, 1, 1})
	w.U32(0)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Details.Money != 500 || len(s.Inventory) != 0 {
		t.Fatal("refused purchase mutated inventory")
	}
	w.Bytes()[2] = 0
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Details.Money != 0 || s.Inventory[[2]uint8{1, 2}].Count != 1 {
		t.Fatal("confirmed purchase missing")
	}
	sell := proto.NewWriter(proto.ShopSell)
	sell.Raw([]byte{0, 1, 2, 1, 1, 0})
	sell.U32(250)
	if _, err := s.Apply(sell.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Details.Money != 250 || len(s.Inventory) != 0 {
		t.Fatal("confirmed full sale did not clear item")
	}
}

func TestConsumableRequiresConfirmedUse(t *testing.T) {
	s := New()
	s.SelfID = 1
	key := [2]uint8{1, 0}
	s.Inventory[key] = proto.InventoryItem{Bag: 1, Slot: 0, Type: 25, TypeID: 1, Count: 2}
	w := proto.NewWriter(proto.UseItem)
	w.U32(1)
	w.Raw([]byte{0, 0, 0, 0, 0})
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Inventory[key].Count != 2 {
		t.Fatal("rejection consumed item")
	}
	ok := proto.NewWriter(proto.UseItem)
	ok.U32(1)
	ok.Raw([]byte{1, 0, 25, 1, 1})
	if _, err := s.Apply(ok.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Inventory[key].Count != 1 {
		t.Fatal("confirmed use missing")
	}
	ok.Bytes()[10] = 0
	if _, err := s.Apply(ok.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(s.Inventory) != 0 {
		t.Fatal("exhausted stack retained")
	}
}

func TestRemoteEquipmentChangesAppearance(t *testing.T) {
	s := New()
	s.Entities[2] = &Entity{Kind: KindUser, ID: 2, Shape: &proto.UserShape{ID: 2, Name: "Remote"}}
	old := s.Entities[2].Shape
	w := proto.NewWriter(proto.CharacterEquipment)
	w.U32(2)
	w.Raw([]byte{5, 1, 11, 3})
	w.Raw(make([]byte, 6))
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if old.Equipment[5][0] != 0 || s.Entities[2].Shape.Equipment[5][1] != 11 {
		t.Fatal("equipment change did not replace appearance independently")
	}
	if _, err := s.Apply(w.Bytes()[:9]); err == nil {
		t.Fatal("accepted truncated equipment")
	}
}

func TestGoldPickupPreservesOccupiedInventorySlot(t *testing.T) {
	s := New()
	key := [2]uint8{1, 0}
	item := proto.InventoryItem{Bag: 1, Slot: 0, Type: 25, TypeID: 4, Count: 3}
	s.Inventory[key] = item
	s.Details.Money = 100
	w := proto.NewWriter(proto.AddItem)
	w.Raw([]byte{1, 0, 26, 0, 1})
	w.U16(25)
	w.Raw(make([]byte, 6))
	w.Fixed("", 21, true)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(s.Inventory) != 1 || s.Inventory[key] != item {
		t.Fatal("gold acknowledgement replaced inventory item")
	}
	if s.Details.Money != 100 {
		t.Fatal("balance changed before server money update")
	}
	w = proto.NewWriter(proto.SetMoney)
	w.U32(125)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Details.Money != 125 {
		t.Fatal("server money update not applied")
	}
}

func TestTargetStateMarksReceivedZeroHPAsDead(t *testing.T) {
	s := New()
	s.Entities[77] = &Entity{ID: 77, Kind: KindMob}
	w := proto.NewWriter(proto.TargetMobGetState)
	w.U32(77)
	w.U32(0)
	w.U8(5)
	w.U8(2)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if e := s.Entities[77]; !e.Dead || !e.HealthKnown {
		t.Fatal("server corpse remained attackable")
	}
}
