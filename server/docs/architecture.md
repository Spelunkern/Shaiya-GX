# Architecture

GX Server is integrated into the parent GX repository and root Go module.
Headless cores share control code with a terminal console and the GX Server
one-button micro app (`gx_server`). No separate server repository is needed.

## Processes

- `gx_console`: terminal operator console ("GX Server Console") --
  `start server` / `shutdown server`.
- `gx_login`: login/authentication TCP server.
- `gx_world`: game/world TCP server.

(`.exe` on Windows, no extension on Linux -- otherwise identical binaries,
built from `cmd/` with Go into `build/`. Storage is embedded SQLite during development, with the original EP4.5
tables/columns retained and procedures translated into parameterized SQL.)

The console owns process lifecycle. The server cores own protocol, state,
database access, and gameplay.

## First Milestone

1. Accept a TCP connection from `Game.exe` on `127.0.0.1:30800`.
2. Send/validate the login handshake.
3. Decode `0xA102` login request using Aries layouts.
4. Return login success and server list.
5. Observe the client connecting to world on `127.0.0.1:30810`.

## Embedded development storage

`internal/db` retains the gameplay-facing API. All former logical databases are
in a single SQLite file with explicit table prefixes. Transactions can therefore
span accounts, characters, guild mirrors and the item-mall log atomically.
Each process owns a bounded connection; WAL and a busy timeout coordinate login
and world. Runtime copies are separate from the versioned development snapshot.
No remote SQL credentials or container lifecycle are needed in the runtime.

SQLite is embedded for development convenience and will be replaced by another backend in the future. The distributed seed intentionally includes shared test accounts and gameplay state. See [storage](sqlite.md) and [operations](../../docs/server.md).
