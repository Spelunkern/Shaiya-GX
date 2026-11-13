# Instructions for coding agents

## Repository and scope

This is Shaiya GX: client at the root, integrated **GX Server** in `server/`. One Git repository and root Go module; no nested server Git repository, submodule or `go.mod`. Rapture is the former server name. Do not recreate it as a separate project. Public docs are English. Local ignored Spanish continuity/research notes may carry newer user context; follow the current user's explicit instructions.

Preserve existing uncommitted work. Do not use subagents or delegate unless the user explicitly requests it. Do not commit, push, publish releases or delete repositories without user authorization. A migration request that explicitly authorizes publication/deletion applies to that migration, not unrelated future changes.

## Boundaries

Keep game rules and persistence authoritative on the server. Preserve protocol framing, opcodes and asset formats. Client owns presentation and movement prediction. Repository integration and rebranding do not authorize unrelated rule/schema changes.

The micro app is **GX Server**, command `server/cmd/gx_server`; headless commands are `gx_login`, `gx_world`, `gx_console`. Keep `RAPTURE_*` environment keys, `rapture.sqlite` and `Rapture_LoginLog` compatible unless explicitly migrating them with tests and documentation.

SQLite is embedded for development simplicity only. It will be replaced by another backend in the future. Keep storage behind `server/internal/db`; do not present SQLite as the final production design or invent a chosen replacement.

## Local safety and fixtures

Check running clients/services before modifying an installation or moving its mutable database. Do not interrupt active personal sessions. Use disposable DB copies, isolated ports and only processes started by your tests. An existing runtime must never be overwritten by packaging or startup. Windows process control must match executable paths, not stop every process with the same filename.

`server/data/` is explicitly distributed: its test DB intentionally includes the author's current accounts, characters and inventory. Refresh only when requested, using SQLite backup and manifest validation. Never snapshot a different personal DB automatically. Keep mutable runtime, `.local`, logs, packet dumps, private backups and credentials out of Git.

Client game archives/assets and original executables are not distributable source. Do not add them or downloaded assets to the repo. Retain source copyrights and licenses, including server/LICENSE. Other local projects are reference-only unless the user authorizes changes.

## Commands and checks

From the GX root:

```sh
go vet ./...
go test ./...
go test -tags integration ./server/... -count=1
python3 -m unittest discover -s server/tools -p 'test_*.py'
python3 tools/check_publication.py --history
```

Use Go 1.27.3. Build Linux with `tools/build.sh`, Windows with `tools/build.ps1`; graphics require CGO and platform prerequisites. N2 uses `tools/n2_smoke.py` with local assets and a disposable server DB. Server portables: `python3 server/tools/package.py --output out/portable`. They must include both GUI and console, validate the seed and pass native relocated startup. Release assets belong to GX releases, never source-history binary blobs.

Run checks appropriate to changed behavior and report actual evidence. Windows cross-compilation or Wine is not native Windows validation. A CI build without game assets cannot establish full client visual parity. Update user-facing docs and local continuity when behavior/workflow changes. Avoid unnecessary architecture layers and keep rendering optimizations measured and visually comparable.
