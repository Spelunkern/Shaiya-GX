# Elements: original EP5 rules

GX uses the elemental rules recovered from the original EP5 server and client.
The shared rules live in `internal/element/element.go`. The world remains
responsible for actual damage; tooltips and icons present definitions and
confirmed inventory without authorizing a hit.

## IDs, names and advantage cycle

| ID | Element | Tier |
|---:|---|---|
| 0 | Neutral / non-elemental | None |
| 1 | Fire | I |
| 2 | Water | I |
| 3 | Earth | I |
| 4 | Wind | I |
| 5 | Fire | II |
| 6 | Water | II |
| 7 | Earth | II |
| 8 | Wind | II |

The advantage cycle is **Fire → Wind → Earth → Water → Fire**. Equal elements
and the nonadjacent pair have no advantage. Tier changes both the offensive
bonus and some defensive interactions; do not reduce the system to a universal
25% modifier or assume two linked elements add together.

Original SysMsg 5204–5207 supplies Fire/Water/Earth/Wind; 5270 supplies
non-elemental. The original client formatter `0x0041e240` and Roman conversion
`0x005bf240` establish family and I/II display. Monster target icons use the
original `enemy_elemental` sheet. Spell particles still use their authored
native recipes; no replacement element-specific art is invented.

## Exact server matrix

Rows are the attacker and columns the defender. Entries are multipliers on the
element-sensitive part of the native formula, **not a promise that final damage
will change by exactly that ratio** after all defenses, level terms, criticals,
flat absorption, floors and caps.

| Attack ↓ / Defense → | Neutral | Fire I | Water I | Earth I | Wind I | Fire II | Water II | Earth II | Wind II |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Neutral | 1.0 | 0.8 | 0.8 | 0.8 | 0.8 | 0.7 | 0.7 | 0.7 | 0.7 |
| Fire I | 1.2 | 1.0 | 0.5 | 1.0 | 1.4 | 1.0 | 0.4 | 1.0 | 1.3 |
| Water I | 1.2 | 1.4 | 1.0 | 0.5 | 1.0 | 1.3 | 1.0 | 0.4 | 1.0 |
| Earth I | 1.2 | 1.0 | 1.4 | 1.0 | 0.5 | 1.0 | 1.3 | 1.0 | 0.4 |
| Wind I | 1.2 | 0.5 | 1.0 | 1.4 | 1.0 | 0.4 | 1.0 | 1.3 | 1.0 |
| Fire II | 1.3 | 1.0 | 0.5 | 1.0 | 1.6 | 1.0 | 0.5 | 1.0 | 1.4 |
| Water II | 1.3 | 1.6 | 1.0 | 0.5 | 1.0 | 1.4 | 1.0 | 0.5 | 1.0 |
| Earth II | 1.3 | 1.0 | 1.6 | 1.0 | 0.5 | 1.0 | 1.4 | 1.0 | 0.5 |
| Wind II | 1.3 | 0.5 | 1.0 | 1.6 | 1.0 | 0.5 | 1.0 | 1.4 | 1.0 |

These are the actual float32 constants at `ps_game.exe` VA `0x00542260`.
All 81 bit patterns are checked against an immutable local original by an
optional regression test. The original executable is not redistributed.

The inspected server SHA-256 is
`0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547`.
This page describes that EP5 behavior; other episodes and custom servers can
use different multipliers.

## Equipment and linked lapis

SQL `Items.Attrib` supplies a definition's element. Values above 8 normalize to
neutral. The player's offensive element comes from the equipped weapon (slot 5).
The defensive element comes only from upper-body armor (slot 1). An elemental
helmet, glove, boot, shield or accessory does not independently change the
player's defensive affinity through this native path.

GX uses this explicit priority for the weapon/armor and its six Type30 sockets:

1. A nonzero intrinsic element always wins. Every linked elemental lapis is
   inactive, including another lapis of the same element or a higher tier.
2. Without an intrinsic element, scan sockets 1–6 in their stored order. The
   first lapis with a valid nonzero element supplies the object's element.
