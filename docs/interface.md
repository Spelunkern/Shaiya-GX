# Interface status

The interface is being reconstructed incrementally from original EP5 artwork and recovered parameters. Phoenix is an auxiliary reference. Data and original artwork are not stored in this repository.

| Area | Current implementation | Remaining work |
|---|---|---|
| Account | Login, server list, faction, character selection/creation and account dialogs | Complete visual comparison and remaining variants |
| HUD | Player portrait, resources, mob level colours, elemental target icon, selection dots, buffs/debuffs with applicable skill levels | Remaining target states and secondary indicators |
| Action bars and skills | Three draggable/rotatable bars with five shared shortcut pages, drag and drop, original single-column skill book, confirmed reuse timers and native press feedback | Additional settings and complete state coverage |
| Inventory | Equipment, bags, drag, right-click use and equipment swaps, tooltips without zero sale prices and larger full-body preview | Remaining contexts and special dialogs |
| Weapon enchantments | Original Weapon.Seff enchantments plus four ITM-selected innate EFT glows, equipped and remote models, both dual bindings | Full visual comparison across animations; see [weapon enchantments](weapon-enchantments.md) |
| Cloaks | Original PC/CTL/3DC loading, equipment slot 7, all body families, native spring modes and animated attachment pins | Body/ground contact adapter and map wind; full physical parity is not yet claimed. Cape emblems (patterns and colours from the emblem NPC) are drawn over the cloth |
| Status | Allocation, EXP and current/max HP/MP/SP bars | Native graph and full attribute help |
| Chat | Phoenix geometry, external Tahoma with Unicode fallbacks, channel colours, persistent grey histories, single input row, six-channel picker and original SysMsg messages | Remaining special notices and complete state comparison |
| Effects | Independently draggable positive/negative strips; 32px icons, 6px gaps, native yellow duration bars and the original white debuff pulse | Full effect categories and all special states |
| Maps | Native World Map with region previews, ring buttons, expanded-map name, original clock frame with date/seconds and portrait resource text styling and native zone/hover labels, draggable minimap/full map, pan and zoom | Remaining map services and indicator variants |
| Options | Menu and six subpages; classic glow/brightness/detail/distance sliders, cloak/helmet visibility, audio, mouse, chat, editable hotkeys and monitor resolution selection | Other service controls |
| Party/raid | Member frames, invitations, leader and selection flows, member effect icons (`0x50abd0`, packets 0x0C04/0x0C05/0x0C0E/0x0C10), union (raid) portrait/slot menus, create/config window and event notices (`0x47ebe0`, `0x48e050`) | Complete badges and visual variants |
| Player context | Original classic TipBox frame, SysMsg rows, hover, character/target/party picking, whisper focus and supported party/trade/duel/guild actions | Inspect, follow, personal shops and remaining social windows/services |
| Quests | Original U button, journal artwork, state icons, seven 31px rows, three scrollbars, selection and confirmed abandonment; NPC dialogs/rewards/progress | Native tracker and remaining markers |
| Trade/duels | Wager/item/money interfaces and confirmation flows | Complete comparison across all states |
| Auction House | Realm-wide NPC market with four tabs, posting, bidding, buyout, claims and watched listings | Legacy visual and protocol details tracked in [Auction House research](auction-house-research.md) |
| Guild | Original G button, single roster sheet, native columns/rows, sorting, applicants, rank actions and notice editor; directory/creation/ranking flows; Buddy, Block and Search Party with native transport | Complete comparison of rare social states and remaining presentation |
| NPC services | Merchant, storage, blacksmith, gems/enchant and original portal window; portal approach, confirmation, same-map and cross-map travel | Remaining service types and special states |
| Mall | Catalog, correct Cash icon atlas, gifts and confirmations | Remaining pages/states and full comparison |
| Actions | Auto attack/pickup acquire nearby targets; selected-target attacks and spells turn immediately and approach before requesting | Complete navigation/collision and remaining actions |
| Combat numbers | Original count atlas with Phoenix damage, critical, miss and green healing billboards; server-confirmed results only | Healing from item pool snapshots lacks an explicit gain event |
| Spell/map effects | Native EFT/3DE particles, mirrored halos, caster/target area announcements and release-clock synchronization | Remaining native effect branches and full per-skill comparison |

## Recent recovered parameters

- Press **L** in game to toggle the local character's light. It brightens the character in fields and dungeons and projects a warm halo at their feet on field terrain. An explicit hotkey assignment to **L** takes precedence.
- Options construction: `0x4fbbb0`; subpage placement: `0x4fa800`. Main buttons are at `(64, 77 + 32*i)`, drawn at 128×32 with 128×22 hit regions. Original coordinates take precedence over Phoenix's compact spacing.
- Cash catalog rendering: `0x4d8b10` calls `0x4a3400` with type 200 and `Cash.Icon - 1`. This selects the type-100 atlas, or type-102 after index 1000; it is not a normal consumable icon.
- Duration bars: `0x4a2fa0`; a 32×5 black background below the icon contains a 30×3 yellow fill.
- Hovering your own buffs or debuffs shows the remaining whole seconds as
  `HH:MM:SS` beneath the description. A timer of at least a day uses
  `DD - HH:MM:SS`. EP4 PS0107 selects the timed tooltip at `0x4b8e02`
  and formats its footer at `0x49eb50`. With zero seconds or an untimed
  effect, the tooltip omits the timer. The server still controls icon removal.
- Status: EXP at `(73,132)`, 183×15; resources at `(73,265 + 20*i)`, 68×15. Values are centered in the original field insets in both layouts. Long values scale to fit their cells, including allocation values on the expanded sheet.
- Inventory preview: 136×217 opening at `(54,33)`, 34-degree FOV and height-based framing. Bind-pose arm span no longer shrinks the idle actor. This framing is calibrated against the reference composition; the complete native camera matrix has not been recovered.
- Inventory gold: native comma grouping (`0x4ce790`/`0x4cb420`) and the `0x4cb4a0` colour table. Boundaries are 1,000 (`80FFFF`), 10,000 (`0080FF`), 100,000 (`00FF40`), 1,000,000 (`FFFF00`), 10,000,000 (`FF8000`), 100,000,000 (`FF00FF`) and 1,000,000,000 (`B673FF`); smaller balances are white. GX lightens the top violet tier.
- Minimap: native frame, button angles −45/−27/40 degrees and a 150×40 information plate at `(9,157)`. Full-map close uses the shared 22px control. The compact view uses the original normalized zoom factor, 1 by default and bounded to 0.5–1.5. The default GX circle shows 136/512 of each calibrated map axis (EP5.4 initialization at `0x4c6b08` and UV scale at `0x4c7860`).
- Bottom bar: atlas slot 5 opens Quest (U); slot 6 opens Guild (G). Both reflect their window's open state. Mouse and keyboard share the same toggle path.
- Guild: `0x4f2960` selects `guild.tga`, not a cropped `guild_master.tga` directory. Column origins are 36/62/184/247/276, with 25/121/61/27/37px hit widths. `0x4ef9d0` draws rows from Y=112 in 23px steps; management controls are at Y=466. The notice uses `guild_message_input.tga` and its original 256×156 visible area.
- Quest journal: `0x4fedc0` and `0x4ff2c0`; seven rows at a 31px pitch, alarm icons `(26,63)`, names `(46,65)`, and original separators. Abandon is at `(402,453)`, drawn 128×32 with an 80×26 hit area. Native scrollbars cover the list `(227,52,13,237)`, objective `(227,320,13,130)` and description `(488,52,13,397)`.
- Portals: `0x494730`/`0x494490`; 512×366 artwork, default merchant anchor, three destination rows and original Move buttons at `(434,80 + 101*i)`. Welcome text is at `(26,92)` with a 193px width, ten visible lines and its own scrollbar. Portraits use the NPC definition's Model, matching the native portrait setter, instead of the previous fixed face 55.

