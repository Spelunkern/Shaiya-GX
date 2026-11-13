#!/usr/bin/env python3
"""Validate the redistributable external font resources before packaging."""
import hashlib
import json
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parents[1]
FONT_PACK = ROOT / 'resources/fonts'


def validate(directory=FONT_PACK):
    directory = Path(directory)
    manifest = json.loads((directory / 'pack.json').read_text())
    if manifest['version'] != 1:
        raise ValueError('unsupported font pack version')
    files = manifest['files']
    actual = {p.relative_to(directory).as_posix() for p in directory.rglob('*') if p.is_file()}
    if actual != set(files) | {'pack.json'}:
        raise ValueError('font pack file inventory mismatch')
    for name, expected in files.items():
        relative = PurePosixPath(name)
        if relative.is_absolute() or '..' in relative.parts:
            raise ValueError('unsafe font resource path')
        data = (directory / name).read_bytes()
        if len(data) != expected['bytes'] or hashlib.sha256(data).hexdigest() != expected['sha256']:
            raise ValueError('font resource checksum mismatch: ' + name)
    for name in ('ui.ttf', 'fonts.json', 'Tahoma-NOTICE.txt', 'LGPL-2.1-or-later.txt',
                 'unicode/LICENSE-OFL.txt', 'unicode/COPYRIGHTS.txt'):
        if name not in files:
            raise ValueError('missing font/license: ' + name)
    coverage = json.loads((directory / 'fonts.json').read_text())
    for name in [coverage['primary']] + [f['file'] for f in coverage['fallbacks']]:
        if not name.startswith('fonts/') or name[6:] not in files:
            raise ValueError('font coverage references an absent resource')
    return manifest


if __name__ == '__main__':
    result = validate()
    print('External font pack verified:', len(result['files']), 'files')
