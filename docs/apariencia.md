# Apariencia

Cada pase y shader: estado original (D3D9) → equivalente GL 4.5, fuente y estado.

| Pase | Original | GL 4.5 | Estado |
|---|---|---|---|
| UI 2D | Quads en píxeles, `SRCALPHA/INVSRCALPHA` | `render.Sprites`, GLSL 450 | Base hecha; filtrado y redondeo de píxel sin verificar |
| Escena de login | `FUN_004bcd60`: `login.wld`/`login2.wld`/`login3.wld`, `login_sky.dds`, rayos `login_lightning1-4.dds`, SMOD `Login_mainstructure`, `Login_floatrock`, `login_effect_*`, `Logineffect_smoke01a`, nubes `Login_cloud.Seff` | — | Ubicado |
| Terreno, objetos, personajes, agua, cielo, efectos | — | — | No empezado |

Texturas: DXT1/3/5 se suben comprimidas (`COMPRESSED_RGBA_S3TC_*`); el
contenido decide el formato (hay BMP y TGA con extensión `.dds`).
