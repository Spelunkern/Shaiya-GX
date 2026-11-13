# GX Server portable distribution

Download portable Linux/Windows builds from [GX releases](https://github.com/Spelunkern/Shaiya-GX/releases/latest). Both include login, world, console, the **GX Server** micro app, server maps/configuration and the requested development database with current test accounts, characters and inventory.

SQLite is embedded for development convenience. It is a temporary backend and will be replaced in the future. Packages do not imply a final production storage choice.

## Build from the GX checkout

From the GX root, with Go 1.27.3, Python 3 and GUI compiler/libraries installed:

```sh
python3 server/tools/package.py --output out/portable
```

From `server/`, the equivalent is `python3 tools/package.py`. Use `--platform linux-amd64` or `--platform windows-amd64` for one target. Linux-to-Windows GUI builds need MinGW; native Windows uses its local x64 GCC. See the source [build guide](../../docs/building.md) for prerequisites.

Output:

- `gx-server-linux-amd64.tar.gz`
- `gx-server-windows-amd64.zip`
- `SHA256SUMS.txt`

Binaries are attached to GX releases, rather than committed into source history. Tagged release automation reproduces the packages on native runners. Archive manifests list every included file's SHA-256. Packaging validates seed integrity, manifest and table counts; it does not silently refresh the seed from personal runtime state.

## Run a package

Keep the directory structure intact in a writable folder. Open `build/gx_server[.exe]` for the one-button app or the `Start GX Server` launcher for console commands `start server` / `shutdown server`. Closing either controller leaves servers running. First start copies the seed to `runtime/rapture.sqlite`; existing runtimes are preserved.

The headless server and console need no Go, Python, Docker, SQL Server or separate SQLite library. The graphical controller additionally needs desktop graphics/drivers and Linux system GUI libraries. Python 3 is optional for administration tools in `tools/`.

Default endpoints are 127.0.0.1:30800 and :30810. Use `RAPTURE_LOGIN_LISTEN`, `RAPTURE_WORLD_LISTEN` and `RAPTURE_PUBLIC_IP` for other addresses. Compatibility variable names and SQLite filenames remain unchanged after the integration. Source operation instructions are in `../../docs/server.md`; download users can also read README.txt in the archive.

## Seed updates and validation

From the GX root:

```sh
python3 server/tools/snapshot_sqlite.py --source /path/to/test/runtime.sqlite
python3 server/tools/package.py --output out/portable
```

This explicitly publishes the chosen development database, including its accounts and inventory. The snapshot uses SQLite backup, includes committed WAL, validates integrity and updates `data/rapture.manifest.json`. Only snapshot data intentionally approved for distribution. The runtime source is not modified. Never include `.local/`, logs, backups or `runtime/` in an archive.

The native packager starts a private relocated copy in a path with spaces and an unrelated working directory, checks both ports, verifies runtime integrity and unchanged seed, then stops only its own processes. CI runs this on native Windows and Linux. Windows cross-builds and Wine checks are additional evidence, not substitutes for native Windows CI or client GPU testing.
