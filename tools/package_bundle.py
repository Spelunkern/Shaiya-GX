#!/usr/bin/env python3
"""Combine verified native releases and local loose data into a dual-OS folder.

Original game resources remain local; this command does not upload them.
"""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import shutil
import tarfile
import tempfile
import zipfile

from font_pack import validate

ARCHIVES = ('gx-client-linux-amd64.tar.gz', 'gx-client-windows-amd64.zip',
            'gx-server-linux-amd64.tar.gz', 'gx-server-windows-amd64.zip')


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def text_content(data):
    if b'\0' in data:
        return None
    for encoding in ('utf-8', 'cp1252'):
        try:
            return data.decode(encoding).replace('\r\n', '\n')
        except UnicodeDecodeError:
            pass
    return None


def merge(source, destination):
    for path in sorted(source.rglob('*')):
        if path.is_symlink():
            raise ValueError('unexpected package symlink')
        if not path.is_file():
            continue
        relative = path.relative_to(source)
        target = destination / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        if target.exists():
            a, b = path.read_bytes(), target.read_bytes()
            if a == b or text_content(a) is not None and text_content(a) == text_content(b):
                continue
            raise ValueError('native packages disagree: ' + str(relative))
        shutil.copy2(path, target)


def verify_cash_copies(client_data, server_data):
    if not (client_data / 'Cash.SData').is_file():
        raise ValueError('client Cash.SData catalog is missing')
    if not (server_data / 'rapture.sqlite').is_file():
        raise ValueError('server ProductList database is missing')


def prepare(archives, data, output, tag):
    if output.exists():
        raise ValueError('output already exists; choose a new folder')
    if not data.is_dir() or output.is_relative_to(data):
        raise ValueError('invalid loose data source/output')
    expected = {}
    for line in (archives / 'SHA256SUMS.txt').read_text().splitlines():
        sha, name = line.split(maxsplit=1)
        expected[name.lstrip('*')] = sha
    for name in ARCHIVES:
        if digest(archives / name) != expected[name]:
            raise ValueError('release checksum mismatch: ' + name)
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='.gx-bundle-', dir=output.parent) as temporary:
        temporary = Path(temporary)
        extracted = temporary / 'native'
        extracted.mkdir()
        for name in ARCHIVES:
            path = archives / name
            if name.endswith('.zip'):
                with zipfile.ZipFile(path) as archive:
                    for member in archive.namelist():
                        relative = PurePosixPath(member)
                        if relative.is_absolute() or '..' in relative.parts:
                            raise ValueError('unsafe release archive path')
                    archive.extractall(extracted)
            else:
                with tarfile.open(path) as archive:
                    archive.extractall(extracted, filter='data')
        stage = temporary / 'bundle'
        client, server = stage / 'Cliente', stage / 'Servidor'
        merge(extracted / 'gx-client-linux-amd64', client)
        merge(extracted / 'gx-client-windows-amd64', client)
        print('Copying local loose game data...', flush=True)
        shutil.copytree(data, client / 'data', dirs_exist_ok=True,
                        ignore=lambda directory, names: ['fonts'] if Path(directory) == data else [])
        validate(client / 'data/fonts')
        for platform in ('linux-amd64', 'windows-amd64'):
            source = extracted / ('gx-server-' + platform)
            manifest = source / 'package.manifest.json'
            record = json.loads(manifest.read_text())
            for name, sha in record['files'].items():
                if digest(source / name) != sha:
                    raise ValueError('native server manifest mismatch: ' + name)
            (server / 'manifests').mkdir(parents=True, exist_ok=True)
            shutil.copy2(manifest, server / 'manifests' / (platform + '.json'))
            manifest.unlink()
            merge(source, server)
        verify_cash_copies(client / 'data', server / 'data')
        files = {p.relative_to(server).as_posix(): digest(p) for p in sorted(server.rglob('*')) if p.is_file()}
        (server / 'package.manifest.json').write_text(json.dumps({'platform': 'linux-amd64+windows-amd64', 'files': files}, indent=2) + '\n')
        (stage / 'README.txt').write_text(f'''Shaiya GX {tag} — Windows and Linux

1. Open Servidor/build/gx_server (Linux) or gx_server.exe (Windows) and start the server.
   Console launchers Start GX Server.sh and Start GX Server.bat are also included.
2. Open Cliente/Shaiya GX (Linux) or Cliente/Shaiya GX.exe (Windows).

The client executables share Cliente/data/, including external fonts and licenses.
Keep the complete folder together. An OpenGL 4.5-capable GPU/driver is required.
Default local endpoints: 127.0.0.1:30800 and 127.0.0.1:30810.

GM role and commands: Servidor/docs/gm.md. Example: /nmake 1 1 creates merchant 1.
The supplied development seed has GM test accounts. Server startup creates a new
independent runtime database; no personal runtime or local credentials are included.
SQLite is embedded development storage and will be replaced in the future.

Original game resources retain their respective ownership. Font and software
licenses are included in Cliente/ and Servidor/. SHA256SUMS.txt verifies this folder.
''')
        (stage / 'SHA256SUMS.txt').write_text(''.join(digest(p) + '  ' + p.relative_to(stage).as_posix() + '\n'
                                                  for p in sorted(stage.rglob('*')) if p.is_file()))
        stage.replace(output)
    print('Distributable folder ready:', output, flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--archives', type=Path, required=True)
    parser.add_argument('--data', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--tag', required=True)
    args = parser.parse_args()
    prepare(args.archives.resolve(), args.data.resolve(), args.output.resolve(), args.tag)


if __name__ == '__main__':
    main()
