#!/usr/bin/env python3
"""Run the local Linux play session, stopping only services started here."""
import argparse
import fcntl
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]
PORTS = (30800, 30810)


def listening(port):
    with socket.socket() as sock:
        sock.settimeout(.2)
        return sock.connect_ex(("127.0.0.1", port)) == 0


def wait_ready(ready, description, children, timeout=90):
    print(description, flush=True)
    deadline = time.monotonic() + timeout
    while not ready():
        for name, process in children:
            if process.poll() is not None:
                raise RuntimeError(f"{name} terminó; revisar work/play-{name}.log")
        if time.monotonic() >= deadline:
            raise RuntimeError(f"Timeout: {description}")
        time.sleep(.25)


def stop_process(process, timeout):
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()


def interrupt(_signum, _frame):
    raise KeyboardInterrupt


def main():
    parser = argparse.ArgumentParser(description="Inicia el cliente y GX Server local; limpia al salir.")
    parser.add_argument("--data", type=Path, default=Path(os.environ.get("SHAIYA_DATA") or Path.home()/"Desktop/Shaiya-Client-Decomp"))
    parser.add_argument("--client", type=Path, default=ROOT/"build/shaiya", help="ejecutable del cliente")
    parser.add_argument("--server", "--rapture", dest="rapture", type=Path, default=Path(os.environ.get("GX_SERVER_DIR") or os.environ.get("RAPTURE_DIR") or ROOT/"server"))
    parser.add_argument("--client-only", action="store_true", help="usar servidores ya levantados, sin administrarlos")
    parser.add_argument("--check", action="store_true", help="comprobar arranque y cierre sin abrir la ventana")
    parser.add_argument("--no-build", action="store_true", help="usar build/shaiya existente")
    parser.add_argument("client_args", nargs=argparse.REMAINDER, help="opciones del cliente después de --")
    args = parser.parse_args()
    args.data = args.data.expanduser().resolve()
    args.rapture = args.rapture.expanduser().resolve()
    args.client = args.client.expanduser().resolve()
    children, client, code = [], None, 1
    signal.signal(signal.SIGTERM, interrupt)
    (ROOT/"work").mkdir(exist_ok=True)
    with (ROOT/"work/play.lock").open("a") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            for name in ("data.sah", "data.saf"):
                if not (args.data/name).is_file():
                    raise RuntimeError(f"Falta {args.data/name}")
            if not args.no_build:
                subprocess.run(["nice", str(ROOT/"tools/build.sh")], cwd=ROOT, check=True)
            if not os.access(args.client, os.X_OK):
                raise RuntimeError(f"Falta {args.client}; ejecutar tools/build.sh")
            if args.client_only:
                if not all(listening(port) for port in PORTS):
                    raise RuntimeError("GX Server no está escuchando en 30800/30810")
            else:
                if any(listening(port) for port in PORTS):
                    raise RuntimeError("Hay servidores escuchando; usar --client-only o build/shaiya")
                for name in ("gx_login", "gx_world"):
                    if not os.access(args.rapture/"build"/name, os.X_OK):
                        raise RuntimeError(f"Falta el binario de GX Server: {name}")
                if not (args.rapture/"data/rapture.sqlite").is_file():
                    raise RuntimeError("Falta la semilla SQLite de GX Server: data/rapture.sqlite")
                env = dict(os.environ, RAPTURE_LOGIN_LISTEN="127.0.0.1:30800",
                           RAPTURE_WORLD_LISTEN="127.0.0.1:30810", RAPTURE_PUBLIC_IP="127.0.0.1")
                for name, port in zip(("gx_login", "gx_world"), PORTS):
                    with (ROOT/f"work/play-{name}.log").open("w") as log:
                        process = subprocess.Popen(["nice", str((args.rapture/"build"/name).resolve())],
                                                   cwd=args.rapture, env=env, stdout=log,
                                                   stderr=subprocess.STDOUT, start_new_session=True)
                    children.append((name, process))
                    wait_ready(lambda: listening(port), f"Esperando {name}…", children)
            if args.check:
                print("GX Server listo; comprobando cierre de la sesión.", flush=True)
                code = 0
            else:
                extra = args.client_args
                if extra[:1] == ["--"]:
                    extra = extra[1:]
                print("Cliente listo. Ingresá tu cuenta en la ventana; al cerrarla finaliza la sesión.", flush=True)
                client = subprocess.Popen(["nice", str(args.client), "-data", str(args.data), *extra],
                                          cwd=ROOT, start_new_session=True)
                code = client.wait()
        except BlockingIOError:
            print("Ya hay una sesión de tools/play.py en curso.", file=sys.stderr)
        except KeyboardInterrupt:
            code = 130
        except (OSError, RuntimeError, subprocess.SubprocessError) as error:
            print(f"No se pudo iniciar la sesión: {error}", file=sys.stderr)
        finally:
            if client is not None:
                stop_process(client, 15)
            for _, process in reversed(children):
                stop_process(process, 5)
    return code


if __name__ == "__main__":
    sys.exit(main())
