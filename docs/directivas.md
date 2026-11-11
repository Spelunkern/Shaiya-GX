# Shaiya Go Client: directivas

Versión 3, del 2026-11-11. Repo: `~/src/shaiya-go-client`, **privado**.
La documentación se escribe en español.

## 0. Objetivo y alcance

- **Objetivo:** un cliente nuevo en Go, de 64 bits y nativo en Windows y
  Linux, que se conecte a **Rapture** y llegue **rápido a producción**: que
  se pueda jugar de verdad, se vea como Shaiya y se use como Shaiya.
- **Qué no es:**
  - no es una reconstrucción religiosa;
  - no es un port del decomp;
  - no busca fidelidad binaria.
  Lo crítico se conserva tal cual. Lo demás se resuelve de la forma más
  directa y mantenible.
- **Qué es crítico y se conserva fiel:**
  - el protocolo: los bytes en el cable;
  - la lectura de los datos originales;
  - la **apariencia**: render, **shaders**, iluminación, niebla, mezclas y
    efectos;
  - la **UI**: el arte original, la posición, el tamaño y el comportamiento
    de las ventanas;
  - las **funcionalidades**: lo que el jugador puede hacer y cómo responde el
    juego.
- **Cliente de referencia:** `~/Desktop/Client EP5/Game.exe`, EP5.4, SHA-256
  `8e5ef14a…3263`. Es el binario que reconstruye `shaiya-client-decomp` y
  contra el que se alineó Rapture.
- **Datos:** los `data.sah` y `data.saf` originales del usuario. Se leen
  directamente y nunca entran en git.
- **Rapture manda en las reglas.** El cliente no calcula resultados de juego.
  Solo predice lo visual: movimiento y animación.
- **Es un proyecto privado.** No hay distribución pública, contribuciones de
  terceros ni releases abiertos.

## 1. Fuentes y su rol

| Fuente | Ruta | Para qué sirve | Para qué NO |
|---|---|---|---|
| shaiya-client-decomp | `~/src/shaiya-client-decomp` | Mapa de la estructura: subsistemas, readers y writers de paquetes, layouts, flujo de pantallas, **layout de UI**, estados de render y shaders (`graphics/native_render_shaders.h`, pases), orden de inicialización | Copiar código línea por línea. Se entiende y se reescribe en Go |
| Ghidra | `~/.cache/phoenix-reverse/ghidra/ghidra_12.1.3_PUBLIC` | Confirmar lo que el decomp tiene en asm: paquetes, structs, estados D3D9 (render states, texture stages), coordenadas de UI | Reemplazar al decomp donde ya hay C++ legible |
| Rapture | `~/Desktop/Rapture` | Contraparte real: crypto, framing, 193 opcodes con shapes, cliente sintético de sus tests | Importarlo (sus paquetes son `internal/`) o modificarlo desde este proyecto |
| Parsec | `~/.local/share/parsec-ep5/Parsec` (MIT) | Especificación de formatos: SAH/SAF, SData + cifrado, WLD, 3DC, 3DO, SMOD, ANI, EFT, svmap… | — |
| Phoenix (Godot) | `~/Desktop/Phoenix` | Experiencia previa con assets de Shaiya: terreno, EFT/GEFFECT, streaming por celdas, UI EP4/EP5 | Arquitectura: es un cliente de AzerothCore |

`~/Projects/game-native-rebuild` **no se usa.** Es ajeno a este proyecto:
no se lee ni se porta nada de ahí.

**Prioridad cuando dos fuentes discrepan:**

- **Protocolo:**
  1. el reader o writer del `Game.exe` EP5.4 (decomp o Ghidra);
  2. Rapture.
- **Apariencia y UI:**
  1. el comportamiento del `Game.exe` original, observado bajo wine y
     confirmado en el código;
  2. el decomp, que en su versión D3D11 tiene diferencias visuales
     declaradas.

Las discrepancias se anotan en `docs/discrepancias.md`. No se arregla
Rapture desde acá.

