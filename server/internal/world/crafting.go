package world

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// Enchanting (lapisia), gem linking and recreation runes. All of them edit
// CharItems.Craftname / Gem1-6 through usp_Save_Char_Item_Mod_E.

const (
	lapisiaItemType      = 95
	maxGemSlots          = 6
	composeResultSuccess = 0
	composeResultFailure = 1
)

func craftString(pairs [10]uint32) string {
	var b strings.Builder
	for _, v := range pairs {
		fmt.Fprintf(&b, "%02d", min(v, 99))
	}
	return b.String()
}

func craftPairs(c CraftStats) [10]uint32 {
	return [10]uint32{c.Str, c.Dex, c.Rec, c.Int, c.Wis, c.Luc, c.MaxHP / 100, c.MaxMP / 100, c.MaxSP / 100, uint32(c.EnchantLevel)}
}

// --- enchant (0x0805) ---

func buildEnchantResult(result, lBag, lSlot uint8, lCount uint16, mBag, mSlot uint8, mCount uint16, iBag, iSlot uint8, money uint32, craft string) []byte {
	w := packet.New(packet.EnchantAdd)
	w.Bytes([]byte{result, lBag, lSlot})
	w.Count(lCount)
	w.Bytes([]byte{mBag, mSlot})
	w.Count(mCount)
	w.Bytes([]byte{iBag, iSlot})
	w.U32(money)
	w.FixedString(craft, 21)
	return w.MustFinish()
}

// lapisiaMoney is GetGemMoneyByGrade (0x0045f6e0) by the lapisia's ReqIg.
func lapisiaMoney(reqIg uint8) uint32 {
	low := [11]uint32{1000, 4095, 11250, 22965, 41280, 137900, 365000, 480000, 627000, 814000, 1040000}
	high := [5]uint32{2500000, 3500000, 5500000, 7500000, 9800000}
	switch {
	case reqIg >= 30 && reqIg <= 40:
		return low[reqIg-30]
	case reqIg >= 96 && reqIg <= 100:
		return high[reqIg-96]
	}
	return 0
}

// lapisiaFits is CUser::EnableLapisianAdd's target check: real type 24
// always; otherwise the lapisia's own fields name what it enchants
// (ReqLevel weapons, Country type 16, the six job flags types 17-22, Grow
// 23, ReqStr 25).
func lapisiaFits(lap *db.ItemDef, rt uint8) bool {
	switch {
	case rt == 24:
		return true
	case rt >= 1 && rt <= 15:
		return lap.ReqLevel != 0
	case rt == 16:
		return lap.Country != 0
	case rt >= 17 && rt <= 22:
		return lap.Classes[rt-17]
	case rt == 23:
		return lap.Grow != 0
	case rt == 25:
		return lap.ReqStr != 0
	}
	return false
}

