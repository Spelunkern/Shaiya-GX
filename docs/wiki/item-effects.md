# Item effects and native field meanings

Item names and descriptions explain content to players; they do not implement an
effect. The world server loads SQL item and skill definitions and executes the
supported behavior. Client Item.SData provides names, artwork, descriptions and
presentation fields. Editing only a description cannot grant a stat or skill.

This page describes current GX development behavior. Historical EP5 or custom
server documentation can describe different rules. See [Blacksmith](blacksmith.md)
for exact forge chances, costs, consumption and safety guarantees.

## Definition versus inventory instance

A definition is identified by `(Type, TypeID)`. A packed identifier is
`Type * 1000 + TypeID`; for example Type 25, ID 1 is 25001. An inventory instance
adds its own unique ID, owner, bag/slot, quantity, quality, six gem entries and
Craftname. Multiple copies share a definition but can have different bonuses.

The server combines equipped instances, their definitions, linked lapis,
recreation attributes and enchantment. Carried equipment contributes no equipped
bonuses. Stacks are limited to 255 and the definition may impose a smaller limit.
Stored quality is retained for compatibility; GX does not implement durability
loss or reduce equipment stats because of old quality values.

## Permanent equipment bonuses

| SQL field | Current use |
|---|---|
| Effect1 / Effect2 | Attack and its added range for weapons, accessories and lapis |
| Effect3 / Effect4 | Defense and resistance for non-weapons |
| ConstStr, ConstDex, ConstRec | Added STR, DEX and REC |
| ConstInt, ConstWis, ConstLuc | Added INT, WIS and LUC |
| ConstHP, ConstMP, ConstSP | Added maximum resources on equipment; see consumables and recovery lapis exceptions below |
| Exp | Flat damage absorption while equipped, not experience gained |
| Attrib | Native element; weapon and upper-body equipment determine offensive/defensive element respectively |
| Slot | Equipment lapis capacity; on a lapis, capacity consumed |

Negative values in the signed stat fields do not subtract stats through the
current equipment accumulation path: it uses nonnegative contributions.
Recreation adds its separately stored values. Weapon enchantment adds its total
attack bonus for the current level; armor enchantment adds flat absorption.
Those bonuses are compiled tables, not sums of preceding levels.

The effective combat totals are calculated on the server. Tooltip descriptions
are not a substitute for confirmed stat packets. In particular, a numeric native
field can have different meaning in a consumable, a gem and a piece of armor.

## Linked lapis: passive and triggered effects

Lapis use Type 30. Linked gem entries store their TypeID and resolve the matching
Type 30 definition. Their ordinary stat bonuses contribute while the containing
item is equipped. Range and AttackTime on a gem additionally modify reach and
attack speed through the equipment path.

Special 16 is the recovery-gem exception: ConstHP/SP/MP contribute to standing and
sitting recovery rather than increasing maximum pools. Other supported passive
stats on that gem continue to apply.

Special 1–15 selects the skill with that ID at **skill level 100**. Only an attack
or debuff skill is accepted by this trigger path. Each resulting skill uses its
own success rules; Special is a skill lookup key, not a percentage.

After a landed hit, current trigger routing is:

| Situation | Gem effects considered |
|---|---|
| Player attacks player | Attacker weapon gems target defender; defender armor gems target attacker |
| Player attacks mob | Attacker weapon gems target mob |
| Mob attacks player | Defender armor gems target mob |

A gem-triggered attack cannot recursively trigger another gem attack. Both
participants must still be alive. Armor scanning follows native order: shield,
helmet, upper armor, gloves, lower armor, shoes, and stops at the first empty
slot. This last detail is implementation behavior, not a recommendation to
leave equipment slots empty.

Source: `server/internal/world/itemoption.go` and `character.go`.

## Elements

See [Elements](elements.md) for the complete original matrix, tier semantics,
spells, player/monster paths, temporary effects and verification evidence.

An intrinsic equipment element always wins. Otherwise the first elemental
lapis in socket order supplies the element. Later elemental lapis stay linked
but inactive and gray; non-elemental lapis remain active. Invalid element values
are ignored. This is GX's explicit priority override; different elements no
longer cancel the equipment to neutral. Element text or a colored icon does not bypass that calculation.

Source: `itemElement` in `server/internal/world/character.go`. Damage interaction
is handled by combat rules; this page does not invent percentages from item
names.

## Healing and item-cast buffs

