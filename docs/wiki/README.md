# Shaiya GX Wiki

The reference for current GX gameplay features, item data, administration and
operation. GX is a Go client with an integrated authoritative world/login
server, using original EP5 interface resources and protocol foundations.

**Version scope:** updated for [gx-v0.1.6](https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-v0.1.6). These pages describe current development behavior. A released
binary may predate a documented change. Use matching client/server builds and
consult release notes when reproducing behavior. Publishing documentation does
not update executable files, client assets or a running database.

## Gameplay and items

- [Quests and map markers](quests-and-map.md): map-wide available missions,
  lightweight NPC metadata and event-driven eligibility caching.

- [Stealth and GM protection](stealth-and-gm-protection.md): authoritative
  concealment, monster disengagement, original ghost rendering and immunity.

- [Elements](elements.md): complete EP5 matrix, item/lapis affinity, PvE/PvP,
  monster and spell rules, buff lifecycle, detection and native evidence.

- [Blacksmith](blacksmith.md): native buttons, continuous attempts, exact gold
  table, linking formulas, hammer values, compiled enhancement tables and cap,
  selective extraction, failure protection and original system messages.
- [Item effects](item-effects.md): passive stats, triggered gem skills, recovery,
  healing, cooldown groups, utility items, elements, native field overloading and
  compatible-target tooltips.
- [Recreation and world loot](recreation-and-loot.md): rune requirements, valid
  random rolls, attribute pools, Craftname encoding, creation versus pickup and
  persistence without free rerolls.
- [Item Mall](item-mall.md): native Cash.SData, SQL ProductList, product bundles,
  255 stacks, catalog synchronization and migration tooling.
- [Weapon visual effects](weapon-effects.md): enchant glow tiers, native weapon
  families, particles and server-confirmed local/remote equipment updates.
- [Interface reference](interface.md): implemented interaction, inventory,
  controls and recovered original presentation details.
- [Examples and troubleshooting](troubleshooting.md): worked forge examples,
  rejected items, stale data, retired INI configuration and version differences.

## Administration and content

- [GM commands](gm-commands.md): role source, complete command syntax, examples,
  limits, transactional inventory changes, temporary spawns and protocol details.
- [Item data and compiled rules](item-data.md): where each setting lives, what
  requires rebuilding, native target flags, safe content migrations and tests.
- [Server operations](server-operations.md): GUI/console, LAN setup, seed versus
  mutable database, persistence, updates, backup and recovery.
- [Data formats](data-formats.md): loose resources, native tables, PNG/Ogg,
  external fonts and importing legally obtained original assets.

## Development and distribution

- [Architecture](architecture.md): ownership boundaries, rendering, network
  state, root module and integrated server storage abstraction.
- [Protocol](protocol.md): explicit packet encoding, authoritative decisions,
  inventory acknowledgment and disposable integration testing.
- [Building and testing](building-and-testing.md): Linux/Windows prerequisites,
  native builds, test commands and portable server packaging.
- [Client portable distribution](client-portable.md): package structure,
  external assets and fonts, launch behavior and validation.

The client requests actions and presents confirmed state. The world validates
live inventory, computes chance and cost, consumes resources and persists the
result. A tooltip, quoted probability or gray material preview cannot grant an
item or authorize a mutation.

## Documentation maintenance

Editable Markdown sources live under `docs/wiki/` in the development checkout.
The GitHub Wiki is a separate Git repository, `Shaiya-GX.wiki.git`; changing a
local Markdown file alone does not publish it. `tools/publish_wiki.py` validates
page links and exports native Wiki page filenames before an explicit push.
Keep rules and examples aligned with executable handlers, not only legacy item
descriptions. Do not publish private runtime files, credentials or original
proprietary assets as documentation attachments.
