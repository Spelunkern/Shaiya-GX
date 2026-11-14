# Blacksmith

## Window behavior

The three views use original EP5 artwork and buttons. Place equipment first.
All placed items and socketed lapis have the same mutable inventory tooltip.
Lapis and lapisias have no sockets of their own in that tooltip. Both show
their compatible equipment categories using original SysMsg labels and native
target flags; see [item effects](item-effects.md).

A placed object is a reference to its inventory slot, not a second copy. Clicking
a placed consumable removes the reference without consuming it. Stacks show
their current count. Successful and failed attempts retain equipment and every
remaining stack; only an exhausted/missing item clears its own selection.
Changing service view clears the view's selections.

The only gray hint is the required enhancement material. Lapisias are chosen by
the player; the window does not suggest one. Success/gold text is not drawn
permanently in the panel. Link and extract show a fresh server quote in their confirmation. Enhancement
executes directly and shows its server-quoted success percentage above the
Enchant button. Recreation also executes immediately without confirmation.

## Gold

The server charges the original table selected by the consumable's SQL `ReqIg`.
There is no character level term, Vg discount, free-testing override or material
quantity discount. `RAPTURE_FORGE_COST_PERCENT` is no longer read.

| ReqIg | Gold |
|---:|---:|
| 30 | 1,000 |
| 31 | 4,095 |
| 32 | 11,250 |
| 33 | 22,965 |
| 34 | 41,280 |
| 35 | 137,900 |
| 36 | 365,000 |
| 37 | 480,000 |
| 38 | 627,000 |
| 39 | 814,000 |
| 40 | 1,040,000 |
| 96 | 2,500,000 |
| 97 | 3,500,000 |
| 98 | 5,500,000 |
| 99 | 7,500,000 |
| 100 | 9,800,000 |

Other ReqIg values have zero cost in the native table. Linking uses the incoming
lapis's ReqIg, extraction the selected socketed lapis's ReqIg, and enhancement
the selected lapisia's ReqIg. Hammer ReqVg does not set the gold cost.

Source: [`lapisiaMoney`](../../server/internal/world/crafting.go), with shared
quote/execution helpers in [`forge_quote.go`](../../server/internal/world/forge_quote.go).

## Linking lapis

The equipment must have room for the lapis and accept its native target flags.
Capacity is `equipment.Slot - sum(linkedLapis.Slot)`; there are at most six stored
socket entries. The same lapis TypeID cannot be linked twice to one item.

Incompatible drops are rejected immediately with original SysMsg 62, duplicate
lapis with 63, and insufficient capacity with 64. Inventory does not change.
The server independently repeats validation before consuming anything.

Probability, expressed as a percentage:

```text
base = max(1, 289 - 8 * lapis.ReqIg + factionBlessingType24)
chance = min(100, base + linkingHammerBonus)
```

The character's level and Vg do not participate. Blessing remains a faction-wide
modifier from the loaded KillStatus tiers. With no hammer, its bonus is zero.
A recognized linking hammer has real item family 26 and `Special=36`;
`max(ReqVg, 0)` adds percentage points, not a multiplier.

| Hammer | ReqVg / added percentage points |
|---|---:|
| Lv1 | 5 |
| Lv2 | 10 |
| Lv3 | 15 |
| Lv4 | 20 |
| Lv5 | 25 |
| Lv6 | 30 |
| Lv7 | 40 |
| Common Linking Hammer | 20 |
| Operator's Exclusive | 100 |

For example, ReqIg 31 gives 41% before blessing. A common hammer makes that 61%.
The operator hammer reaches 100%. These are the current content values, editable
in SQL and the matching client definition; see [item data](item-data.md).

A valid attempt consumes gold, one lapis and one selected hammer whether it
succeeds or fails. A refused attempt consumes nothing. Failure preserves the
entire equipment instance and its already linked gems. Lucky Charms are not
consumed. The native confirmation uses the cost/rate/question wording of SysMsg 10011;
its obsolete tendency-dependent cost introduction is omitted. Results use 55/56.

## Enhancement with lapisias

The target must have `Server>0`, a required material in `ReqLuc`, and a target
family compatible with the lapisia. The material must have packed ID
`Type*1000+TypeID == equipment.ReqLuc`. Each valid attempt consumes exactly one lapisia, one material and gold.
There is no quantity selector or confirmation. Legacy requests with a larger
positive material count also consume only one. The server supplies the chance
for the placed equipment before consumables are selected; this preview does
not authorize execution.