Click an NPC to approach its displayed position and talk within two units. NPC conversations and service windows close beyond three units in 3D. Manual movement cancels the approach; moving NPCs also retain the server interaction-range check. Choose a populated destination, press Move and confirm. The client sends the NPC ID and destination index; GX Server validates availability, faction, distance and money, then supplies the balance and teleport position. Empty destinations remain inactive, including correct support for valid map-zero destinations with nonzero coordinates. Paid cross-map tests also verify scene replacement and persistence after reconnecting.

Press **H** to toggle white catalog IDs. Monster IDs appear on nameplates already shown by the current selection and name settings. Every visible NPC displays its type and ID, for example `6 1`. Item tooltips use the same two-number format, for example `100 120`. The toggle starts off each time the client launches.

Inventory (I), Status (T), Skills (K) and Actions (B) open independently through either hotkeys or the bottom bar. Opening one keeps the others visible. New frames use free horizontal space when possible; manually positioned frames keep their layout. Overlapping personal panels route clicks, title dragging and Escape in their drawing order: Status, Actions, Skills, Inventory. Escape closes one panel at a time.

Options remains nonmodal: opening it preserves keyboard movement and automatic travel. Pointer gestures belong to the window where they started. Logout confirmations remain modal and await the server response.

Click a binding in **Keys** or **Extra Hotkey**, press its new key and confirm with OK. Press Escape to cancel a pending assignment. **User Option → Invert Mouse** exchanges the two mouse buttons. Click walkable terrain or a model surface to move there while keeping the selected target. **F12** and **Print Screen** save a PNG in the `screenshot` directory next to the executable. The capture is written in the background, and the system chat shows the filename once it is saved.

Visual approval and complete comparison against the original remain ongoing. Opt-in rendering tests exercise real archives and preserve captures only in ignored local directories.

The player popup follows `0x47de10`/`0x47e1e0`: text starts at `(10,10)` with 15px rows. Its classic skin uses `TipBox.TGA`, 4px borders and the native `CD1D1F24` background. The `RightMenu_*` feature skin is absent from the local archive. The native minimum width is 90px of content plus borders; longer translated labels expand the width to fit the current font. Existing actions use original SysMsg labels, including party request/kick/leadership/leave (5002–5005), whisper (5007), battle (5008), transaction (5091), guild duel (5339) and raid leadership controls (6507–6509).

Right-click the selected character in the world, the target frame or a party/raid member to open their actions. Right-click the local portrait in a group to leave it. Whisper selects the recipient and focuses the existing chat input. The popup preserves held movement and owns its pointer gestures; changed rows cannot activate an action that replaced the one originally pressed. Group leadership, capacity, faction and guild eligibility filter supported requests; membership, trades and duels still await GX Server responses. Friends/ignore, inspect, follow and personal-shop services are not represented by invented actions.

## Cloaks and combat presentation

Cloaks use original archive files directly, without converted or recreated meshes. Equipment types 24/39 in slot 7 select a zero-based item Model: Model 0 is the first cloak. The server's chest ReqVg minus one selects the PC fit, with out-of-range fits falling back to zero. The server publishes this cosmetic fit because older Item.SData catalogs can contain a stale zero that hides the shoulder beneath equipped armor. Each PC record selects its own body, shoulder and racial CTL texture. The shoulder is an opaque, unlit skinned part rendered on both sides, matching the original pass. Its texture alpha is not a visibility mask; actor opacity controls the final surface alpha, preserving deliberate stealth. The cloth follows the record's bone pins. The same appearance mapping serves selection, inventory, the local character and other players.

The CPU solver reproduces native directed springs, masses, forces, Euler integration and both cloth modes. Synthetic one-step tests compare outputs with the original executable. Real-archive GL checks load all 3,072 body/fit/model combinations and simulate/render all 16 body families. The current runtime uses the native no-contact branch; body/ground contacts and map wind remain incomplete. Cape emblem decals are rendered on a 3x3 cloth patch (see [emblem research](emblem-research.md)).

Attacking or using a targeted skill turns toward the current target immediately and publishes the heading before the action packet. Targets outside weapon or skill range start an approach that updates as they move. A double click keeps the basic attack queued until it can be requested in range. Monster picking uses the original EP5 camera-facing rectangle, sized by Monster.SData Size and scaled MON height. Movement keys and clicks on walkable ground interrupt basic attacks and end the chase while keeping the target selected. Changing targets or issuing a new action also replaces the current request. This uses existing movement collision handling rather than a new pathfinding system. Results and resource changes remain server-owned.

Floating numbers use `count.tga` digit cells of 26×32, the Critical region at `(260,0,156,32)` and Miss at `(416,0,96,32)`. Phoenix parameters are a one-second lifetime, 0.3-unit rise per second, 0.2-unit digit width and 1.1875 height ratio. Critical digits are 1.5 times wider; healing is native RGB `00FF00` (digit renderer `004274ff`). Numbers detach from the target's displayed position, retain world scale under camera movement and expire from a queue capped at 128 entries. Attack, skill, drain and periodic result packets supply the displayed values; ordinary HP snapshots do not invent healing events. A lethal basic attack with two contact markers starts the monster’s death animation at the first impact and presents the second damage number at its own marker.

Drain results (0x0512, 0x0518, 0x060C) show the actual resources gained above the caster using the healing-number animation. HP uses native recovery green `#00FF00`, SP uses the original gauge yellow `#F9BD0F`, and MP uses the original gauge blue `#00AFE5`. The examined EP4/EP5 digit renderer draws all three recovery types green. GX uses the tones from `main_stats_bar` to distinguish SP and MP.

EFT `Unknown29` is the mirrored-texture flag. Billboards with this flag use four reflected quadrants, matching Phoenix, so light and spell halos meet inward. The baked UVs use clamp addressing to avoid opposite-edge sampling; ordinary billboards and authored 3DE meshes retain their addressing. Original map-one `lamp003` is covered by an opt-in GL check.

