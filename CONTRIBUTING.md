# Contributing to Shaiya GX

GX includes both the client and **GX Server** (`server/`, formerly Rapture). Contribute to this repository; do not open changes against a standalone Rapture repository. Public documentation, commit messages and review discussions should be in English.

## Project direction

The goal is a functional foundation that faithfully reproduces original Shaiya behavior. Contributions should advance that shared foundation. GX is not tailored to each person's private-server preferences, custom rules or personal feature requests. An original Shaiya feature is welcome; unrelated additions, unusual graphical overhauls and visual redesigns outside the original game's direction are out of scope.

The technical direction is fixed: **Go**, the current **OpenGL 4.5 graphics API**, and **Linux as the primary development platform**. Contributions must not replace the language or graphics API, or make Windows the primary development platform. Windows remains a supported build and runtime target; preserve its compatibility while developing primarily on Linux.

## Ways to contribute

Contributions do not have to be code or complete systems. We welcome:

- Bug reports and fixes, with reproduction steps when possible.
- Refactors that improve clarity or maintainability while preserving behavior.
- Complete implementations of missing original features.
- Corrections to existing features that behave differently from original Shaiya.
- Reverse engineering, documented findings, format/protocol analysis and reproducible evidence.
- Optimizations backed by measurements that preserve original behavior and appearance.
- Suggestions and discussions aligned with this project's direction.

For parity work, explain the original behavior, GX's current behavior and the evidence for the proposed correction. Incomplete findings can still be useful when their limits and open questions are clear. You can contribute through an issue or discussion without submitting a pull request.

## Set up

1. Read [README](README.md), [build instructions](docs/building.md), [architecture](docs/architecture.md) and the component's status/parity notes.
2. Install Go 1.27.3, an x64 C compiler and the documented platform dependencies.
3. Develop primarily on Linux and build with `tools/build.sh`. Use `tools/build.ps1` for the supported Windows target.
4. Run `go vet ./...`, `go test ./...` and, for server changes, `go test -tags integration ./server/... -count=1`.
5. Use compatible loose client `data/` locally; see [data formats](docs/data-formats.md) for importing archives. Server maps and the requested test DB are included; client game data is not.

## Scope and design

Keep modules small, readable and focused. The server is authoritative for combat, inventory and persistence. Client prediction is presentation/movement only. Preserve packet bytes and original asset interpretation; document deliberate visual adaptations and known differences. Gameplay corrections must be grounded in original behavior and supported by evidence/tests. Personal rule changes and unrelated visual redesigns do not belong in the shared foundation.

SQLite is a temporary embedded development backend for easy setup. A different database backend is planned. Keep storage inside `server/internal/db` and avoid spreading SQL/backend-specific assumptions through gameplay or UI code.

## Tests and evidence

Use unit tests for behavioral changes and isolated integration tests for wire/storage behavior. For UI/rendering changes provide comparable captures and describe the scene, resolution and data version. Use `tools/n2_smoke.py` for the Linux end-to-end disposable client/server probe. Never aim tests at a personal or production runtime. Native Windows CI and Linux CI must pass for supported targets; cross-build success alone is not runtime validation.

Report what was actually tested. Call out skipped tests, Wine versus native Windows, GPU/driver limitations and remaining parity gaps. CI lacks the proprietary client archives and cannot establish complete visual/gameplay parity.

## Data and publication policy

Allowed fixtures are `server/data/` (supplied server maps/configuration and the intentionally shared development DB/manifest) and the authored GX Server icon. The shared DB intentionally includes test accounts, characters and inventory; do not refresh it from a different runtime without explicit authorization. SQLite snapshot/manifest must be updated together via `server/tools/snapshot_sqlite.py`.

Never commit `data.sah`, `data.saf`, original executables, client textures/models/music, private fonts, personal runtime DBs, local overrides, credentials, packets, screenshots, research dumps, build outputs or private backups. Keep game-asset rights separate from the source license. Preserve [NOTICE](NOTICE) and [server/LICENSE](server/LICENSE).

Run `python3 tools/check_publication.py --history` after staging the intended changes. Publish portable server binaries as GX release assets; use `python3 server/tools/package.py --output out/portable` and attach its checksums. Do not commit generated ZIPs/EXEs into source history.

## Submit a change

Open an issue for substantial behavior/design changes, then submit a focused pull request. Explain the concrete before/after behavior, why it is needed, the affected client/server component and validation. Link an issue if applicable. Keep unrelated reformatting out of the diff. Document user-visible configuration or workflow changes in the guides. Include no account passwords or private screenshots in issue reports.
