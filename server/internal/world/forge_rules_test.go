package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
)

func forgeRuleSession(t *testing.T) (*Session, func(db.Item) db.Item) {
	t.Helper()
	old := db.MustDefs()
	rates := enchantConfig
	t.Cleanup(func() { db.SetDefsForTest(old); enchantConfig = rates; buildDropCatalog() })
	defs := &db.Defs{Items: map[db.ItemKey]*db.ItemDef{
		{Type: 1, TypeID: 1}:     {Type: 1, TypeID: 1, Count: 1, Slot: 6, Server: 3, ReqWis: 9, ReqLuc: 25043, Grade: 1},
		{Type: 16, TypeID: 1}:    {Type: 16, TypeID: 1, Count: 1, Slot: 6, Server: 3, ReqWis: 9},
		{Type: 30, TypeID: 1}:    {Type: 30, TypeID: 1, Count: 1, Slot: 1, ReqLevel: 1, ReqIg: 30, ReqVg: 1},
		{Type: 30, TypeID: 2}:    {Type: 30, TypeID: 2, Count: 1, Slot: 1, ReqLevel: 1, ReqIg: 30},
		{Type: 95, TypeID: 1}:    {Type: 95, TypeID: 1, Count: 255, ReqLevel: 1, ReqIg: 30, ReqVg: 1},
		{Type: 25, TypeID: 43}:   {Type: 25, TypeID: 43, Count: 255},
		{Type: 100, TypeID: 49}:  {Type: 100, TypeID: 49, Count: 255, Special: 36, ReqVg: 100},
		{Type: 100, TypeID: 140}: {Type: 100, TypeID: 140, Count: 255, Special: 58, ReqVg: 100},
		{Type: 100, TypeID: 141}: {Type: 100, TypeID: 141, Count: 255, Special: 58, ReqVg: 0},
		{Type: 100, TypeID: 171}: {Type: 100, TypeID: 171, Count: 255, Special: specialRecreationRune},
	}, MobDrops: map[uint16][]db.MobDrop{}}
	db.SetDefsForTest(defs)
	id, err := db.CreateCharacter(db.Account{UserUID: 1, UserID: "admin1"}, db.NewCharacter{Slot: 4, Name: "ForgeRuleProbe", HP: 100})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.Pool(db.GameData)
	t.Cleanup(func() {
		pool.Exec("DELETE FROM PS_GameData_CharItems WHERE CharID=?", id)
		pool.Exec("DELETE FROM PS_GameData_Chars WHERE CharID=?", id)
	})
	s, _ := gmWireSession(t, db.Character{ID: id, Level: 1, Money: 20000000, MapID: 60091, CurrentHP: 100}, false)
	add := func(it db.Item) db.Item {
		t.Helper()
		if err := db.AddCharacterItem(id, &it); err != nil {
			t.Fatal(err)
		}
		s.items = append(s.items, it)
		return it
	}
	return s, add
}
func TestForgeCostsAndRatesIgnoreCharacterLevelAndVg(t *testing.T) {
	s, _ := forgeRuleSession(t)
	gem := db.MustDefs().Item(30, 1)
	hammer := db.MustDefs().Item(100, 49)
	for _, level := range []uint16{1, 60, 255} {
		for _, vg := range []int16{-200, 0, 200} {
			s.char.Level = level
			s.char.Vg = vg
			if gemCost(s.char, gem) != 1000 || gemChance(s.char, s.faction, gem, nil) != 49 || gemChance(s.char, s.faction, gem, hammer) != 100 {
				t.Fatal("character altered costs/rates")
			}
		}
	}
	hammer.ReqVg = 20
	if got := gemChance(s.char, s.faction, gem, hammer); got != 69 {
		t.Fatal("hammer is not additive", got)
	}
	enchantConfig.WeaponRate[0] = 900000
	for _, count := range []uint16{1, 10, 100} {
		if got, ok := enchantRate(db.Item{Type: 1}, count); !ok || got != 900000 {
			t.Fatal("material quantity altered probability", got)
		}
	}
}
func TestRepeatedEnchantAndRecreationPersistSameEquipment(t *testing.T) {
	s, add := forgeRuleSession(t)
	item := add(db.Item{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1, Quality: 500})
	lap := add(db.Item{Bag: 1, Slot: 1, Type: 95, TypeID: 1, Count: 3})
	mat := add(db.Item{Bag: 1, Slot: 2, Type: 25, TypeID: 43, Count: 3})
	rune := add(db.Item{Bag: 1, Slot: 3, Type: 100, TypeID: 171, Count: 3})
	enchantConfig.WeaponRate[0], enchantConfig.WeaponRate[1] = 1000000, 1000000
	for n := 0; n < 2; n++ {
		s.handleEnchantAdd(proto.EnchantPacket(1, 1, 1, 2, 1, 1, 0))
	}
	before := s.items[findItemByUID(s.items, item.UID)]
	if c, _ := ParseCraftName(before.CraftName); c.EnchantLevel != 2 {
		t.Fatal("continuous enchant did not advance", c)
	}
	enchantConfig.WeaponRate[2] = 0
	s.handleEnchantAdd(proto.EnchantPacket(1, 1, 1, 2, 1, 1, 0))
	if got := s.items[findItemByUID(s.items, item.UID)]; got.CraftName != before.CraftName || got.Quality != 500 {
		t.Fatal("failed lapisia changed equipment", got)
	}
	if findItemByUID(s.items, lap.UID) >= 0 || findItemByUID(s.items, mat.UID) >= 0 {
		t.Fatal("wrong consumptions")
	}
	for range 2 {
		s.handleItemCompose(proto.ComposePacket(1, 3, 1, 0))
	}
	live := s.items[findItemByUID(s.items, item.UID)]
	validateRecreationRoll(t, db.MustDefs().Item(1, 1), live.CraftName, 2)
	if s.items[findItemByUID(s.items, rune.UID)].Count != 1 {
		t.Fatal("rune stack lost")
	}
	saved, err := db.CharacterItems(s.char.ID)
	if err != nil {
		t.Fatal(err)
	}
	i := findItemByUID(saved, item.UID)
	if i < 0 || saved[i].CraftName != live.CraftName || saved[i].Quality != 500 {
		t.Fatal("craft not persisted")
	}
	if s.char.Money != 20000000-3000 {
		t.Fatal("wrong native costs", s.char.Money)
	}
}
func validateRecreationRoll(t *testing.T, d *db.ItemDef, name string, enchant uint8) {
	t.Helper()
	c, ok := ParseCraftName(name)
	if !ok || c.EnchantLevel != enchant {
		t.Fatal("invalid Craftname", name)
	}
	values := []uint32{c.Str, c.Dex, c.Rec, c.Int, c.Wis, c.Luc, c.MaxHP / 100, c.MaxMP / 100, c.MaxSP / 100}
	nonzero := 0
	for i, v := range values {
		if v > 0 {
			nonzero++
			if v > uint32(d.ReqWis) || rollsStatsOnly(d.Type) && i >= 6 {
				t.Fatal("invalid roll for original requirements", values)
			}
		}
	}
	pool := 9
	if rollsStatsOnly(d.Type) {
		pool = 6
	}
	if nonzero != min(int(d.Server), pool) {
		t.Fatal("wrong number of rolled stats", values)
	}
}
func TestExtractionHammerSuccessFailureOnlyTouchesChosenLapis(t *testing.T) {
	s, add := forgeRuleSession(t)
	item := add(db.Item{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1, Quality: 500, Gems: [6]uint8{1, 2}, CraftName: "01020300000000000002"})
	tool := add(db.Item{Bag: 1, Slot: 1, Type: 100, TypeID: 140, Count: 2})
	fail := add(db.Item{Bag: 1, Slot: 2, Type: 100, TypeID: 141, Count: 2})
	s.handleGemRemove(proto.GemRemoveHammerPacket([2]uint8{1, 0}, [2]uint8{1, 1}, 0))
	it := s.items[findItemByUID(s.items, item.UID)]
	if it.Gems != [6]uint8{0, 2} || s.items[findItemByUID(s.items, tool.UID)].Count != 1 {
		t.Fatal("success touched wrong socket or stack")
	}
	recovered := 0
	for _, g := range s.items {
		if g.Type == 30 && g.TypeID == 1 {
			recovered += int(g.Count)
		}
	}
	if recovered != 1 {
		t.Fatal("recovered lapis missing")
	}
	s.handleGemRemove(proto.GemRemoveHammerPacket([2]uint8{1, 0}, [2]uint8{1, 2}, 1))
	it = s.items[findItemByUID(s.items, item.UID)]
	if it.Gems != [6]uint8{} || it.CraftName != item.CraftName || it.Quality != item.Quality || s.items[findItemByUID(s.items, fail.UID)].Count != 1 {
		t.Fatal("failure destroyed equipment or wrong stack", it)
	}
	for _, g := range s.items {
		if g.Type == 30 && g.TypeID == 2 {
			t.Fatal("failed lapis was granted")
		}
	}
	saved, err := db.CharacterItems(s.char.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved[findItemByUID(saved, item.UID)].Gems != [6]uint8{} {
		t.Fatal("failed extraction not persisted")
	}
	if s.char.Money != 20000000-2000 {
		t.Fatal("extraction quote cost mismatch")
	}
}
func TestNewDropRecreationSurvivesPickupAndRedrop(t *testing.T) {
	s, _ := forgeRuleSession(t)
	defs := db.MustDefs()
	defer clearMapItems(s.char.MapID)
	defs.MobDrops[1] = []db.MobDrop{{ItemOrder: 1, Grade: 1, DropRate: 100}}
	buildDropCatalog()
	drops := rollMobDrops(s.char.MapID, &MapMob{MobID: 1, ID: 1, Level: 1}, s.char.ID, nil, s.log)
	if len(drops) != 1 {
		t.Fatal("missing forced drop", len(drops))
	}
	d := drops[0]
	validateRecreationRoll(t, defs.Item(1, 1), d.Item.CraftName, 0)
	s.handlePickupItem(proto.PickupPacket(d.ID))
	saved, err := db.CharacterItems(s.char.ID)
	if err != nil || len(saved) != 1 || saved[0].CraftName != d.Item.CraftName {
		t.Fatal("pickup changed or lost initial roll", saved, err)
	}
	old := saved[0]
	p := proto.NewWriter(proto.RemoveItem)
	p.U8(old.Bag)
	p.U8(old.Slot)
	p.Count(1)
	s.handleDropItem(p.Bytes())
	ground := itemsOnMap(s.char.MapID)
	if len(ground) != 1 || ground[0].Item.CraftName != old.CraftName {
		t.Fatal("redrop rerolled stats", ground)
	}
	s.handlePickupItem(proto.PickupPacket(ground[0].ID))
	saved, err = db.CharacterItems(s.char.ID)
	if err != nil || len(saved) != 1 || saved[0].CraftName != old.CraftName {
		t.Fatal("second pickup rerolled stats", saved, err)
	}
	plain := newDroppedItem(&db.ItemDef{Type: 25, TypeID: 1, Count: 255})
	if plain.CraftName != "" {
		t.Fatal("non-recreatable item rolled stats")
	}
	for _, typ := range []uint8{1, 16} {
		def := defs.Item(typ, 1)
		for range 20 {
			validateRecreationRoll(t, def, newDroppedItem(def).CraftName, 0)
		}
	}
}
func TestCompiledEnchantCapAndRates(t *testing.T) {
	_, _ = forgeRuleSession(t)
	if enchantConfig.MaxLevel != 20 {
		t.Fatal("default cap must be 20")
	}
	enchantConfig.MaxLevel = 21
	enchantConfig.WeaponRate[20] = 1000000
	it := db.Item{Type: 1, CraftName: "00000000000000000020"}
	if _, ok := enchantRate(it, 1); !ok {
		t.Fatal("configured +20->21 blocked")
	}
	it.CraftName = "00000000000000000021"
	if _, ok := enchantRate(it, 1); ok {
		t.Fatal("configured cap ignored")
	}
	if err := validateEnchantRules(); err != nil {
		t.Fatal(err)
	}
	enchantConfig.MaxLevel = 50
	if err := validateEnchantRules(); err == nil {
		t.Fatal("unrepresentable cap accepted")
	}
	enchantConfig.MaxLevel = 20
	enchantConfig.WeaponRate[0] = 1000001
	if err := validateEnchantRules(); err == nil {
		t.Fatal("rate over 100% accepted")
	}
}