Caster-centred area skills send no selected victim and render their body from the server's area announcement, even when there are no recipients. Target-centred areas use their announced target. Secondary hits independently select their recipient effect through the native damage/debuff/heal branches; a library's `DA_area_` filename does not suppress a recipient effect. Player effect anchors use the existing foot-space transform; the native logical-origin subtraction of 0.9 must not be applied twice. Projectile launch height is consequently 1.6 above the feet, with user/mob arrival points 0.9 above their feet.

Instant skill release animations begin when the request is sent; matching confirmations preserve the running action instead of restarting it. Direct effects catch up to that release clock, and projectiles catch up both their native 12-key windup and elapsed flight. A late result can start directly at its elapsed impact. Melee cues retain frame overshoot instead of restarting their effect clock. Melee contact markers and authored EFT sequence delays remain in use. Timed casts start from server cast-start messages; damage and effects still require confirmed results.

A server cast-start supersedes any predicted instant release. If local SData declares no preparation duration, the character and hand effects prepare until the server supplies a result or cancellation. Preparation time is not included in the subsequent release/impact clock. This matters for Frozen Ice: the local archive declares an instant skill, while the test server announces a three-second cast. The integration test now renders throughout that wait, checks fresh release clocks and verifies changed framebuffer pixels with and without the area effect, both with and without recipients.

Confirmed skills also create the separate caster release sequence used by native `0x57bdc0`/`0x57e320`. `IN_*` sequence zero supplies the release body; sequence one remains the preparation on the hands. Physical skills select the activation library through `0x5768e0`, except the native ranged weapon families, which use the generic release. This fills the missing early visual phase without deleting the later offsets authored in the target library. Release effects capture their world transform, recover elapsed animation time and are not recreated by secondary area hits. Native attack-speed modulation and the remaining special effect branches still need full comparison.

Login VANI assets use the frame interval stored at header offset 48. The original login water and rocks specify 33 ms per pose. Animated UVs use repeat addressing, including the negative and greater-than-one coordinates in the basin water.

The WLD placement group selects how VANI textures use alpha. Group A writes depth and keeps fragments with alpha at least 80/255 (`Game.exe` 0x44867a and 0x448a72). Group B blends the texture's full alpha range back to front, after opaque geometry and characters (`Game.exe` 0x44a666–0x44a769). Depth testing stays enabled and depth writes are disabled during this pass. This preserves low-alpha fabric in the Light selection scene and the intended transparency of login water. Both groups share the same texture and animation data.

## Phoenix chat and native targets

The login logo stays visible during server selection and disappears with the login scenery.

Chat uses runtime `sysmsg-uni.txt` entries, including channel templates 353/354/355/370 and channel names 5086–5089/5377/5388. The formatter replaces native placeholders once; message bodies cannot introduce more placeholders. Normal messages use the native `name : message` format. Combat, experience, acquired gold/items, item use, learned skills, level changes, party invitations/membership, quest outcomes, trade and supported guild lifecycle notices use their original message IDs. Diagnostic commands retain explanatory fallback text where no native equivalent has been connected. Special service and third-party combat notices remain incomplete; ordinary speech bubbles follow confirmed chat echoes.

The chat retains the chosen flat Phoenix-style plates while using recovered native geometry. Histories are 350px wide with an eight-pixel gap and default heights of 112/144px. They now fit seven/nine 16px rows; the newest text uses the bottom complete font cell instead of leaving an empty row. A full cell, descenders and one-pixel shadow are clipped inside each flat frame. The background is native ARGB `9A1D1F24`; text uses the external primary font at the native English 15px cell height with Unicode/emoji fallbacks.

The authored entry row sits at `height - 76`: a 64×22 selector followed by a 292×22 field with its right-hand language icon removed by request. The combined visible width is 356px. Input text starts at X69 and the selector label at X7, both vertically centred in the 22px row (Y+3 with the 15px English chat font; a fixed Y+6 left the larger font touching the bottom border). A blinking vertical caret replaces the permanent underscore. Pointer bounds follow the corrected row.

**Use Mouse** controls whether clicking walkable ground starts movement. Mouse
camera controls work with either setting.

Both chat histories remain visible. Press Enter to open the chat entry row. Auto-run and click-to-move continue while typing, and the chat panel handles clicks on its own controls.

**Banned words Filter** in User Options masks matching text in received player chat and
speech bubbles. Edit `data/Filter.INI` next to `sysmsg-uni.txt` and restart the
client to load your rules. Add one `Word=` line per word or phrase under
`[Filter]`, such as `Word=example`. The file accepts UTF-16LE and UTF-8 and
starts with no rules, and the checkbox starts off. Matching is case-insensitive and treats each rule as literal text.

Golden frames, Alpha/filter/help panels and the separate native whisper-recipient row remain retired under the chosen flat presentation. Histories retain a shared 300-message buffer and independent scroll anchors. Settings preserve row sizes. Chat starts in Normal on each login, and channel changes apply to the current session. Whisper accepts the existing recipient syntax, including `!name message` and `/w name message`. Raid is outside the six-channel picker.

| Channel | Native RGB |
|---|---|
| Normal | `D5D4D3` |
| Party | `A8F3A0` |
| Whisper | `F3A0A0` |
| Guild | `EEA4ED` |
| Trade | `F3DEA0` |
| Area | `FF3000` |

System notices retain runtime wording and the native base palette. Combat amounts are highlighted in yellow; HP damage omits its HP suffix by explicit user request. Healing and MP/SP labels remain localized. Skill-use/area announcements, additional damage and user periodic effects now have their missing chat paths connected. Native mob-periodic and reflection callbacks do not emit chat and retain their floating-number presentation. The picker remains above movable action bars and owns its pointer gestures.

See the [EP5.4 chat audit](chat-parity.md) for recovered coordinates, callback/message coverage, intentional differences and remaining editing/filter/service gaps.

Mob names and target captions share `0x4cb250`'s signed monster-level minus player-level table. This intentionally retains the original high-level grey range. Element tiers repeat the four original target icons using `(element - 1) % 4`.

| Level difference | RGB |
|---|---|
| ≤ −6 | `FFFFFF` |
| −5…−4 | `80FFFF` |
| −3…−2 | `0000FF` |
| −1…+1 | `00FF00` |
| +2…+3 | `FFFF00` |
| +4…+5 | `FF8000` |
| +6…+7 | `FF0000` |
| +8…+9 | `FF00FF` |
| ≥ +10 | `808080` |

