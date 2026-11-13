#!/usr/bin/env python3
"""Publish a consistent SQLite snapshot and row-count/checksum manifest."""
import argparse
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import sqlite3
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]


def snapshot(source, destination):
    source = source.expanduser().resolve()
    destination = destination.expanduser().resolve()
    if source == destination:
        raise ValueError("source and destination must be different files")
    if not source.is_file():
        raise FileNotFoundError(source)
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".snapshot-", dir=destination.parent) as temporary:
        staged = Path(temporary) / destination.name
        deadline = time.monotonic() + 60

        def progress(_status, _remaining, _total):
            if time.monotonic() > deadline:
                raise TimeoutError("SQLite snapshot exceeded 60 seconds")

        with closing(sqlite3.connect(source.as_uri() + "?mode=ro", uri=True, timeout=10)) as original:
            with closing(sqlite3.connect(staged)) as target:
                original.backup(target, pages=256, progress=progress, sleep=0.01)
                if target.execute("PRAGMA user_version").fetchone()[0] != 1:
                    raise ValueError("unsupported schema version")
                if target.execute("PRAGMA integrity_check").fetchall() != [("ok",)]:
                    raise ValueError("SQLite integrity check failed")
                target.execute("PRAGMA journal_mode=DELETE")
                tables = {}
                for (table,) in target.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"):
                    identifier = '"' + table.replace('"', '""') + '"'
                    tables[table] = target.execute("SELECT COUNT(*) FROM " + identifier).fetchone()[0]
        # Connections must be closed before publishing a self-contained file.
        manifest = staged.with_suffix(".manifest.json")
        manifest.write_text(json.dumps({"tables": tables, "sha256": hashlib.sha256(staged.read_bytes()).hexdigest()}, indent=2) + "\n")
        for path in (staged, manifest):
            with path.open("rb") as stream:
                os.fsync(stream.fileno())
        os.replace(staged, destination)
        os.replace(manifest, destination.with_suffix(".manifest.json"))
    return tables


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, default=Path(os.environ.get("RAPTURE_DB_PATH", ROOT / "runtime/rapture.sqlite")))
    parser.add_argument("--output", type=Path, default=ROOT / "data/rapture.sqlite")
    args = parser.parse_args()
    tables = snapshot(args.source, args.output)
    print(f"Snapshot ready: {args.output.resolve()} ({len(tables)} tables)")


if __name__ == "__main__":
    main()
