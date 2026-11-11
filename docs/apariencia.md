# Apariencia

Cada pase y shader: estado original (D3D9) → equivalente GL 4.5, fuente y estado.

| Pase | Original | GL 4.5 | Estado |
|---|---|---|---|
| UI 2D | Quads en píxeles, `SRCALPHA/INVSRCALPHA` | `render.Sprites`, GLSL 450 | Base hecha; filtrado y redondeo de píxel sin verificar |
| Escena de login | `FUN_004bcd60`. Este cliente tiene el feature 15 activo (deducido: el decomp calibró la cámara y las rocas solo para `login3`), así que carga `login3.wld` (DUN, `Login_eternity_territory_01.dg`, 11 edificios animados por MAni, 2 árboles, 6 VANI, `world_a1.EFT`) más 6 SMOD a mano: `Login_mainstructure`, `Logineffect_smoke01a`, `login_effect_acrossup`/`acrossdown`, `Login_floatrock`, `login_effect_back01a` | — | Formatos leídos (WLD/DG/SMOD); render pendiente |
| Terreno de campo | Una pasada por capa: RGB del tile (`Terrain/Detail`, repetido cada `TileSize`), alfa de la máscara `<mapa>_00_A<i>.dds`; primera opaca, luego alfa (corte 7.5/255); pasada final × lightmap `_L.dds` (port D3D11 del decomp, `native_terrain_ports.cpp`) | `scene.World`, modos 1 y 2 | Hecho; altura `(muestra-10000)/50` confirmada en `0x5aef30` |
| Objetos SMOD | textura × difusa Gouraud clamp(amb + max(0, N·−L)), amb 128/255, sin culling (Phoenix `account_ep5`) | modo 0 | Hecho; corte de alfa 0.5 NO VERIFICADO |
| Cielo | cúpula `sky.3do` + gradiente del mapa + 2 nubes (`0x54c9b0`, `0x54dd00`) | `scene.Sky` | Hecho; orden y estados NO VERIFICADOS |
| Personajes | 3DC (B por parte) × finales ANI, 2 pesos; partes por MLT según `0x417220`; amb 0.8 | `render.Skinner`, `scene.Character` | Hecho; reflejo esférico (ps_1_1 `0x6c87b8`) pendiente |
| Agua, VANI, MAni, efectos EFT | — | — | No empezado |

Texturas: DXT1/3/5 se suben comprimidas (`COMPRESSED_RGBA_S3TC_*`); el
contenido decide el formato (hay BMP y TGA con extensión `.dds`).

## Cámara y proyección

- Proyección por defecto `0x542400`: FOV `0.785398` (45°), aspecto `w/h`,
  near 0.1, far 400 (`FUN_00408160`: 200/300/400 según distancia de visión).
  La escena de login proyecta sus pasadas con (10, 100000), (0.1, 1e6) y
  (near, far) de configuración (`0x4ba919`, `0x4bb06f`, `0x4bb83c`).
- Selección de personaje (`0x46a5a0`): ojo (271.097, 1.866, 178.515), objetivo
  (275.03, 1.957, 180.196), up (0, 1, 0).
- `D3DXMatrixLookAtLH` / `PerspectiveFovLH`: en GL 4.5 se usará
  `glClipControl(GL_LOWER_LEFT, GL_ZERO_TO_ONE)` para el mismo rango de profundidad.

## Pendientes conocidos del 3D

- Los nombres de textura de DG/SMOD dicen `.tga`, pero los archivos son `.dds`
  en `Entity/Texture`: falta ubicar la regla del original.
- `Login_eternity_territory_01.dg` declara 4 lightmaps sin archivos `_L<n>`; hay un
  `.dg_pv` (~73 bytes por vértice), posiblemente iluminación precalculada. Sin formato todavía.
- Referencia de combinación (decomp D3D11, solo para entender): textura × color de
  vértice × lightmap (1x, clamp) y niebla lineal por profundidad de vista.
- UI 2D: D3D9 ubica los centros de píxel en enteros; el decomp suma 0.5 px.
  Falta aplicarlo en `render.Sprites`.