World names project a fixed offset from the actor origin. NPCs display one
yellow line at their authored `Npc.MON` height plus 0.3 (`0x4344cd`), with
the native half-height dead NPCs and one-fifth-height animals 541–543.
Monsters use their `Monster.MON` height times the model scale plus 0.2
(`0x431770`, `0x44c39b`). A selected monster has a name within 100 units and
two 16-pixel `mon_*.png` markers matching its element. Hovering a different
monster shows its unmarked name. Text starts at the projected Y coordinate;
the markers start three pixels above it with a one-pixel side gap.

 The English client's grave-accent key (`0x43cd51`, DirectInput `0x29`) cycles living monsters by increasing distance and wraps. Backspace (`0x43c7a5`, `0x4cb2d0`) restores the previous selection; text editing still receives repeated Backspace events. The native locale-specific Tab variant is not substituted for the existing walk/run binding.

## Local chat protection

The client checks incoming player chat before it reaches the history or speech bubbles. Links stay plain text. The client hides links to local files and private network destinations, app launch links, direct downloads of executables or scripts, and some lookalikes of common service domains. It also removes characters that can invisibly reorder chat, limits extreme glyph repetition, and hides credential and download requests that come with an unverified staff claim. Staff verification uses server-provided character roles and admin channels when available. Links are checked locally without visiting them.

Repeated messages from the same player are grouped for 15 seconds after the latest repeat. A `×N` counter updates the existing line without reopening hidden chat or showing another bubble. Public channels share a limit of six new messages per player every ten seconds. Incoming whispers have a separate limit of six. Party and guild each allow twelve. A local notice appears when a sender reaches a limit. Players can mute that sender via **Guild → Block**.

Profanity, system notices and in-game item links are outside the content checks. Item links remain available when an external URL on the same line is hidden.

## Item details, cooldowns and cursors

Item properties use original SysMsg labels, yellow titles/base bonuses, orange remake values and white prices. Recovery amounts on consumable families are no longer shown as permanent equipment bonuses. Classes, stack metadata and extra attribute requirements have been removed from the tooltip. Original inline description controls and confirmed gem sockets remain. Full modifier-key/rarity presentation still needs comparison.

K has nine rows in one column: icons at `(46,78+43*i)`, names at `(88,81+43*i)`, requirements at y=97+43*i, and 32px learn buttons at `(278,84+43*i)` with 22px hit regions. Original SysMsg labels and red unmet requirements replace the compact two-column labels. Learning, scrolling, drag/drop, tooltips and skill use share those rows.

GX Server already validates skill reuse by SkillID and consumables by ReqIg. The client now suppresses requests during confirmed reuse, restores skill delays from the login snapshot and shares consumable timers across inventory/shortcut instances. ReqIg groups 1/2/3/4/5/6/10/11 mean 15/20/25/30/60/120/600/2 seconds. Timers start on accepted results. Inventory, K, the action bars and Revolver show the eight-triangle sweep (`0x4a27d0`) in red with yellow remaining seconds. Potentials assigned to the action bars or Revolver display their reuse as soon as World announces the automatic trigger. They remain passive when assigned. The action bars show item stack quantities and skill levels appear in tooltips. Action icons use the original white `0x50FFFFFF` press overlay (`0x4e0160`), including held number/keypad keys. Consumable reconnect persistence follows the current GX Server session behavior; the protocol supplies no consumable timer snapshot.

Accepted healing potions followed by the authoritative resource snapshot emit green SysMsg 100 for each restored pool, using the actual clamped gain. Ordinary regeneration does not create potion notices. Buff badges display their transmitted level except absent/special level 100.

All seven `Cursor/01.png` through `07.png` files load from loose data into OS cursors. Lossless conversion preserves native dimensions and RGBA pixels, including the original AND-mask transparency. All supplied cursors use a `(0,0)` hotspot; the loader applies that directly. The CUR decoder remains an import utility. Context selection covers attack, loot, NPC conversation, bar interaction, dragging and skill aiming, with the default cursor elsewhere. Remaining specialized service/targeting states still need comparison. Cursor handles are released before the window system closes.

## Inventory, monster presentation and graphics controls

Press C to sit or stand. Holding C triggers a single request, and a new press
can change posture after the current transition finishes. Hold a movement key
to stand once the sit animation finishes. Equipment changes retain the seated
pose or the progress of the sit or stand animation.

Equipment sounds play when World confirms a change to an equipped slot.

Items are used with the **right** button (a deliberate departure from the original,
which used the left click, to match modern clients): right-clicking a bag item
requests its use, and an equippable item selects its equipment slot and requests
the native move/swap, including replacement. A left click only selects, and a drag
still moves items; a release outside its source does not consume it. The Z/X potion
bar follows the same rule: right click uses, left click does nothing. The action
bars keep the original left click (hotkeys are unchanged). Accessory pairs choose an empty slot first. Results remain
server-authoritative. Provisional inventory instructions/waiting messages are
removed, and items with no sale value omit that line entirely.

Monsters use Monster.SData normal/chase timing and attack animation fields.
Movement and confirmed attacks interrupt idle variations. If a hit draws aggro,
the chase interrupts the damage reaction. The run cycle continues across brief
gaps between chase steps, and walking ends on arrival. Selecting a monster
updates its health and speed without restarting its movement. Damage reactions
cannot interrupt attacks or locomotion, and attack facing follows the victim. Monster skill results use their own 0x060b/c/d layouts,
including the attack-slot byte, level-one definition lookup and sequence-two
native impact route. Wire level100 is a marker. Repeated recipients of one area
attack do not restart its animation/audio. Speed changes and target snapshots
retain the monster's current server modifiers. Full per-monster comparison is
still ongoing.

The original debuff overlay (`0x4bff80`/`0x4c02d0`) uses a one-second triangular white pulse
from zero to 127/255 alpha and back to zero. Icon hit regions and authoritative buff
lifetime remain intact. Login/server selection music is `Sound/Music/bgm_title.ogg`,
confirmed at `0x42caf0`; it replaces the previous `bgm_login001.wav` selection.

The four graphic thumbs use the original slider art at Y184/234/284/334, with
ranges glow0–2, brightness0–4, details0–2 and distance0–2. Glow uses native
neighbor/center weights (0/0,22/17,44/35). Brightness applies the windowed power
curve and +10 black offset to the completed game frame, including UI, without
changing desktop gamma. The original byte input divisor is256 (`0x541540`).
Details default to **2 (High)**, retaining the existing texture bias, subpixel
static-object LOD, vegetation and half-size water reflections. Medium/Low keep
texture biases 1/2 and additionally raise static-mesh screen-error limits to
1.25/3 pixels, trim distant decorative grass/trees and thin grass beyond 25
units to one-half/one-quarter density. Grass ranges are 120/60 units; ordinary
tree ranges are 300/160 units (large landmark trees remain). Buildings, terrain,
actors and collision data are retained. Water reflection dimensions become
one-quarter/one-eighth of the main viewport, reducing reflection pixels by
4x/16x relative to High. Preset changes clear LOD hysteresis so restoring High
restores its previous image. These optional budgets are GX adaptations, not a
claim of recovered native quality rules. Distance uses
half/three-quarter/full view and fog ranges; this scaling is an implementation
adaptation, not a claim that the entire native distance pipeline is recovered.
Checked Cape/Helmet boxes mean visible; the stored hide flags remain compatible
with existing settings. Changes preview live, Cancel restores them, Reset restores defaults, and OK
persists them. Opening Options remains nonmodal.

