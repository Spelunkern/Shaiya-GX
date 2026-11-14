package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

// Costs are the original ReqIg table, with no character-dependent discount.
func forgeCost(original uint32) uint32 { return original }
func gemChance(_ *db.Character, faction uint8, gem, hammer *db.ItemDef) int {
	chance := max(289-8*int(gem.ReqIg)+int(kcWorldType(faction, 24)), 1)
	if validForgeHammer(hammer, specialGemSuccessUp) {
		chance += max(int(hammer.ReqVg), 0)
	}
	return min(chance, 100)
}
func validForgeHammer(d *db.ItemDef, special uint8) bool {
	return d != nil && realType(d.Type) == 26 && d.Special == special
}
func extractionChance(d *db.ItemDef) int              { return min(max(int(d.ReqVg), 0), 100) }
func gemCost(_ *db.Character, gem *db.ItemDef) uint32 { return lapisiaMoney(gem.ReqIg) }
func enchantRate(item db.Item, _ uint16) (uint32, bool) {
	craft, _ := ParseCraftName(item.CraftName)
	step := craft.EnchantLevel
	if step > 50 {
		step -= 50
	}
	if step >= enchantConfig.MaxLevel {
		return 0, false
	}
	rate := enchantConfig.DefenseRate[step]
	if rt := realType(item.Type); rt >= 1 && rt <= 15 {
		rate = enchantConfig.WeaponRate[step]
	}
	return min(rate, 1000000), true
}
func (s *Session) forgeQuote(q proto.ForgeQuoteRequest) proto.ForgeQuote {
	out := proto.ForgeQuote{Serial: q.Serial, Kind: q.Kind}
	if s.char == nil {
		return out
	}
	defs := db.MustDefs()
	get := func(k [2]uint8) (*db.ItemDef, *db.Item) {
		i := findItem(s.items, k[0], k[1])
		if i < 0 || s.items[i].Count == 0 {
			return nil, nil
		}
		it := &s.items[i]
		return defs.Item(it.Type, it.TypeID), it
	}
	target, item := get(q.Item)
	if item == nil || target == nil || item.Count != 1 {
		return out
	}
	lap, lapItem := get(q.Lapis)
	switch q.Kind {
	case proto.ForgeQuoteLink:
		if lap == nil || lapItem.Type != gemItemType || !lapisiaFits(lap, realType(item.Type)) || !gemFits(item, target, lap) {
			return out
		}
		hammer, _ := get(q.Hammer)
		out.Rate = uint32(gemChance(s.char, s.faction, lap, hammer)) * 10000
		out.Gold = gemCost(s.char, lap)
		out.Valid = true
	case proto.ForgeQuoteExtract:
		hammer, tool := get(q.Hammer)
		if !validForgeHammer(hammer, specialGemExtract) || tool.Bag == 0 || q.Count >= maxGemSlots || item.Gems[q.Count] == 0 || realType(item.Type) == 24 {
			return out
		}
		gem := defs.Item(gemItemType, item.Gems[q.Count])
		if gem == nil {
			return out
		}
		out.Rate = uint32(extractionChance(hammer)) * 10000
		out.Gold = gemCost(s.char, gem)
		out.Valid = true
	case proto.ForgeQuoteEnchant:
		if target.Server == 0 || target.ReqLuc <= 0 {
			return out
		}
		out.Material = uint32(target.ReqLuc)
		var ok bool
		out.Rate, ok = enchantRate(*item, 1)
		if q.Lapis != [2]uint8{255, 255} && (lap == nil || lapItem.Type != lapisiaItemType || !lapisiaFits(lap, realType(item.Type))) {
			return out
		}
		if q.Material != [2]uint8{255, 255} {
			_, mat := get(q.Material)
			if mat == nil || mat.ItemID() != target.ReqLuc || mat.Count < 1 {
				return out
			}
		}
		if lap == nil || lapItem.Type != lapisiaItemType || !lapisiaFits(lap, realType(item.Type)) {
			return out
		}

		out.Gold = forgeCost(lapisiaMoney(lap.ReqIg))
		out.Valid = ok
	}
	return out
}
func (s *Session) handleForgeQuote(p []byte) {
	q, err := proto.DecodeForgeQuoteRequest(p)
	if err != nil {
		return
	}
	out := s.forgeQuote(q)
	w := packet.New(packet.GXForgeQuote)
	w.U32(out.Serial)
	w.U8(out.Kind)
	w.Bool(out.Valid)
	w.U32(out.Rate)
	w.U32(out.Gold)
	w.U8(out.Lapis)
	w.U32(out.Material)
	s.send(w.MustFinish(), "GX forge quote")
}
