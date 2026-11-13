# Embedded development database

**SQLite is a development choice because its embedded operation simplifies setup, testing and portable distribution. GX plans to replace it with a different database backend in the future. The final engine, migration format and timing are not selected.** Keep gameplay/storage boundaries explicit; do not assume SQLite is the permanent production architecture.

## Files and first launch

- `data/rapture.sqlite`: the explicitly requested shared test snapshot, including current development accounts, characters, inventory, guild/social state and game definitions.
- `data/rapture.manifest.json`: table counts and seed SHA-256.
- `runtime/rapture.sqlite`: mutable working copy, ignored by Git. Startup copies the seed only when absent. An existing runtime is never reset by an update.
- `RAPTURE_DB_PATH`, `RAPTURE_DB_SEED`, `RAPTURE_DATA_DIR`: compatible overrides for runtime, seed and supplied server maps/configuration.

The names remain compatible with the former Rapture server. They no longer identify a separate project. Login and world share one SQLite file using `modernc.org/sqlite`; no C SQLite library, credentials, Docker or database daemon is required. Each process owns a bounded connection. WAL, a busy timeout, full synchronization and immediate write transactions coordinate writes.

The snapshot includes test account credentials and gameplay state by explicit choice. Personal future runtime files, backups, local overrides and logs remain outside Git and releases.

## Schema and behavior

Original `PS_UserData`, `PS_GameData`, `PS_GameDefs` and `PS_Billing` names become table prefixes in one file. The schema retains source columns, primary keys, indexes and IDs. Former stored procedures are parameterized queries/transactions in `server/internal/db`. Character creation/deletion/rename, guild changes, item claims, login/logout and point deductions preserve the existing API. The original `Rapture_LoginLog` name remains intact.

Dates use UTC text; integer/blob data is not exported through floating-point CSV. SQLite NOCASE is not identical to every SQL Server non-ASCII collation. SQLite initialization is serialized and atomic. The distributed seed is closed/checkpointed with no required WAL/SHM; a running runtime can have both companions.

## Administration

From `server/` in source, or the extracted portable folder:

```sh
python3 tools/sqlite.py init
python3 tools/sqlite.py account player
python3 tools/sqlite.py query --file query.sql
```

On Windows use `py -3`. Account creation prompts for a password. Automation may use `RAPTURE_ACCOUNT_PASSWORD` in the environment. The account tool never overwrites an existing account. SQL files use SQLite syntax and prefixed table names.

For a consistent backup:

```sh
python3 tools/snapshot_sqlite.py --source runtime/rapture.sqlite --output /private/backups/gx.sqlite
```

To deliberately update the distributed test seed, omit `--output` or specify `data/rapture.sqlite`. This includes accounts and gameplay state; review the intended source before publishing. Keep seed and manifest together. It does not touch the source runtime and includes committed WAL. Rebuilding a portable alone does not take a fresh snapshot.

Stop both services before restoring/replacing a runtime. Use the backup API for a live copy; never copy only an open main file and discard its WAL. Detailed upgrade/LAN/restore instructions are in the source [operations guide](../../docs/server.md).

## Legacy migration and tests

`tools/export_sqlite.py` and `tools/sqlserver.py` remain migration/reference utilities. Ordinary play needs no SQL Server. Existing external SQL Server volumes are not altered by GX integration. A future engine migration must include an explicit validated conversion, rather than pretending the SQLite file can simply be reused by another engine.

Storage tests and server integration tests use disposable runtime copies. Snapshot tests verify WAL inclusion, 64-bit IDs, Unicode/blob preservation and preservation of previous output on validation failure. Portables validate both hash and row counts. The gameplay/server rules do not change as part of naming and repository integration.
