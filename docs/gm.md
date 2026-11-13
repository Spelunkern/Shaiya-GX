# GX GM role and commands

GM authority comes exclusively from `PS_UserData_Users_Master.Admin`: `0` is
normal and `1` is GM. It is loaded on world login; changing it requires a new
session. `AdminLevel` does not grant this role. The distributed seed enables GM
for its ten test accounts. Existing runtime databases are preserved; the narrow
SQL in `server/tools/enable_test_gms.sql` can update those named accounts in an
intentionally selected offline database.

GM characters display a light-blue `(GM)` prefix (`#66CCFF`) beside their white
world name, visible to themselves and other players. Stored character names are
unchanged. Turbo starts enabled for GM sessions and multiplies movement speed by
12, including backward movement, mounts and swimming; server immobilization and
death still prevent movement.

| Command | Result |
|---|---|
| `/iclear` | Permanently deletes all of the issuing character's inventory, including equipment, and dismounts. Other characters and account storage are untouched. |
| `/turbo` | Toggles the issuing GM's turbo off/on for this session. |
| `/getitem type id quantity` | Grants the complete quantity, respecting definition stack limits and free bag slots. `/getitem 25 1 200` grants 200 apples in twenty stacks of ten with the current definitions. |
| `/mmake mobid quantity` | Spawns 1–100 live mobs around the GM, with their normal stats, AI and drops, without automatic respawn. |
| `/nmake type id` | Spawns exactly one static NPC around the GM, retaining its definition and services. `/nmake 1 1` selects merchant 1; `/nmake 2 1` selects gatekeeper 1. Type and row ID are separate arguments. |
| `/cmove mapid` | Teleports to the nearest safe arrival relative to the current map position. Uses rebirth boxes, or authored portal/gatekeeper arrivals when no box is available. GM travel can inspect base instance maps without party/guild entry requirements. |

Item quantities are positive integers, up to 30,600; grants that do not fit are
rejected entirely. Inventory changes are transactional. Spawns use expanding
rings, reject blocked/occupied positions, and enter nearby clients' visibility
immediately. Spawned entities live in memory and disappear on server restart.
Maps without a known safe arrival are rejected. Commands are case-insensitive,
validated on the server, and never broadcast as ordinary chat when handled.
Finish an active exchange before issuing a GM command.

The client sends commands through the normal chat request independently of the
selected UI channel. GX server response `0x7F01` contains self character ID `u32`,
GM `u8`, turbo `u8`, UTF-8 message byte length `u16`, and message bytes, all numeric
fields little-endian. GM/turbo bytes must be 0 or 1; turbo requires GM. Ordinary
`0x0201` marker 2 supplies GM role metadata for remote nameplates. A forged client
response cannot grant server privileges. This extension requires the updated GX
client and world server; the original client does not implement turbo state.