// handleEnchantAdd (0x0805) is CUser::ItemLapisianAdd (0x0045fb20):
// lapisia bag/slot, material bag/slot, material count, item bag/slot.
// The item (count 1, Server set) needs its ReqLuc as material; below
// the compiled cap, ReqIg selects the gold cost and the compiled rate
// determines success independently of material quantity. Either way
// the lapisia and the material count are used up; equipment is preserved.
func (s *Session) handleEnchantAdd(p []byte) {
	if len(p) < 9 || s.char == nil {
		return
	}
	ch := s.char
	r := packet.NewReader(p)
	lBag, lSlot, mBag, mSlot := r.U8(), r.U8(), r.U8(), r.U8()
	requestedCount := r.Count()
	count := uint16(1)
	iBag, iSlot := r.U8(), r.U8()
	if r.Err() != nil {
		return
	}
	if !isValidItemPosition(lBag, lSlot) || !isValidItemPosition(mBag, mSlot) || !isValidItemPosition(iBag, iSlot) {
		return
	}
	li, mi, ii := findItem(s.items, lBag, lSlot), findItem(s.items, mBag, mSlot), findItem(s.items, iBag, iSlot)
	if li < 0 || mi < 0 || ii < 0 {
		return
	}
	lap, mat, item := s.items[li], s.items[mi], s.items[ii]
	fail := func(why string) {
		s.log.Printf("GX Server World rejected enchant: %s", why)
		s.send(buildEnchantResult(0, lBag, lSlot, lap.Count, mBag, mSlot, mat.Count, iBag, iSlot, ch.Money, item.CraftName), "0x0805 enchant add result")
	}
	defs := db.MustDefs()
	lapDef, def := defs.Item(lap.Type, lap.TypeID), defs.Item(item.Type, item.TypeID)
	switch {
	case li == mi || li == ii || mi == ii || lBag == 0 || mBag == 0:
		fail("distinct carried consumables required")
		return
	case lapDef == nil || def == nil || lap.Type != lapisiaItemType || item.Count != 1:
		fail("not a lapisia or not a single item")
		return
	case requestedCount == 0 || mat.Count < count:
		fail("material count")
		return
	case def.Server == 0 || def.ReqLuc == 0 || mat.ItemID() != def.ReqLuc:
		fail(fmt.Sprintf("material %d does not match required %d", mat.ItemID(), def.ReqLuc))
		return
	case !lapisiaFits(lapDef, realType(item.Type)):
		fail("lapisia does not fit the item")
		return
	}
	weapon := realType(item.Type) >= 1 && realType(item.Type) <= 15
	craft, _ := ParseCraftName(item.CraftName)
	step := craft.EnchantLevel
	if step > 50 {
		step -= 50
	}
	if step >= enchantConfig.MaxLevel {
		fail("already at max level")
		return
	}
	cost := forgeCost(lapisiaMoney(lapDef.ReqIg))
	if ch.Money < cost {
		fail("not enough gold")
		return
	}
	ch.Money -= cost
	lapLeft, matLeft := lap.Count-1, mat.Count-count
	uidI := item.UID
	s.adjustCount(lap.UID, lapLeft)
	s.adjustCount(mat.UID, matLeft)
	rate, _ := enchantRate(item, count)
	if rand.Uint32N(1_000_000) >= rate {
		name := item.CraftName
		s.persist()
		s.log.Printf("GX Server World enchant failed item=%d/%d step=%d", item.Type, item.TypeID, step)
		s.send(buildEnchantResult(0, lBag, lSlot, lapLeft, mBag, mSlot, matLeft, iBag, iSlot, ch.Money, name), "0x0805 enchant add result")
		return
	}
	pairs := craftPairs(craft)
	next := uint32(step) + 1
	if !weapon {
		next += 50
	}
	pairs[9] = next
	name := craftString(pairs)
	if i := findItemByUID(s.items, uidI); i >= 0 {
		s.items[i].CraftName = name
		db.SaveCharacterItem(ch.ID, &s.items[i])
	}
	s.persist()
	s.log.Printf("GX Server World enchant success item=%d/%d new_step=%d craftname=%s money=%d", item.Type, item.TypeID, step+1, name, ch.Money)
	s.send(buildEnchantResult(1, lBag, lSlot, lapLeft, mBag, mSlot, matLeft, iBag, iSlot, ch.Money, name), "0x0805 enchant add result")
	if iBag == 0 {
		s.equipmentChanged([]uint8{iSlot})
	}
}

// adjustCount sets an item's count by UID (deleting at zero) and persists.
func (s *Session) adjustCount(uid int64, count uint16) {
	i := findItemByUID(s.items, uid)
	if i < 0 {
		return
	}
	s.items[i].Count = count
	if err := db.SaveCharacterItem(s.char.ID, &s.items[i]); err != nil {
		s.log.Printf("GX Server World failed to update item count: %v", err)
	}
	if count == 0 {
		s.items = append(s.items[:i], s.items[i+1:]...)
	}
}

// --- gems (0x0801 / 0x0802) ---

func buildGemAddResult(result, gBag, gSlot uint8, gCount uint16, iBag, iSlot, pos, gemID uint8, money uint32) []byte {
	w := packet.New(packet.GemAdd)
	w.Bytes([]byte{result, gBag, gSlot})
	w.Count(gCount)
	w.Bytes([]byte{iBag, iSlot, pos, gemID})
	w.U32(0) // four unused bytes (client reader 0x0801: 12 bytes + money)
	w.U32(money)
	return w.MustFinish()
}

// maxPools returns the effective max HP/SP/MP (0x050B order 0, 1, 2).
func (s *Session) maxPools() [3]uint32 {
	st := s.stats()
	return [3]uint32{st.MaxHP, st.MaxSP, st.MaxMP}
}

// sendChangedPools sends 0x050B only for pools that changed; a full stat
// resend crashes the client after item mutations.
func (s *Session) sendChangedPools(before [3]uint32, why string) {
	after := s.maxPools()
	for t := range 3 {
		if before[t] != after[t] {
			s.send(buildMaxEnergy(s.char, uint8(t), after[t]), "0x050B max energy ("+why+")")
		}
	}
}