All probabilities, the maximum level and resulting stat bonuses are **compiled
Go rules**, in [`enchant_rules.go`](../../server/internal/world/enchant_rules.go).
There is no ItemEnchant.ini runtime dependency. Default cap: **+20**.

| Resulting level | Weapon chance | Armor chance | Weapon attack bonus | Armor absorption |
|---:|---:|---:|---:|---:|
| +1 | 90% | 99% | +7 | 5 |
| +2 | 80% | 98% | +14 | 10 |
| +3 | 70% | 97% | +21 | 15 |
| +4 | 60% | 80% | +31 | 20 |
| +5 | 40% | 60% | +41 | 25 |
| +6 | 20% | 40% | +51 | 30 |
| +7 | 10% | 20% | +64 | 35 |
| +8 | 8% | 10% | +77 | 40 |
| +9 | 6% | 8% | +90 | 45 |
| +10 | 1% | 6% | +106 | 50 |
| +11 | 0.6% | 1% | +122 | 55 |
| +12 | 0.2% | 0.6% | +138 | 60 |
| +13 | 0.1% | 0.2% | +157 | 65 |
| +14 | 0.06% | 0.1% | +176 | 70 |
| +15 | 0.02% | 0.06% | +195 | 75 |
| +16 | 0.02% | 0.02% | +217 | 80 |
| +17 | 0.02% | 0.02% | +239 | 85 |
| +18 | 0.02% | 0.02% | +261 | 90 |
| +19 | 0.02% | 0.02% | +286 | 95 |
| +20 | 0.02% | 0.02% | +311 | 100 |

Rates are stored per million: 900000 means 90%. Index N describes N -> N+1;
bonus arrays instead index the resulting level. Bonuses are totals at that
level, not a sum of all preceding rows. Weapon bonuses add equipment attack;
armor absorption subtracts a flat amount from incoming damage, with the
existing minimum-damage rule. Changing MaxLevel, rates or
bonuses requires rebuilding and restarting gx_world. Raising the cap requires
editing the corresponding table entries as well. Zero rates mean zero success.
The original two-digit Craftname encoding supports a shared cap of 0–49;
0 disables new enhancement attempts. Existing stored levels are not lowered
when the cap changes. Values outside the encoding or rates above 100% are rejected
at startup.

Lapisia variants do not add a chance bonus. No lapisia can break equipment:
failure preserves its quality, existing enchantment and recreated stats.
Compatible lapisias stack up to 255 in the current definitions.

## Extraction

Place an extraction hammer in the hammer well and click the socketed lapis to
choose it. If no occupied socket is currently selected, the first occupied one
is used. Each attempt processes **one** lapis, not all sockets.

A recognized extraction hammer has real family 26 and `Special=58`.
Its ReqVg is the direct percentage, clamped to 0–100. Small Extraction Hammer
currently has 30%; Large Extraction Hammer has 90%. Character level, Vg,
blessing and linking hammer bonuses do not modify extraction hammer success.

The server checks the socket, hammer, gold and room for the recovered lapis
before consuming anything. A valid attempt consumes one hammer and the selected
lapis's gold cost. On success, the lapis enters inventory. On failure, that lapis
is destroyed. Equipment, enchantment, recreation and every other socket are
preserved in both cases. A surviving hammer stack stays placed for the next
attempt. Requests without a valid extraction hammer are rejected.

Confirmation uses native SysMsg 840; results use 57/58. The selected-socket
request extends native 0x0802 with hammer coordinates, socket index and a zero
reserved byte; its result retains the original packet layout. After either
outcome the client refreshes both equipment and hammer from the server, including
failed extraction whose socket state cannot be inferred from the result alone.

## Confirmed result feedback

Linking, extraction, enhancement and recreation play the original
`ui_rapis001.ogg` success sound or `ui_rapis002.ogg` failure sound once their
server result has been resolved. Successful operations flash the original
`Lensflare/flare0.jpg` over the placed equipment for 500 ms, using additive
blending. A failure clears a previous success flash. Native WAV sources are
imported as Ogg resources; original game assets remain external to Git.


## Client-side tooltip rates

Lapis show a light-green `Base link chance: N%` footer below their original
description, like the Max OJ equipment footer. The client derives this from
Item.SData ReqIg as `min(100,max(1,289-8*ReqIg))`; blessing and hammers are excluded
from this base display. The actual attempt still uses the server quote.
Linking hammers (`Special=36`, real family26) show `Linking bonus: +N%` from
ReqVg as added percentage points. Extraction hammers (`Special=58`) show
`Extraction chance: N%`, clamped0–100. These footers use client definitions only;
no additional quote request or server mutation is needed to hover an item.
