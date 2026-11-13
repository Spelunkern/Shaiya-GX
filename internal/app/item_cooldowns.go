package app

import (
	"fmt"
	"math"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// ReqIg is a shared reuse group, not a delay in seconds. This is the native
// ItemUse table, also enforced by Rapture cooldownForReqIg.
func itemReuseSeconds(group uint8) float64 {
	switch group {
	case 1:
		return 15
	case 2:
		return 20
	case 3:
		return 25
	case 4:
		return 30
	case 5:
		return 60
	case 6:
		return 120
	case 10:
		return 600
	case 11:
		return 2
	}
	return 0
}

func (a *App) itemCooldown(it proto.InventoryItem) float64 {
	if a.arc == nil && a.items == nil {
		return 0
	}
	if d := a.itemsItem(it.Type, it.TypeID); d != nil {
		return max(0, a.inventory.cooldowns[d.ReqIg]-a.clock)
	}
	return 0
}

func cooldownFlash(until, now float64) float32 {
	elapsed := now - until
	if until <= 0 || elapsed < 0 || elapsed >= .8 {
		return 0
	}
	return float32((1 + math.Cos(elapsed*math.Pi*8)) * .35 * (1 - elapsed/.8))
}

type potionRestoration struct {
	name     string
	before   proto.Pools
	pools    [3]bool
	deadline float64
}

// Never start a reuse timer on a request or a refusal. One ACK starts every
// shortcut/inventory icon in the same group, regardless of bag or item ID.
func (a *App) confirmedItems(op proto.Opcode) {
	v := &a.inventory
	if op == proto.UseItem {
		it := a.state.LastItemUse
		if it.ID != a.state.SelfID || it.Type == 0 {
			return
		}
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			if seconds := itemReuseSeconds(d.ReqIg); seconds > 0 {
				if v.cooldowns == nil {
					v.cooldowns = map[uint8]float64{}
				}
				v.cooldowns[d.ReqIg] = a.clock + seconds
			}
			if (d.Special == 0 || d.Special == 17) && (d.RecoveryHP > 0 || d.RecoveryMP > 0 || d.RecoverySP > 0) {
				v.restoration = &potionRestoration{d.Name, a.state.Pools, [3]bool{d.RecoveryHP > 0, d.RecoveryMP > 0, d.RecoverySP > 0}, a.clock + 2}
			}
		}
	}
	if op != proto.CharacterCurrentHitpoints || v.restoration == nil {
		return
	}
	r := v.restoration
	v.restoration = nil
	if a.clock > r.deadline {
		return
	}
	for i, pool := range []struct {
		name          string
		before, after uint32
	}{
		{"HP", r.before.HP, a.state.Pools.HP}, {"MP", r.before.MP, a.state.Pools.MP}, {"SP", r.before.SP, a.state.Pools.SP},
	} {
		if r.pools[i] && pool.after > pool.before {
			a.systemMessage(4, 100, map[string]string{"v": strconv.FormatUint(uint64(pool.after-pool.before), 10), "hp": pool.name, "i": r.name})
		}
	}
}

func (a *App) drawItemCooldown(it proto.InventoryItem, x, y, size int) {
	if left := a.itemCooldown(it); left > 0 {
		d := a.itemsItem(it.Type, it.TypeID)
		a.sprites.DrawCooldown(a.white, float32(x), float32(y), float32(size), float32(left/itemReuseSeconds(d.ReqIg)), [4]float32{0, 0, 0, 160.0 / 255})
		text := fmt.Sprintf("%.0f", math.Ceil(left))
		a.font.Draw(a.sprites, text, x+(size-a.font.Width(text))/2, y+(size-13)/2, white, size)
	}
}
