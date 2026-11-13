# Building, testing and packaging GX

Run commands from the repository root unless noted. Client and server share the root `go.mod` / `go.sum`; `server/` has no separate Go module or Git repository. Go 1.27.3 is the tested toolchain. Use an amd64 toolchain and `go version` to confirm it.

## Linux

Install Go, GCC, `pkg-config`, OpenGL development headers, X11, Wayland/xkbcommon headers and ALSA development libraries. For Debian/Ubuntu:

```sh
sudo apt-get install build-essential pkg-config libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev libasound2-dev
./tools/build.sh
```

The script vets, builds client utilities into `build/`, builds server programs into `server/build/`, and runs ordinary unit tests. On other distributions install equivalent packages. The client needs an OpenGL 4.5-capable GPU/driver. The Fyne micro app also needs desktop graphics. The portable headless server/console does not require these development packages.

## Portable runtime requirements

The published Linux amd64 micro app is built on the Ubuntu CI runner and requires **glibc 2.38 or newer** (for example Ubuntu 24.04), plus OpenGL, X11/Wayland and xkbcommon runtime libraries. It is not a universal static GUI binary for every Linux distribution. On an older distribution, use the CGO-free console/headless services, or build the GUI locally against that distribution's libraries.

For the GUI, install your distribution's runtime packages providing `libGL.so.1`, `libX11.so.6`, `libXrandr.so.2`, `libXxf86vm.so.1`, `libXi.so.6`, `libXcursor.so.1`, `libXinerama.so.1`, Wayland client/cursor/EGL and `libxkbcommon.so.0`. Check missing libraries with `ldd build/gx_server` inside an extracted Linux server package. No Go toolchain or development headers are needed to run a prebuilt app. The client additionally needs a working OpenGL 4.5 driver and its audio runtime libraries.

Windows archives use the system graphics/runtime DLLs; no Go toolchain or C compiler is needed to run them. Install the GPU vendor's OpenGL-capable driver for the graphical applications. Native CI covers the hosted Windows environment; a specific desktop/GPU still needs its own gameplay validation.

## Windows (native)

Install the Go 1.27.3 amd64 toolchain and a 64-bit MinGW-w64 GCC (for example MSYS2 UCRT64's `mingw-w64-ucrt-x86_64-gcc`). Add Go and that GCC's `bin` directory to your PowerShell PATH, then confirm:

```powershell
go version
gcc --version
$env:CGO_ENABLED = '1'
powershell -NoProfile -ExecutionPolicy Bypass -File tools/build.ps1
```

Do not use a 32-bit compiler or MSVC `cl` as the CGO compiler. Output: `build/shaiya.exe`, `build/shaiyacli.exe`, `build/assets.exe` and `server/build/gx_{login,world,console,server}.exe`. `gx_server.exe` is a GUI-subsystem app. Audio uses the Windows backend; ALSA is only a Linux dependency. Run from a normal desktop with your GPU vendor's OpenGL 4.5 driver.

## Windows cross build from Linux

With `x86_64-w64-mingw32-gcc` installed:

```sh
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
  go build -o build/shaiya.exe ./cmd/shaiya
python3 server/tools/package.py --platform windows-amd64 --output out/portable
```

The packager cross-builds the GUI with MinGW (`GX_WINDOWS_CC` overrides the compiler). Login/world/console use CGO=0. Cross-compilation proves build compatibility; native Windows execution is checked separately by CI. [Fyne's cross-build guide](https://docs.fyne.io/started/cross-compiling/) explains the GUI toolchain requirement.

## Tests and continuous integration

```sh
go vet ./...
go test ./...
go test -tags integration ./server/... -count=1
python3 -m unittest discover -s server/tools -p 'test_*.py'
python3 tools/check_publication.py --history
```

The publication check inspects staged source and, with `--history`, reachable history. Stage intended files first. It permits the explicitly distributed `server/data/` fixtures and the authored micro app icon, while rejecting client assets, local runtime copies, credentials and generated binaries.

`.github/workflows/ci.yml` builds and tests on native Linux and Windows runners, including server integration, portable startup from a relocated directory containing spaces, and console Start/Stop isolation from a second running installation. Portable packages are uploaded as workflow artifacts. Tagged GX releases publish the portable archives and checksums. A green run does not establish full visual parity: graphical tests need licensed local assets and a real OpenGL context.

## Local assets and graphics checks

Set `SHAIYA_DATA` to the directory containing your compatible `data.sah` and `data.saf`, and `SHAIYA_FONT` to a Tahoma-compatible font. The client also checks for archives and `fonts/tahoma.ttf` beside its executable. Open Sans SemiBold is embedded under its included SIL license; Tahoma is not bundled publicly.

```sh
SHAIYA_TEST_RENDER=1 SHAIYA_DATA=/path/to/game-data \
SHAIYA_FONT=/path/to/tahoma.ttf SHAIYA_TEST_ARTIFACTS=/path/to/private/captures \
  go test -tags integration ./internal/app \
  -run 'TestHUDPolishOriginalAssetsGL|TestPhoenixChatAndLoginLogoGL' -count=1 -v
```

`python3 tools/n2_smoke.py` is the Linux end-to-end client/server probe. It uses `server/build`, copies the distributed seed to temporary storage, creates disposable fixtures and uses ports 40800/40810. It does not use your mutable runtime DB or stop your personal services. `GX_SERVER_DIR` overrides the server directory; legacy `RAPTURE_DIR` remains accepted. Legacy SQL Server-only probes are disabled; use N2 for SQLite.

## Package a portable server

```sh
python3 server/tools/package.py --output out/portable
```

This requires Go, Python 3 and GUI build dependencies. Building both targets on Linux also requires MinGW. It emits `gx-server-linux-amd64.tar.gz`, `gx-server-windows-amd64.zip` and `SHA256SUMS.txt`. Every package includes the GUI, console, login/world, seed, server data, docs, license and optional account/backup tools. Packaging fails if the GUI fails to build; it cannot silently disappear from a release. The packager validates seed hashes, table counts and integrity before shipping and starts the native target in a disposable relocated directory.

Rebuilding does not take a new snapshot from a personal runtime. Refresh deliberately using [server storage instructions](../server/docs/sqlite.md), then build. Release binaries are attached to the GX repository's releases rather than committed into source history.

## Prepare a private client folder (Linux)

`python3 tools/package.py --source /path/to/local/assets --output /path/to/play-folder` copies your own game files, font, client, docs and launcher for personal use. Set `GX_SERVER_DIR` to your source `server/` folder or an extracted portable server. `--update` replaces the Linux executable atomically and preserves installed archives. This helper is for private Linux play folders; Windows users can run `build/shaiya.exe` with the environment variables above. Never publish a client play folder containing game assets.
