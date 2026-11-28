"""Load each client's pristine option art for repeatable patching."""

import shutil

from PIL import Image


def load_option_art(data, name, backup=None):
    data = data.resolve()
    target = data / "interface" / name
    if backup is None:
        backup = data.parent / ".local" / "option-art" / data.name / f"{target.stem}.original.png"
    if not backup.exists():
        backup.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy(target, backup)
    with Image.open(backup) as original:
        return target, original.convert("RGBA")
