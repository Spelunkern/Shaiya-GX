#!/usr/bin/env python3
"""Prepare external GX UI/Unicode fonts and a lazy coverage index (no game build).

Requires fontTools for this offline import step, never for the running game.
Copy the primary font's license beside ui.ttf and retain fallback licenses.
"""
import argparse
import json
from pathlib import Path
import shutil
from fontTools.ttLib import TTFont


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--primary', required=True, type=Path)
    parser.add_argument('--fallbacks', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    primary, source, output = (p.expanduser().resolve() for p in
                               (args.primary, args.fallbacks, args.output))
    if output == source or output.is_relative_to(source):
        parser.error('output must be separate from the fallback source')
    output.mkdir(parents=True, exist_ok=True)
    if primary != output/'ui.ttf':
        shutil.copy2(primary, output/'ui.ttf')
    target = output/'unicode'
    shutil.copytree(source, target, dirs_exist_ok=True)
    priority = ['NotoSans-Regular.ttf', 'NotoSansArabic-Regular.ttf',
                'NotoSansDevanagari-Regular.ttf', 'NotoSansCJKsc-Regular.otf',
                'NotoSansSymbols2-Regular.ttf']
    files = sorted((p for p in target.iterdir() if p.suffix.lower() in {'.ttf', '.otf'}),
                   key=lambda p: (priority.index(p.name) if p.name in priority else len(priority), p.name))
    entries = []
    for file in files:
        with TTFont(file, lazy=True) as font:
            codes = sorted(font.getBestCmap())
        ranges = []
        for code in codes:
            if ranges and code == ranges[-1][1] + 1:
                ranges[-1][1] = code
            else:
                ranges.append([code, code])
        entry = {'file': 'fonts/unicode/'+file.name, 'ranges': ranges}
        if file.name == 'NotoColorEmoji.ttf':
            entry['color'] = True
        entries.append(entry)
    (output/'fonts.json').write_text(json.dumps(
        {'primary': 'fonts/ui.ttf', 'fallbacks': entries}, separators=(',', ':'))+'\n')
    print(f'Indexed {len(entries)} external fallback fonts in {output}')


if __name__ == '__main__':
    main()
