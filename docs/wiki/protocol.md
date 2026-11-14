# Protocol and server authority

Shaiya GX targets the EP5.4 client protocol implemented by GX Server. Packets are serialized field by field with explicit widths and ordering; Go struct layout is never used as a wire format.

`internal/proto` provides one reader/writer implementation per supported packet family, with defensive bounds checks and codec tests. `internal/netx` implements framing and cryptography; `internal/session` owns the login-to-world transition. Passwords are supplied through the graphical login or an environment variable for diagnostics, never stored in the repository.

The server owns authentication, character rules, combat, skills, inventory, equipment, purchases, quests, trade, guild state and persistence. Client interfaces wait for confirmation and handle timeout/rejection without inventing successful state changes.

Examples of implemented packet families include character entry/movement, target selection, normal and player attacks, skill activation/cooldown, resource updates, bags and equipment, pickup, chat, party/raid, NPC services, storage, quests, trade, duels, guild and Item Mall operations.

Automatic action approach is client movement prediction. It selects a known nearby target, follows its position and sends an attack or pickup request in reach. Item removal, damage and rewards still require server packets. Manual movement, cancellation or target disappearance cancels approach.

Native addresses and counterpart behavior are cited in source comments. Unknown fields remain explicitly unknown; protocol parity is not inferred from a visually working screen. Integration tests run against isolated GX Server instances with disposable accounts and a temporary database.

GX extension `0x7F02` communicates authorized character concealment (`u32` ID,
`u8` boolean). Unauthorized viewers receive native `0x0202` leave and no entity
updates. See [Stealth and GM protection](stealth-and-gm-protection.md).

GX `0x7F04` sends an atomically assembled, chunked map quest-NPC catalog. It
contains definition keys and X/Z anchors without interactive entity IDs. See
[Quests and map markers](quests-and-map.md) for layout and cache invalidation.