## 2. Arquitectura

### 2.1 Stack

- **Go 1.27.3:** el local de `~/.local/opt/go-1.27.3`, con
  `GOTOOLCHAIN=local`. Targets: `linux/amd64` y `windows/amd64`.
- **Ventana e input:** GLFW 3.4 (`go-gl/glfw`).
- **Gráficos:** **OpenGL 4.5 core** (`go-gl/gl/v4.5-core`). Se usa DSA
  (`glCreate*` / `glNamed*`) y `GL_KHR_debug` en debug. El shader se escribe
  en GLSL 450 y hay un renderer por pase, sin capa de abstracción
  multi-API: se apunta a GL 4.5 y punto.
- **Audio:** `ebitengine/oto/v3` con decodificador WAV propio.
- **Texto:** las fuentes que usa el original, rasterizadas a un atlas
  (`golang.org/x/image/font`). El tamaño, el contorno y el color tienen que
  coincidir con los de la UI original.
- **cgo, sin Zig:**
  - Linux: con `gcc` del sistema.
  - Windows: cross-compilado con **MinGW-w64**:
    `CC=x86_64-w64-mingw32-gcc GOOS=windows CGO_ENABLED=1`. Ya está
    instalado: `mingw-w64-gcc` 16.2.
  - **Por ahora solo se prueba en Linux.** Windows se compila y se prueba
    más adelante, cuando el usuario lo pida.
- **Prohibido:** motores completos (Godot, Unity, raylib) y dependencias sin
  mantenimiento.

### 2.2 Estructura del repo

```
cmd/
  shaiya/          cliente
  shaiyacli/       cliente headless (bot/probe) con la misma lógica de juego
  assets/          inspección/extracción de data.saf
internal/
  netx/            framing, AES-CTR LE, RSA login, ExpandedXOR, salto login→world
  proto/           opcodes + un codec por paquete
  session/         Login → ServerList → World → CharSelect → InWorld
  data/            archive (SAH/SAF), sdata, image (DDS/TGA/BMP), world (WLD/svmap/DG),
                   model (3DC/3DO/SMOD/ANI), effect (EFT/3DE/VANI/MAni/WTR), ini
  game/            estado del juego: entidades, jugador, inventario, chat, target (sin GL ni sockets)
  render/          GL 4.5: device, pases (terreno, objetos, personajes, agua, cielo, efectos, UI), shaders/
  ui/              ventanas de la UI original, layouts recuperados, input
  audio/
  platform/        ventana, input, rutas, logs, config
docs/
  directivas.md    este archivo (a partir de ahora la copia del repo es la que manda)
  protocolo.md     opcodes: dirección, layout, fuente, estado
  apariencia.md    cada pase y shader: estado original (D3D9) → equivalente GL, estado
  ui.md            cada ventana: origen del layout, estado
  discrepancias.md
tools/ghidra/      scripts headless
```

### 2.3 Reglas de arquitectura

1. **Un solo hilo dueño del estado.** El loop principal, con
   `runtime.LockOSThread`, muta `game` y dibuja. La red corre en goroutines y
   entrega paquetes decodificados por un canal.
2. **`game` no depende de `render`, `netx` ni `platform`.**
3. **Un codec por paquete,** con test de ida y vuelta y tabla de tamaños
   fijos. Los paquetes desconocidos se loguean y se ignoran.
4. **Assets perezosos y con caché acotada.** El mundo se carga por celdas.
5. **Configuración propia,** con el host de login configurable y
   `127.0.0.1:30800` por defecto.

## 3. Fidelidad

