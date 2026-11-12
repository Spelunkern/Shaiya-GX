# Building and testing

## Linux build

Use Go 1.27.3 with CGO, a C compiler, `pkg-config`, OpenGL development headers, X11/GLFW dependencies and ALSA development libraries. A functioning OpenGL 4.5 driver is required at runtime.

```sh
go build -o build/shaiya ./cmd/shaiya
go build -o build/shaiyacli ./cmd/shaiyacli
go build -o build/assets ./cmd/assets
go vet ./...
go test ./...
staticcheck ./...
```

`tools/build.sh` runs vet, builds all three executables and runs unit tests. Build outputs are ignored. Windows cross-compilation is not part of the currently validated release path.

## Local assets

Keep `data.sah` and `data.saf` outside the source tree. Set `SHAIYA_DATA` to their directory and `SHAIYA_FONT` to a local Tahoma-compatible font. A packaged executable also looks beside itself for archives and `fonts/tahoma.ttf`.

`tools/package.py --source /path/to/local/assets --output /path/to/play-folder` prepares a local play folder. This copies local game assets for personal testing; **never add that output folder to Git**. The helper assumes a local font and Rapture setup; inspect its options before using it on a different workstation. `--update` replaces an existing Linux executable atomically without truncating the running process's executable inode.

## Opt-in graphics checks

```sh
SHAIYA_TEST_RENDER=1 \
SHAIYA_DATA=/path/to/local/game-data \
SHAIYA_FONT=/path/to/tahoma.ttf \
SHAIYA_TEST_ARTIFACTS=/path/to/local/captures \
go test -tags integration ./internal/app -run TestHUDPolishOriginalAssetsGL -count=1 -v
```

These checks need a graphics context and compatible original archives. They do not download assets. Use an ignored directory for captures, logs and analysis output.

## Server integration

`tools/n2_smoke.py` expects a local Rapture build and seed database. It copies the seed into a temporary directory, provisions disposable accounts/items, starts login/world on ports 40800/40810 and removes its own processes/database afterwards. Existing game services and personal runtime databases are left alone. `RAPTURE_DIR` overrides the local server directory.

The current probe covers movement and remote animation, jump/posture/evasions, mouse controls, frame-rate consistency, approach-and-pickup, approach-and-attack, chat and acknowledged return to selection. Other integration tests require their documented opt-in environment variables and suitable disposable fixtures.
