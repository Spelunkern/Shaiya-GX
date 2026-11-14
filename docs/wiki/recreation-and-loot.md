# Recreation and world drops

## Recreation requirements

Recreation uses the existing native requirements in SQL item definitions:

- The rune has `Special=62` and is carried in inventory.
- The target is one equipment item with `Server>0` and `ReqWis>0`.
- `Server` sets the number of distinct attributes rolled, limited by the
  applicable attribute pool.
- `ReqWis` sets the maximum rolled value, capped at 99 by the native encoding.

The player places equipment and a rune stack and presses the original action
button. There is no confirmation. A valid operation consumes one rune and
replaces the recreation attributes, preserving the current enchantment and
socketed lapis. It has no additional gold charge. The item stays placed; the
rune stack stays until exhausted or removed by clicking its slot.

The inventory tooltip remains visible beside the recreation panel. Its stats
are rebuilt from the latest confirmed inventory instance, not the old selection
snapshot. The client does not predict random stats or consume runes locally.

## Attribute pools and encoding

Weapons (real families 1–15) and real families 22, 23 and 25 roll only the six
base attributes. Other eligible equipment rolls from all nine:

```text
STR, DEX, REC, INT, WIS, LUC, HP, MP, SP
```

Each selected attribute receives an independent integer from 1 through
`min(ReqWis,99)`. Attributes are distinct and unselected attributes become zero.
The number selected is `min(Server, poolSize)`; this is not a guaranteed upgrade
and there is no character-level modifier.

`Craftname` contains 20 decimal digits, ten two-digit fields:

```text
STR DEX REC INT WIS LUC HP MP SP ENCHANT
```

HP/MP/SP fields represent hundreds: field 07 contributes 700. Weapon enchantment
uses the actual level; armor enchantment stores 50+level. Recreation preserves
the last pair. Code and validation:
[`rollCraftName`](../../server/internal/world/crafting.go),
[`ParseCraftName`](../../server/internal/world/character.go).

## New world loot

A newly generated eligible mob or sequence drop receives its recreation roll
once, when its ground item is created. The rules are the same Server/ReqWis and
attribute-pool rules as recreation; its initial enchantment is zero. Ineligible
items receive no recreation roll. Existing drop membership, probabilities,
ownership and party distribution are unchanged.

The ground item carries that Craftname in memory. On pickup, the normal item
persistence path stores it in `PS_GameData_CharItems.Craftname`. It survives
inventory moves, reconnects and later storage/trade through the existing item
metadata paths. A ground item is not itself a separately persisted inventory row
before someone picks it up.

Dropping an existing item, losing an existing item through PvP, or picking it up
again does **not** generate another roll. Those paths copy existing Craftname,
gems and quality. Randomizing only at the new-loot creation point prevents
throwing and picking up the same equipment to reroll it for free.

Sources: [`newDroppedItem` / mob loot](../../server/internal/world/drops.go),
[pickup and player drops](../../server/internal/world/inventory.go),
[item persistence](../../server/internal/db/items.go).

## Targeted custom recreation runes

All variants retain `Special=62`; `ReqVg` selects their extra behavior. The
ordinary rune uses 0 and retains the full native reroll. Custom rune keys use
Type 101, TypeID 1–18, sharing the original rune artwork and service effect.

| Stat | Random TypeID / ReqVg | Perfect TypeID / ReqVg |
|---|---|---|
| STR | 1 / 101 | 10 / 201 |
| DEX | 2 / 102 | 11 / 202 |
| REC | 3 / 103 | 12 / 203 |
| INT | 4 / 104 | 13 / 204 |
| WIS | 5 / 105 | 14 / 205 |
| LUC | 6 / 106 | 15 / 206 |
| HP | 7 / 107 | 16 / 207 |
| MP | 8 / 108 | 17 / 208 |
| SP | 9 / 109 | 18 / 209 |

Random runes roll only their named stat from 1 through `min(ReqWis,99)`.
Perfect runes set only that stat to this maximum. HP/MP/SP still use hundreds.
Every other pair, including enchantment, is preserved; socketed lapis survive.
These are not guaranteed upgrades except for the selected perfect stat.

The original six-stat equipment pool still disallows HP/MP/SP. Adding a stat
that was zero requires room under `min(Server,poolSize)`; if the item is already
at its distinct-stat limit, the operation is rejected rather than deleting a
sibling stat. Updating a stat already present does not need another OJ slot.
Unknown nonzero ReqVg values are rejected. Refused operations consume nothing.
Each accepted operation consumes exactly one rune and keeps the remaining stack
placed. All 18 variants stack to 255 and have dedicated 255-unit mall products.

`tools/update_recreation_content.py` creates the definitions and aligned native
mall products in explicitly selected offline databases/client data, with backups
and trial migrations. It updates only Items and ProductList and never copies a
personal runtime over the seed or rewrites owned item quantities.

## Tooltip limits

Eligible equipment shows light-green `Max OJ X/Y` beneath its description.
X is `min(ReqWis,99)` for ordinary stat pairs; Y is X multiplied by 100 for
resource pairs. For example, `Max OJ 13/1300` means at most 13 STR/DEX/etc. or
1300 HP/MP/SP for an allowed stat. It does not mean 13 distinct attributes.
The distinct-stat count remains Server and the equipment pool.
The required enhancement material is displayed once with a text-sized icon
before its name; that inline icon does not have an independent hover tooltip.
