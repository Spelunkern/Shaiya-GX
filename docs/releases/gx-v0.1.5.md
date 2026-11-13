# Shaiya GX gx-v0.1.5 — native interface, audio, dungeon performance and GM tools

## Downloads and updating

- Native Linux/Windows client and GX Server archives are published with `SHA256SUMS.txt`.
- Client packages now include the **external font pack** in `data/fonts/`: unmodified Wine Tahoma, 217 lazy Noto Unicode/colour-emoji fallbacks, coverage metadata and licenses. Extract the complete client archive, merging its fonts with your loose game data. Windows `Shaiya GX.exe` and Linux `Shaiya GX` can share the same client folder and data.
- Original game artwork, maps and audio remain separate from these GitHub archives. Use matching converted resources for the new loading/World Map artwork. The older Drive bundle has not been refreshed by this release.
- Update client and server together for the GX GM controls. Preserve an existing server's runtime database when upgrading; the included development seed has ten GM test accounts. Packaging never replaces a personal runtime or takes a fresh snapshot of it.
- Published Linux packages target Ubuntu 24.04/glibc 2.39+. Both clients require an OpenGL 4.5-capable GPU/driver. SQLite remains temporary embedded development storage.

## Gameplay, interface and maps

- Clicking a ground item approaches it and picks it up. Clicking an NPC approaches and interacts on arrival; server distance checks remain authoritative.
- NPC windows are movable and start at the merchant anchor. Supported NPC face portraits appear in their original frame; the added merchant instruction text is removed.
- Static NPCs retain server coordinates, including height. Active patrol movement follows terrain and uses walking animation and definition timing.
- Add the movable native Z/X potion bar, inventory drag/drop bindings, counts, reuse feedback and persistence. Correct the subaction animation/button mapping.
- Restore missing Prayer/group/area skill audio paths and deduplicate repeated skill announcements.
- Casting fills the vertical bar from bottom to top. NPC type captions sit closely above names; character/NPC names have a subtle stronger stroke. Inventory/action-bar counts share their colour and black outline. The inventory character preview uses brighter lighting.
- Own portrait, target frames and minimap are movable. The minimap clock keeps its original frame and fits day/month/year plus hours/minutes/seconds inside it.
- Add WLD hover-name text lookup and original worldname PNG zone banners, native World Map region previews/faction handling and M/W navigation. Auxiliary World Map help/transparency controls remain incomplete.
- Add original loading artwork and expose the shared game options from character selection.
- Restore native chat RGB values and six-channel selection, remove the added Raid selector/footer hints, and add confirmed ordinary-chat speech bubbles with original timing/geometry. External Unicode and colour-emoji fonts load only as needed.

## Audio and water

- Add map music and environmental audio playback, including Ogg resource aliases and an environmental-volume slider. Audio settings migrate while retaining existing preferences.
- Add terrain footsteps, swimming/floating transitions and water-entry effects. Existing water/reflection settings remain available.

## Rendering and measured performance

- Keep only the local character's ground/dungeon shadow. Remove the costly object shadow-receiver system and other actor/mount shadows.
- Use compact 16-bit static mesh indices where their range permits it, retaining exact topology and 32-bit indices where required.
- Dungeons no longer inherit white field fog. Their previous projection distance remains available after fog is disabled.
- Cull off-camera DG geometry and MAni/SMOD/VAni placements. A conservative current-camera CPU mask uses actual opaque static surfaces, preserving doors/transparency and excluding moving occluders. There are no blocking GPU occlusion queries or previous-frame visibility assumptions.
- Instance consecutive repeated single-material objects while preserving native material order. Release dungeon meshes when their world closes.
- Seven real dungeons across 176 views, aspect ratios and animation samples produced byte-identical reference/optimized images. Mean world-geometry draw-call reductions ranged from **84.3% to 89.7%**; map 3 measured **2,332 to 309 mean calls**. These scene-specific world-pass measurements are not whole-frame FPS guarantees.

## GM account role and commands

`PS_UserData_Users_Master.Admin` is the authoritative flag: `0` normal, `1` GM. GM characters show a light-blue `(GM)` prefix beside their name. GM turbo starts enabled for the session. All command arguments and permissions are checked on GX Server.

| Command | Behaviour |
|---|---|
| `/iclear` | Permanently clears only the issuing character's inventory, including equipped items, and dismounts; account storage and other characters remain untouched. |
| `/turbo` | Toggles the GM's ×12 movement multiplier. |
| `/getitem type id quantity` | Grants the full quantity transactionally using definition stack limits; `/getitem 25 1 200` grants 200 apples. |
| `/mmake mobid quantity` | Spawns 1–100 normal live mobs around the GM without stacking or automatic respawn. |
| `/nmake type id` | Spawns exactly one NPC; `/nmake 1 1` is merchant 1, `/nmake 2 1` gatekeeper 1. |
| `/cmove mapid` | Teleports to a safe authored arrival relative to the current position, including GM inspection of base instance maps. |

See the complete [GM documentation](https://github.com/Spelunkern/Shaiya-GX/blob/gx-v0.1.5/docs/gm.md), also shipped as `docs/gm.md` in the server packages. Existing databases keep their current role flags unless explicitly updated; the narrow offline SQL for named test accounts is included.

## Validation

Native Linux/Windows CI checks ordinary tests, external font coverage, server integration, seed integrity and relocated portable-server/controller startup. Local licensed-data GL/audio tests cover the changed interface, water, dungeon/field visibility, own shadow and ambient playback; an isolated live GX session verifies GM role/turbo and both two-argument NPC examples. Native Windows compilation/startup checks do not establish complete gameplay parity on physical Windows graphics hardware.