3. All subsequent elemental lapis are inactive, even if they repeat the first
   element. Non-elemental stat lapis are not affected by this priority.
4. Inactive elemental lapis stay linked and occupy capacity, but supply no
   elemental/stat/recovery/speed bonus or triggered gem skill.
5. Extracting the first active elemental lapis promotes the next remaining one.
   An intrinsic element continues to win after any extraction.

Multiple elemental lapis may still be linked, subject to the normal target,
capacity and duplicate-TypeID checks. They do not cancel the item to neutral or
combine their elements. For example, intrinsic Fire I plus linked Water II
remains Fire I; a neutral item with Water I then Fire II uses Water I.

**This priority is an intentional GX override requested after the original EP5
research.** The recovered `SetAttrAtk`/`SetAttrDef` conflict-to-neutral behavior
is historical evidence and is no longer the configured item rule. The original
81-entry combat matrix and temporary character-element rules remain unchanged.

Shared inventory tooltips show the resulting element of an instance, including
its current linked lapis, in the original yellow style. A neutral result has no
positive element line. Lapis definitions themselves show their own element and
their compatible target categories. Extraction, linking, equipment moves and
server inventory refreshes update the presentation from the latest instance.
Cached Item.SData definitions are not mutated by a tooltip calculation.
Inactive elemental socket icons use real grayscale in inventory-style tooltips
and blacksmith socket rows; hovering an inactive socket also shows a muted
lapis tooltip. A carried, unlinked lapis retains its original color. Gray status
is recalculated from the current instance, so extraction updates it immediately.

## Element badges on item icons

Weapons, armor, shields and capes with a nonzero effective item element display
a small symbol from the original monster target atlas (`enemy_elemental.png`).
It occupies the upper-left corner and scales with the item icon, up to 14px.
Shortcut numbers and lower-right stack quantities retain their own space.
Tier I and II share their family's symbol; the tooltip names the exact tier.

The badge uses the same intrinsic-plus-lapis calculation as the tooltip. It
refreshes from confirmed instance gems, including linking/extraction and item
moves. The badge follows intrinsic/first-socket priority. Temporary
character buffs do not rewrite an item's own property or badge.

This applies to inventory and equipped cells, horizontal/vertical action bars,
held-item drags, blacksmith slots, personal/guild storage, trade and duel offers,
NPC merchandise and single-item mall/gift previews. Definition-only previews use
the intrinsic element because no instance gems exist yet. A mixed mall bundle
has no single gear affinity. Consumables, lapis/lapisias and jewelry do not gain
an equipment badge merely because their native fields contain an element.

## Player and monster attacks

Player normal attacks use the effective weapon element against the target's
effective defense element, in both PvE and PvP. Physical, ranged and magical
native formulas share the same matrix while retaining their own order of
operations. Element does not independently bypass hit chance, range, attack
category blocks, critical rules, shields or absorption.

A monster's SQL `Mobs.Attrib` is its base **defensive** element. Original
`CGameData::LoadConstMob` (`0x004c1720`) accepts only 0–4 for that field; values
5–8 or other invalid base values normalize to neutral. Client Monster.SData is
normalized to the same supported range, avoiding an icon that contradicts the
server's base affinity. Original content rows remain stored unchanged.

A normal monster attack deliberately uses **attacker row 0**, even when the
monster has an elemental defensive affinity. This is confirmed in
`CMob::AttackDamage` (`0x004a2920`); do not automatically make every Fire monster's
normal hit Fire. Monster skills follow the separate skill rules below.

`Attackattrib1/2` are native attack-template values, normalized to 0–4. The third
attack slot's `Attackattrib3` is overloaded as country/hostility metadata; it is
not a third independent element to add to the creature.

## Spells, area attacks and damage modes

For weapon-based skill damage, a nonzero `Skills.AttrType` overrides the
attacker's ordinary element for that hit. A zero value uses the ordinary weapon
or monster attack-template element. This covers player skills against monsters
and players, and monster skills against players. Each area recipient uses its
own defense element; there is no one multiplier shared by an entire mixed group.