| Área | Exigencia |
|---|---|
| **Cable** | **Exacta:** ancho, signo, orden, padding, strings fijos, versión de cliente |
| **Datos** | **Exacta** en la interpretación. Se pueden ignorar campos, pero nunca leerlos mal |
| **Apariencia** | **Fiel al original:** mismas texturas, mismas mezclas y alpha test, niebla, iluminación (luz de mapa, lightmaps), animaciones, efectos y sombras. Los estados fijos D3D9 se traducen a GLSL equivalente. Se aceptan diferencias solo por resolución, filtrado o precisión, y quedan anotadas en `apariencia.md` |
| **UI** | **Fiel:** arte original, posiciones, tamaños, fuentes, tooltips, arrastre, atajos de teclado y comportamiento de cada ventana |
| **Funcionalidad** | **Fiel:** la misma acción del jugador produce las mismas peticiones y la misma respuesta visible |
| **Implementación interna** | **Libre:** Go idiomático, estructuras propias, sin copiar la arquitectura del original |

Reglas:

1. **Fuente citada** en cada paquete, shader y ventana. Puede ser una
   dirección del `Game.exe` o un archivo del decomp. Lo que no tiene fuente
   queda marcado `NO VERIFICADO`.
2. **No inventar** campos, reglas ni elementos de UI. Los bytes que no se
   entienden se llaman `unk_XX` y se conservan.
3. **Los números que la UI necesita y el servidor no manda** salen del SData
   original.
4. **Los structs de cable no son structs de memoria.** El original es de 32
   bits y este de 64. Se serializa campo por campo, sin `unsafe` ni
   `binary.Read` sobre structs.
5. **La resolución libre está permitida.** La UI respeta el layout original
   anclado a los bordes como lo hace el cliente. El escalado de UI es una
   opción, no el modo por defecto.

## 4. Código Go

1. `gofmt`, `go vet` y `staticcheck` limpios.
2. **Errores:** nunca se descartan en silencio. Los datos de red o de archivo
   corruptos no pueden tirar el cliente; `panic` solo por un invariante
   interno.
3. **Parsers defensivos:** largos validados antes de reservar, límites duros
   y fuzz tests en codecs y formatos.
4. **`unsafe` solo en `render`,** para pasar buffers a GL.
5. **Comentarios de código en inglés. `docs/` y las respuestas al usuario,
   en español.**
6. **Pragmatismo:** si una solución directa da el mismo resultado visible y
   el mismo cable, se elige la directa. No se recrean sistemas internos del
   original sin un efecto observable.

## 5. Ghidra y herramientas

1. **Proyecto Ghidra propio** para `Client EP5/Game.exe`, en
   `~/src/shaiya-go-client/work/ghidra/` (fuera de git). Se importa una vez y
   después se usa `-readOnly`.
   `JAVA_HOME=~/.cache/phoenix-reverse/ghidra/jdk-21.0.12.1+1`.
2. **Nombres del decomp** importados como etiquetas con
   `tools/ghidra/ImportDecompNames`.
3. **Scripts headless:**
   - `DecompAt`;
   - `ListPacketHandlers`: opcode → función;
   - `DumpStruct`;
   - `DumpRenderStates`: secuencias `SetRenderState`, `SetTextureStageState`
     y shaders por pase;
   - `DumpUILayout`: coordenadas y recursos de cada ventana.
4. **Entregables de análisis, en orden:**
   1. el mapa de opcodes;
   2. el mapa de pases de render y shaders;
   3. el mapa de ventanas de UI.
5. **Referencia visual:** se toman capturas del `Game.exe` original bajo wine
   contra Rapture, en las mismas escenas y posiciones de cámara, para
   comparar a ojo y con diff de imagen.
6. **`--packet-log`** del propio cliente: vuelca el tráfico descifrado, con
   la contraseña enmascarada.

## 6. Verificación

Las pruebas son livianas porque el objetivo es producción. Se exige lo que
detecta errores reales.

| Nivel | Qué prueba | Cuándo es obligatorio |
|---|---|---|
| **N0** | Build Linux, vet, unit tests (Windows: más adelante) | Siempre |
| **N1** | Codecs con fixtures; parsers contra el `data.saf` real | Paquete o formato nuevo |
| **N2** | Integración con Rapture vivo (`-tags integration`) | Cambios en red o sesión |
| **N3** | Comparación con el original: paquetes recibidos por Rapture (funcionalidad) y capturas lado a lado (apariencia y UI) | Cierre de cada hito |
| **N4** | Prueba del usuario en juego | Cierre de cada hito |