Guild's side navigation now reaches the original Buddy, Block and Search Party
frames. Buddy supports invitation/acceptance, presence, whisper, deletion and
notes; right click a row to edit its note in the name field and confirm with
Enter. Block persists independently and filters chat. The fourth native page is
party search; mail in the auxiliary AzerothCore UI is not an EP5 transport.
GX Server supplies existing 0x2201–0x220e packets and existing SQLite tables. Native
Buddy result codes are0 rejection/1 acceptance/2 different area (SysMsg399–401);
block uses402/403 and search registration404/405. Candidate filtering and rare
original service states still need full comparison. Presence refreshes every
three seconds; membership changes follow ACKs.


## Personal markets

Open **Sub Skill** with **B** and choose **Setup Market**. This action can also be assigned to a shortcut. Drag items from a bag into any of the 20 listing slots and enter a gold price per unit. Select an existing listing to change its price or remove it. Enter a market message and choose **Open Market**. The seller sits and the message appears above the character. **Stop selling** returns to setup with unsold offers available for editing. Standing, moving, dying, changing maps or logging out closes the market. The last sale closes an empty market.

To buy, right-click a nearby seller and choose **View markets**. Select a listing, choose **Buy**, and confirm the quantity and total. The server updates both inventories and balances in one SQLite transaction, then updates the visible stock. Available space, both gold balances, the original item's identity and the buyer's distance are checked on purchase. The closing notice also dismisses the purchase dialog if a seller stops while someone is browsing.

Markets can be opened in maps 35, 36 and 42. Buyers must be within ten world units of the seller. Map 42 allows buyers from either faction. A market has up to 16 active browsers.

The client can use the original `interface/p_market.png` and `interface/p_market_button.png`. They can be converted from their `.tga` entries in an owned client archive with the [asset tool](data-formats.md). The panel supports older loose data sets through a built-in layout for missing textures.

## Resource checks, self-targeting and loot

Skill requests check the known MP/SP budget before sending or starting predicted animation. Failures use SysMsg 214 with the original MP/SP labels. Queued approach skills recheck on arrival; the server remains authoritative. The original cost value 1 retains its consume-current-pool meaning.

F1–F7 select party members in roster order. F1–F6 select within your raid subgroup. Solo players can use F2 or the local portrait to select themselves. Press F with a nearby group member selected to copy their target. Click dropped items or use the pickup action to approach and collect them within one horizontal unit. The server confirms pickup and removal.

Friendly skill results are sent after the recipient applies the heal. Green numbers show effective HP gained rather than the requested amount; zero gain displays no number. Apply the updated GX Server build as well as the client to use the corrected healing announcements.

## Display selection

Graphics options list numeric resolutions only. The client always uses a
resizable window; there are no Window/Fullscreen entries or mode suffixes.
Common sizes, monitor-advertised sizes and the current custom size remain
available. Common presets fit the logical work area; a native monitor-size
choice preserves its requested client size and keeps the title bar reachable.
Old saved fullscreen resolutions migrate to ordinary window sizes. Resolution
changes apply on OK, Cancel discards them, and explicit `-w`/`-h` arguments retain
their startup override. The existing OpenGL context is preserved.

VSync retains the fixed startup behavior (enabled); there is no VSync setting.
Help remains visible but disabled in the options menu.
Validation covers window-only transitions, real GL resize/resource preservation
and graphics settings save/restart with real AA rendering.

The five Graphics checkboxes control shadows, water surface effects, cloak and
helmet visibility, and effects. Older data packs with the extended graphics frame are
compacted at load time. The client reads the existing PNG without modifying
the file or requiring the old artwork patch script.

## Antialiasing

The former `Color Bit` slot offers **Off** (default), FXAA, TAA and supported
MSAA levels. Existing settings remain Off; unsupported saved MSAA counts fall
back to the nearest lower supported count. Colour/depth format support is queried
separately. Selection previews immediately; OK saves and Cancel restores it.

MSAA resolves scene colour and float depth before glow/UI, including water
refraction captures. FXAA filters the scene in one pass. TAA uses subpixel Halton
sampling, camera/depth reprojection, depth rejection and neighbourhood/variance
clamping, with resets for scene/camera cuts, resize and mode changes. It restores
stable colour/depth before annotations; HUD/text are not filtered. Visibility
culling uses the jittered projection, while UI anchors and input rays stay stable.
FXAA, TAA and supported MSAA have been verified to reload after saving with OK
and activate their actual rendering resources on a fresh application instance.
TAA currently has no per-object/skinned motion-vector buffer, so fast animation
and translucent effects still need broader ghosting evaluation. At 1080p its
three colour/depth surfaces use about 47.5 MiB; Off allocates no AA surfaces and
switching Off releases them. Shadows and clipped reflections remain unjittered.

## Water and scene transparency

**Water Surface Effect** changes reflection/refraction work, not the visibility
of the surface. With the option disabled, the original animated WTR water stays
visible and the extra reflection scene and refraction capture are skipped.
Re-enabling the option restores the advanced path, including when a reflection
target already exists from an earlier frame.

Scene destination alpha stays opaque throughout drawing, including reflected
scenes, so Dispel's destination-alpha blend layer cannot darken the background
into black shapes. Individual textures retain source-alpha transparency. The
previous colour-write mask is restored afterward. Shadow silhouette targets
still write their own alpha coverage and restore the enclosing scene mask.
Regression checks exercise the real Dispel EFT, translucent layering, nested
shadow captures and the water option in a rendered map.


## Local presentation adjustments after gx-v0.1.4

The blue cast column grows from its bottom edge. Player guild captions are
centred 15px above character names. Player plates use the native standing or
lowered height, with adjustments for race, creature forms and mounts. The raw guild
name has no added brackets: members use `#E4E4E4`, and the guild master uses
`#E0AFFA` on the guild caption only. Character-name/GM/faction colours remain
independent. The confirmed directory master or own-guild role determines the
leader colour; remote shape packets supply guild text when the directory lacks
that guild. Nameplates use a subtle half-pixel stroke and black outline.
Inventory and action-slot item counts share white text and a black outline,
with counts remaining readable above cooldown shading. Skill levels keep
their existing colour. The inventory actor has frontal light and full ambient
fill; this affects only its preview, preserving the world and portrait lighting.

Player names with PvP kills show the original kill rank icon from `grade2.tga`
(31 ranks, from 1 to 1,000,000 kills; ranks 1-16 in the left column, 17-31 in
the right) 1px to the right of the name and 3px above it. The name stays centred;
the native temporary -6px offset is compensated before its final draw.
Own kills come from 0x020E/0x0105 and other players' from 0x0522; a character
without kills shows no icon.

