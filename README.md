# Shaiya Go Client

Cliente Shaiya EP5.4 en Go, 64 bits, OpenGL 4.5, para jugar contra
Rapture (`~/Desktop/Rapture`). Proyecto privado. Directivas:
[docs/directivas.md](docs/directivas.md).

## Estado (2026-11-11)

**Hito actual: M3 casi cerrado (falta la escena 3D del login y la creación de personaje); comienza M4 (mundo).**

| Área | Estado |
|---|---|
| Red (`netx`, `proto`, `session`) | **Funciona contra Rapture (N2).** RSA 0xA101, AES-CTR, login, lista de servidores, handshake de world, cambio a la tabla XOR tras 0x010B y vuelta a AES tras 0x0107 |
| Cliente headless (`shaiyacli`) | **Funciona:** login → personajes → entrar al mapa → moverse → chat → logout |
| Protocolo | 406 opcodes recibidos mapeados; 27 decodificados; 15 writers verificados en Ghidra. Ver [docs/protocolo.md](docs/protocolo.md) |
| Datos | `data.sah`/`data.saf` (24.655 archivos), TGA/DDS/BMP: 11.867 texturas decodifican (N1) |
| Ventana y render | GLFW + OpenGL 4.5 core, texturas DXT subidas comprimidas, batch 2D |
| UI | **Login, lista de servidores y selección de personaje (2D) fieles**: layout, botones de 5 estados, checkbox, Tahoma 13, mensajes de `sysmsg-uni.txt`, nombres de mapa. Sin escena 3D todavía; el mundo es una pantalla provisoria de texto |
| Cliente con ventana | **Funciona contra Rapture (N2):** login → servidores → personajes → mundo (`-autoplay`) |
| 3D | **Selección de personaje completa**: terreno de campo (capas + máscaras + lightmap), objetos SMOD, cielo con nubes, personaje con skinning en GPU y animación ANI, equipo real vía `Item.SData` (SEED) + MLT |
| Mundo de juego, efectos, VANI/MAni, audio | No empezado |
| Windows | Sin compilar ni probar (se hará más adelante) |

Lo que **no** funciona todavía: escena 3D del login, creación de personaje,
mundo de juego en 3D, combate, inventario y UI de juego.

## Uso

```sh
./tools/build.sh                       # build/shaiya, build/shaiyacli, build/assets
export SHAIYA_DATA="$HOME/Desktop/Client EP5"
build/shaiya                           # cliente con ventana
SHAIYA_PASSWORD=... build/shaiya -autoplay admin1 -screenshot w.png   # flujo completo oculto
SHAIYA_PASSWORD=... build/shaiyacli -user admin1 -slot 0
build/assets stats | ls interface | cat interface/login.tga | find .wld
```

## Pruebas

```sh
go test ./...                                              # N0
SHAIYA_DATA="$HOME/Desktop/Client EP5" go test ./internal/data/...   # N1
SHAIYA_TEST_USER=admin1 SHAIYA_PASSWORD=... go test -tags integration ./cmd/shaiyacli  # N2 (Rapture levantado)
```

## Diferencias con el original

- Ninguna en el protocolo. Ver [docs/discrepancias.md](docs/discrepancias.md).

## Herramientas de análisis

`tools/ghidra/` (proyecto en `work/ghidra`, fuera de git): `source
tools/ghidra/env.sh` y `ghidra_run Script.java args`. Scripts: `DecompAt`,
`Insns`, `FindOperand`, `FindStrings`, `OpcodeWriters`, `Refs`, `Sym`.
`tools/protomap.py` regenera `docs/protocolo.md`.