// gemFits is CItem::EnableGem (0x004ba050): the item's Slot is a
// capacity each linked gem uses its own Slot of; the same gem twice is
// refused.
func gemFits(item *db.Item, target, gemDef *db.ItemDef) bool {
	used := 0
	for _, g := range item.Gems {
		if g == 0 {
			continue
		}
		if g == gemDef.TypeID {
			return false
		}
		if d := db.MustDefs().Item(gemItemType, g); d != nil {
			used += int(d.Slot)
		}
	}
	return int(gemDef.Slot) <= int(target.Slot)-used
}

// findSpecialItem finds a usable item of real type 26 with the given
// Special in the bags (CUser::FindItemGemAddItemProtect looks for '9',
// the Lucky Charm).
func (s *Session) findSpecialItem(special uint8) int {
	for i := range s.items {
		it := &s.items[i]
		if it.Bag == 0 || it.Count == 0 || realType(it.Type) != 26 {
			continue
		}
		if d := db.MustDefs().Item(it.Type, it.TypeID); d != nil && d.Special == special {
			return i
		}
	}
	return -1
}

const (
	specialGemSuccessUp = 36 // additive percentage-point bonus
	specialGemExtract   = 58 // direct extraction success percentage
	specialGemProtect   = 57 // legacy Lucky Charm ID; no longer needed/consumed
)

// handleGemAdd (0x0801) is CUser::ItemGemAdd (0x00461310): gem bag/slot,
// item bag/slot, optional hammer bag/slot. The item (count 1) must take
// the gem (the lapisia-style target fields and EnableGem's capacity); the
// gem's GetGemMoneyByGrade is paid; base chance is 289 - 8 x ReqIg
// plus blessing (at least 1), plus hammer ReqVg percentage points, up to 100%.
// The gem is always used up. GX preserves the equipment on every failure;
// legacy ReqVg destruction flags and Lucky Charms do not affect linking.
func (s *Session) handleGemAdd(p []byte) {
	if len(p) < 6 || s.char == nil {
		return
	}
	ch := s.char
	gBag, gSlot, iBag, iSlot := p[2], p[3], p[4], p[5]
	var hBag, hSlot uint8
	if len(p) >= 8 {
		hBag, hSlot = p[6], p[7]
	}
	if !isValidItemPosition(gBag, gSlot) || !isValidItemPosition(iBag, iSlot) {
		return
	}
	gi, ii := findItem(s.items, gBag, gSlot), findItem(s.items, iBag, iSlot)
	if gi < 0 || ii < 0 {
		return
	}
	gem, item := s.items[gi], s.items[ii]
	fail := func(why string) {
		s.log.Printf("GX Server World rejected gem add: %s", why)
		s.send(buildGemAddResult(0, gBag, gSlot, gem.Count, iBag, iSlot, 0, 0, ch.Money), "0x0801 gem add result")
	}
	defs := db.MustDefs()
	target, gemDef := defs.Item(item.Type, item.TypeID), defs.Item(gemItemType, gem.TypeID)
	switch {
	case gem.Type != gemItemType || gem.Count == 0 || gemDef == nil || target == nil || item.Count != 1:
		fail("not a gem or not a single item")
		return
	case !lapisiaFits(gemDef, realType(item.Type)) || !gemFits(&item, target, gemDef):
		fail("gem does not fit the item")
		return
	}
	pos := -1
	for i, g := range item.Gems {
		if g == 0 {
			pos = i
			break
		}
	}
	if pos < 0 {
		fail("no free socket")
		return
	}
	var hammer *db.ItemDef
	var hammerUID int64
	if len(p) >= 8 {
		hi := findItem(s.items, hBag, hSlot)
		if !isValidItemPosition(hBag, hSlot) || hBag == 0 || hi < 0 || s.items[hi].Count == 0 || s.items[hi].UID == item.UID || s.items[hi].UID == gem.UID {
			fail("invalid hammer slot")
			return
		}
		hammer = defs.Item(s.items[hi].Type, s.items[hi].TypeID)
		if !validForgeHammer(hammer, specialGemSuccessUp) {
			fail("not a linking hammer")
			return
		}
		hammerUID = s.items[hi].UID
	}
	cost := gemCost(ch, gemDef)
	if ch.Money < cost {
		fail("not enough gold")
		return
	}
	chance := gemChance(ch, s.faction, gemDef, hammer)
	ch.Money -= cost
	if hammerUID != 0 {
		hi := findItemByUID(s.items, hammerUID)
		s.adjustCount(hammerUID, s.items[hi].Count-1)
	}
	before := s.maxPools()
	uidI := item.UID
	s.adjustCount(gem.UID, gem.Count-1)
	if rand.IntN(100) >= chance {
		s.persist()
		s.log.Printf("GX Server World gem add failed gem=%d chance=%d", gem.TypeID, chance)
		s.send(buildGemAddResult(0, gBag, gSlot, gem.Count-1, iBag, iSlot, 0, 0, ch.Money), "0x0801 gem add result")
		if s.player != nil {
			s.player.UpdateItems(s.items)
		}
		return
	}
	if i := findItemByUID(s.items, uidI); i >= 0 {
		s.items[i].Gems[pos] = gem.TypeID
		db.SaveCharacterItem(ch.ID, &s.items[i])
	}
	s.persist()
	s.log.Printf("GX Server World gem add gem=%d item=%d/%d position=%d chance=%d", gem.TypeID, item.Type, item.TypeID, pos, chance)
	s.send(buildGemAddResult(1, gBag, gSlot, gem.Count-1, iBag, iSlot, uint8(pos), gem.TypeID, ch.Money), "0x0801 gem add result")
	if iBag == 0 {
		s.sendChangedPools(before, "gem add")
	}
	if s.player != nil {
		s.player.UpdateItems(s.items)
	}
}

