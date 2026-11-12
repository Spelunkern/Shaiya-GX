# Interface status

The interface is being reconstructed incrementally from original EP5 artwork and recovered parameters. Phoenix is an auxiliary reference. Data and original artwork are not stored in this repository.

| Area | Current implementation | Remaining work |
|---|---|---|
| Account | Login, server list, faction, character selection/creation and account dialogs | Complete visual comparison and remaining variants |
| HUD | Player portrait, resources, target, buffs and debuffs | Complete target states and secondary indicators |
| Action bars and skills | Two draggable/rotatable bars, shortcuts, drag and drop, two-column skill book | Additional settings and complete state coverage |
| Inventory | Equipment, bags, item drag, tooltips and larger full-body preview | Remaining contexts and special dialogs |
| Status | Allocation, EXP and current/max HP/MP/SP bars | Native graph and full attribute help |
| Chat | Phoenix geometry/font/colours, two resizable histories, channel selection and input history | Filters, visual whisper recipient selection and speech bubbles |
| Effects | Independently draggable positive/negative strips; 32px icons, 6px gaps and native yellow duration bars | Full effect categories and all special states |
| Maps | Native map art, ring buttons, name/clock plate, draggable minimap/full map, pan and zoom | Remaining map services and indicator variants |
| Options | Menu and six subpages; supported audio, mouse, visibility and chat settings | Unconnected original graphics/service controls and shortcut editing |
| Party/raid | Member frames, invitations, leader and selection flows | Complete badges, buffs and visual variants |
| Quests | Journal, NPC dialogs, rewards and progress | Native tracker and remaining markers |
| Trade/duels | Wager/item/money interfaces and confirmation flows | Complete comparison across all states |
| Guild | Members, applicants, directory, creation/notice and ranking-related flows | Remaining presentation and service routes |
| NPC services | Merchant, storage, blacksmith, gems/enchant and portals | Remaining service types and special states |
| Mall | Catalog, correct Cash icon atlas, gifts and confirmations | Remaining pages/states and full comparison |
| Actions | Auto attack/pickup acquire nearby targets and approach before requesting | Complete navigation/collision and remaining actions |

## Recent recovered parameters

- Options construction: `0x4fbbb0`; subpage placement: `0x4fa800`. Main buttons are at `(64, 77 + 32*i)`, drawn at 128×32 with 128×22 hit regions. Original coordinates take precedence over Phoenix's compact spacing.
- Cash catalog rendering: `0x4d8b10` calls `0x4a3400` with type 200 and `Cash.Icon - 1`. This selects the type-100 atlas, or type-102 after index 1000; it is not a normal consumable icon.
- Duration bars: `0x4a2fa0`; a 32×5 black background below the icon contains a 30×3 yellow fill.
- Status: EXP at `(73,132)`, 183×15; resources at `(73,265 + 20*i)`, 68×15, following Phoenix's restored layout.
- Inventory preview: 136×207 viewport, 34-degree FOV, bounds-based framing without the former forced 3.6-unit camera distance.
- Minimap: native frame, button angles −45/−27/40 degrees and a 150×40 information plate at `(9,157)`. Full-map close uses the shared 22px control.

Options remains nonmodal: opening it preserves keyboard movement and automatic travel. Pointer gestures belong to the window where they started. Logout confirmations remain modal and await the server response.

Visual approval and complete comparison against the original remain ongoing. Opt-in rendering tests exercise real archives and preserve captures only in ignored local directories.
