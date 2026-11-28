#!/usr/bin/env python3
"""Patch interface/option_graphic.png of a GX data folder (idempotent).

* bakes "Antialiasing" into the frame art in place of "Color Bit";
* adds 146 px of black Etc. Option area (and shifts the bottom frame down) so
  four more unlabeled checkboxes fit.

* paints thirteen empty checkbox frames (ClientSettings): one beside
  "Effect On/Off" at (156,451), then rows of two at y 475, 499, 523 ... 595;
  (156,451) is labelled VSync and (35,475) Anisotropic, the rest stay spare;
* fills the grown area with real panel bands so it does not look stretched.

Usage: tools/patch_option_graphic.py <data dir> [pristine.png] [font.ttf]
The default pristine backup is stored in .local/option-art/<data-dir-name>
beside the data directory. Supply pristine.png to use a different backup.
"""
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
from option_art import load_option_art

GROW, SPLIT = 146, 469
LABEL_SIZE = 11          # extra black rows inserted before row 469 (below the last checkbox)
data = Path(sys.argv[1])
root = Path(__file__).resolve().parents[1]
font_path = sys.argv[3] if len(sys.argv) > 3 else str(root / "resources/fonts/ui.ttf")
backup = Path(sys.argv[2]) if len(sys.argv) > 2 else None
target, src = load_option_art(data, "option_graphic.png", backup)
W, H = src.size
out = Image.new("RGBA", (W, H), (0, 0, 0, 0))
out.paste(src.crop((0, 0, W, SPLIT)), (0, 0))
# Fill the grown area with real 24-row bands of the Etc. panel instead of
# repeating single rows (which looked stretched): the frame columns come from
# rows 446-470 and the interior from the blank patch right of "Effect On/Off",
# mirrored horizontally to cover the left half; every other band is flipped
# vertically so band seams join on identical rows.
BAND = 24
band = src.crop((0, 446, W, 446 + BAND))
blank = src.crop((140, 446, 290, 446 + BAND))
band.paste(blank.transpose(Image.FLIP_LEFT_RIGHT).crop((150 - 124, 0, 150, BAND)), (16, 0))
for i, y in enumerate(range(SPLIT, SPLIT + GROW, BAND)):
    piece = band if i % 2 == 0 else band.transpose(Image.FLIP_TOP_BOTTOM)
    out.paste(piece.crop((0, 0, W, min(BAND, SPLIT + GROW - y))), (0, y))
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

# Labels of the spare checkboxes now in use, in the style of the original
# "Shadow Effect" rows (bold, 1 px tracking, drop shadow).
LABELS = {(156, 451): "VSync", (35, 475): "Anisotropic"}
font = ImageFont.truetype(font_path, LABEL_SIZE)
d.fontmode = "L"                  # the row labels are antialiased, unlike "Resolution"
for (cx, cy), label in LABELS.items():
    for dx, dy, fill in ((1, 1, (0, 0, 0, 220)), (0, 0, (226, 222, 214, 255))):
        x = cx + 21 + dx
        for ch in label:          # bold = two passes; the labels below use 1 px tracking
            d.text((x, cy + 2 + dy), ch, font=font, fill=fill)
            d.text((x + 1, cy + 2 + dy), ch, font=font, fill=fill)
            x += d.textlength(ch, font=font) + 1.5
out.save(target)
print("patched", target)