Detail 24 treats the defender's element as neutral for that calculation. It does
not forcibly clear the attacker element: a Fire I attack with detail 24 still
uses Fire I → Neutral (1.2). It does not permanently remove a target's affinity.

Native fixed/resource-based modes bypass the weapon element branch:

| DamageType | Element behavior |
|---:|---|
| 0 | Uses direct HP/SP/MP damage values; no matrix multiplication |
| 1 | Adds damage values to the weapon attack; uses element |
| 2 | Scales the weapon result by damage values; uses element |
| 3 | Caster-resource calculation; no weapon element branch |
| 4 | Target maximum-resource percentage; no weapon element branch |
| 5 | Target current-resource percentage; no weapon element branch |

Periodic damage and healing retain their native periodic/resource formulas.
An elemental description does not introduce a second multiplier on every tick.
Skill tooltips name explicit AttribType elements and tiers. Their element line
uses light blue for Water, gray for Wind, light green for Earth and light orange
for Fire; both tiers share their family color. Existing original
cast/projectile/impact recipes and monster wire-level-100 handling are retained.

Sources include `CUser::AttackDamage` (`0x0044d540` / `0x00457030`) and
`CMob::AttackDamage` (`0x004a2920` / `0x004a3070`).

## Elemental buffs, removal and equipment changes

| Skill detail | Role |
|---:|---|
| 94, buff | Sets the attack element, including an explicit neutral override |
| 95, buff | Sets the defense element |
| 68, debuff | Clears the defense element temporarily |

These fields are **event state**, following native application and removal.
They are not recomputed by blindly replaying whichever elemental buffs remain:

- A different attack-element buff removes prior different attack-element buffs.
  When the replacement expires, an older removed element cannot reappear.
- Removing a weapon also removes its attack-element buffs. A mere inventory
  self-refresh does not remove them. Linked-gem changes on the same equipped
  instance retain an active protected attack override.
- Removing detail 94 restores the current weapon element.
- Removing detail 95 or 68 restores the current armor element, even if a different
  elemental buff remains listed. Replaying that old buff would differ from EP5.
- Original armor removal/addition protection is retained while detail 95 is
  active. Equipped-item changes and expiry can therefore differ from simply
  taking the last listed buff.
- Monsters similarly lose their defense element when detail 68 is applied.
  Expiration restores their base element; overlapping removals follow native
  application/removal order rather than a permanent "any active removal" flag.

Effect lifetimes, cancellation, death handling and existing buff persistence
remain in the normal skill system. On character restoration, saved buffs are
applied to a fresh equipment-derived elemental state. GX does not add a new
permanent character element column or persist a tooltip preview.

## Detection and visibility

Detection (`TypeEffect=6`, detail 123, native response `0x0558`) reads the target
player's live authoritative defensive element. It includes buffs, removal,
expiry and equipment changes. The target's session publishes a locked read-only
snapshot; other sessions never mutate its buff list. The client presents the
original element name/tier rather than an unexplained numeric ID.

Detection's existing target validation, success, costs, level/mode fields and
packet layout remain intact. Mob target icons retain their native definition
presentation; the original mob-removal path does not broadcast a new elemental
HUD packet. GX does not expose every remote player's element through a new
unsolicited protocol extension.

## Configuration and verification

Element IDs reside in SQL item/mob/skill definitions and corresponding external
Item/Monster/Skill.SData. Changing content requires matching data and restarting
the affected component. The fixed original matrix is Go code; it is not a new
INI, environment multiplier or player-level modifier.

Run ordinary and server integration checks from the repository root. To verify
the table against the local original without redistributing it:

```sh
SHAIYA_ORIGINAL_SERVER=/path/to/original/ps_game.exe go test ./internal/element -run TestOriginalEP5ExecutableMatrix -count=1 -v
```

Regression coverage includes all 81 pairs in damage paths, magical damage,
spell overrides, detail24, fixed-damage bypass, intrinsic/first-socket priority,
original labels/tiers, buff replacement, removal and weapon swaps, overlapping
monster removal expiry, live detection and real-asset tooltip rendering.
