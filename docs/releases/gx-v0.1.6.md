# Shaiya GX gx-v0.1.6 — blacksmith, elements, stealth and world fixes

Update client and server together. This release includes new GX visibility,
forge-quote and map-quest messages. Native Linux and Windows client/server
archives include checksums; client archives include the external licensed font
pack, but game artwork, maps and audio remain separate. Preserve your existing
runtime database when upgrading. The distributed development seed is intentional
test data, not an empty production database.

## Blacksmith, inventory and Item Mall

- Restore original blacksmith buttons and internal controls, inventory-style
  hover tooltips, compatible-item information and original success/failure system
  messages. Remove temporary NPC headings and placeholder instructions.
- Linking and extraction never destroy equipment. Failed extraction destroys
  only the selected lapis. Equipment stays placed after every outcome; remaining
  material, rune and lapisia stacks stay ready for continuous attempts.
- Linking retains original ReqIg gold costs without level or character-stat
  discounts. Base success is `clamp(289 - 8 * ReqIg, 1, 100)` percent; linking
  hammers add ReqVg percentage points, with a final cap of 100%. Extraction tools
  use ReqVg directly as their success percentage. Item descriptions and client
  tooltips explain these values.
- Enhancement consumes one lapisia and one material per attempt, without a
  quantity prompt or confirmation. Success chance appears above Enchant.
  Compatible enhancement materials have a gray preview and normal hover.
- Enhancement rates, bonuses and the configurable default cap of +20 now live
  in Go source (`server/internal/world/enchant_rules.go`). ItemEnchant.ini is
  no longer a runtime dependency; the legacy table remains a reference.
- Restore forge success/failure sounds and the success flare. Recreate executes
  immediately, displays the placed item's live tooltip and retains rune counts.
- Add 18 directed recreation runes: random and perfect versions of STR, DEX,
  REC, INT, WIS, LUC, HP, MP and SP. ReqVg selects the operation. Each changes
  only its designated stat, respecting eligible item pools and maxima. New drops
  receive valid recreation rolls persisted in Craftname. Rejected attempts do
  not consume a rune.
- Recreatable equipment shows its own light-green Max OJ limits below its
  description. Enhancement names include their material icon.
- Repair dragging between inventory bags and repeatable equipment switching
  from either action bar: using the same binding swaps back to the previous item.
- Restore native Cash.SData presentation with SQL ProductList transactions.
  Lapis, lapisias and recreation runes support stacks of 255; matching mall
  products deliver full stacks. Equipment durability and wear penalties are
  removed; repair requests do not charge gold.

## Elements and combat

- Implement the original EP5 elemental damage matrix across player/monster
  combat, weapons, armor and spells, including temporary element effects,
  removal, restoration and detection.
- Add elemental item badges consistently across equipped slots, inventory,
  action bars, blacksmith and other item views. Spell tooltips use elemental
  colors. Lapis tooltips list compatible equipment and base linking chance;
  linking/extraction hammers display their percentage behavior.
- An intrinsic item element takes priority. Otherwise the first elemental lapis
  in socket order is active. Additional elemental lapis may be linked, but stay
  gray and contribute no bonuses or triggered effects. Extracting the active
  lapis promotes the next eligible one.
- Monsters no longer passively regenerate health or refill it while returning
  home. Spawn/respawn health and explicit healing skills retain their rules.

## Stealth and GM controls

- Make stealth server-authoritative: enemies cannot see or target concealed
  players, while authorized allies and GMs see the translucent model. Stealth
  keeps original colors with a subtle fade and correct front/back rendering.
- Concealed players do not attract monsters. Existing attackers retarget eligible
  combat participants or return home. Ordinary stealth breaks on accepted
  attacks or actual damage, without slowing movement or forcing walking.
- GMs enter invisible and invulnerable by default. `/char off` reveals and
  `/char on` conceals; `/attack off` disables protection and `/attack on`
  restores it. GM invisibility persists through attacks and spells.
- Protected GMs can use their own positive buffs, heals and cures with normal
  confirmations, effects and sound. External effects, including allied buffs,
  remain blocked. Restoring protection retains positive buffs and removes
  adverse effects. Positive saved buffs survive login.
- GM command responses and usage messages now appear in green.

## Interface, world and rendering

- Restore native portrait masking, placement and HP/MP/SP bar geometry.
- Restore ground-item labels so their drawing and click rectangles agree.
- Show all available quest locations on the current map, independently of 3D
  NPC view range. Lightweight metadata and event-driven caches avoid repeatedly
  scanning quests; irrelevant inventory changes and movement do not rebuild
  availability. Quest markers remain static and have stable ordering.
- Fix creature depth-state inheritance that made NPCs/monsters flicker or appear
  through terrain, including Windtail Forest Wood Spirits.
- Restore the original finest terrain triangulation and consistent height,
  picking, water, culling and shadow queries. Terrain triangle count drops by
  87.5% without discarding native surface detail.
- Improve window-resolution handling, chat bounds and lower-detail presets;
  keep fixed VSync and the original right-mouse steering behavior. Add placed
  dungeon geometry, bone attachments, monster skill presentation and local-map
  cursor coordinates. Preserve optional AA and persistent chat settings.

## Documentation and validation

The expanded [GitHub Wiki](https://github.com/Spelunkern/Shaiya-GX/wiki) covers
exact blacksmith formulas and tables, item fields/effects, recreation, elements,
stealth, GM commands, quests, protocol, builds and server operation. Its English
sources are versioned under `docs/wiki/`.

Validation covers Go vet/unit tests, server integration, storage/content tools,
race checks, real-resource OpenGL/audio regressions and isolated encrypted
login/world gameplay tests. Release CI builds and packages on native Linux and
Windows runners and verifies relocated server startup. Automated checks do not
substitute for full visual/gameplay testing on every Windows GPU.