// handleGemRemove extracts one selected socket with an extraction hammer.
// Obsolete requests without a hammer are rejected without consuming anything.
func (s *Session) handleGemRemove(p []byte) {
	if len(p) < 4 || s.char == nil {
		return
	}
	ch := s.char
	iBag, iSlot := p[2], p[3]
	if !isValidItemPosition(iBag, iSlot) {
		return
	}
	ii := findItem(s.items, iBag, iSlot)
	if ii < 0 {
		return
	}
	item := s.items[ii]
	var removed, ids, bags, slots [maxGemSlots]uint8
	result := func(ok bool) {
		w := packet.New(packet.GemRemove)
		w.Bool(ok)
		w.U8(iBag)
		w.U8(iSlot)
		w.Bytes(removed[:])
		w.Bytes(ids[:])
		w.Bytes(bags[:])
		w.Bytes(slots[:])
		w.U32(ch.Money)
		s.send(w.MustFinish(), "0x0802 gem remove result")
	}
	defs := db.MustDefs()
	positions := []int{}
	toolUID := int64(0)
	rate := -1
	var cost uint32
	if len(p) == 8 {
		hi := findItem(s.items, p[4], p[5])
		pos := int(p[6])
		if hi < 0 || p[4] == 0 || !isValidItemPosition(p[4], p[5]) || pos >= maxGemSlots || p[7] != 0 || s.items[hi].UID == item.UID {
			result(false)
			return
		}
		tool := s.items[hi]
		hd := defs.Item(tool.Type, tool.TypeID)
		if tool.Count == 0 || !validForgeHammer(hd, specialGemExtract) || item.Gems[pos] == 0 {
			result(false)
			return
		}
		rate = extractionChance(hd)
		toolUID = tool.UID
		positions = append(positions, pos)
	} else {
		result(false)
		return
	}
	for _, pos := range positions {
		d := defs.Item(gemItemType, item.Gems[pos])
		if d == nil {
			result(false)
			return
		}
		cost += gemCost(ch, d)
	}
	if len(positions) == 0 || item.Count != 1 || realType(item.Type) == 24 || ch.Money < cost || len(freeInventorySlots(s.items)) < len(positions) {
		result(false)
		return
	}
	before := s.maxPools()
	ch.Money -= cost
	if toolUID != 0 {
		hi := findItemByUID(s.items, toolUID)
		s.adjustCount(toolUID, s.items[hi].Count-1)
	}
	extracted := false
	for _, pos := range positions {
		id := item.Gems[pos]
		chance := rate
		if chance < 0 {
			chance = gemChance(ch, s.faction, defs.Item(gemItemType, id), nil)
		}
		i := findItemByUID(s.items, item.UID)
		if rand.IntN(100) < chance {
			free := freeInventorySlots(s.items)
			g := db.Item{Bag: free[0][0], Slot: free[0][1], Type: gemItemType, TypeID: id, Count: 1}
			if err := db.AddCharacterItem(ch.ID, &g); err != nil {
				s.log.Printf("gem extraction insert: %v", err)
				continue
			}
			s.items = append(s.items, g)
			s.send(buildAddItem(&g), "0x0205 extracted lapis")
			removed[pos], ids[pos], bags[pos], slots[pos] = 1, id, g.Bag, g.Slot
			extracted = true
		}
		// A failed roll destroys only this lapis; other sockets stay untouched.
		s.items[i].Gems[pos] = 0
	}
	i := findItemByUID(s.items, item.UID)
	db.SaveCharacterItem(ch.ID, &s.items[i])
	s.persist()
	result(extracted)
	if iBag == 0 {
		s.sendChangedPools(before, "gem extraction")
	}
	if s.player != nil {
		s.player.UpdateItems(s.items)
	}
}

