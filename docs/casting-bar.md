# Original EP5 casting indicator

The current implementation follows the original executable rather than a guessed
HUD texture. Reference executable SHA-256:
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`.
Recovered routines are available in the author's read-only local client decomp.

| Routine | Recovered behavior |
| --- | --- |
| `0x0058fea0` | Reads cast-start `0x0510`; session vtable callback at offset `0x144`. |
| `0x005780a0` | Starts the local preparation indicator. Duration is skill `ReadyTime × 250 ms`; records total, remaining and tick. |
| `0x004bff80` | Updates remaining time; clears preparation state when finished. |
| `0x004c02d0` | Projects player position plus 0.5 world units along horizontal camera-right, rounds screen coordinates and draws the indicator. |
| `0x004a3020` | Black rectangle **5 pixels wide × 64 high**. Blue interior at `(x+1,y+1)`, width **3**, height proportional to the supplied current/total value. GX supplies elapsed progress; see the direction correction below. |
| `0x004a2690` | Solid vertex-colored rectangle; no dedicated cast-bar artwork. |

The indicator is **vertical**, pure blue (`ARGB 0xff0000ff`), with a black
one-pixel border. The blue column starts empty and fills as preparation advances.
This direction was corrected following the author's in-game verification; the
initial remaining-time interpretation of the recovered state was reversed.
It has no skill-name text, numeric timer, ornament, icon or fixed bottom-HUD
anchor. World projection makes its screen position follow the character and
camera. Subsequent interface windows may cover it, following the original draw
order. Native half-pixel D3D raster placement maps to integer GL rectangle edges.

GX uses the existing actor preparation clock started by the server announcement.
Movement, cancellation and results end it. An unavailable/zero duration produces
no invented progress bar. Packet sizes and server rules are unchanged.
The former framed red HUD implementation and its restored optional artwork have
been removed from use. Source: `internal/ui/hud.go`, `internal/app/combat.go`.

Validation includes duration/rounding cases, a GL pixel comparison of the whole
indicator at full/half/finished time, an in-world capture and N2 server casting.
Private disassembly/extracts are retained under `work/cast-reverse/`.
