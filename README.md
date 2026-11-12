# Shaiya GX

A native 64-bit Shaiya EP5.4 client written in Go, using OpenGL 4.5 and the Rapture protocol.

Shaiya GX is an independent client implementation in active development. It reads your existing game archives locally. **This repository does not include game data, original executables, textures, models, audio, account databases, or credentials.**

## Current status

The Linux client supports login, character selection and creation, world rendering, character movement, combat, skills, inventory, equipment, chat, NPC services, and a growing set of social and gameplay interfaces. It connects to Rapture; the server remains authoritative for game rules and outcomes.

Implemented systems include:

- Original archive and asset readers, skinned characters, terrain and objects, map fog, effects, water and actor shadows.
- Third-person and first-person cameras, jumping, evasive steps, walking, automatic movement and click-to-move.
- Player/target HUD, draggable action bars, skill-book drag and drop, item/skill tooltips, inventory character preview, Status resource bars and Item Mall icons.
- Draggable buff/debuff strips with native duration bars, minimap and full map, map information plate, and a resizable two-pane chat with channel selection and input history.
- Nonmodal options, audio and mouse settings, party/raid, quests, trade, duels, guild interfaces, storage, blacksmith services and portals.
- Action-panel auto attack and pickup that approach their selected nearby target before sending the request.

This is a **development build**, not a finished replacement for the original client. Interface parity, navigation/collision, visual effects and several service variants remain incomplete. Windows is a planned target and is not currently validated. See [interface status](docs/interface.md) and [architecture](docs/architecture.md).

## Build on Linux

Requirements:

- Go 1.27.3 and a C compiler; CGO must be enabled.
- OpenGL 4.5-capable graphics drivers.
- Development libraries for OpenGL, X11/GLFW and ALSA, plus `pkg-config`.
- Your own compatible `data.sah` and `data.saf` archives, kept outside this repository.
- A Tahoma-compatible TrueType font. Set `SHAIYA_FONT` to its local path; the default lookup also checks `fonts/tahoma.ttf` beside the executable and Wine's system font directory.

```sh
go build -o build/shaiya ./cmd/shaiya
SHAIYA_DATA=/path/to/local/game-data \
SHAIYA_FONT=/path/to/tahoma.ttf \
./build/shaiya -login 127.0.0.1:30800
```

The graphical client asks for the account at login. Passwords are not persisted. The headless diagnostic client reads its password from `SHAIYA_PASSWORD`.

Open Sans SemiBold is embedded for the interface elements that require it. Its [SIL Open Font License](internal/ui/fonts/OFL.txt) is included. No original game font is bundled.

## Controls

| Input | Action |
|---|---|
| W / S | Forward / backward |
| A / D | Turn |
| Q / E | Side step |
| Right mouse | Steer; A / D become side steps |
| Both mouse buttons | Move forward |
| Mouse wheel | Zoom |
| Space / C | Jump / sit or stand |
| Tab / NumLock | Walk or run / automatic movement |
| F / X / H | Attack selected target / stop attack / approach nearby loot |
| I / T / K / B | Inventory / Status / skills / actions |
| U / G / M | Quests / guild / map |
| O / Escape | Options; Escape closes an active panel first |
| Enter / Up / Down | Chat; navigate sent-message history while typing |
| 1–9, 0 / numeric keypad | First / second action bar |
| F9 / F10 | Reset camera / first-person view |

Drag window headers to move supported windows. Buff/debuff strips drag from their icons; drag the minimap rim or information plate, and the full-map header. Resize chat histories from their top edges. UI positions and supported settings are stored in the user's configuration directory.

## Validation

```sh
go vet ./...
go test ./...
staticcheck ./...
python3 tools/check_publication.py --history
```

Tests that require a graphics context, local game archives or a server are opt-in. [Building and testing](docs/building.md) describes these checks. Integration probes use disposable accounts and an isolated database; they must not be pointed at a production account or database.

## Repository contents

`cmd/` contains the graphical client, headless client and archive-inspection utility. `internal/` contains networking, protocol, state, data readers, rendering, UI and audio. `tools/` contains local build, packaging and analysis helpers. `docs/` contains English public documentation.

Game assets and generated files belong outside Git. See [.gitignore](.gitignore), [third-party notices](NOTICE), and the [protocol notes](docs/protocol.md).

Shaiya and its original game content belong to their respective rights holders. Public source availability does not grant rights to the original game's assets or branding.
