#!/usr/bin/env python3
"""Build self-contained server archives with the versioned database and game data."""
import argparse
from contextlib import closing
import hashlib
import json
import os
import platform as host_platform
import sys
from pathlib import Path
import shutil
import socket
import sqlite3
import subprocess
import tarfile
import tempfile
import time
import zipfile

ROOT = Path(__file__).resolve().parents[1]
PROGRAMS = ("gx_login", "gx_world", "gx_console")


def checksum(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def check_seed(root):
    seed = root / "data/rapture.sqlite"
    manifest = json.loads(seed.with_suffix(".manifest.json").read_text())
    if checksum(seed) != manifest["sha256"]:
        raise ValueError("SQLite seed and manifest disagree; run tools/snapshot_sqlite.py")
    with closing(sqlite3.connect(seed.as_uri() + "?mode=ro", uri=True)) as database:
        if database.execute("PRAGMA integrity_check").fetchall() != [("ok",)]:
            raise ValueError("SQLite seed integrity check failed")
        for table, expected in manifest["tables"].items():
            name = '"' + table.replace('"', '""') + '"'
            if database.execute("SELECT COUNT(*) FROM " + name).fetchone()[0] != expected:
                raise ValueError("SQLite seed row-count mismatch: " + table)
    if not list((root / "data/maps").glob("*.svmap")):
        raise ValueError("server maps are missing")
    for name in ("Map.ini", "ItemEnchant.ini", "NpcQuest.SData", "KillStatus.SData", "GuildHouse.ini", "SequenceItemCreate.ini"):
        if not (root / "data/game" / name).is_file():
            raise ValueError("required server data is missing: " + name)


def verify_native(package):
    # The binaries discover this relocated copy from an unrelated working directory.
    with tempfile.TemporaryDirectory(prefix="gx-server-package-check-") as temporary:
        temporary = Path(temporary)
        copied = temporary / "GX Server portable copy"
        shutil.copytree(package, copied)
        environment = {key: value for key, value in os.environ.items() if not key.startswith("RAPTURE_")}
        reservations = []
        for _ in range(2):
            sock = socket.socket()
            sock.bind(("127.0.0.1", 0))
            reservations.append(sock)
        ports = [sock.getsockname()[1] for sock in reservations]
        for sock in reservations:
            sock.close()
        environment.update(RAPTURE_LOGIN_LISTEN=f"127.0.0.1:{ports[0]}", RAPTURE_WORLD_LISTEN=f"127.0.0.1:{ports[1]}")
        before = checksum(copied / "data/rapture.sqlite")
        processes = []
        try:
            suffix = ".exe" if sys.platform == "win32" else ""
            for name in PROGRAMS[:2]:
                log = (temporary / (name + ".log")).open("w")
                process = subprocess.Popen([str(copied / "build" / (name + suffix))], cwd=temporary,
                                           env=environment, stdout=log, stderr=subprocess.STDOUT)
                log.close()
                processes.append(process)
            for port in ports:
                deadline = time.monotonic() + 30
                while True:
                    if any(process.poll() is not None for process in processes):
                        details = "\n".join(path.read_text()[-2000:] for path in temporary.glob("*.log"))
                        raise RuntimeError("portable server exited:\n" + details)
                    with socket.socket() as sock:
                        sock.settimeout(.2)
                        if sock.connect_ex(("127.0.0.1", port)) == 0:
                            break
                    if time.monotonic() > deadline:
                        raise TimeoutError("portable server startup timed out")
                    time.sleep(.05)
            if not (copied / "runtime/rapture.sqlite").is_file():
                raise ValueError("runtime copy was not created inside the package")
            if checksum(copied / "data/rapture.sqlite") != before:
                raise ValueError("server modified its distributed seed")
            with closing(sqlite3.connect(copied / "runtime/rapture.sqlite")) as database:
                if database.execute("PRAGMA integrity_check").fetchone()[0] != "ok":
                    raise ValueError("runtime integrity check failed")
            print(f"{sys.platform} verified: relocated folder, unrelated cwd, isolated ports, unchanged seed", flush=True)
        finally:
            for process in reversed(processes):
                if process.poll() is None:
                    process.terminate()
                    try:
                        process.wait(timeout=10)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait()
        verify_controller(copied, temporary, environment, ports)


def verify_controller(package, temporary, environment, ports):
    """Exercise the real console and ensure a second installation survives."""
    suffix = ".exe" if sys.platform == "win32" else ""
    foreign = temporary / "Another GX Server copy"
    shutil.copytree(package, foreign)
    with socket.socket() as reservation:
        reservation.bind(("127.0.0.1", 0))
        other_port = reservation.getsockname()[1]
    other_env = dict(environment, RAPTURE_LOGIN_LISTEN=f"127.0.0.1:{other_port}")
    with (temporary / "foreign.log").open("w") as log:
        other = subprocess.Popen([str(foreign / "build" / ("gx_login" + suffix))],
                                 cwd=temporary, env=other_env, stdout=log, stderr=subprocess.STDOUT)
    console = None

    def listening(port):
        with socket.socket() as sock:
            sock.settimeout(.2)
            return sock.connect_ex(("127.0.0.1", port)) == 0

    try:
        deadline = time.monotonic() + 30
        while not listening(other_port):
            if other.poll() is not None or time.monotonic() > deadline:
                raise RuntimeError("second installation did not start")
            time.sleep(.1)
        with (temporary / "console.log").open("w") as log:
            console = subprocess.Popen([str(package / "build" / ("gx_console" + suffix))],
                                       cwd=temporary, env=environment, stdin=subprocess.PIPE,
                                       stdout=log, stderr=subprocess.STDOUT, text=True)
        console.stdin.write("start server\n")
        console.stdin.flush()
        deadline = time.monotonic() + 60
        while not all(listening(port) for port in ports):
            if console.poll() is not None or time.monotonic() > deadline:
                raise RuntimeError("console failed to start login/world")
            time.sleep(.1)
        console.stdin.write("shutdown server\n")
        console.stdin.close()
        console.wait(timeout=60)
        if console.returncode != 0 or any(listening(port) for port in ports):
            raise RuntimeError("console failed to stop its own installation")
        if other.poll() is not None or not listening(other_port):
            raise RuntimeError("controller stopped another portable installation")
        print("Console verified: start/stop and second-installation isolation", flush=True)
    finally:
        if console is not None and console.poll() is None:
            # Cleanup is scoped to this copied package, never by image name.
            if console.stdin and not console.stdin.closed:
                console.stdin.write("shutdown server\n")
                console.stdin.close()
            try:
                console.wait(timeout=60)
            except subprocess.TimeoutExpired:
                console.kill()
                console.wait()
        if other.poll() is None:
            other.terminate()
            other.wait(timeout=10)


def build_package(platform, output, go):
    system, architecture = platform.split("-", 1)
    suffix = ".exe" if system == "windows" else ""
    name = "gx-server-" + platform
    with tempfile.TemporaryDirectory(prefix=".package-", dir=output) as temporary:
        root = Path(temporary) / name
        (root / "build").mkdir(parents=True)
        environment = dict(os.environ, CGO_ENABLED="0", GOOS=system, GOARCH=architecture, GOTOOLCHAIN="local")
        for program in PROGRAMS:
            subprocess.run([go, "build", "-trimpath", "-ldflags=-s -w", "-o", str(root / "build" / (program + suffix)),
                            "./cmd/" + program], cwd=ROOT, env=environment, check=True)
        gui_environment = dict(environment, CGO_ENABLED="1")
        if system == "windows" and sys.platform != "win32":
            gui_environment["CC"] = os.environ.get("GX_WINDOWS_CC", "x86_64-w64-mingw32-gcc")
        flags = "-s -w -H windowsgui" if system == "windows" else "-s -w"
        subprocess.run([go, "build", "-trimpath", "-ldflags=" + flags, "-o",
                        str(root / "build" / ("gx_server" + suffix)), "./cmd/gx_server"],
                       cwd=ROOT, env=gui_environment, check=True)
        shutil.copytree(ROOT / "data", root / "data", ignore=shutil.ignore_patterns("vssver.scc", "__pycache__", "*.sqlite-wal", "*.sqlite-shm"))
        shutil.copy2(ROOT / "LICENSE", root / "LICENSE")
        (root / "docs").mkdir()
        for document in ("sqlite.md", "distribution.md", "social.md"):
            text = (ROOT / "docs" / document).read_text().replace("../../docs/", "")
            (root / "docs" / document).write_text(text)
        for document in ("server.md", "quickstart.md", "building.md", "gm.md"):
            text = (ROOT.parent / "docs" / document).read_text().replace("../server/docs/", "")
            (root / "docs" / document).write_text(text)
        (root / "tools").mkdir()
        for tool in ("sqlite.py", "snapshot_sqlite.py", "enable_test_gms.sql"):
            shutil.copy2(ROOT / "tools" / tool, root / "tools" / tool)
        if system == "windows":
            (root / "Start GX Server.bat").write_bytes(b'@echo off\r\ncd /d "%~dp0"\r\n"%~dp0build\\gx_console.exe"\r\n')
        else:
            launcher = root / "Start GX Server.sh"
            launcher.write_text('#!/bin/sh\nset -eu\ncd "$(dirname "$0")"\nexec ./build/gx_console\n')
            launcher.chmod(0o755)
        (root / "README.txt").write_text("""GX Server — the integrated Shaiya GX server

Open build/gx_server (Linux) or build/gx_server.exe (Windows) for the
one-button GX Server app. Start/Stop works as before. Closing the app
leaves servers running. The console launcher below is also included.

SQLite is embedded for development convenience. A future database backend
will replace it; SQLite is not the final production storage design.

Run Start GX Server.sh (Linux) or Start GX Server.bat (Windows).
Type: start server
To stop: shutdown server
Closing the console leaves the server processes running.

GM commands and Admin 0/1 account roles: see docs/gm.md.
Example: /nmake 1 1 creates merchant 1. Test accounts in the seed are GMs.

No Go, Docker, SQL Server, Python or database installation is needed to run
this package. Python 3 is optional for the tools in tools/.

Keep build/ and data/ together. First startup copies the included database
to runtime/rapture.sqlite. Later starts preserve your working database.
The included snapshot retains the current development accounts and characters.
To create another account: python3 tools/sqlite.py account NAME

Default endpoints are 127.0.0.1:30800 (login) and 127.0.0.1:30810 (world).
Set RAPTURE_LOGIN_LISTEN, RAPTURE_WORLD_LISTEN and RAPTURE_PUBLIC_IP to use
other addresses. See docs/distribution.md and docs/sqlite.md for details.
""")
        check_seed(root)
        files = {str(path.relative_to(root)).replace(os.sep, "/"): checksum(path)
                 for path in sorted(root.rglob("*")) if path.is_file()}
        (root / "package.manifest.json").write_text(json.dumps({"platform": platform, "files": files}, indent=2) + "\n")
        if (system == "linux" and sys.platform.startswith("linux") or system == "windows" and sys.platform == "win32") and host_platform.machine().lower() in ("x86_64", "amd64"):
            verify_native(root)
        if system == "windows":
            staged = Path(temporary) / (name + ".zip")
            with zipfile.ZipFile(staged, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
                for path in sorted(root.rglob("*")):
                    if path.is_file():
                        archive.write(path, name + "/" + path.relative_to(root).as_posix())
        else:
            staged = Path(temporary) / (name + ".tar.gz")
            with tarfile.open(staged, "w:gz", compresslevel=9) as archive:
                archive.add(root, arcname=name)
        destination = output / staged.name
        os.replace(staged, destination)
        print("Ready:", destination, flush=True)
        return destination


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--platform", action="append", choices=("linux-amd64", "windows-amd64"))
    parser.add_argument("--output", type=Path, default=ROOT / "out/portable")
    args = parser.parse_args()
    go = shutil.which("go")
    if go is None:
        candidates = sorted((Path.home() / ".local/opt").glob("go-*/bin/go"))
        go = str(candidates[-1]) if candidates else None
    if go is None:
        parser.error("Go 1.27 or later is needed to build the packages")
    check_seed(ROOT)
    output = args.output.expanduser().resolve()
    output.mkdir(parents=True, exist_ok=True)
    archives = [build_package(platform, output, go) for platform in (args.platform or ("linux-amd64", "windows-amd64"))]
    staged = output / ".SHA256SUMS.next"
    staged.write_text("".join(checksum(path) + "  " + path.name + "\n" for path in archives))
    os.replace(staged, output / "SHA256SUMS.txt")


if __name__ == "__main__":
    main()