In the expanded Status window, tap **+** to allocate one point or hold it to add ten points per second after a short delay. **Apply** sends the draft to World for confirmation.

Status (T) shows server-confirmed kills, deaths, duel victories and defeats.
Its Title field uses the same rank table as the world icon: ranks 1-15 combine
SysMsg 5450 with 5389-5403, ranks 16-30 use 5451 with those names, and rank 31
uses 5404 alone. Zero kills leaves Title blank. Names are read from the external
SysMsg table; they are not translated or hardcoded in the client.

The PvP Manager (NPC type 4) opens the original Warrior Reward panel, with its
portrait, coloured explanation, rank guide, scroll controls and two confirmed
claim actions. Claiming requires level 16. Native
0x0218/0x0219/0x021A carry next claim stages, awarded stat points and death-benefit
gold. Each action redeems one earned stage; reconnects retain claimed stages.
GX uses all 31 client-table ranks and reward stages, with kills/deaths capped
at 1,000,000 by explicit request. The unified stat points by rank and all
death-benefit amounts come from the verified EP5 client tables; no high-rank
placeholder values are used. A stat reset preserves earned counters and both claimed reward stages.
Allocated points return to the unspent pool.
The kill progress strip shows kills divided by the next claim threshold;
the death-benefit strip shows the next claim level divided by 31. Their
native widths are 153px and 200px, using the red and violet atlas strips.

Diving below the swimming surface starts a 30-second oxygen bar. It uses the
cast bar's shape and drains from full to empty. At the surface or on dry land
it refills in up to three seconds and hides when full. Diving again uses the air
currently available. At zero air, drowning deals 20% of maximum HP every two
seconds. World tracks oxygen from the character's position and applies normal
environmental death handling.

Swimming and floating use the original 007_swim and 006_swnormal clips, including
race/sex filename aliases. Entering swimming releases a previous locked pose;
water locomotion takes precedence over sitting in presentation. Surface contact
leaves the extracted Cwater.EFT rings in place, using Phoenix's contact spacing
and a ring on entering flotation. Deep divers, mounted/dead actors and dry ground
do not create new surface marks. Existing particles expire normally. Contact
scheduling is adapted from Phoenix, not a claim of exact native emission cadence.


Sound options include the original third Environment Sound slider at y=206,
with its own persisted Ambient value. Music, effects and ambience apply
independently to active playback; Cancel, Reset and older settings retain their
expected behaviour. Older settings default the newly added channel to full volume.


## Loading and character-screen options

The character selection Option button opens the same menu used in the world;
O and Escape also open it. Its graphics, sound, persistence and display controls
share the existing settings. Closing or interacting with options consumes the
input before character selection. Returning to Characters while already there
closes the menu; Exit uses confirmation without sending an in-world logout.

Character scenery, first map entry and map transfers show Phoenix-style loading
art before synchronous scene setup. The overlay remains during initial cell and
resource preparation, blocks gameplay input, and reports monotonic progress from
the actual initial residency plan. Subsequent background streaming does not
reopen it. Network processing continues and protocol framing/ACKs are preserved.
The loading BGM starts after the first overlay frame has been presented. Missing
optional loading artwork falls back to a plain progress bar. The local data uses
Phoenix's existing random_loading01/loadbar_bg/loadbar/loadbar_light PNG assets.


## Current shadow scope

Only the local character casts a dynamic shadow, onto terrain or dungeon floor
geometry. NPCs, monsters, remote players and mounts no longer schedule shadow
captures. Projecting character shadows onto SMOD/MAni/VANI scene objects has been
removed, including its triangle copies and receiver traversal. The existing
Shadow Effect setting controls the local character's remaining shadow.


## NPC, potion and map presentation

NPC service windows share the merchant's default screen anchor and retain
independent draggable positions with Remember Layout. Quest conversations now
participate in the same layout persistence and pointer ownership. Portraits use
`NpcFace/<Model>.png`, with the original face 55 as a missing-resource fallback.
The merchant's purchase/drag hint is removed; transaction errors remain visible.
Merchant sales that would exceed the 4,000,000,000 gold cap are refused with
SysMsg 467. Sale and drop quantity fields accept digits from the number row
and numeric keypad.
Player portrait clicks still select self, while a drag moves its frame. The whole
target frame and minimap surface/information plate can be dragged; minimap buttons
and wheel zoom retain their functions. The original clock plate is preserved unchanged; only its text displays
`dd/mm/yyyy hh:mm:ss`, fitted to its existing interior with the same white text and shadow as the portrait resource values.

The movable 94×54 potion bar uses original `QuickPotion.png`, 32px slots at
`(11 + 40*i,13)`, left Z and right X, outlined stack counts, shared confirmed
cooldowns and depleted-icon dimming. Drag a consumable from inventory to bind it,
drag outside to clear, or drag between the two slots to swap. Clicks never remove bindings. Bindings are local
per server/character and follow the item type when its bag position changes.
X now activates the potion slot. Native server item-use requests and responses
remain responsible for consuming items and starting reuse timers. The server removes action-bar shortcuts to a stack when its last unit is consumed.

NPC patrol smoothing uses `MoveDistance / MoveSpeed` (milliseconds per step)
rather than the player packet interpolation interval, and display Y follows terrain during an active patrol segment and for GM summons. Other stationary
NPCs retain the server-supplied position, including elevated/floating placements. Group/area spells, including Prayer, use the existing native
cast/release cues with a short per-caster deduplication window. The final social
buttons map to `emoti009` (Provoke) and `emoti010` (Insult); Defeat uses `emoti007`.

WLD mode-zero areas ray-test their authored boxes (Phoenix picking margin), using
the original WLD record index as the sparse ID in UTF-16 `world/<map>.txt`.
Bitmap and other records do not renumber those IDs. Mode-two regions show the
original `interface/worldname/*.png` artwork on entry, with Phoenix fade timing.
Missing text/art is omitted. The checked map-one assets contain 48 localized
hover areas and 13 bitmap regions. These are presentation features; no server
map, patrol or interaction rules are changed.


## Native chat and World Map revision

Chat palette bytes now follow `Game.exe` 0x41cfe0, rather than Phoenix colour
approximations: Normal `D5D4D3`, Party `A8F3A0`, Whisper `F3A0A0`, Guild `EEA4ED`,
Trade `F3DEA0`, Area `FF3000`. SysMsg colours retain their corresponding original
style rows. Raid is removed from this chat selector and no longer overrides Party
colour. Older saved channel selections are ignored at startup.
The histories use the native `9A1D1F24` background, eight-pixel separation and
`rows+2`/`rows+4` visual-line capacities on the 16px grid. Footer button labels and
custom hover boxes are removed.

