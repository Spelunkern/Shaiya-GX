"""Storage checks for publishing the distributable development database."""
from contextlib import closing
import hashlib
import json
from pathlib import Path
import sqlite3
import tempfile
import unittest

from snapshot_sqlite import snapshot


class SnapshotTests(unittest.TestCase):
    def test_snapshot_includes_committed_wal_and_preserves_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "runtime.sqlite"
            destination = root / "data/rapture.sqlite"
            with closing(sqlite3.connect(source)) as live:
                live.execute("PRAGMA journal_mode=WAL")
                live.execute("PRAGMA wal_autocheckpoint=0")
                live.execute("PRAGMA user_version=1")
                live.execute("CREATE TABLE sample(id INTEGER PRIMARY KEY,value TEXT,payload BLOB)")
                live.execute("INSERT INTO sample VALUES(?,?,?)", (2**60 + 1, "área", b"\x00\xff"))
                live.commit()
                self.assertTrue(Path(str(source) + "-wal").stat().st_size > 0)
                original = source.read_bytes()
                self.assertEqual(snapshot(source, destination), {"sample": 1})
                self.assertEqual(source.read_bytes(), original)
                self.assertEqual(live.execute("SELECT COUNT(*) FROM sample").fetchone()[0], 1)
            with closing(sqlite3.connect(destination)) as copy:
                self.assertEqual(copy.execute("SELECT * FROM sample").fetchone(), (2**60 + 1, "área", b"\x00\xff"))
                self.assertEqual(copy.execute("PRAGMA journal_mode").fetchone()[0], "delete")
            metadata = json.loads(destination.with_suffix(".manifest.json").read_text())
            self.assertEqual(metadata["sha256"], hashlib.sha256(destination.read_bytes()).hexdigest())
            self.assertFalse(Path(str(destination) + "-wal").exists())

    def test_failed_snapshot_preserves_previous_distribution(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, destination = root / "runtime.sqlite", root / "seed.sqlite"
            destination.write_bytes(b"previous distribution")
            manifest = destination.with_suffix(".manifest.json")
            manifest.write_bytes(b"previous manifest")
            with closing(sqlite3.connect(source)) as database:
                database.execute("PRAGMA user_version=99")
            with self.assertRaises(ValueError):
                snapshot(source, destination)
            self.assertEqual(destination.read_bytes(), b"previous distribution")
            self.assertEqual(manifest.read_bytes(), b"previous manifest")
            with self.assertRaises(ValueError):
                snapshot(source, source)


if __name__ == "__main__":
    unittest.main()
