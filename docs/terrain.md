# Original terrain surface

GX previously rendered every two-unit FLD height sample, with one fixed cell
diagonal. Valid original files can contain values between the native mesh
vertices that create spikes or pits when rendered this way.

The reference EP5 `Game.exe` SHA-256 is
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`.
Direct execution in an isolated x86 emulator establishes:

- `0x5b3720` queries four triangle planes in a four-sample square. Its vertices
  are the four corners and the centre: an eight-unit square with a four-unit
  centre offset. It does not interpolate every source sample or blur neighbours.
- `0x5b75c0`, reached from `0x5b77b0`, limits ROAM subdivision when both base
  coordinate differences are below eight samples, after splitting that node.
- `0x5b6160` emits the selected near-camera mesh. A 32-unit block produces
  41 shared vertices and 64 triangles, exactly sixteen four-triangle fans.
  Original corner/centre heights survive unchanged; unused intermediate samples
  do not become vertices. An isolated spike fixture confirms this distinction.

GX now uses this **finest native surface** at every distance. Distance-based
ROAM collapse is not implemented by this change. Existing streaming, material
sections, visibility culling and batching remain in use. A 64-unit cell changes
from 1,089 vertices / 2,048 triangles to 145 vertices / 256 triangles: 87.5% fewer
terrain triangles, not an equivalent reduction in total scene cost.

Height queries, terrain drawing, camera/picking queries, water classification,
bounds, occlusion and character-shadow receivers use the same surface. Derived
height storage fills the unused lattice points from the five native anchors;
FLD source samples and all asset files remain unmodified. Unusual non-native
small grids retain the previous fallback. Edge queries clamp safely to the map
rather than reproducing the executable's out-of-buffer corner reads.

`tools/terrain_oracle.py` requires optional `pefile` and `unicorn` analysis
packages. It accepts only the reference hash and runs original height, variance,
near-camera selection and mesh emission. Its only substituted game operation is
heap allocation for nodes; it does not initialize the game or invoke OS,
network or graphics APIs.

```sh
python tools/terrain_oracle.py /path/to/original/Game.exe \
  --data /path/to/loose/data --output-dir work/terrain-native/oracle
```

Validation covered 1,000 synthetic height queries, 24 real outdoor-map crops
(16 native queries each), and exact synthetic mesh topology. Native float-store
rounding differs by less than 0.00024 world units in these height probes. Unit
fixtures retain 160 synthetic native queries and the original emitted synthetic
mesh; no original map data or executable bytes are included.

The complete map-1 sample comparison found a maximum 71.275-unit deviation at
X=1938, Z=1936 (old sample 220.54; native surface 149.265). This is a geometry
comparison, not a claim that every player-visible discontinuity is removed.
Real OpenGL checks cover maps 1, 2 and 19, shared borders, shadow receivers,
streaming, water, occlusion and AA. Local logs and before/after captures are kept
under `work/terrain-native/`.