Only confirmed ordinary-chat echoes create a speech bubble. New text replaces
that actor's bubble and resets its five-second lifetime. The native single-line
grey `80484848` rectangle has measured text width plus 20px and 30px height;
text stays white, with coloured emoji preserved. Projection follows the actor,
hides off camera/beyond 50 units and keeps the frame above the posed crown/name
in GX's rig coordinate system. No client-side chat echo or server rule is added.

Open the expanded minimap, then W for the native World Map. Its 19 regions come
from `WorldMap.INI`; hover draws authored `wm_e` overlays and `wm_f` title art
under the original `wm_letter` labels. The current map has its marker. Clicking a
friendly/neutral region previews its minimap image; clicking the current region
returns to the live local map. Opposing-faction previews show the native warning.
W returns from a preview to the overview; M returns from the overview to the local
map. Right click follows the same back navigation. This never teleports or sends
a map-change request. It shares the existing draggable full-map frame. Native
translucency/help controls are not implemented by this revision.


The expanded local map shows the character's current X/Z coordinates in its
bottom border. Hovering the map image shows the pointed X/Z beside the cursor,
using the same crop, zoom and pan as the displayed map. Coordinates are hidden
outside the image and on the World Map overview.

Targeted skills require a present, living combat target before a request or
predicted animation begins. Corpse skills use their authored corpse requirement;
self, party and caster-area skills retain their target rules. Queued approaches
revalidate the target when arriving. The server remains authoritative.

Monster skill identity, names, debuff icons and timers come from
`Monster/NpcSkill.SData`, with wire level 100 selecting that namespace. Its visual
selector is separate from its one-based skill ID. Model attack/death EFT and
MON bone attachments are rendered independently, with a shared pose clock and
bounded nearby effect preparation. The existing Skill Effects option controls
these effects too.


With no selected target or an enemy selected, friendly healing/buff skills use the caster
(including Aura of Peace). Directed damage/debuff skills select the nearest
known living enemy, using stable ID ordering for equal distances, and use the
existing approach-to-range flow. NPCs, dead targets and ordinary friendly
players are excluded. Explicit targets are preserved. Caster-area skills do
not select a victim. Requests and predicted animations still require a valid
resolved target.

Quest minimap marks share available/ready eligibility with NPC indicators;
unfinished kill objectives mark their known living target mobs. Markers of the
same kind are deduplicated within an eight-pixel cell. Nearby NPC indicators use
the four original Quest.EFT sequences; confirmed completion uses up.EFT sequence
one and the original quest jingle. Their preparation is bounded and respects
Skill Effects.

Generic NPC dialogues have native Okay/Quest/service actions. Quest details use
the original take/complete/cancel sheets, correct disabled/pressed states,
separate story/reward regions and native scrollbars. Cancel closes the
conversation; a service action returns to that NPC's service. List and detail
windows retain their respective authored heights.

Blacksmith operations have separate Link Lapis, Enhance and Recreate views with
original rapis, rapisian and Remake3 frames. Their equipment, sockets, hammer,
lapisia, material and rune positions follow those panels. Changing operation
clears stale selections without sending requests. Repair actions and durability
information are removed.


Item Mall reads native `Cash.SData`; the server uses SQL ProductList. Purchases,
gifts and claims use the original workflow without a catalog hash handshake.
See [catalog format](cash-catalog.md).

All three action bars can display any of the five shared shortcut pages. Use
the arrows at the numbered end of each bar to change its page. The +/− buttons
on the first and second bars show or hide the next bar. The third bar starts
hidden and opens on page three. The number row controls the primary bar, the
keypad controls the visible secondary bar, and **Ctrl + 1–0** controls the
visible third bar. The native icon overlay placement comes from EP5.4
`0x4df520`, `0x4e0160`, and `0x4a1810`.

The Revolver bar displays the previous, current and next actions in a sequence.
Press R to use the current action, Alt + R to reset the sequence or Alt + 1–4
to select a page. Open its editor to arrange learned skills and basic actions.
Each page has three to six cells. See the [Revolver bar reference](revolver.md).

Action-bar bindings also require an explicit drag outside the bar to remove.
Left clicks activate and right clicks preserve assignments. Inventory, bars and
quantity dialogs support 255-item stacks using server-defined stackability.
Item Mall point updates refresh the wallet interface without adding chat lines.

Equipment ignores legacy character/item mode fields and tooltips omit mode requirements. Refused moves retain both slots and send a private System message explaining level, race, class, slot or equipment-combination restrictions. GM authority does not bypass those requirements.

Lapis linking and extraction always preserve the equipment on failure. Lapis and linking hammer consumption still applies; gold follows the native ReqIg table. Lucky Charms are not consumed. Results use original SysMsgTxt 55/56 (linking) and 57/58 (extraction), with the equipment and lapis names. This safety rule does not change lapisia enhancement.


Blacksmith selectors and action buttons use original EP5 sprites at the existing
positions: `repair_button3`, `repair_button4`, `rapis_button2` and
`Btn_Remodeling`. There are no blank button faces with replacement labels.
Equipment, socketed/new lapis, hammers, lapisias, materials and recreation runes
share inventory tooltips. Only the required enchantment material uses a muted icon/tooltip hint; lapisias
are chosen by the player.

A read-only server quote provides the current success rate and gold cost.
Linking and extraction confirmations format original SysMsgTxt 10011/840;
confirmation text keeps cost, probability and the question visible inside the original modal art.
Enchantment quotes validate the selected material quantity without changing the rate. Recreation keeps
its selected equipment and shows its inventory tooltip beside the panel,
rebuilding the stats from confirmed inventory updates.

NPC service windows omit NPC names and generic greeting/context text. Portraits,
functional instructions, quest objectives/rewards and destination names remain.

The [feature wiki](wiki/README.md) documents current forge rules. Placed equipment
and surviving consumable stacks remain after success or failure, with visible
stack counts and click-to-remove references. Incompatible lapis drops are
rejected with native SysMsg 62/63/64. Percent/gold information appears only in
operation confirmations. Recreation runs immediately without confirmation.


## Continuous forge operations and inventory bag dragging

Dropping an inventory item on another bag tab moves it to the first empty slot
in that bag and keeps the current bag visible. A full bag leaves the item in place.
Enchant now consumes exactly one material and lapisia per click, without quantity
or confirmation dialogs; the server chance is displayed above its button.
Native success/failure sounds and a 500ms additive Lensflare flash accompany
confirmed forge outcomes. Recreatable equipment shows light-green Max OJ beneath
the description and a text-sized icon before its enhancement material name.
Targeted random/perfect recreation runes preserve all other stats and obey the
original attribute pool/count/max restrictions; see the local wiki source.


Elemental weapons/armor now carry a small original monster-element badge on
their item icons across inventory/equipment, both action-bar orientations,
forge, personal/guild storage, trade/duel offers, drags and merchandise/mall.
The badge shares the tooltip's confirmed intrinsic/lapis calculation; neutral
results have no badge; intrinsic or first elemental lapis sets the badge. See `docs/wiki/elements.md`.


