#!/usr/bin/env python3
"""Patch interface/option_graphic.png of a GX data folder (idempotent).

* bakes "Antialiasing" into the frame art in place of "Color Bit";
* adds 146 px of black Etc. Option area (and shifts the bottom frame down) so
  four more unlabeled checkboxes fit.

* paints thirteen empty checkbox frames (ClientSettings.Spare): one beside
  "Effect On/Off" at (156,451), then rows of two at y 475, 499, 523 ... 595;
  the client toggles them with no label.

Usage: tools/patch_option_graphic.py <data dir> [pristine.png] [font.ttf]
The pristine art is kept at work/option_graphic.original.png (outside the
client folder) and copied from the data folder on first use.
"""
import shutil
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

GROW, SPLIT = 146, 469          # extra black rows inserted before row 469 (below the last checkbox)
data = Path(sys.argv[1])
root = Path(__file__).resolve().parents[1]
font_path = sys.argv[3] if len(sys.argv) > 3 else str(root / "resources/fonts/ui.ttf")
target = data / "interface/option_graphic.png"
backup = Path(sys.argv[2]) if len(sys.argv) > 2 else root / "work/option_graphic.original.png"
if not backup.exists():
    shutil.copy(target, backup)
src = Image.open(backup).convert("RGBA")
W, H = src.size
out = Image.new("RGBA", (W, H), (0, 0, 0, 0))
out.paste(src.crop((0, 0, W, SPLIT)), (0, 0))
tile = [src.crop((0, y, W, y + 1)) for y in range(SPLIT, SPLIT + 4)]
seq = tile + tile[::-1]
for i in range(GROW):
    out.paste(seq[i % len(seq)], (0, SPLIT + i))
out.paste(src.crop((0, SPLIT, W, H - GROW)), (0, SPLIT + GROW))

# "Color Bit" -> "Antialiasing": clear the old label with clean panel pixels.
patch = out.crop((141, 84, 157, 101))
for x in range(150, 206, 16):
    out.paste(patch, (x, 84))
out.paste(patch.crop((0, 0, 11, 17)), (195, 84))
font = ImageFont.truetype(font_path, 10)
text = "AA"
d = ImageDraw.Draw(out)
d.fontmode = "1"                  # hard pixels, like the game's own labels
def spaced(x, y, fill):
    for ch in text:               # bold = two passes; 1 px tracking as in "Resolution"
        d.text((x, y), ch, font=font, fill=fill)
        d.text((x + 1, y), ch, font=font, fill=fill)
        x += d.textlength(ch, font=font) + 2
width = sum(d.textlength(c, font=font) + 2 for c in text)
x, y = 204 - width, 87
spaced(x + 1, y + 1, (0, 0, 0, 200))
spaced(x, y, (232, 232, 232, 255))
box = src.crop((32, 401, 54, 421))
SPARE = [(35, 475), (156, 475), (35, 499), (156, 499), (156, 451)] + [(x, y) for y in (523, 547, 571, 595) for x in (35, 156)]
for cx, cy in SPARE:
    out.paste(box, (cx - 3, cy - 3))
out.save(target)
print("patched", target)
