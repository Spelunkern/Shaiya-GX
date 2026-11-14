# Architecture

Shaiya GX contains the Go client and its integrated GX Server (formerly Rapture), speaking the EP5.4 wire protocol. The implementation is independent; assets are read from the adjacent loose data directory at runtime.

## Modules

| Directory | Responsibility |
|---|---|
| `cmd/shaiya` | Graphical client and offscreen capture entry point |
| `cmd/shaiyacli` | Headless diagnostic client |
| `cmd/assets` | Read-only archive inspection |
| `internal/netx` | Transport, framing and cryptography |
| `internal/proto` | Explicit packet readers and writers |
| `internal/session` | Login, world connection and session transitions |
| `internal/game` | Authoritative state received from the server; no GL dependency |
| `internal/data` | Archive, image, model, animation, world and SData readers |
| `internal/scene` | Asset caches, actors, map streaming and scene presentation |
| `internal/render` | OpenGL 4.5 rendering passes and GPU resources |
| `internal/ui` | Original interface geometry, widgets and interaction |
| `internal/app` | Input routing and integration between state and presentation |
| `internal/audio` | Ogg Vorbis decoding, playback and channel mixing |

One thread owns application state and OpenGL. Network goroutines deliver decoded packets to that thread. The server decides game rules, inventory changes, combat results and persistence. The client predicts movement and presents animations; it does not invent successful outcomes.

World rendering uses reverse depth. Character previews use conventional depth and restore the surrounding render state. UI capture, portrait composition, world effects and map overlays have separate rendering responsibilities.

Assets load from the configured local data directory and are cached. `internal/data/archive` retains its legacy name and SAH/SAF reader for import tooling, while game startup explicitly uses its loose-directory reader. Lookup remains case-insensitive on both Linux and Windows. Interface resources are lossless PNG; Vorbis is decoded in Go to the audio mixer format. Terrain and repeated objects are batched and culled. UI layout persists normalized window centers while preserving native artwork dimensions.

## Reference policy

Original EP5 behavior and recovered constants define appearance and protocol. Phoenix supplies useful interface and interaction references, while its unrelated server rules are not imported. Comments cite native addresses or reference modules where available. A working panel does not imply complete parity with every original state.

## Integrated server

The root Go module owns `server/cmd`, `server/internal`, `server/data` and server tools. This is a direct source integration, not a submodule, external repository or remote module dependency. Server `internal` boundaries remain private to the server subtree.

`gx_login` handles authentication/session handoff; `gx_world` owns maps, entities, combat and persistence. `gx_console` and the Fyne `gx_server` micro app share process control. Build outputs are under `server/build`; runtime, data and logs stay relative to that server installation. Compatibility environment names and database filenames are preserved. The advertised name and micro app title are GX Server.

SQLite is embedded for development ease and will be replaced by another backend in the future. Storage remains behind `server/internal/db`. The included seed deliberately contains the author's shared test accounts and gameplay state; a mutable copy is created only when absent. Portables attach to GX releases. See [server operations](server.md) and [storage](../server/docs/sqlite.md).