Elemental socket priority is intrinsic equipment first, otherwise the first
valid elemental lapis among six sockets. Subsequent elemental lapis remain
linked but gray/inactive in tooltip and blacksmith socket rows. Gray is actual
shader desaturation and does not affect later unrelated sprites. Extracting the
active lapis promotes the next; an intrinsic element keeps every elemental
lapis inactive. The server uses the same selection for affinity and gem bonuses.

## Confirmed equipment switching and HUD geometry

Using a carried equipment item from any action bar submits the same native
inventory-to-equipment swap as inventory equip. The shortcut remains bound to
its carried bag/slot. The old equipped item returns to that slot and therefore
becomes the next item on the shortcut. A refused swap leaves both items and the
shortcut intact. Server class, level, compatibility and equipment restrictions
still apply. Repeated input waits for the authoritative inventory reply.

The player frame follows `Game.exe` `0x005097B0`: resource atlas rectangles start
at root `(70,40)`, `(70,56)`, `(70,72)`, each 128 by 16 pixels, clipped by resource
ratio with matching UVs. The atlas includes transparent leading padding; moving
it to the visible colour edge would shift and shrink the fill a second time.
Values are centred at root X 135, with native Y offsets 41, 58 and 73. Portraits
use the original 64 by 64 mask at root `(16,26)`, preserving the recovered family,
class and gender camera. The image is not rescaled into a substitute ellipse.

Quest markers use stable paint order and remain individually present. They no
longer compete for a changing 8-pixel clustering cell, which could make nearby
markers jump or alternate between frames.

Character, monster and NPC skin passes explicitly restore depth testing, depth
writes and the current normal/reversed-depth convention before drawing. A prior
UI, shadow or transparent emitter pass cannot make the next actor ignore terrain
or leave its first material without depth writes. The regression includes the
native Windtail Forest Wood Spirit model (ID 2025) and other Windtail creatures.

## Map-wide quest availability

Quest giver and ready turn-in markers cover the complete map independently of
NPC visibility range. Their eligibility is cached and refreshed on quest,
character, relevant inventory, catalog or minimum-time changes; moving the
camera or player does not rescan every quest. Distant NPCs remain metadata, not
3D entities. Party and nearby creature-objective positions stay live overlays.

Cooldowns use a red sweep and centered yellow seconds across item/skill icons,
without flashing when ready. Reuse timing remains server-confirmed.

Portrait HP/MP/SP values use the native 12-cell resource font, white text with
a one-pixel black shadow and the original five-pixel centering/row offsets.
Names retain their separate warm font. Values above maximum are clamped for display.

Floating combat numbers now use the native anchor policy from `0x450a10`:
players add 1.2 to their logical origin (2.1 above GX feet), while creatures use
the authored MON height plus native 0.2 headroom (`0x431770`) instead of a fixed
1.2 offset. Their sampled position
still detaches from the actor when the number is emitted.

Physical bows, crossbows and javelins now use the original arrow/thrown weapon
models for basic attacks and offensive physical skills. Recipient effects and
numbers wait for arrival; health still follows the server. See the
[ranged-projectile guide](ranged-projectiles.md).

See [skill targeting and effects](skill-targeting-and-effects.md) for friendly
self fallback, fast instant-skill chaining and native selected-target effect icons.


Guild-caption source audit (EP5.4 client SHA-256 `8e5ef14a…bb3263`):
`0x0044c0cf–0x0044c125` selects ARGB `FFE0AFFA` for the master flag and
`FFE4E4E4` otherwise, independently of the character-name draw. It subtracts
float `0x006bc894` (15.0) from the projected label Y and centres using half of
the guild string's own measured width. The directory comparison at
`0x0044bc9a–0x0044bd0e` identifies masters by character name. The guild text
updates at `0x005699e0`/`0x00569c95` copy the supplied name verbatim; no angle
brackets are inserted. These rules apply to both local and remote labels.


### Monster appearance and corpse fade

Fresh `0601` / GX `7F05` mob spawns (`New=true`) play the original
`Effect/start.EFT`, sequence zero, at the spawn packet's ground position.
Its world transform is captured once: it never follows later movement, an old
interpolated actor position, or the entity's removal.
Its fixed world axes are independent of the monster's facing and scale, as in
Phoenix's separate spawn anchor.
Ordinary visibility snapshots do not replay it. This is Phoenix's `Spawn`
recipe, with the original emitters and authored lifetime, not a custom effect.

Dead monsters keep their death animation. As explicitly requested, GX adopts
Phoenix's alternative to native sinking: hold for 3.5 seconds, then use a
1.5-second opaque screen-door fade. A 4x4 threshold in the existing skin shader
reduces pixel coverage while preserving material RGB/depth writes. There is no
additional pass, transparent sorting or generated texture. Repeated death
observations cannot extend the lifetime; respawn clears the visual clock.
At death, the last rendered location and orientation are frozen; pending movement,
height corrections and late attack/reaction/death cues cannot move or restart the
corpse. The death animation finishes and holds its last pose. A real respawn clears
the frozen interpolation state and resumes live presentation.
Server corpse removal, loot and respawn timers remain authoritative.

Native news uses the English locale's regular 15px Arial-compatible font
(`00544620`, actor font slot `+3A8`, drawn through `00541AA0`), shared with chat.
The 13px tooltip font is not used for news. Centering, outline, queue, colors,
background and lifetime follow the native news presentation.


News background `0050ff10` uses the original `tipbox.png` atlas: 4px border
pieces around the interior, native UV slices, and the ARGB CD1D1F24 solid center.
Its supplied width/height are interior dimensions, not the complete outer size.
Full-map character coordinates sit four pixels higher above the map surface.

## Status fields and potion quantities

Stat in the Status (T) window shows the equipped upper armor's effective intrinsic/active-lapis
element, using the original SysMsg names and tier, or SysMsg 5270 for neutral.
Long localized element names fit inside the existing field.

The Z/X potion bar uses the native item-count anchor `(iconX + 12, iconY + 16)`
from Game.exe 0x4a3400, called by 0x4ddc30. Shortcut captions are drawn before
item icons, matching the original layer order; quantities remain legible on
filled cells, including three-digit stacks.


## Personal warehouse

Merchants and bankers open the inventory flush to the left of the service window. The inventory returns to its saved position when the service closes.

A personal banker opens account storage. While Warehouse Call is active,
click its buff icon to reopen personal storage and the inventory. Drag the
icon to reposition the buff strip. The remote window has an empty portrait
slot. Both personal warehouse entry points use the same six storage pages.
Drop a stored item on a page tab to move it to that page’s first free slot. For an exact destination, select the item, switch tabs and click the slot. Guild storage opens through its own banker in the guild house.