Supported consumables must be in a valid carried slot and have remaining count.
Ordinary healing uses positive ConstHP/MP/SP as flat restoration. Special 17
interprets those values as percentages of the effective maximum resource. The
server caps the resulting resources through its normal healing path and consumes
one unit for a valid use.

A Special 0 item with nonzero Range and AttackTime can instead cast a skill:
Range selects SkillID and AttackTime selects skill level. The definition must
resolve to a buff skill. Buff duration and abilities come from that skill's row,
not from an invented universal potion multiplier.

Healing and item-cast buffs use ReqIg as a shared cooldown-group key:

| ReqIg | Cooldown |
|---:|---:|
| 1 | 15 seconds |
| 2 | 20 seconds |
| 3 | 25 seconds |
| 4 | 30 seconds |
| 5 | 60 seconds |
| 6 | 120 seconds |
| 10 | 600 seconds |
| 11 | 2 seconds |
| Other | No delay from this lookup |

This is a different use of ReqIg from the forge gold table. Items in the same
cooldown group share that session's group timer. Unsupported Special values are
rejected by the generic use path; they are not silently treated as potions and
are not consumed by that rejection.

Source: `handleUseItem` and `cooldownForReqIg` in
`server/internal/world/inventory.go`.

## Service and utility Special values

| Special | Implemented role / service |
|---:|---|
| 0 | Ordinary healing, or a supported item-cast buff |
| 1–15 on lapis | Trigger a defined attack/debuff skill at level 100 |
| 16 on lapis | Resource recovery gem |
| 17 | Percentage healing potion |
| 34 | Reset allocated character stats through the item-use path |
| 36 | Linking hammer; ReqVg adds percentage points |
| 37 | Rebirth-item handling; validated by the separate rebirth service |
| 44 | Appearance change; validated through the appearance workflow |
| 45 | Sex change; validated through the appearance workflow |
| 51 | Enable the character rename workflow |
| 57 | Historical Lucky Charm; not required or consumed by current safe linking |
| 58 | Extraction hammer; ReqVg is direct success percent |
| 62 | Recreation rune; consumed by the recreation service |

This is a catalog of supported paths, not a claim that every Special value from
other servers works. A service item is not necessarily usable by double-clicking
it. For example, recreation requires equipment and a rune at the blacksmith;
extraction requires a selected socket and an extraction hammer.

## Compatible targets in gem tooltips

Both lapis (Type 30) and lapisias (Type 95) list compatible equipment categories
under native SysMsg 5181, **Item possible to equip**. Category labels come from
original system messages, and placement uses the same native flags:

| Equipment category | Lapis/lapisia flag |
|---|---|
| All weapon families | ReqLevel != 0 |
| Helmets | Country != 0 |
| Upper armor | First class flag != 0 |
| Lower armor | Second class flag != 0 |
| Gauntlets | Third class flag != 0 |
| Shoes | Fourth class flag != 0 |
| Shields | Fifth class flag != 0 |
| Rings | Sixth class flag != 0 |
| Necklaces | Grow != 0 |
| Coat / cape | Accepted by the current native-family compatibility rule |
| Bracelets | ReqStr != 0 |

These flags do not impose player level, faction, profession, mode or STR
requirements on linking. The tooltip therefore does not display their ReqLevel
flag as a character level requirement. Normal equipment retains its usual level
label. Alternate equipment Types are normalized to their native family.

Compatibility is only one requirement. The item must also have room, not already
contain the same lapis TypeID, and satisfy the selected service's remaining
checks. A listed category does not grant sockets or make every item of that
category forgeable. Neither lapis nor lapisias show sockets of their own.

## Visual effects versus gameplay

Weapon enchant glows are presentation. They start at +5; +5 through +10 select
six native tiers, and +11 through +49 reuse the +10 visual tier. The glow does
not set chance, cost or damage. Those remain authoritative server rules.
See [Weapon effects](weapon-effects.md) for family selection, particles,
remote-player updates and limitations.

## How to add or correct an effect

1. Identify the exact Type/TypeID and inspect its SQL definition and matching
   client SData row. Check field overloading for that item family.
2. Confirm that the intended server handler exists. A new Special behavior
   requires code and verification; description edits alone are insufficient.
3. Change the selected database and client definition consistently. Stop the
   affected component and preserve a database backup before content migrations.
4. For skills, validate SkillID, level, effect category, duration and abilities.
5. Verify consumption, rejection, persistence, relogging and the displayed
   tooltip. Use disposable test state for destructive scenarios.

See [Item data](item-data.md), [Building and testing](building-and-testing.md),
and [Server operations](server-operations.md).
