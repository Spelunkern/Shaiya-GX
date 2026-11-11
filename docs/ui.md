# UI

Cada ventana: función del original que la construye, recursos y estado.

## Framework (pendiente de portar)

- `FUN_005497a0(dir, file, w, h)`: carga de una textura de UI con su tamaño.
- `FUN_00528830(...)`: constructor de botón. Recibe x, y, ancho, alto,
  archivo, tamaño de textura y las UV de 4 estados (normal, hover,
  presionado, deshabilitado). Ejemplo: `ok_button.tga` 128x128, estados
  en franjas de 0.25 en V.
- `FUN_00527000`: inicializador de los controles hijos (7 en la ventana de
  login).

## Ventanas

| Ventana | Constructor | Recursos | Estado |
|---|---|---|---|
| Login | `FUN_004b8ff0` | `login.tga` (256x256), `shaiyalogo01.TGA` (512x256), `ok_button.tga`, `login_motp.tga`, guarda `LOGIN_ID_SAVE` | Ubicado, sin portar |
| Lista de servidores | `FUN_004eb520` | `login_server.tga` | Ubicado, sin portar |
| Selección de personaje | `FUN_0046ce30` | `character_select.tga`, `character_select_button.tga`, `character_select_button2.tga`, `character_select_start.tga` | Ubicado, sin portar |
