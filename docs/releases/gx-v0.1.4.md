# Shaiya GX gx-v0.1.4 — accumulated gameplay and rendering update

## Downloads and updating

- Client: `gx-client-linux-amd64.tar.gz` or `gx-client-windows-amd64.zip`.
- GX Server: matching Linux/Windows portable archives, including the development seed.
- Verify downloads with `SHA256SUMS.txt`. Extract the complete archive and retain notices/licenses.
- Client executables are `Shaiya GX` and `Shaiya GX.exe`. Both can share one adjacent `data/` folder. Client packages contain no game data; the [Drive bundle](https://drive.google.com/file/d/15BFHAGpLi_JJZbZ4fi8jpDsFQLdHmOyq/view?usp=sharing) supplies data separately and has not been reuploaded for this release.
- Update **both client and GX Server** for UTF-8 text and effective-healing/duel updates. Stop your server before upgrading, back up its personal runtime database, and preserve that runtime rather than replacing it with the supplied test seed.
- Windows amd64 and Linux amd64 require OpenGL 4.5. Published Linux clients target Ubuntu 24.04/glibc 2.39+; see the portable README for graphics/audio runtime libraries.

## Gameplay and interface

- Skills with insufficient known MP/SP fail locally with the appropriate SysMsg before predicted animation. Queued skills recheck resources upon reaching their target; GX Server remains authoritative.
- Dropped items no longer have floating labels. Model clicks and pickup approach use a responsive range inside the server's ten-unit limit.
- Select yourself through your portrait or F2, with self HP/name in the target frame and compatible previous-target selection.
- Healing numbers reflect HP actually restored, including friendly healing and mob drain. Zero healing shows no number.
- Own portrait/resources and target frames can be dragged independently by their name headers, with layout persistence.
- The compact minimap plate displays only the clock. The expanded map displays its localized name along the upper border.
- Replace the provisional casting panel with the original simple vertical blue bar. It fills as preparation progresses, tracks the character/camera, and disappears on completion or cancellation.
- Graphics options now list the monitor's advertised fullscreen resolutions and Window. Mouse, wheel, scrollbar and keyboard navigation select a pending mode; OK applies/saves and Cancel discards. Returning to Window restores its geometry/maximized state. Explicit `-w`/`-h` override a saved fullscreen preference for that launch.
- Resolution-list keyboard input does not open chat or interrupt movement already held.
- Duel victories and defeats refresh immediately through the live statistics packet, without reconnecting. Ordinary PvP kill/death counts are unchanged.

## Text and embedded font

- Client and GX Server use UTF-8, retaining Windows-1252 decoding for legacy resource text. Editing, wrapping and byte truncation preserve whole grapheme clusters.
- Wire sizes stay fixed: character names 18 bytes, UI account names 18 bytes, passwords 16 bytes and chat payloads up to 127 bytes. These are byte budgets, not character counts.
- Embed only **Liberation Sans Regular 2.1.5**, an unmodified SIL OFL font with Arial-compatible metrics. The font and license are included in source and release notices; no private font or runtime font installation is required.
- UTF-8 transport does not imply every visible glyph. Latin, Greek, Cyrillic and Hebrew are covered; Arabic, CJK and emoji have incomplete or absent glyph coverage. The large fallback/color-emoji pack is not included.

## Rendering corrections and measured optimization

- Dispel no longer produces black background shapes from its destination-alpha layer. Scene alpha remains opaque while texture alpha still controls transparency; the previous write mask is restored. Shadow captures retain their own alpha coverage.
- Water Surface Effect now controls reflection/refraction, not water visibility. Unchecked leaves animated water visible and skips the extra scene/capture work; checked restores advanced effects.
- Request the dedicated GPU before context creation where recognized. Linux respects explicit PRIME/vendor/software overrides; Windows exports NVIDIA Optimus/AMD PowerXpress performance requests. This is a driver preference, not a benchmark guaranteeing the fastest GPU in every machine.
- Conservative current-camera terrain occlusion rejects fully hidden geometry, actors, loot, effects and labels. Incomplete residency and uncertain visibility remain visible; shadows/reflections retain their own passes.
- Sector bounds reject groups before individual object visibility tests. Resource handles, prepared draw lists and matrix/uniform reuse reduce repeated allocation, path hashing and state work.
- Two CPU workers read/decode/parse streamed assets with bounded outstanding work. OpenGL uploads remain on the render thread, with a soft 2 ms asset budget, generation cancellation and pinned dependencies. Individual uploads can exceed the budget; initial lightmap mip generation remains a known stall source.
- Conservative SMOD LOD uses measured projected error and hysteresis. Near geometry, collision, characters, terrain, VANI and special passes retain full detail. Benefits are modest for the original low-poly assets.
- Local route measurement reduced mean streaming update cost from 2.918 ms to 0.792 ms (maximum 12.532 ms to 3.226 ms). A fixed map-1 view measured 0.718 ms to 0.495 ms with the same 159 draws. These scene-specific development measurements are not universal FPS claims; see [performance](https://github.com/Spelunkern/Shaiya-GX/blob/gx-v0.1.4/docs/performance.md) and [streaming/LOD](https://github.com/Spelunkern/Shaiya-GX/blob/gx-v0.1.4/docs/streaming-lod.md).

## Verification and limits

Native Linux/Windows CI builds and ordinary tests, server integration and relocated portable-server startup are required for this release. Local licensed-data checks cover two-client gameplay/Unicode, real duel counters, graphics options, cast pixels, occlusion/LOD comparisons, streaming lifetimes, water toggles and the real Dispel EFT. Native Windows compilation is distinct from full gameplay validation on a physical Windows GPU. Physical hybrid-GPU selection and real-monitor fullscreen transitions still need broader hardware testing.

SQLite remains embedded development storage and will be replaced in the future. No replacement backend is selected. The packaged seed intentionally contains the author's shared test fixtures; packaging does not refresh it from personal runtime state.
