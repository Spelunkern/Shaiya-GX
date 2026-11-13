# Contributing to Shaiya GX

GX includes both the client and **GX Server** (`server/`, formerly Rapture). Contribute to this repository; do not open changes against a standalone Rapture repository. Public documentation, commit messages and review discussions should be in English.

## Set up

1. Read [README](README.md), [build instructions](docs/building.md), [architecture](docs/architecture.md) and the component's status/parity notes.
2. Install Go 1.27.3, an x64 C compiler and the documented platform dependencies.
3. Build with `tools/build.sh` on Linux or `tools/build.ps1` on Windows.
4. Run `go vet ./...`, `go test ./...` and, for server changes, `go test -tags integration ./server/... -count=1`.
5. Use your own compatible client archives locally. Server maps and the requested test DB are included; client archives are not.

## Scope and design

Keep modules small, readable and focused. The server is authoritative for combat, inventory and persistence. Client prediction is presentation/movement only. Preserve packet bytes and original asset interpretation; document deliberate visual adaptations and known differences. Changes to gameplay rules need a stated purpose and tests/evidence, rather than being hidden in a refactor.

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
