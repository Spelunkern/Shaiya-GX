# First run: Windows and Linux

GX consists of a graphical client and GX Server in the same repository. They can run on one computer or on separate LAN computers. Begin locally; the default addresses are loopback.

## Download a portable server

Open [GX releases](https://github.com/Spelunkern/Shaiya-GX/releases/latest). Download the archive for your 64-bit OS and `SHA256SUMS.txt`:

- Windows: `gx-server-windows-amd64.zip` (Explorer → Extract All).
- Linux: `gx-server-linux-amd64.tar.gz` (`tar -xzf gx-server-linux-amd64.tar.gz`).

Choose a writable folder in your user directory, not Program Files or a protected system directory. Keep `build`, `data`, `docs` and the launchers together. No Go, Python, Docker or database server is needed for ordinary server use. Linux's GUI needs system graphics libraries; the headless binaries and console are alternatives.

### Verify a download

Windows PowerShell:

```powershell
Get-FileHash .\gx-server-windows-amd64.zip -Algorithm SHA256
Get-Content .\SHA256SUMS.txt
```

Compare the full displayed hash to the line for that exact filename (case does not matter). Linux:

```sh
sha256sum --ignore-missing -c SHA256SUMS.txt
```

Each extracted package also contains `package.manifest.json` with per-file hashes. It excludes itself and runtime files created after startup.

## Start and stop

Windows: double-click `build\gx_server.exe`. Linux: run `./build/gx_server`. The **GX Server** window has the same one-button start/stop behavior as the former Rapture app. Click **Start server** and wait until login/world show ON. DB READY means a runtime file exists; it is not a full integrity diagnostic.

For a terminal instead, run `Start GX Server.bat` or `./Start\ GX\ Server.sh`. Enter:

```text
start server
shutdown server
```

Closing either control window leaves the detached processes running. Stop explicitly before moving the folder, replacing binaries or resetting the database. Logs are in `logs/gx_login.log` and `logs/gx_world.log` inside the server folder.

The first start copies `data/rapture.sqlite` to `runtime/rapture.sqlite`. Future starts reuse the runtime file. The shipped seed contains the author's requested current **test** accounts, characters and inventory. A fresh extraction starts from that snapshot; an existing runtime is never silently overwritten.

**SQLite is an embedded development convenience. The project will change database engines in the future; no replacement engine or migration schedule is promised yet.**

## Accounts

Use a known test account from the distributed snapshot, or create your own. Python 3 is only required for this optional administration tool. Run from the extracted server directory:

Windows:

```powershell
py -3 tools/sqlite.py account player
```

Linux:

```sh
python3 tools/sqlite.py account player
```

The tool prompts for a password; avoid putting passwords in shell commands. It initializes the runtime from the seed if needed. Account names must fit 1–18 characters and passwords 1–32 characters in the protocol's encoding. An account already present is not overwritten. In a source checkout, use `python3 server/tools/sqlite.py account player` from the GX root.

## Run the client

Build using [the OS-specific instructions](building.md). Obtain your own compatible EP5.4 `data.sah` and `data.saf` plus a Tahoma-compatible font. Keep those files outside Git. Do not use the original `Game.exe` to run GX.

Windows PowerShell, from the GX checkout:

```powershell
$env:SHAIYA_DATA = 'C:\Games\ShaiyaData'
$env:SHAIYA_FONT = 'C:\Windows\Fonts\tahoma.ttf'
.\build\shaiya.exe -login 127.0.0.1:30800
```

Linux:

```sh
SHAIYA_DATA=/path/to/game-data SHAIYA_FONT=/path/to/tahoma.ttf \
  ./build/shaiya -login 127.0.0.1:30800
```

Sign in, select the server, choose/create your character and enter the world. GX Server decides outcomes and saves progress. Copying archives and `fonts/tahoma.ttf` beside a locally packaged client also works. Public GX server downloads do not include those client assets.

## If startup fails

- No login/world ON: read the end of the server logs. Check that `build/` and `data/` are still together and the folder is writable.
- Address already in use: another instance is using 30800 or 30810. Reuse it or stop it through its own control app; do not kill unrelated game sessions.
- Client cannot create a graphics context: install your GPU vendor's driver with OpenGL 4.5 support. A working server does not imply working client graphics.
- Login succeeds but world fails: check the advertised world address, world port and firewall. See [LAN configuration](server.md#lan-connections).
- Changes appear missing: confirm which `runtime/rapture.sqlite` is being used. Extracting a new folder creates a different independent runtime. Preserve your old runtime when updating.
