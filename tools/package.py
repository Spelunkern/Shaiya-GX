#!/usr/bin/env python3
"""Build the private Linux play folder with independent copies of local assets."""
import argparse
import hashlib
import os
import re
from pathlib import Path
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, default=Path.home()/"Desktop/Shaiya-Client-Decomp")
    parser.add_argument("--output", type=Path, default=Path.home()/"Desktop/Go project")
    parser.add_argument("--update", action="store_true", help="publish a new executable/docs atomically, keeping the installed assets")
    args = parser.parse_args()
    source, output = args.source.expanduser().resolve(), args.output.expanduser().resolve()
    if source == output:
        parser.error("source and output must differ")
    asset_root = output if args.update else source
    for name in ("data.sah", "data.saf"):
        if not (asset_root/name).is_file():
            parser.error(f"missing {asset_root/name}")
    output.mkdir(parents=True, exist_ok=True)
    for name in ("tools", "fonts", "docs", "work"):
        (output/name).mkdir(exist_ok=True)
    env = dict(os.environ, PATH=str(Path.home()/".local/opt/go-1.27.3/bin")+":"+os.environ.get("PATH", ""),
               GOTOOLCHAIN="local", GOPATH=os.environ.get("GOPATH", str(Path.home()/".local/share/go")))
    staged = output/".shaiya.next"
    subprocess.run(["nice", "go", "build", "-o", str(staged), "./cmd/shaiya"], cwd=ROOT, env=env, check=True)
    # Existing Linux processes retain their original executable inode. Never
    # truncate a binary that the user's currently running window may be using.
    staged.replace(output/"shaiya")
    if not args.update:
        for name in ("data.sah", "data.saf"):
            print(f"Copiando {name}…", flush=True)
            subprocess.run(["cp", "--reflink=auto", "--preserve=mode,timestamps", "--", str(source/name), str(output/name)], check=True)
        for file in source.glob("*.txt"):
            name = "LEEME-client-decomp.txt" if file.name.lower() == "leeme.txt" else file.name
            shutil.copy2(file, output/name)
    shutil.copy2(os.environ.get("SHAIYA_FONT", "/usr/share/wine/fonts/tahoma.ttf"), output/"fonts/tahoma.ttf")
    shutil.copy2(ROOT/"tools/play.py", output/"tools/play.py")
    shutil.copy2(ROOT/"internal/ui/fonts/OFL.txt", output/"fonts/OpenSans-OFL.txt")
    progress = ROOT/"docs/continuar.md"
    if progress.exists():
        with (output/"shaiya").open("rb") as binary:
            build_hash = hashlib.file_digest(binary, "sha256").hexdigest()
        progress.write_text(re.sub(r"(?<=Ejecutable SHA-256:\n  `)[0-9a-f]{64}", build_hash, progress.read_text()))
    for file in (ROOT/"docs").glob("*.md"):
        shutil.copy2(file, output/"docs"/file.name)
    progress = ROOT/"docs/continuar.md"
    if progress.exists():
        shutil.copy2(progress, output/"PROGRESO.md")
    (output/"Jugar.sh").write_text('''#!/bin/sh
set -eu
bundle_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec python3 "$bundle_dir/tools/play.py" --no-build --client "$bundle_dir/shaiya" --data "$bundle_dir" -- "$@"
''')
    (output/"Jugar.sh").chmod(0o755)
    (output/"Jugar.desktop").write_text(f'''[Desktop Entry]
Type=Application
Name=Shaiya Go
Exec="{output}/Jugar.sh"
Path={output}
Terminal=false
Categories=Game;
''')
    (output/"Jugar.desktop").chmod(0o755)
    (output/"LEEME.txt").write_text('''SHAIYA GO — PRIMERA BUILD DE PRUEBA (Linux 64 bits)

Abrir Jugar.sh o Jugar.desktop. Ingresar la cuenta local en el cliente.
El lanzador inicia Rapture de ~/Desktop/Rapture y su SQL configurado.
Al cerrar la ventana, guarda la salida normal y apaga los servicios que inició.
Si Rapture ya está levantado, ejecutar ./shaiya directamente.

Controles: W/S avanzar/retroceder, A/D girar, Q/E lateral; botón derecho dirige.
Botón derecho+A/D: lateral; ambos botones: avanzar. Rueda: zoom; flechas: cámara.
M mapa completo; rueda para zoom y arrastre para desplazarlo. Escape cierra el mapa.
Clic/Tab target; doble clic o F atacar; X detener; K skills; 1..0 usar barra.
I inventario/equipo; T Status/atributos; N NPC; V vender seleccionado; G loot.
Arrastrar el título mueve inventario, skills, Status y tienda; su posición se guarda.
Enter chat; flechas arriba/abajo: historial escrito; selector junto a la entrada: canal.
O/Escape: opciones; Escape primero cierra paneles. Regreso/salida desde el menú.
R revivir estando muerto. Matriz de UI completa y pendientes: docs/reconstruccion-ui.md.
Grupo: /invite nombre, /leave, /kick nombre y /leader nombre; /party texto.

data.sah, data.saf y textos: copia de Desktop/Shaiya-Client-Decomp.
La fuente Tahoma va en fonts/. No requiere instalar Go ni Wine para ejecutarse.
Usa las bibliotecas gráficas y de audio de Linux: OpenGL 4.5 y controladores.
Las dependencias de esta máquina están en dependencias.txt.
El backend Rapture/SQL es externo al cliente; el lanzador usa su instalación local.

PROGRESO.md y docs/ contienen estado, verificación, fuentes y pendientes.
Agua WTR animada y sombras de actores en campos; pases avanzados y sistemas restantes en desarrollo.
Login al ritmo de Phoenix, captura del retrato HUD mejorada y tamaño de ventana guardado.
Buffs: iconos/tiempo restante y clic derecho para solicitar cancelación al servidor.
Métricas fijas arriba al centro: FPS, ms, CPU/GPU, RAM y VRAM estimada.
Sus valores usan verde/amarillo/naranja/rojo según presupuesto de cuadro o memoria.
Niebla RGB/min/max del WLD. Muestreo de texturas restaurado al anterior.
Las actualizaciones de build se publican sin interrumpir la ventana que ya está abierta.
El render descarta terreno/objetos fuera de cámara y agrupa objetos repetidos en GPU.
Volver a abrir Jugar.sh ejecuta la última build publicada.
''')
    result = subprocess.run(["ldd", str(output/"shaiya")], capture_output=True, text=True, check=True)
    if "not found" in result.stdout:
        raise RuntimeError(result.stdout)
    (output/"dependencias.txt").write_text(result.stdout)
    files = [p for p in output.rglob("*") if p.is_file() and "work" not in p.relative_to(output).parts and p.name != "SHA256SUMS.txt"]
    previous = {}
    checksum_file = output/"SHA256SUMS.txt"
    if args.update and checksum_file.exists():
        previous = {name: digest for digest, name in (line.split("  ", 1) for line in checksum_file.read_text().splitlines())}
    rows = []
    for file in sorted(files):
        name = str(file.relative_to(output))
        if args.update and name in ("data.sah", "data.saf") and name in previous:
            digest = previous[name] # Installed assets were not written by this update.
        else:
            with file.open("rb") as stream:
                digest = hashlib.file_digest(stream, "sha256").hexdigest()
        rows.append(f"{digest}  {file.relative_to(output)}")
    (output/"SHA256SUMS.txt").write_text("\n".join(rows)+"\n")
    print(f"Build preparada en {output}", flush=True)


if __name__ == "__main__":
    main()
