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
| Selección de personaje | `0x46ce30` / `0x46a960`, `0x469870`, `0x4697d0` | **Portada la parte 2D** (`ui.CharSelect`); falta la escena 3D y los modelos |
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

### Selección de personaje (`ui.CharSelect`)

- Panel `character_select.tga` (256x1024) en `(w/16, h/2 - 0x127)`; cadenas
  `character_chain.tga` en (+0x20/+0xd0, -0x136/-0x9b).
- Slots `character_select_button2.tga`: (43, 244 + 36·i), dibujo 256x32, clic 162x25;
  UV normal/hover/presionado/seleccionado (= hover)/deshabilitado. Nombre centrado
  en el área clicable (`0x543eb0`).
- Create (29,440), Delete (29,478), Option (136,440), Exit (136,478):
  columnas 0/2/1/3 de `character_select_button.tga` (512x256), dibujo 128x64, clic 88x32.
- Game Start `character_select_start.tga` (62,527), dibujo 256x64, clic 131x43.
- Cruz `character_navi_bg.tga` en `(w - w/12 - 0x82, h - h/6 - 100)`; botones de 32x32
  (clic 23x23): rotar ← (8,38), rotar → (99,38), pausa (41,39), stop (69,39),
  zoom + (53,7), zoom − (53,76).
- Info: nombre en `(x+0x7e - 3·len, y+0x28)`; `Level` (msg 5065), `Title` (clase:
  msgs 5050.. para Luz, 5047.. para Furia), `Mode` (470..473) y `exit location`
  (mapa: tabla generada de `CPlayerData::GetMapName` `0x4d2e90`) en x+0x1f, y+0x49/0x5b/0x6d/0x7f.
- Cámara de la escena (`0x46a5a0`): ojo (271.097, 1.866, 178.515), mira (275.03, 1.957, …).
- **NO VERIFICADO / pendiente:** confirmación de Delete (diálogo original), Create
  (pantalla de creación), Option, doble clic, slots vacíos (hoy deshabilitados), navegación 3D.
