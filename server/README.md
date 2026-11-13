# GX Server

The integrated server component of [Shaiya GX](../README.md), formerly Rapture. This directory is part of the parent GX repository and Go module; it is not a separate project or submodule.

The micro app is **GX Server** (`build/gx_server[.exe]`). It keeps the same one-button Start/Stop behavior. The terminal console uses `start server` / `shutdown server`. Closing either controller leaves login/world running.

From the GX root:

```sh
./server/tools/build.sh
python3 server/tools/package.py --output out/portable
```

On Windows use `server/tools/build.ps1`. Portable Linux/Windows archives, including micro app and console, are published in [GX releases](https://github.com/Spelunkern/Shaiya-GX/releases/latest).

**SQLite is embedded for ease of development and distribution. It is temporary and will be replaced by a different database backend in the future.** The distributed seed retains the author's current test accounts, characters, items and definitions. First startup copies it into an independent mutable runtime; updates preserve existing runtime state.

Read [first run](../docs/quickstart.md), [build instructions](../docs/building.md), [operations](../docs/server.md), [storage](docs/sqlite.md), [distribution](docs/distribution.md) and [parity notes](docs/original-parity.md). Keep existing `RAPTURE_*` environment variables and `rapture.sqlite` filenames for compatibility. Source copyright/license is preserved in [LICENSE](LICENSE).

GM role and the six server-authorized commands are documented in [GM controls](../docs/gm.md).
