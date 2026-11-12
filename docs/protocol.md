# Protocol and server authority

Shaiya GX targets the EP5.4 client protocol implemented by Rapture. Packets are serialized field by field with explicit widths and ordering; Go struct layout is never used as a wire format.

`internal/proto` provides one reader/writer implementation per supported packet family, with defensive bounds checks and codec tests. `internal/netx` implements framing and cryptography; `internal/session` owns the login-to-world transition. Passwords are supplied through the graphical login or an environment variable for diagnostics, never stored in the repository.

The server owns authentication, character rules, combat, skills, inventory, equipment, purchases, quests, trade, guild state and persistence. Client interfaces wait for confirmation and handle timeout/rejection without inventing successful state changes.

Examples of implemented packet families include character entry/movement, target selection, normal and player attacks, skill activation/cooldown, resource updates, bags and equipment, pickup, chat, party/raid, NPC services, storage, quests, trade, duels, guild and Item Mall operations.

Automatic action approach is client movement prediction. It selects a known nearby target, follows its position and sends an attack or pickup request in reach. Item removal, damage and rewards still require server packets. Manual movement, cancellation or target disappearance cancels approach.

Native addresses and counterpart behavior are cited in source comments. Unknown fields remain explicitly unknown; protocol parity is not inferred from a visually working screen. Integration tests run against isolated Rapture instances with disposable accounts and a temporary database.