Rapture se levanta solo para probar y se apaga después
(`docker stop --time 60`). `eternity-sqlserver` no se toca.

## 7. Hitos (camino a producción)

| Hito | Contenido | Cierre |
|---|---|---|
| **M0 Base** | Repo, build Linux, ventana GL 4.5 con contexto debug, Ghidra importado, mapa de opcodes v1 | N0 en Linux |
| **M1 Red** | `netx`, `proto` y `session`; `shaiyacli` juega el flujo completo: login → lista de servidores → facción → personajes → mapa → mover → chat → logout | N2 y N3 de cable |
| **M2 Datos + UI base** | SAH/SAF, DDS/TGA, SData, fuentes; sistema de ventanas de la UI original | N1 |
| **M3 Cuenta** | Pantallas de login, servidor, selección y creación, fieles, con personaje 3D (3DC + ANI) y la escena de login | N3 visual y N4 |
| **M4 Mundo** | Terreno, objetos, cielo, agua, niebla, luz, entidades con modelo, nombre y animación; movimiento | N3 visual y N4 |
| **M5 Juego** | Target, ataque, skills, barras, exp, inventario y equipo, loot, muerte y revivir, chat, NPC y tienda, efectos de skills, audio | N3 y N4 → **primera versión de producción** |
| **M6 Sistemas** | Party/raid, quests, comercio, banco, guilds, enchant/gemas, mall, PvP/duelos y lo que Rapture ya soporta | Uno por uno |

Se trabaja en paralelo donde no hay dependencia: M2 puede avanzar mientras se
cierra M1.

## 8. No objetivos

- Fidelidad binaria y arquitectura interna igual a la del original.
- 32 bits y macOS.
- Updater, `CONFIG.exe` (se reemplaza por opciones dentro del juego),
  `sdev.dll`, anti-cheat, `ijl15.dll`, IME coreano.
- Saltar validaciones del servidor.
- Otros servidores aparte de Rapture.

## 9. Seguridad y datos

1. **La contraseña** viaja solo dentro del paquete 0xA102 cifrado. No se
   loguea ni se persiste.
2. **La clave RSA** se toma del 0xA101.
3. **Nunca se commitean:**
   - `data.sah`, `data.saf` ni el `Game.exe`;
   - capturas con credenciales;
   - salidas de Ghidra;
   - `runtime.env`.
   El `.gitignore` va desde el primer commit.
4. **Las rutas a datos** se configuran por variable de entorno o por config.

## 10. Git y flujo

1. Repo **privado** en GitHub. El remoto se crea solo cuando el usuario lo
   pida.
2. Se trabaja solo en `main`, sin ramas sueltas, worktrees ni backups.
3. Commit y push solo a pedido explícito. Commits chicos, con el nivel de
   verificación en el mensaje.
4. `README.md` abre con **Estado**: hito actual, qué funciona y qué no.
5. Los trabajos pesados corren con `nice` y se apagan al terminar.
6. **Los demás proyectos no se tocan.** Decomp, Rapture y Phoenix solo se
   leen.

## 11. Riesgos

| Riesgo | Mitigación |
|---|---|
| Build de Windows (cgo + MinGW) sin probar | Se valida cuando se active Windows |
| Traducir los estados fijos de D3D9 a GLSL sin perder el aspecto | `DumpRenderStates` + capturas comparadas; un shader por combinación real usada, no un emulador genérico |
| GL 4.5 bajo wine o con drivers viejos | Se prueba en M0. El mínimo declarado es GL 4.5; no hay fallback |
| SData cifrado en EP5.4 | Parsec documenta el descifrado; se valida en M2 |
| Protocolo que solo está en asm | Ghidra + mapa de opcodes; se prioriza lo que Rapture implementa |
| El alcance se infla | M5 define la producción; M6 se hace un sistema por vez |
