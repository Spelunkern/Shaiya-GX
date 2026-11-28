import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from PIL import Image, ImageDraw


TOOLS = Path(__file__).resolve().parent


class PatchOptionArtTest(unittest.TestCase):
    def setUp(self):
        scratch = tempfile.TemporaryDirectory(prefix="gx-option-art-")
        self.addCleanup(scratch.cleanup)
        self.root = Path(scratch.name)
        self.tools = self.root / "fresh checkout" / "tools"
        self.tools.mkdir(parents=True)
        for name in ("option_art.py", "patch_option_graphic.py", "patch_option_sound.py"):
            shutil.copyfile(TOOLS / name, self.tools / name)
        fonts = self.tools.parent / "resources" / "fonts"
        fonts.mkdir(parents=True)
        shutil.copyfile(TOOLS.parent / "resources" / "fonts" / "ui.ttf", fonts / "ui.ttf")

    def make_art(self, path, color):
        path.parent.mkdir(parents=True, exist_ok=True)
        with Image.new("RGBA", (384, 768), color) as image:
            ImageDraw.Draw(image).rectangle((129, 286, 153, 301), fill="black")
            image.save(path)
        return path.read_bytes()

    def patch(self, kind, data, *args):
        result = subprocess.run(
            [sys.executable, str(self.tools / f"patch_option_{kind}.py"), str(data), *map(str, args)],
            cwd=self.root,
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)

    def assert_background(self, path, color):
        with Image.open(path) as image:
            self.assertEqual(image.getpixel((10, 10)), color)

    def test_two_disposable_clients_keep_their_own_art_on_first_run_and_rerun(self):
        clients = {
            self.root / "Client A" / "data": (221, 55, 55, 255),
            self.root / "Client B" / "data": (44, 100, 220, 255),
        }
        for kind in ("graphic", "sound"):
            with self.subTest(patcher=kind):
                first_pass = {}
                for data, color in clients.items():
                    target = data / "interface" / f"option_{kind}.png"
                    original = self.make_art(target, color)
                    self.patch(kind, data)
                    self.assert_background(target, color)
                    backup = data.parent / ".local" / "option-art" / data.name / f"option_{kind}.original.png"
                    self.assertEqual(backup.read_bytes(), original)
                    first_pass[data] = target.read_bytes()
                    self.assertNotEqual(first_pass[data], original)
                for data in reversed(clients):
                    self.patch(kind, data.relative_to(self.root))
                    target = data / "interface" / f"option_{kind}.png"
                    self.assertEqual(target.read_bytes(), first_pass[data])
        self.assertFalse((self.tools.parent / "work").exists())

    def test_explicit_pristine_file_remains_supported(self):
        data = self.root / "Client" / "data"
        for kind in ("graphic", "sound"):
            with self.subTest(patcher=kind):
                target = data / "interface" / f"option_{kind}.png"
                pristine = self.root / "previous backups" / f"option_{kind}.png"
                original_color = (8, 90, 113, 255)
                self.make_art(target, (101, 102, 103, 255))
                original = self.make_art(pristine, original_color)
                self.patch(kind, data, pristine)
                self.assert_background(target, original_color)
                self.assertEqual(pristine.read_bytes(), original)
                self.assertFalse((data.parent / ".local").exists())


if __name__ == "__main__":
    unittest.main()
