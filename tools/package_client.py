#!/usr/bin/env python3
"""Package an already built GX client with notices, without game data."""
import argparse
from pathlib import Path
import shutil
import tarfile
import zipfile

ROOT = Path(__file__).resolve().parents[1]
p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--platform', choices=['linux-amd64', 'windows-amd64'], required=True)
p.add_argument('--output', type=Path, default=ROOT/'out/client')
a = p.parse_args()
output = a.output.resolve()
name = 'gx-client-'+a.platform
stage = output/name
if stage.exists(): shutil.rmtree(stage)
stage.mkdir(parents=True)
exe = 'Shaiya GX.exe' if a.platform.startswith('windows') else 'Shaiya GX'
shutil.copy2(ROOT/'build'/exe, stage/exe)
for file in ['LICENSE', 'NOTICE']:
    shutil.copy2(ROOT/file, stage/file)
licenses = stage/'licenses'
licenses.mkdir()
for file in ['OFL.txt', 'Tahoma-NOTICE.txt', 'LGPL-2.1-or-later.txt']:
    shutil.copy2(ROOT/'internal/ui/fonts'/file, licenses/file)
for file in (ROOT/'licenses').glob('*.txt'):
    shutil.copy2(file, licenses/file.name)
shutil.copy2(ROOT/'docs/client-portable.md', stage/'README.md')
if a.platform.startswith('windows'):
    target = output/(name+'.zip')
    with zipfile.ZipFile(target, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for file in sorted(stage.rglob('*')):
            if file.is_file(): z.write(file, file.relative_to(output))
else:
    target = output/(name+'.tar.gz')
    with tarfile.open(target, 'w:gz') as t: t.add(stage, arcname=name)
print(target)
