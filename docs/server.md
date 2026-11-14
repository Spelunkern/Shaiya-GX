# Operating GX Server

GX Server is part of GX under `server/`. In a source checkout, server paths in this guide are relative to `server/`; in a portable archive they are relative to the extracted top-level folder. The former standalone Rapture repository is retired after migration.

## Micro app and console

`build/gx_server[.exe]` opens **GX Server**. Click Start server or Stop server. Login/world remain detached when you close the window. `Start GX Server.sh` / `.bat` opens the terminal console; commands are `start server` and `shutdown server`. The same embedded database and game services back both controls. Use one controller per installation at a time.

Alternatively run `build/gx_login[.exe]` and `build/gx_world[.exe]` directly and manage those process lifetimes yourself. Each binary finds `data/` relative to its executable when no data/database overrides are provided, so launching from another working directory works. Do not mix binaries from different versions.

The controller checks executable paths, including on Windows, so stopping one installation does not stop a separate copy merely because its process names match. An app status refresh may take several seconds, especially Windows process enumeration.

## Database and persistence

- Distributed seed: `data/rapture.sqlite`; accompanying hash/table counts: `data/rapture.manifest.json`.
- Mutable state: `runtime/rapture.sqlite`. Created from the seed only when absent.
- Login/world handoff keys: `runtime/sessions/`.
- Local overrides: `.local/runtime.env` (KEY=VALUE, not committed).
- Logs: `logs/gx_login.log`, `logs/gx_world.log`.

The distributed database intentionally retains the current development accounts, characters and inventory requested by the author. Treat it as a shared test fixture. **SQLite is temporary development storage, chosen for embedded simplicity. A future backend will replace it; the replacement has not been chosen.** The gameplay-facing database API remains isolated in `server/internal/db`.

Existing `RAPTURE_*` configuration names and SQLite filenames are retained for compatibility. They do not imply an external Rapture installation is required.

## LAN connections

Defaults listen on `127.0.0.1:30800` (login) and `127.0.0.1:30810` (world). To accept another computer, set the following before starting the controller.

Windows PowerShell (replace the example IP with your server's reachable IPv4 address):

```powershell
$env:RAPTURE_LOGIN_LISTEN = '0.0.0.0:30800'
$env:RAPTURE_WORLD_LISTEN = '0.0.0.0:30810'
$env:RAPTURE_PUBLIC_IP = '192.168.1.20'
.\build\gx_server.exe
```

Linux:

```sh
RAPTURE_LOGIN_LISTEN=0.0.0.0:30800 RAPTURE_WORLD_LISTEN=0.0.0.0:30810 \
RAPTURE_PUBLIC_IP=192.168.1.20 ./build/gx_server
```

Allow inbound TCP 30800/30810 on the server firewall. Connect the client with `-login 192.168.1.20:30800`. If using another world port, also pass `-world-port PORT` to the GX client. `RAPTURE_PUBLIC_IP` is the address advertised to the client for world, so it must be reachable from the client's computer; `0.0.0.0` is a listen wildcard, not an advertised address.

For persistent overrides create `.local/runtime.env` in the server directory. Explicit environment values take priority. `RAPTURE_DATA_DIR`, `RAPTURE_DB_PATH` and `RAPTURE_DB_SEED` select data/runtime/seed locations independently. Relative paths in overrides are resolved against the server folder when using the controllers. Prefer absolute paths for direct headless launches with explicit overrides.

## Back up and restore

For a consistent live backup, use the SQLite backup API through the optional Python tool:

```sh
python3 tools/snapshot_sqlite.py --source runtime/rapture.sqlite --output /private/backups/gx.sqlite
```

On Windows replace `python3` with `py -3` and use your destination path. This preserves committed WAL data, validates integrity and emits a manifest. Backups can contain account credentials and player state; keep personal backups outside Git.

To restore, stop both services, verify they are stopped, replace the runtime database with the validated backup and remove any stale `-wal` / `-shm` companions from the stopped runtime before restarting. Never overwrite an open SQLite database or restore only its main file while its old WAL is still present. Keep the previous stopped runtime until restoration is verified.

## Update an installation

1. Stop login/world using the controller.
2. Back up `runtime/rapture.sqlite` consistently and retain `.local/runtime.env`.
3. Extract the new release to a new folder and copy your stopped runtime into its `runtime/` directory. Preserve your override settings. Do not replace the new seed with your runtime unless deliberately preparing a shared test fixture.
4. Start the new version and verify login, character selection and expected progress. Keep the previous folder until verified.

Starting a newly extracted package without copying a runtime creates a new world from the included test snapshot. Replacing the seed alone never resets an existing runtime. Future database-engine migrations will need explicit migration instructions; no automatic cross-engine conversion exists today.

## Renaming and migration

Rapture's source is now `server/` in the GX module. Commands and binaries use `gx_*`; the small app and advertised server name use GX Server. Protocol opcodes, gameplay rules, schema and database API behavior are unchanged by integration. The legacy `Rapture_LoginLog` table and `rapture.sqlite` filenames intentionally remain, preserving compatibility. Existing clients still use the same login/world endpoints.


## Equipment condition

GX equipment has no durability: attacks never wear items, legacy stored quality
never reduces equipment statistics, and repair requests are ignored without
charging gold. Enhancement failure does not create a broken/zero-quality item.
Legacy database and wire quality fields remain compatible; authored quality
metadata still has unrelated uses such as quest-item drop probabilities. No
personal database migration is required. Source changes take effect after the
world server is rebuilt and restarted.


The static Item Mall catalog loads from SQL `PS_GameDefs_ProductList`. The
client presents native `Cash.SData`; keep both catalogs aligned. See
[cash catalog](cash-catalog.md). Balances, transactions and gifts use SQL.


Blacksmith previews use the read-only GX forge quote (`0x7F03`), with request
serial, live inventory slot references and material count. Replies supply the
success rate per million, gold cost and compatible enchantment hints; no gold or
items change during preview. Actual attempts still validate inventory and use
the same rate/cost functions. The client discards stale quotes and requests a
fresh quote before opening a link/extract confirmation. Enchant attempts execute
immediately; their success percentage is displayed above the action button.

Forge costs use the native ReqIg table without Vg/level discounts or the old
free-cost environment override. Enchant rates, cap (default20) and bonuses are
compiled in `server/internal/world/enchant_rules.go`; ItemEnchant.ini is no
longer read or required. See the [feature wiki](wiki/README.md) for exact rules,
item content fields, extraction, recreation and drop persistence.
