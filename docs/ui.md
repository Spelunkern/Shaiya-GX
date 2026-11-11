# UI

Cada ventana: función del original que la construye, recursos, constantes y estado.
Todas las coordenadas están en píxeles de pantalla, como en el cliente D3D9.

## Framework

- **Widget base** (`FUN_00527000`, ~0x5d8 bytes): origen en `+0x38/+0x3c`, offset
  en `+0x10/+0x14`, tamaño de dibujo en `+0x20/+0x24`, tamaño clicable en
  `+0x28/+0x2c`, tipo en `+0x08` (1 = botón, 3 = checkbox). UV de 5 estados
  en arrays: `u0 @+0x48`, `u1 @+0x5c`, `v0 @+0x70`, `v1 @+0x84` (5 floats cada uno).
  Estados: normal, hover, presionado, marcado, deshabilitado.
- **Botón** `FUN_00528830(origen, offset, dibujo 128x32, clic 65x25, 0, archivo,
  128, 128, 1, uv..., tooltip)`. Hoja de 4 filas: normal 0–.25, hover .25–.5,
  presionado y marcado .5–.75, deshabilitado .75–1. Port: `ui.Button`.
- **Checkbox** `FUN_00528700`, tipo 3: solo los estados presionado y marcado muestran
  la textura (la caja está dibujada en la ventana). Port: `ui.Button{Toggle}`.
- **Textura de ventana**: `FUN_005497a0(dir, archivo, w, h)` carga y
  `FUN_00549a70(x, y)` dibuja.
- **Texto**: `FUN_00543d30(x, y, r, g, b, 0, 0x96, texto)` y
  `FUN_005418c0(fuente, x, y, ARGB, 0, formato, ...)` con la fuente global
  `0x2209408`. Locale 6 (`DAT_00708598`): Tahoma de 13 px y de 12 px (`FUN_00544620`).
  Port: `ui.Font` con `/usr/share/wine/fonts/tahoma.ttf` (`SHAIYA_FONT`).
- **Mensajes**: `FUN_0041d4a0(id)` lee `sysmsg-uni.txt` (UTF-16). Port: `text.SysMsg`.

## Ventanas

| Ventana | Constructor / dibujo | Estado |
|---|---|---|
| Login | `0x4b8ff0` / `0x4ba650` (al final) / lógica `0x4bd680` | **Portada** (`ui.Login`) |
| Lista de servidores | `0x4eb520` / `0x4ea5d0` | **Portada** (`ui.ServerList`) |
| Selección de personaje | `0x46ce30` | Ubicada; hay una pantalla provisoria |
| Escena 3D del login | `0x4bcd60` | Ubicada (ver apariencia.md) |

### Login (`ui.Login`)

- Origen: `(w/2 - 0x6f, h - h/32 - 0x114)`. Logo `shaiyalogo01.TGA` en `(w/2 - 256, h/16)`.
- Offsets (floats `0x709d40..0x709d74`): OK (37,159), Cancel (122,159), Save ID
  (147,102), texto del ID (84,56), contraseña con `*` (84,81).
- Cursor: temporizador `0x8655c0`; se dibuja `"%s_"` mientras pasa de 0.5 s y se
  reinicia en 1.0 s. Foco en `+0x23c` (0 = ID, 1 = contraseña).
- **NO VERIFICADO:** largo máximo de los campos (18/16), Enter en el ID pasa a la
  contraseña, clic en los campos, texto y posición de los mensajes de error.

### Lista de servidores (`ui.ServerList`)

- Origen: `(w/2 - 0x9c, h - h/20 - 0x1c2)`, panel `login_server.tga` (512x512).
- OK (67,397), Cancel (180,397), scrollbar `scrollbar.tga` en (281,65), 13x312.
- Filas cada 0x16 px, 13 visibles; resaltado `0x7aa0a0a0` en (31,78) de 240x19;
  texto `"%d.%s"` en (35,82) (`0x70a1c4 + 4`, inicializador estático `0x6b6700`);
  estado en x = 225.
- Estado: Lock / Closed (`0xfff3a0a0`); usuarios > 0.85·cap = "Full" (`0xfff3a0a0`),
  > 0.45 = msg 10080 (`0xfff3dea0`), > 0.3 = msg 10081 (`0xffa8f3a0`), si no msg 10082
  (`0xffa0cff3`).
- **NO VERIFICADO:** selección por clic y doble clic, scrollbar sin portar.
