#!/usr/bin/env python3
"""Reject game assets, local runtime files and recognizable secrets before publishing.

Checks the staged source tree, and optionally every reachable history blob.
Only reports paths, never the contents of a potential credential.
"""
import argparse
from pathlib import PurePosixPath
import re
import subprocess

GAME_SUFFIXES = {'.sah', '.saf', '.sdata', '.exe', '.dll', '.dds', '.tga', '.bmp',
                 '.3dc', '.3do', '.smod', '.ani', '.wld', '.svmap', '.wav', '.mp3', '.ogg',
                 '.png', '.jpg', '.jpeg', '.gif', '.webp', '.eft', '.3de', '.mlt',
                 '.sqlite', '.db', '.pem', '.key', '.pcap', '.packetlog', '.zip', '.7z'}
LOCAL_DIRS = {'work', 'build', 'logs', 'runtime', 'data'}
GALLERY_IMAGES = {
    'gallery/2026-11-13-login.png',
    'gallery/2026-11-13-character-selection.png',
    'gallery/2026-11-13-in-game.png',
}
SECRETS = [re.compile(rb'-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----'),
           re.compile(rb'\bgh[pousr]_[A-Za-z0-9]{30,}\b'),
           re.compile(rb'\bgithub_pat_[A-Za-z0-9_]{40,}\b'),
           re.compile(rb'\bAKIA[A-Z0-9]{16}\b')]


def git(*args):
    return subprocess.check_output(['git', *args])


def forbidden(path):
    p = PurePosixPath(path)
    # Explicitly distributed server fixtures and authored app icon. This does
    # not permit personal runtime copies, client archives or build artifacts.
    if path in GALLERY_IMAGES or path in {'server/cmd/gx_server/icon.png', 'internal/audio/testdata/tone.ogg'}:
        return False
    if p.parts[:2] == ('server', 'data'):
        return (p.suffix.lower() in {'.sah', '.saf', '.exe', '.dll', '.pem', '.key', '.env'}
                or p.name.startswith('.') or p.name.endswith(('-wal', '-shm')))
    return (p.suffix.lower() in GAME_SUFFIXES or p.parts[0] in LOCAL_DIRS
            or any(part in {'runtime', '.local', 'build', 'out', 'logs'} for part in p.parts[:-1])
            or p.name == '.env' or p.name.startswith('.env.') or p.name.endswith('.env')
            or p.name in {'shaiya-go.toml', 'runtime.env'})


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--history', action='store_true')
    args = parser.parse_args()
    blobs = {}
    errors = []
    for record in git('ls-files', '--stage', '-z').split(b'\0'):
        if not record:
            continue
        metadata, path = record.split(b'\t', 1)
        path = path.decode()
        mode, oid, stage = metadata.split()
        if stage != b'0':
            errors.append(f'unmerged path: {path}')
        if forbidden(path):
            errors.append(f'forbidden staged path: {path}')
        if mode == b'100644' or mode == b'100755':
            blobs[oid.decode()] = path
    if args.history:
        for line in git('rev-list', '--objects', '--all').splitlines():
            oid, _, path = line.partition(b' ')
            if path:
                path = path.decode()
                if forbidden(path):
                    errors.append(f'forbidden history path: {path}')
                blobs.setdefault(oid.decode(), path)
    reader = subprocess.Popen(['git', 'cat-file', '--batch'], stdin=subprocess.PIPE,
                              stdout=subprocess.PIPE)
    checked = 0
    try:
        for oid, path in blobs.items():
            reader.stdin.write((oid + '\n').encode())
            reader.stdin.flush()
            header = reader.stdout.readline().split()
            if len(header) != 3:
                errors.append(f'unreadable object: {path}')
                continue
            size = int(header[2])
            content = reader.stdout.read(size)
            reader.stdout.read(1)
            if header[1] != b'blob':
                continue
            checked += 1
            if size > (67_108_864 if path.startswith('server/data/') else 8_388_608 if path in GALLERY_IMAGES else 1_048_576):
                errors.append(f'oversized source blob: {path}')
            if any(pattern.search(content) for pattern in SECRETS):
                errors.append(f'potential secret: {path}')
    finally:
        reader.stdin.close()
        reader.wait()
    if errors:
        raise SystemExit('\n'.join(sorted(set(errors))))
    print(f'Publication check passed: {checked} source/history blobs; only approved server fixtures; no client archives, local runtime files or recognizable secrets.')


if __name__ == '__main__':
    main()
