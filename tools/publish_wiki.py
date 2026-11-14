#!/usr/bin/env python3
"""Validate/export English repository wiki sources. Publication is explicit."""
import argparse
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / 'docs/wiki'
NAMES = {
    'README.md': 'Home', 'quests-and-map.md': 'Quests-and-Map', 'stealth-and-gm-protection.md': 'Stealth-and-GM-Protection', 'elements.md': 'Elements', 'blacksmith.md': 'Blacksmith',
    'item-effects.md': 'Item-Effects', 'gm-commands.md': 'GM-Commands',
    'item-data.md': 'Item-Data', 'recreation-and-loot.md': 'Recreation-and-Loot',
    'item-mall.md': 'Item-Mall', 'server-operations.md': 'Server-Operations',
    'architecture.md': 'Architecture', 'building-and-testing.md': 'Building-and-Testing',
    'data-formats.md': 'Data-Formats', 'protocol.md': 'Protocol',
    'interface.md': 'Interface', 'weapon-effects.md': 'Weapon-Effects',
    'client-portable.md': 'Client-Portable', 'troubleshooting.md': 'Troubleshooting',
}
# References inherited from repository guides now point to their Wiki pages.
ALIASES = {'server.md': 'server-operations.md', 'gm.md': 'gm-commands.md',
           'cash-catalog.md': 'item-mall.md', 'building.md': 'building-and-testing.md',
           'weapon-enchantments.md': 'weapon-effects.md'}
LINK = re.compile(r'(?<!!)\[([^\]]+)\]\(([^)]+)\)')

def export(destination):
    destination.mkdir(parents=True, exist_ok=True)
    for filename, title in NAMES.items():
        source = SOURCE / filename
        def replace(match):
            label, target = match.groups()
            if target.startswith(('https:', 'http:', '#', 'mailto:')):
                return match.group(0)
            raw, sep, anchor = target.partition('#')
            name = Path(raw).name
            name = ALIASES.get(name, name)
            if name in NAMES:
                return f'[{label}]({NAMES[name]}{sep}{anchor})'
            resolved = (source.parent / raw).resolve()
            # Copied repository guides retain their original relative paths.
            if not resolved.exists():
                for base in (ROOT/'docs', ROOT):
                    candidate = (base/raw).resolve()
                    if candidate.exists():
                        resolved = candidate
                        break
            if not resolved.exists() or not resolved.is_relative_to(ROOT):
                raise ValueError(f'{source.name}: unresolved link {target}')
            repository_path = resolved.relative_to(ROOT).as_posix()
            if repository_path.startswith('docs/wiki/'):
                raise ValueError(f'Unmapped wiki page: {repository_path}')
            return f'[{label}](https://github.com/Spelunkern/Shaiya-GX/blob/main/{repository_path}{sep}{anchor})'
        content = LINK.sub(replace, source.read_text())
        (destination / (title+'.md')).write_text(content)
    sidebar = ['## Shaiya GX', '', '[Home](Home)', '']
    sidebar.extend(f'- [{title.replace("-", " ")}]({title})' for title in NAMES.values() if title != 'Home')
    (destination/'_Sidebar.md').write_text('\n'.join(sidebar)+'\n')
    (destination/'_Footer.md').write_text('Current GX development behavior. Released binaries may lag behind this reference. [Repository](https://github.com/Spelunkern/Shaiya-GX) · [Home](Home)\n')
    print(f'Exported {len(NAMES)} pages, sidebar and footer to {destination}')

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--checkout', required=True, type=Path, help='Existing cloned Wiki repository')
    parser.add_argument('--push', action='store_true', help='Explicitly commit and push exported documentation')
    args = parser.parse_args()
    if not (args.checkout/'.git').is_dir():
        parser.error('--checkout must be a cloned Git repository')
    remote = subprocess.check_output(['git', '-C', str(args.checkout), 'remote', 'get-url', 'origin'], text=True).strip()
    if remote not in ('https://github.com/Spelunkern/Shaiya-GX.wiki.git', 'git@github.com:Spelunkern/Shaiya-GX.wiki.git'):
        parser.error('Refusing to export into a different repository')
    export(args.checkout)
    if args.push:
        subprocess.run(['git', '-C', str(args.checkout), 'add', '--', '*.md'], check=True)
        changed = subprocess.run(['git', '-C', str(args.checkout), 'diff', '--cached', '--quiet']).returncode
        if changed == 1:
            subprocess.run(['git', '-C', str(args.checkout), 'commit', '-m', 'Document GX gameplay, items, administration and operations'], check=True)
            subprocess.run(['git', '-C', str(args.checkout), 'push', 'origin', 'HEAD'], check=True)
        elif changed != 0:
            raise RuntimeError('Cannot inspect staged wiki changes')

if __name__ == '__main__':
    main()
