# Architecture

Shaiya GX is a Go client for the EP5.4 wire protocol supported by Rapture. The implementation is independent; original assets are read from user-supplied archives at runtime.

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
| `internal/audio` | Local WAV decoding, playback and channel mixing |

One thread owns application state and OpenGL. Network goroutines deliver decoded packets to that thread. The server decides game rules, inventory changes, combat results and persistence. The client predicts movement and presents animations; it does not invent successful outcomes.

World rendering uses reverse depth. Character previews use conventional depth and restore the surrounding render state. UI capture, portrait composition, world effects and map overlays have separate rendering responsibilities.

Assets load from the configured local archives and are cached. Terrain and repeated objects are batched and culled. UI layout persists normalized window centers while preserving native artwork dimensions.

## Reference policy

Original EP5 behavior and recovered constants define appearance and protocol. Phoenix supplies useful interface and interaction references, while its unrelated server rules are not imported. Comments cite native addresses or reference modules where available. A working panel does not imply complete parity with every original state.