// --- recreation rune (0x0806) ---

var jewelryItemTypes = []uint8{22, 23, 40}

// rollsStatsOnly is CItem::ReGenerationCraftExpansion's split: weapons
// (real types 1-15) and real types 22, 23 and 25 roll the six stats only.
func rollsStatsOnly(t uint8) bool {
	rt := realType(t)
	return (rt >= 1 && rt <= 15) || rt == 22 || rt == 23 || rt == 25
}

// rollCraftName picks `count` distinct stats (6 for weapons/jewelry, 9
// otherwise) at 1..maxValue each, keeping the enchant level.
func rollCraftName(count, maxValue uint8, statsOnly bool, previous string) string {
	pool := 9
	if statsOnly {
		pool = 6
	}
	var pairs [10]uint32
	for _, i := range rand.Perm(pool)[:min(int(count), pool)] {
		pairs[i] = uint32(rand.IntN(int(max(maxValue, 1)))) + 1
	}
	if c, ok := ParseCraftName(previous); ok {
		pairs[9] = uint32(c.EnchantLevel)
	}
	return craftString(pairs)
}

func buildComposeResult(result, bag, slot uint8, craft string) []byte {
	w := packet.New(packet.ItemCompose)
	w.Bytes([]byte{result, bag, slot})
	w.FixedString(craft, 21)
	return w.MustFinish()
}

// handleItemCompose: rune bag/slot, item bag/slot. Items.ReqWis is the max
// roll and Items.Server the number of rolled stats.
func (s *Session) handleItemCompose(p []byte) {
	if len(p) < 6 || s.char == nil {
		return
	}
	ch := s.char
	rBag, rSlot, iBag, iSlot := p[2], p[3], p[4], p[5]
	if !isValidItemPosition(rBag, rSlot) || !isValidItemPosition(iBag, iSlot) {
		return
	}
	ri, ii := findItem(s.items, rBag, rSlot), findItem(s.items, iBag, iSlot)
	if ri < 0 || ii < 0 {
		s.send(buildComposeResult(composeResultFailure, iBag, iSlot, ""), "0x0806 item compose result")
		return
	}
	rune, item := s.items[ri], s.items[ii]
	fail := func(why string) {
		s.log.Printf("GX Server World rejected item compose: %s", why)
		s.send(buildComposeResult(composeResultFailure, iBag, iSlot, item.CraftName), "0x0806 item compose result")
	}
	defs := db.MustDefs()
	runeDef, itemDef := defs.Item(rune.Type, rune.TypeID), defs.Item(item.Type, item.TypeID)
	switch {
	case rBag == 0 || ri == ii || item.Count != 1:
		fail("distinct carried rune and single equipment required")
		return
	case rune.Count == 0 || runeDef == nil || runeDef.Special != specialRecreationRune:
		fail("not a recreation rune")
		return
	case itemDef == nil || itemDef.ReqWis <= 0 || itemDef.Server == 0:
		fail("item not recreatable")
		return
	}
	name, err := recreateWithRune(runeDef, itemDef, item)
	if err != nil {
		fail(err.Error())
		return
	}
	before := s.maxPools()
	s.items[ii].CraftName = name
	db.SaveCharacterItem(ch.ID, &s.items[ii])
	s.adjustCount(rune.UID, rune.Count-1)
	consumed := rune
	consumed.Count--
	s.log.Printf("GX Server World item compose item=%d/%d craftname=%s", item.Type, item.TypeID, name)
	// Captured order: 0x050B (changed pools), 0x0806, 0x050A.
	if iBag == 0 {
		s.sendChangedPools(before, "compose")
	}
	s.send(buildComposeResult(composeResultSuccess, iBag, iSlot, name), "0x0806 item compose result")
	s.send(buildUseItemResult(ch.ID, &consumed), "0x050A rune consume (compose)")
	if s.player != nil {
		s.player.UpdateItems(s.items)
	}
}
