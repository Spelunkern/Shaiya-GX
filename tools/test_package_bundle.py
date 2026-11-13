import tempfile
from pathlib import Path
import unittest

from package_bundle import merge


class NativeBundleMergeTests(unittest.TestCase):
    def test_windows_ansi_readme_matches_linux_utf8_without_changing_linux_bytes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, destination = root/'windows', root/'shared'
            source.mkdir(); destination.mkdir()
            text = 'GX Server — portable\nGM commands: /nmake 1 1\n'
            (source/'README.txt').write_bytes(text.replace('\n', '\r\n').encode('cp1252'))
            (source/'gx_world.exe').write_bytes(b'PE\0windows')
            (destination/'README.txt').write_bytes(text.encode('utf8'))
            merge(source, destination)
            self.assertEqual((destination/'README.txt').read_bytes(), text.encode('utf8'))
            self.assertEqual((destination/'gx_world.exe').read_bytes(), b'PE\0windows')

    def test_different_binary_content_is_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, destination = root/'a', root/'b'
            source.mkdir(); destination.mkdir()
            (source/'seed.sqlite').write_bytes(b'SQLite\0one')
            (destination/'seed.sqlite').write_bytes(b'SQLite\0two')
            with self.assertRaises(ValueError):
                merge(source, destination)
