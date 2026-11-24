# On-screen performance metrics

The small colored line centered at the top of the client is a development diagnostic. It is visible in the gallery screenshots and measures the running client, not project completion or server performance.

Typical layout:

```text
FPS | frame ms | CPU ms / GPU ms | RAM / VRAM~ MiB | DC
```

| Field | Meaning | Scope |
|---|---|---|
| FPS | Frames per second over the sampling interval | Scene and frame-limit dependent |
| Frame ms | Average elapsed time per frame | Includes frame pacing/waits; not just CPU or GPU work |
| CPU | Measured client update plus draw-submission time, in milliseconds | Not processor utilization, thread-wide profiling or a percentage |
| GPU | OpenGL timestamp duration of the measured draw section, in milliseconds | Asynchronous results can lag; unavailable timing shows `--` |
| RAM | Resident memory of the client process, in MiB | Not total system usage or all virtual/reserved memory |
| VRAM~ | Estimated GPU image/buffer storage tracked by GX, in MiB | Excludes driver overhead and the window-system framebuffer; not exact device-wide VRAM usage |
| DC | Instrumented draw calls from the sampled frame | A rendering workload indicator, not a count of visible objects or triangles |

Displayed values refresh approximately every 250 ms. FPS, frame time and CPU time use the frames in that interval. GPU time averages the timestamp results that became available during the interval and shows `--` if none arrived. Memory and draw-call counters are sampled when the display refreshes. GPU timestamps are read only when available, avoiding a forced GPU wait. CPU and GPU overlap, so adding their times does not give total frame time.

## Colors

Colors are diagnostic heuristics, not pass/fail certification:

| Measurement | Green | Yellow | Orange | Red |
|---|---|---|---|---|
| FPS | 60 or more | 40–59 | 30–39 | Below 30 |
| Frame/CPU/GPU time | Up to 16.67 ms | Above 16.67 to 25 ms | Above 25 to 33.33 ms | Above 33.33 ms |
| RAM/estimated VRAM | Up to 50% of reported capacity | Above 50% to 70% | Above 70% to 85% | Above 85% |
| Draw calls | Up to 2,000 | 2,001–5,000 | 5,001–10,000 | Above 10,000 |

Unavailable or zero timing/capacity values use a neutral color. The memory ratio compares client usage with the reported machine/device capacity; it does not account for other applications' usage.

## Comparing performance

Use the same scene, camera, resolution, graphics settings, frame cap/VSync, data version and hardware/driver. Let loading settle before comparing. A frame cap can keep FPS fixed while CPU/GPU work changes; inspect timings too. Memory depends on cached resources and where the client has already traveled. A single gallery capture is an illustration, not a controlled benchmark or a guarantee for other machines.

Implementation: [HUD sampling and colors](../internal/app/performance.go), [GPU timestamps and counters](../internal/render/metrics.go), [platform memory reporting](../internal/platform). For optimization contributions, include reproducible before/after measurements and preserve original behavior and appearance.

## Local rendering optimizations

Current local work preserves original graphics, animation timing, visibility
margins and material ordering. Linux remains the development platform. These
changes are not yet published in the linked release packages.

- **Prepare static work when residency changes.** Resolve model/animation
  references, static bounding spheres and stable material groups when streamed
  cells load or unload. Per-frame rendering avoids asset-path hashing, repeated
  static transforms and sorting model groups. Animated rotors retain current
  transforms and bounds; native MAni/ordinary alpha thresholds stay separate.
- **Reuse matrix upload memory.** World and skin renderers keep upload arrays
  instead of allocating three arrays per part through CGO. Identical world/view/
  projection uniforms are not uploaded again. New scene passes invalidate that
  cache, and changed cameras/projections are compared explicitly.
- **Reuse identical terrain masks.** The conservative occlusion mask is reused
  only when view, projection, eye and streamed terrain revision are unchanged.
  Movement, rotation or residency changes rebuild it. Incomplete streaming and
  unsupported cameras continue to fail open. This is not previous-frame
  visibility prediction.

### Measured scope

A real-asset Linux OpenGL benchmark (`BenchmarkDefaultGlowOriginalMapWith`, map 1,
640×360, frozen camera/time, original glow enabled) measured:

| Metric | Before | After |
| --- | ---: | ---: |
| Time per benchmark frame | 1.421 ms | 0.653 ms |
| Heap allocations per frame | 377 | 41 |
| Allocated bytes per frame | 40,255 | 11,159 |
| Draw calls per frame | 159 | 159 |

Hardware: NVIDIA RTX 3050, driver 615.78.08. These are local benchmark results,
not universal FPS or isolated GPU timings. The benchmark includes a final
`glFinish`; CPU profiles include asset setup, while the benchmark timer excludes
initialization. A stationary-camera mask benefits from exact reuse; moving
cameras still rebuild it. CPU/heap profiles and logs are private in
`work/optimization-before.*` and `work/optimization-after.*`.

Validation covers real-map pixel comparisons, stable equal-depth material
ordering, terrain streaming routes, MAni/VANI, actor/weapon animation, shadows,
reflection/water state restoration, UI and isolated N2 gameplay. The VANI
residency test now advances using the resource's actual authored interval rather
than assuming every file has a 100 ms interval. No quality reduction or shader
replacement was used to obtain these measurements.


## Subsequent sector/streaming/LOD work

See [streaming and LOD](streaming-lod.md) for visibility sectors, asynchronous
CPU loading, bounded GPU work, conservative object LOD and later measurements.
Initial compressed-mipmap uploads can still exceed the soft frame budget.

## GPU preference at startup

The client requests a dedicated GPU before GLFW creates its OpenGL context.
Linux discovers accessible PCI render devices through DRM/sysfs, recognizes
NVIDIA/nouveau and discrete AMDGPU adapters, and uses reported VRAM as a
secondary preference. AMD's kernel FUSION flag distinguishes an APU from a
discrete card; shared memory size alone is not sufficient. Unknown devices keep
the driver default when no recognized dedicated adapter is available. A single
GPU needs no offload override. This is a startup preference, not a performance
benchmark or a guarantee of the fastest adapter on every multi-GPU machine.

Mesa selection uses the explicit PCI form of [DRI_PRIME](https://docs.mesa3d.org/envvars.html#dri-prime).
Proprietary NVIDIA additionally uses its documented [PRIME offload variables](https://download.nvidia.com/XFree86/Linux-x86_64/545.29.06/README/primerenderoffload.html).
Existing PRIME, GLX/EGL vendor and software-rendering overrides are respected,
including `DRI_PRIME=0` to retain the system default. The startup OpenGL log
reports the renderer actually selected by the driver.

Windows source exports the high-performance requests documented by
[NVIDIA](https://developer.download.nvidia.com/devzone/devcenter/gamegraphics/files/OptimusRenderingPolicies.pdf)
and [AMD](https://gpuopen.com/learn/amdpowerxpressrequesthighperformance/).
System/user graphics preferences still take precedence. This local patch has
Linux policy tests and a native single-GPU smoke test; physical hybrid hardware
and the Windows exports have not been validated at runtime.


## Compact geometry indices

Immutable static, animated-topology and skinned meshes use unsigned 16-bit GPU
indices when every referenced vertex fits. Larger index ranges keep 32 bits;
dynamic triangle streams keep their existing format. Direct, instanced, terrain
indirect and offset draws use the stored element type and byte width. Topology,
vertex data, UVs, textures, collisions and draw order are unchanged.

On the checked 640x360 map 1 scene, resident GPU storage dropped from 138,869,468
to 133,606,844 bytes (5,262,624 bytes saved), with identical rendered pixels. The
full-frame benchmark kept 159 draws and measured 0.566 versus 0.522 ms in this
run; this is local evidence, not a promised FPS gain. Benchmark warmup now waits
for initial residency to finish. Index buffers shrink by 50% where eligible,
not total VRAM by 50%. The real-asset audit found no exact duplicated SMOD
vertices to remove, so it does not justify lossy model or texture changes.


## Map 1 populated-scene bottleneck

A desktop-resolution reproduction at `(996.458, 12.76, 787.152)`, using a private
disposable database replica, the equipped character and nearby server entities,
measured 47–49 ms per frame across four headings. The CPU profile attributed
about 79% to projecting character/NPC shadows onto nearby scene objects: each
shadow repeatedly transformed and tested entire building meshes. A scenery-only
fixture with frozen/empty actor state did not expose this cost.

The requested scope is now the local character shadow only, on terrain/dungeon
floor geometry. Object receiver projection and all other actor shadow captures
were removed. The same populated-scene probe measured 4.25–5.64 ms per frame.
This is the result of removing that feature's work, not an appearance-identical
benchmark of the previous all-actor shadow system. Desktop/GPU contention and
actual camera/input can affect user FPS. GPU timestamp intervals can include
periods waiting for CPU submissions; a large interval alone does not establish
that shaders or asset geometry are the bottleneck.


## Dungeon visibility and fog

DUN worlds use authored WLD fog when available. The stock placeholder keeps fog
disabled while retaining the WLD distance for the projection far plane. Field
and underwater rendering retain their existing settings.

The old eager dungeon path submitted every DG fragment and every MAni/SMOD/VAni
placement. It now rejects off-camera bounds and uses a conservative CPU depth
mask from actual opaque DG/static-model triangles. Alpha coverage must be complete
at every authored mip, including DXT alpha modes; doors, grates and unknown alpha
remain openings. Collision hulls and room-sized solid boxes are not occluders.
Moving rotors and vertex animations do not contribute occluders.

The mask is 160×90, uses spatial bins and at most 512 nearby triangles per camera
update, and is reused only for an identical view/projection. It has no GPU readback,
blocking occlusion query or previous-frame visibility. Clipped reflection and
shadow passes do not reuse the main-view mask. Bounds cover full animated frames.
Consecutive single-material model instances share draws without changing native
material order. DG merges are likewise limited to consecutive matching spatial
materials; multipart object order is preserved. Dungeon GPU meshes are released
when their world closes.

A real-assets GL comparison covers seven dungeons, 176 camera views, animation,
three aspect ratios and a map-3 arrival position. The optimized and complete
reference traversals produced byte-identical pixels in those views. These are
world-geometry draw calls, excluding UI/entities/effects outside that traversal;
they are not whole-frame FPS claims.

| Dungeon map | Previous draws | Mean optimized draws | Mean reduction |
|---|---:|---:|---:|
| 3 | 2332 | 308.9 | 86.8% |
| 4 | 450 | 70.8 | 84.3% |
| 5 | 1236 | 153.6 | 87.6% |
| 6 | 545 | 83.2 | 84.7% |
| 7 | 1541 | 158.3 | 89.7% |
| 8 | 545 | 83.2 | 84.7% |
| 9 | 1891 | 218.9 | 88.4% |

Regression checks also cover the ordinary field batching/terrain mask, the login
MAni backdrop, the character's dungeon shadow, alpha mip handling, an open doorway
and camera changes. Evidence is kept locally under `work/dungeon-render/` and
`work/dungeon-*-gl.log`; these resources and captures are not distribution assets.


## Stable lighting on coincident world surfaces

Some SMOD assets author front and back surfaces at the same position with
opposite normals and different triangle diagonals. Drawing both sheets into the
float depth buffer makes subpixel camera movement alternate between their
lighting. The map 1 market wood walls are a reproducible example.

Static model preparation now identifies opposite coplanar coverage, checking
vertices, edge midpoints and triangle centers. These faces use clockwise
back-face culling. Unpaired surfaces retain two-sided coverage, including
cutout foliage and thin decorations. Mixed material parts split into compact
vertex/index buffers; UVs, normals, textures, collisions and authored files stay
unchanged. Async streaming performs this classification on its CPU workers,
before GL upload, and LOD meshes retain the resulting surface contract.

The rendering change applies to streamed and eager world objects and instanced
copies. It does not depend on an asset filename or an anti-aliasing option and
uses no depth bias. Mixed parts may add a draw call; no general FPS improvement
is claimed. A float-depth map 1 regression checks camera motion from eight views
in normal and reverse depth. Separate GL tests check both sides, cutout holes,
unpaired foliage and raster-state cleanup; field/dungeon batching comparisons
use the same surface contract.

## Dungeon cursor and stationary mob queries

Cursor hover uses a three-slot, one-pixel PBO readback ring with zero-timeout
fences. CPU reads occur only after a transfer completes; an unavailable sample
keeps the cursor neutral. Samples match cursor coordinates, viewport and depth
convention, and expire after three captured frames. Hover feedback can therefore
lag briefly; actual clicks still validate the captured scene depth synchronously
with the original tolerance. Resize, map changes and shutdown discard pending
samples and release their buffers. Off-target fallback candidates are rejected
before requesting depth.

Each mob caches its visual grounding result while its interpolated XYZ, world,
height-authority flag and position-reset serial remain identical. Movement,
a changed server height, respawn/teleport or a new map recomputes the result.
Replacing an entity also replaces its actor/cache. Network positions, original
support queries and different-floor handling remain unchanged; animations and
effects continue updating for stationary mobs.

Local three-run median measurements on Ryzen 5700G / RTX 3050, NVIDIA 615.78.08:

| Isolated operation | Before | After |
| --- | ---: | ---: |
| 1280×720 depth copy plus cursor hover | 1.685 ms | 0.148 ms |
| Ground queries for 256 stationary mobs, original dungeon map 15 | 0.583 ms | 0.00102 ms |

These benchmarks isolate the reported costs; they do not establish whole-scene
FPS or eliminate the need for live dungeon validation. GPU benchmark cases each
own a current context on their locked OS thread. Real-asset regressions cover
hover/click occlusion, standard/reversed depth, map/resize invalidation and
matching visual heights on dungeon floors and outdoor terrain.

## Reproducible CPU/render baseline

Use the [client measurement harness](client-measurement.md) for repeated real-asset
scenes, verified visible water, dungeons, 200/400-actor stress and synthetic packet
pumping. It separates update/draw submission, explicit GPU queue wait and distinct
GPU timestamp samples, with p50/p95/p99, captures and isolated profiles. Hidden
throughput is not presented FPS; shared-GPU runs are explicitly classified.

## Client baseline without a competing game (2026-11-16)

A separate quiet-desktop run of the [measurement harness](client-measurement.md)
used Linux, Ryzen 7 5700G, RTX 3050, NVIDIA 615.78.08 and Go 1.27.4 at 1280×720.
WoW was closed and absent from the recorded GPU process snapshots. Eleven scenes
ran three times with 900 frames each (29,700 unprofiled frames); two additional
3,000-frame CPU profiles were kept separate. These are hidden-window submission
timings with bounded GPU queue depth, not presented FPS or a player-capacity claim.

| Scene | CPU submission p50 ms | p95 ms | p99 ms |
|---|---:|---:|---:|
| world-stream | 0.831 | 0.966 | 1.070 |
| water-visible | 5.328 | 6.586 | 7.597 |
| crowd-200 | 6.519 | 7.481 | 8.438 |
| crowd-400 | 13.018 | 14.659 | 16.305 |
| crowd-effects | 7.866 | 8.519 | 9.745 |
| crowd-network | 6.693 | 7.610 | 8.744 |
| dungeon | 13.529 | 14.960 | 16.015 |

The follow-up priorities are actor pose/bounds preparation for crowds, and
effect submission/buffer streaming for the visible map-15 dungeon. Preserve
visible animation, shadow/reflection participation and transparent draw order;
validate each change independently before introducing general parallelism or
changing glow. No renderer optimization was included in this measurement update.
Machine contention, camera/workload and timing mode must match for comparisons.

## Pose sampling and identical inverse-bind palettes (2026-11-16)

The client now reuses actor-owned final-matrix storage, decomposes immutable ANI
rest transforms once when loading the shared clip, and evaluates one palette for
parts whose complete inverse-bind arrays match exactly. Different bind arrays
retain independent palettes. Animation sampling times, matrix multiplication
order, bounds, reflection and shadow visibility remain unchanged.

A focused Linux/OpenGL comparison used the original baseline binary and the new
binary, four scenes, three shuffled repetitions per binary and 300 frames per run
(7,200 measured frames). These are short before/after batches, not an alternating
binary experiment; normal desktop noise remains. All GPU samples were valid and
all expected assets/actors loaded. Median CPU submission wall times:

| Scene | Before ms | After ms | Reduction |
|---|---:|---:|---:|
| 200 actors | 6.319 | 5.161 | 18.3% |
| 400 actors | 12.915 | 10.099 | 21.8% |
| Field | 0.729 | 0.729 | unchanged |
| Visible water | 5.068 | 4.708 | 7.1% observed |

The water change is not attributed to this pose optimization: scene/driver noise
can affect it. Do not extrapolate presented FPS or universal performance gains.
Allocated Go bytes per frame fell from about 631 KB to 12 KB for 200 actors and
1,259 KB to 22 KB for 400 actors; allocation counts fell by one per actor/frame.

Unit tests cover storage reuse, actor isolation, different inverse binds, outfit
replacement, clip growth/shrinkage, identity fallback and exact bounds. The native
GL test also compares original player/monster/vehicle part matrices against
independent per-part evaluation with uncached rest decomposition. Existing twelve
mixed camera/clip-plane views and 144 shadow placements preserve culling pixels.
The harness captures themselves differ between processes because of animated
scene content; no whole-scene pixel equality is claimed for those captures.
Private measurements and captures: `work/client-measure/pose-before/` and
`work/client-measure/pose-after/`. Conservative pre-pose rejection and dungeon
effect improvements are documented in the follow-up sections below.

## Consecutive map effects share scene setup (2026-11-16)

Map effects now enter and leave the scene shader pass once per camera, retaining
placement/emitter/particle order and independent draws. Standalone effect callers
keep their existing setup/restore behavior. No textures, blending, particle counts
or camera visibility rules changed. Real-map traversal and synthetic mixed
single/instanced 3DE effects preserve pixels, including clipping and a following
ordinary mesh draw. Original mirrored lamps and zero-life emitters also pass.

Focused short measurements use the same three repetitions of 300 frames per case
and binary. See private `work/client-measure/effects-before/` and `effects-after/`.
Dungeon median CPU submission was 13.069 → 12.607 ms (3.5% observed), with
per-run medians of 12.858–13.319 and 12.262–12.890 ms. Crowd-effects was
6.707 → 6.340 ms (5.5% observed), with overlapping per-run ranges.
Do not attribute all timing variation to this small state-call reduction: desktop
noise and wide ranges remain. No streaming/storage strategy changed in this step.

A subsequent experiment retained streamed mesh indices when consecutive topology
arrays were identical, while still orphaning/updating vertex storage. It was
**not retained**: three 300-frame repetitions produced overlapping dungeon
medians and no consistent improvement; the crowd-effects case was slower.
Private experiment output is `work/client-measure/stream-after/`; the retained
comparison is `effects-after/`. Existing streaming/orphaning remains in place.

## Conservative actor rejection before pose sampling (2026-11-16)

`Character.Visible` first tests an enclosing box that needs no current pose. It
falls through to the unchanged exact `WorldBounds` test whenever that box is
visible or unavailable. Each clip stores a per-bone bound on global rotation
stretch and translation reach, built in the sampler's parent order. Translation
interpolation stays inside its endpoint hull; normalized interior quaternion
interpolation preserves length. Raw endpoint norm error, float32 arithmetic and
outward box rounding receive conservative margins. Equipment and inverse-bind
boxes contribute to the actor envelope. Affine world transforms account for
nonuniform scale, shear and reflection.

The actor cache tracks clip, part identities, equipment transforms and mounted
state; asset geometry/bind data remain immutable. Cloth, unknown geometry,
non-affine/nonfinite or overflow-risk matrices, uncertain quaternion/key data and unsupported
large keyframe indices retain exact culling. A rejection is evaluated separately
for each camera/clip plane; it never suppresses shadow fitting. Name anchors
sample their crown binds on demand when the body palette is stale, preserving
labels without preparing every skin part.

Focused Linux/OpenGL comparison: three 300-frame repetitions per scene and
binary, 18 runs / 5,400 measured frames. All GPU samples were valid and no assets
were missing. CPU submission wall times (not presented FPS):

| Scene | Before p50 / p99 ms | After p50 / p99 ms |
|---|---:|---:|
| 200 actors | 5.052 / 6.866 | 4.681 / 6.474 |
| 400 actors | 9.918 / 11.839 | 9.920 / 12.179 |
| Visible water | 4.596 / 6.082 | 4.655 / 6.689 |

The 200-actor median fell 7.3%; per-run medians were 5.014–5.089 before and
4.644–4.761 after. The 400-actor median was unchanged. Water and tail timings
fluctuated; no improvement is claimed for them. These are short sequential
before/after batches on a normal desktop, not exclusive hardware measurements.
A subsequent final-binary check showed desktop drift, so an additional adjacent
old/new pair was run: 200 actors 4.684 → 4.509 ms (3.7% observed), and 400 actors
9.335 → 9.333 ms (unchanged). The initial 7.3% is therefore not a guaranteed gain.
These checks add six runs / 1,800 frames (7,200 frames overall); private outputs
are `precull-final-check/` and `precull-paired-{before,after}/`. Reused binaries
are identified by their hashes; the paired old binary is the same original
`effects-after.test`, despite the runner also recording current checkout source.
Benefit depends on how many actors can be rejected by the coarse envelope;
visible actors still use exact poses and bounds.

Validation includes 500 random affine-world/time comparisons against exact
animated/equipment boxes, camera switching, clip planes, mount/outfit/clip
invalidation, uncertain-data fallback and current crown sampling after skipped
body poses. Original player, monster and vehicle bounds are also checked. Twelve
main/reflected views and 144 shadow placements preserve pixels; original NPC
nameplate, picking and lazy name-depth tests pass. Unit/vet, integration vet,
scene race tests and native Linux build pass. Private artifacts:
`work/client-measure/precull-before/`, `precull-after/`, `precull-install/`.

## Shared instance upload for effects

The renderer packs instance matrices and tint colors into one OpenGL buffer
upload. The VAO reads the matrix prefix and the shader reads the colors at
the end of the buffer. This removes one buffer upload per colored batch
while retaining the particle order, blending and draw count. Ground item
batches use the same path.

An interleaved Linux/OpenGL comparison on the RTX 3050 at 1280×720 used three
repetitions of 360 frames per scene and binary. The table reports the median
of the three per-run submission medians. Submission wall time includes the
driver, and the hidden window does not present frames.

| Scene | Separate uploads | Shared upload |
|---|---:|---:|
| Map 15 dungeon | 12.671 ms | 12.096 ms |
| 200 actors and combat effects | 6.406 ms | 6.413 ms |
| Visible water | 5.080 ms | 5.077 ms |
| Field | 0.747 ms | 0.735 ms |

All three dungeon pairings improved. Whole-loop time fell from 12.715 to
12.219 ms per frame in that scene. The other scenes did not show a material
submission change. A third binary also grouped adjacent compatible quads.
It produced no consistent improvement over the shared upload and was
discarded. The experiment recorded 12,960 valid GPU timestamp samples
with no missing assets. Existing OpenGL comparisons cover blend modes,
animated 3DE frames, clip planes, water and subsequent draws.

## Affine pose products and shared world transforms

Skeletal composition, inverse-bind palettes and skinned bounds use the affine
matrix product where applicable. Projective or nonfinite inputs retain the
full matrix product. During each bounds calculation, parts with identical
inverse-bind palettes share bone-to-world transforms for the bones they use.
Each mesh keeps its own bounds, and animation sampling and draw order are
unchanged.

The same Linux/OpenGL fixture at 1280×720 ran three interleaved repetitions
of 360 frames per scene and binary. Values are medians of the per-run
submission p50 or p99 and whole-loop averages, in milliseconds per frame.

| Scene | Submission p50 before → after | Submission p99 before → after | Whole loop before → after |
|---|---:|---:|---:|
| 200 actors | 5.156 → 4.475 | 6.887 → 6.227 | 5.293 → 4.576 |
| 400 actors | 10.649 → 8.985 | 12.768 → 10.896 | 10.812 → 9.095 |
| Field | 0.770 → 0.738 | 1.117 → 0.922 | 0.891 → 0.856 |
| Visible water | 5.231 → 5.188 | 6.683 → 6.454 | 5.299 → 5.198 |

Each crowd pairing improved at p50. Tail times varied; one optimized
400-actor run had a higher p99 than its baseline. Field and water differences
were small. The isolated 64-bone pose benchmark fell from 11.30 µs to 8.34 µs
per operation, with zero steady-state allocations in both versions.
Submission includes driver work, and the fixture runs without presentation.
In separate 3,000-frame profiles of the 400-actor case, direct matrix
product samples fell from 8.65 s to 3.97 s, counting the affine and full
products together in the optimized profile.

Tests compare the affine product with the full product, including fallback
inputs, and compare real character, monster and vehicle palettes with an
independent full-product reference. OpenGL checks cover main and reflected
views, shadows, cloth and nameplates.

## Player crowd rendering with native cloth

Character rendering sends the world and camera matrices once per actor and
reuses a palette across consecutive parts with identical inverse binds.
Visible player and bot capes are prepared on up to four CPU threads in large
crowds. This uses the same per-actor solver and substeps. OpenGL mesh uploads
and ordered draws remain on the render thread.

The seeded 300-bot fixture used map 1 at 1280×720, with weapons and cloth
equipped and skill effects disabled. Three alternating runs of 240 frames
per version returned the following medians of per-run statistics.

| Version | Submission p50 | Submission p99 | Whole loop | Draws/frame |
|---|---:|---:|---:|---:|
| Baseline | 34.566 ms | 37.272 ms | 34.738 ms | 3,293 |
| Shared world matrices and palette uploads | 34.055 ms | 41.785 ms | 34.509 ms | 3,293 |
| Shared uploads and parallel cloth | 20.322 ms | 22.983 ms | 20.529 ms | 3,293 |

In the separate baseline CPU profile, cloth preparation accounted for 51.1%
of samples and the local bot simulation accounted for 1.2%. The parallel
pass improves frame submission time. It retains the cloth simulation work.
Precomputing additional cloth coefficients did not consistently improve
submission in a follow-up comparison and was discarded.

Phoenix groups bot instances by preset and shared pose and batches the
mesh draws. GX retains each actor's animation clock and cloth state.
The remaining draw count makes instancing with independent palette offsets
a separate candidate. See [Phoenix's bot renderer](https://github.com/Spelunkern/Phoenix-Engine/blob/f1ba55122de7def1e3c49f69eaaccf6922f89a1a/src/character/bot_manager.h#L849-L893).

## Instanced PvP characters and projected cloth detail

In large crowds with skill effects disabled, GX groups compatible skinned
parts and rigid equipment. The skin shader reads a separate bone palette,
camera transform, and corpse fade value for each instance. Mesh, texture,
cutout and lighting state remain part of the group key. Concealed characters,
blended attachments, and shadow captures use the serial path. Mount draws
keep their order. The body-only clip envelope also rejects off-camera
cloaked characters when it can enclose their original skinned bounds.

Small projected capes use staggered refresh times and a shorter simulation
budget. Large capes keep their full solver. The sampled cape mesh follows the
actor between updates, and clipped captures use the full cloth path.

A seeded, interleaved 300-bot comparison with skill effects off and a shared
desktop GPU used 240 frames in each of three runs per version at 1280×720.
The following values are medians of the corresponding per-run statistics.

| Version | Submission p50 | Submission p99 | Whole loop | Full-frame draws |
|---|---:|---:|---:|---:|
| Previous parallel cloth | 19.854 ms | 22.437 ms | 20.005 ms | 3,293 |
| Instancing, body rejection and cloth detail | 14.343 ms | 16.553 ms | 14.463 ms | 1,278 |

The final main-camera bot pass had 300 skinned batches, 33 rigid equipment
batches and 288 cape draws. The full-frame counts also include the map,
overlays and secondary passes. A wider 600-bot view measured 24.308 ms per
frame for the whole loop and 1,703 full-frame draws in one run. These
measurements do not establish presented FPS.

A controlled OpenGL fixture with 24 independent animation phases, a
fading corpse and original equipment produced identical pixels in two
camera passes while reducing draws from 216 to 32. Tests also cover nearby
full cloth, reduced distant cadence, fallback in clipped passes, and
rejection before preparing poses for cloaked actors.

## Instanced cloth and depth-tested nameplates

Opaque capes share draws by mesh and material. Each instance reads its own
deformed positions from a resident storage buffer. A changed pose uploads
one packed list of position ranges and a compute pass copies it to the
resident slots. A fully unchanged pose sends only the current instance
transform. A growing or heavily changed buffer uses a full position upload.
Cloth uses the authored UVs and its existing unlit shader. The CPU no longer
reconstructs normals for that unlit surface.

Nameplates carry depth per quad, and solid health bars share the active glyph
batch. This preserves their draw order and per-label depth test with fewer
sprite submissions. Capes below roughly 24 projected actor pixels use a
12 Hz refresh with a smaller solver budget. Larger capes retain the previous
quality tiers, and near, clipped and shadow passes retain the full solver.

Three interleaved Linux runs of 240 frames per variant used the seeded
600-bot scene at 1280×720 with skill effects disabled.

| Variant | Whole loop ms/frame | Full-frame draws |
|---|---:|---:|
| Before | 24.295 | 1,703 |
| Instanced capes, dirty uploads and no normal rebuild | 22.701 | 1,166 |
| Also batch nameplates | 22.484 | 576 |

In three additional 180-frame paired tests, the far tier reduced whole-loop
time from 22.260 to 20.921 ms. This is an isolated comparison with the prior
tier. In the 600-bot scene, cloak draws fell from 585 to 47 and UI draws fell
from 624 to 38. The position stream averaged 580 kB per frame when the
instanced path uploaded the entire buffer. Dirty-range uploads averaged
203 kB, or 162 kB with the far tier. The total excludes instance transforms.
In matched 900-frame profiles, CPU samples fell from 30.94 s to 25.19 s.
Cloth preparation and bounds still account for about 37% each in the final
profile, with pose work included in the bounds count.

Native OpenGL tests compare serial and instanced cloak pixels in two cameras,
test moving unchanged meshes and one changed range, and verify label depth
across mixed depths and solid quads. The maximum bot count also exercises
buffer release.

## Parallel crowd visibility and cheaper skin poses

Remote players and local PvP bots prepare frustum and clip visibility with
up to four workers. Each worker owns its actor's pose, so submission follows
the original order and the GL thread still does all drawing. The same
animated bounds, terrain occlusion and mount ordering apply.

Skin palette preparation computes only the bones weighted by the pieces
that share an inverse-bind palette. Crowd uploads omit trailing unused
matrices while retaining the authored bone indices. Unknown weights use
the full palette. Attachment and cloth pin joints still use the complete
skeleton. Cloth reuses a spring force when the native graph repeats the
same spring in reverse. Both contributions retain their original order,
and each cloth keeps its own position and velocity.

Three interleaved 240-frame runs per variant used the seeded 600-bot scene
with skill effects disabled, at 1280×720 on a shared desktop. These are
medians of the individual run statistics.

| Variant | Whole loop ms/frame | Submission p50 / p99 ms | Full-frame draws |
|---|---:|---:|---:|
| Previous build | 21.307 | 21.117 / 24.797 | 576 |
| Parallel visibility and sparse palettes | 12.286 | 12.194 / 14.469 | 576 |
| Also reuse reverse spring forces | 11.991 | 11.713 / 14.657 | 576 |

At this camera the final frame submitted 5.80 MB of skinning matrices in
300 skinned batches. The timings are from a hidden window without presentation.
Separate 900-frame profiles collected 25.49 s, 22.24 s and 20.34 s of CPU
samples for the three variants. In the final profile, posed bounds and cloth
preparation took 7.43 s and 6.75 s respectively. The bounds count includes
pose preparation. Two paired 300-bot runs measured 13.170 → 8.010 ms for
the whole loop. Two paired scenes with 200 remote players, 120 monsters and
80 NPCs measured 9.021 → 6.973 ms.

## Crowd preparation scaling and skinned multi-draw

The crowd distributes camera visibility, cloth preparation and per-actor draw
preparation across scheduler threads. The worker count grows with both
`GOMAXPROCS` and the number of characters in the pass, keeping at least 24
candidate actors per worker on average. Workers read the finished terrain
mask. The render thread owns OpenGL, submission order and occlusion counters.

If the clip envelope sits entirely inside the camera, the crowd accepts it
without posed bounds. A bound intersecting a plane is refined with the posed
geometry. A point known to lie inside the body bounds can also rule out
terrain occlusion before visiting each posed part. Uncertain cases use the
full animated bounds and terrain test.

Skinned geometry is packed into bounded GPU buffers per crowd. A multi-draw
combines consecutive groups that share the same index width and back-face
culling state. Every command keeps its texture, alpha cutoff, lighting and
palette offsets. Texture views reuse the authored mip storage and preserve
wrapping. This path uses `GL_ARB_bindless_texture` and
`GL_ARB_shader_draw_parameters`. When either extension is unavailable, the
renderer uses ordinary instancing. [Khronos bindless texture extension](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_bindless_texture.txt),
[Khronos shader draw parameters extension](https://registry.khronos.org/OpenGL/extensions/ARB/ARB_shader_draw_parameters.txt).

Three interleaved 240-frame runs per variant used the seeded 600-bot scene
with skill effects disabled at 1280×720. These medians measure a hidden
window on a shared Ryzen 7 5700G and RTX 3050 desktop.

| Variant | Whole loop ms/frame | Submission p50 / p99 ms | Full-frame GL calls |
|---|---:|---:|---:|
| Previous build | 11.335 | 11.205 / 13.821 | 576 |
| Scaled workers and conservative visibility | 9.055 | 8.696 / 10.650 | 576 |
| Also submit skinned groups with multi-draw | 8.176 | 8.059 / 9.807 | 370 |

The 300 skin groups occupied 94 GL submissions. The packed skin cache
added about 9.45 MB of renderer-owned storage in this scene. A separate
three-run comparison of the final binary at `GOMAXPROCS` 4, 8 and 16 measured
10.404, 8.617 and 8.141 ms for the whole loop. These timings exclude display
presentation.

In matched 900-frame profiles at `GOMAXPROCS=16`, posed bounds fell from
7.55 to 3.82 seconds of CPU samples. Across all threads the profile increased
from 20.99 to 23.31 CPU seconds. A trial capping only cloth at eight workers
increased the whole loop from 7.864 to 8.275 ms in two paired runs. The
automatic worker count keeps the lower frame time. A smaller 300-bot case
measured 7.840 → 5.731 ms. Two paired 400-actor scenes with skill effects off
measured 5.419 → 5.118 ms and 606 → 511 GL submissions.

## Reduced simulation grids for distant capes

Crowd capes switch to a reduced simulation grid below 38 projected actor
pixels. The grid keeps the original columns and the top, bottom and pinned
rows, simulating every other unpinned row. The renderer interpolates movement
across the existing vertices and retains the authored rest shape, triangles and
UVs. Each character owns its positions and velocities. A cloak stays on the
small grid until it reaches 44 projected pixels, then resumes the full grid
from the displayed surface. Nearby characters and clipped or shadow passes
use the original grid.

Five interleaved runs at 1280×720 in the seeded 600-bot fight with skill
effects disabled measured 7.649 → 7.390 ms for the whole loop, with submission
p99 at 9.177 → 8.672 ms. Draw submissions and cloth position upload size
remained at about 370 calls and 162 kB per frame, respectively. The simulation
reduction preserves the GPU mesh. Measurements were made in a hidden window
on a shared desktop.

The 600-bot diagnostic counted roughly 21,300 live simulation points out of
36,200 authored vertices. In separate 900-frame profiles, the cloth update
fell from 9.89 to 5.74 seconds of CPU samples and total CPU samples went from
23.18 to 20.07 seconds. Two 300-bot pairs measured 5.603 → 5.336 ms for the
whole loop. An exact animation-phase diagnostic found a potential duplicate
for about 6% of visible actors, so this change adds no pose cache.

## PvP effects with ordered parallel preparation

Crowd character geometry stays batched with skill effects enabled. Weapon
transparency follows opaque geometry and uses its completed depth buffer.
Within each transparent pass, actor, emitter, particle and blend order are
preserved. Workers advance independent effect instances and prepare their
placements, materials and animation samples. Effects following a 3DE path
read the birth vertex directly from immutable keys. The GL thread performs
resource creation, uploads, blend changes and submission.

Original 3DE topology and authored position/UV keys stay in bounded GPU
storage. Each particle supplies its original key pair and interpolation
factor. Particles at different animation phases can share an instanced draw.
The transparent pass shares one instance upload per bounded range and joins
consecutive draws with identical geometry, texture, wrap and blend state.
Geometry that exceeds the resident budget uses CPU sampling. Draw calls for
opaque crowds use the existing instancing path. Shadow and ordinary UI passes
retain their original path. Particle counts, lifetime, cadence and material
definitions are kept.

The seeded 600-bot case at 1280×720 with skill effects enabled measured the
following on Linux and RTX 3050 in a hidden shared-desktop window. Each row
is the median of three 180-frame runs. The two rows use the same fixture in
separate interleaved stage comparisons.

| Version | Whole loop ms/frame | Submission p99 ms | GL calls/frame |
|---|---:|---:|---:|
| Before the effects work | 52.248 | 58.677 | 9,860 |
| Complete effects path | 17.700 | 20.063 | 2,167 |

GPU key sampling, with opaque batching and parallel preparation, reached
28.865 ms before the shared upload and adjacent draw merges. At the final
revision one trial each with `GOMAXPROCS=4` and `=8` measured 20.866 and
18.788 ms. The 16-thread trial set above measured 17.700 ms. Real-asset GL
comparisons cover glows, enchantments, reflections, map effects and cooldowns.
The animated 3DE fixture checks sampled UVs and colors pixel-for-pixel against
CPU sampling with mixed blend modes and a clip plane.

Separate 900-frame profiles collected 61.28 → 32.98 CPU seconds. One 300-bot
effects-on comparison measured 31.578 → 12.529 ms. The map-15 dungeon control
measured 11.792 → 8.236 ms. The measurements exclude display presentation.

## Ordered multi-draw for combat effects

Consecutive effect draws with the same blend factors and index format share
an indirect OpenGL submission. A bounded scene cache packs immutable mesh
topology and the original 3DE position and UV keys. Each command selects its
own geometry, texture and wrap mode. The GL thread submits commands in their
original order and keeps depth testing active with depth writes disabled.
Drivers without bindless textures use the compact instanced path. Animated
geometry beyond the resident cache can still use CPU sampling, and small
passes use ordinary instancing.

Effects are unlit and unfogged. The compact shader needs only a projected
matrix, tint and 3DE key selection, reducing the main-camera instance upload
from 224 to 96 bytes. The clipped reflection pass also supplies the original
world matrix. Workers use the current camera's read-only terrain mask to
discard fully hidden groups of billboard particles and compact the surviving
placements in order. Simulation and particle lifetimes continue normally.

Three interleaved 180-frame pairs per case on Ryzen 7 5700G and RTX 3050,
at 1280×720 in a hidden window on a shared desktop with effects enabled.

| Local PvP fixture | Whole loop ms/frame before → after | Submission p99 ms before → after | GL calls/frame before → after |
|---|---:|---:|---:|
| 300 bots | 12.428 → 7.764 | 14.562 → 9.500 | 1,437 → 624 |
| 600 bots | 17.753 → 10.791 | 19.947 → 12.255 | 2,167 → 851 |

A single 600-bot run per intermediate stage measured 16.968 ms with compact
uploads and 14.264 ms with worker-side occlusion added. Single control pairs
measured 7.342 → 7.237 ms with 600 bots and skill effects disabled, and
8.256 → 4.537 ms in the map-15 dungeon. Renderer-owned memory for the 600-bot
effects run rose by about 4.8 MiB. The controlled pixel comparison covers mixed
textures, wrap modes, 3DE keyframes, alpha and additive blending, opaque depth,
reverse depth, temporal jitter and a reflection clip plane.

## Cheaper particle transforms

The compact effect queue uses affine model-view multiplication and skips
zero terms in the native perspective projection. Camera-relative translation
still subtracts the eye before rotation in double precision. Temporal jitter
and reverse depth use the existing projection coefficients. Other projection
layouts use general multiplication.

Each EFT particle stores the two coefficients of its initial tilt at birth.
The renderer composes the animated spin with that tilt in the authored order
for meshes and billboards. This also preserves the falling meteor's axis.

In interleaved 240-frame trials at 1280×720 with effects enabled, the
whole loop measured 7.833 → 7.193 ms for 300 local PvP bots (three pairs)
and 11.032 → 10.178 ms for 600 bots (four pairs). The 600-bot projection-only
variant measured 10.251 ms. The draws remained near 630 and 867 per frame.
The measurements use a hidden Linux window on a shared desktop with an
RTX 3050 and Ryzen 7 5700G and exclude display presentation.

## Map geometry, terrain streaming and reflections

Static map geometry stays in a bounded GPU cache with its placement
transforms. On drivers with bindless textures and shader draw parameters,
each camera uploads the visible placement IDs, materials and indirect
commands. The renderer preserves native draw order and groups compatible
commands. Drivers missing these extensions retain ordinary and instanced
draws. Visibility workers test cached dungeon bounds and field placements.
Animated map models retain their original animation and draw paths.

Terrain workers prepare fixed four-by-four blocks. A block remains resident
while any of its cells is needed, and the render thread submits the visible
ranges. Terrain overlays keep the original framebuffer blending and use one
indirect submission per block when supported. Masks whose authored alpha
levels all fall below the original threshold can be skipped after the base
layer. Water reflection rendering uses a conservative viewport region padded
for normal-map displacement and filtering. A receiver near the camera or
uncertain coverage uses the full reflection target.

Three 420-frame runs per case at 1280×720, with fixed cameras on the RTX 3050
and Ryzen 7 5700G shared Linux desktop. CPU times are median submission times
including driver waits. GPU figures are mean timestamp intervals.

| Fixture | CPU ms before → after | GPU ms before → after | GL draws/frame before → after |
|---|---:|---:|---:|
| Dungeon map 15 | 4.580 → 1.840 | 4.658 → 1.885 | 1,501 → 418 |
| Map 1 coast with reflections | 4.738 → 2.450 | 4.788 → 2.457 | 816 → 198 |
| Map 1 coast with reflections disabled | 2.884 → 1.383 | 2.907 → 1.408 | 412 → 117 |
| Map 1 field | 0.605 → 0.541 | 0.736 → 0.690 | 165 → 71 |

Renderer-owned GPU memory rose by 5–9 MiB in these scenes. Real map draw
comparisons test dungeon alpha and lightmaps, field terrain and placements,
and the final water composite with full and cropped reflections. The packed
field path differed by up to two channels per captured frame, each by one
8-bit level. Asynchronous route tests cover movement and terrain residency.

## Low-spec GPU, Crowd & UI Optimization Pass

An audit on integrated AMD APU hardware exposed several CPU, driver, and UI
overhead hotspots across dense scenes and `Glow = 0` configurations:

- **Water-target color-only FBO (`internal/render/water_target.go`,
  `name_depth.go`, `hover_depth.go`, `internal/app/world.go`)**: when `Glow` is
  disabled, binding `worldDepth` as a shader sampler while it remained attached
  to the active `frameTarget` FBO triggered a depth-feedback stall on AMD
  OpenGL 4.5 (`21.5.2`). `BeginColorOnly()` binds a dedicated `colorFBO` with
  no depth attachment before sampling `worldDepth` for nameplates/water, and
  `SceneGlow` is skipped completely when `Glow == 0`.
- **Rigid crowd attachment batching (`internal/render/rigid_crowd.go`,
  `internal/scene/character_batch.go`)**: static rigid weapons and shields in
  dense crowds are packed into an immutable vertex/index heap and drawn via the
  bindless `mapVS`/`mapFS`/`mapCS` multi-draw indirect path (`gl.CULL_FACE`
  disabled), cutting crowd draw calls by ~60% (falls back cleanly when `< 2`
  groups or bindless MDI is unavailable). Concealed/blended draws defer after
  the opaque crowd flush.
- **Pre-outlined Latin-1 font atlas & nameplate icon batching
  (`internal/ui/font.go`, `internal/app/entities.go`)**: Latin-1 glyph cells
  pack a 1 px dilated outline directly into a `512x512` atlas (`outlineGlyphs`
  / `hasOutline`), reducing `DrawOutlined` from 5 passes to 2 passes while
  keeping the 5-pass path for dynamic Unicode fallbacks. World nameplate icons
  are grouped/sorted by `tex.ID`.
- **Skinned UBO & sprite ring buffers, cached locations, terrain layer skip,
  and GL VRAM fallback (`internal/render/skinned.go`, `sprites.go`,
  `shadow.go`, `effect_pipeline.go`, `terrain_overlay.go`, `scene.go`,
  `device.go`)**: replaced per-draw buffer orphaning with a 64-slot
  offset-aligned skinned UBO ring buffer and a 4-slot sprite VBO ring buffer;
  cached `ShadowMap` / `uReversedDepth` uniform locations; bucketed
  `effectUpload` capacities to powers of two; added early-out before sampling
  disabled/low-alpha terrain layers; and added `GL_ATI_meminfo` VRAM query
  fallback alongside Windows `ResidentBytes`/`TotalMemoryBytes`
  (`internal/platform/memory_windows.go`).
- **Animation keyframe binary search, coarse bounds, and 32-triangle camera
  collider blocks (`internal/scene/character.go`, `character_bounds.go`,
  `camera_collision.go`, `assets.go`, `field.go`, `internal/app/world_camera.go`,
  `monsters.go`, `npcs.go`)**: added monotonic keyframe checks (`rotKeysSorted`
  / `transKeysSorted`) for exact binary-search sampling; cached coarse AABBs
  (`CoarseBounds`) and upgraded `Visible()` to `frustum.ClassifyBox`; added
  stationary camera-ray caching in `World` (`v.world.Ready()`) plus
  `segmentSphere` broad-phase and lazy 32-triangle `colliderBlock` AABBs; and
  precomputed `vp` in `combatGlyphRectVP` (`combat_numbers.go`).
- **VSync configuration (`cmd/shaiya/main.go`, `internal/platform/window.go`,
  `internal/app/game_menu.go`, `app.go`)**: persisted `NoVSync` in settings and
  added `-vsync` / runtime `ApplyVSync`.

### Validation Benchmarks (AMD A12-9720P, Radeon R7 Graphics, OpenGL 4.5.14831 / 21.5.2, Windows 10)

Each run renders 180 frames using `SHAIYA_TEST_RENDER=1` with real local assets.

**Baseline (`1280x720`, default settings before pass):**

| Scene | Throughput (ms/f) | CPU p50 (ms) | GPU p50 (ms) | Avg DC |
| --- | --- | --- | --- | --- |
| `world-stream` | 7.688 | 6.101 | 7.094 | 72.8 |
| `world-mobs` | 12.265 | 6.998 (p95 16.009) | 8.795 (p95 14.619) | 77.0 |
| `world-hud` | 10.197 | 7.004 | 9.677 (`gpu_wait` 2.908) | 108.3 |
| `crowd-200` | 19.322 | 19.007 | 19.255 | 202.8 |

**After depth-feedback reproduction vs. final optimizations (`1366x705`):**

Note: before adding `BeginColorOnly()`, disabling Glow (`Glow=0`) regressed
`crowd-200` at `1366x705` (`120` frames) from `22.459 ms/f` (`Glow=2`, `83.0`
DC) to `36.599 ms/f` (`Glow=0`, `81.0` DC) due to the `worldDepth` feedback
loop on AMD `21.5.2`.

| Scene / Preset (`1366x705`, 180 frames) | Throughput (ms/f) | FPS | CPU p50 / p95 (ms) | GPU p50 / p95 (ms) | Wall p50 / p95 (ms) | Avg DC |
| --- | --- | --- | --- | --- | --- | --- |
| `world-stream-no-glow` | 7.273 | 137.5 | 6.106 / 10.000 | 6.381 / 9.485 | 6.999 / 10.003 | 70.8 |
| `world-mobs-no-glow` | 8.889 | 112.5 | 7.053 / 9.939 | 8.402 / 8.798 | 8.887 / 12.014 | 76.0 |
| `world-hud-no-glow` | 9.436 | 106.0 | 8.002 / 11.006 | 8.965 / 9.519 | 9.005 / 13.103 | 107.0 |
| `crowd-200-no-glow` | 15.542 | 64.3 | 15.002 / 18.006 | 15.082 / 18.474 | 15.552 / 18.620 | 80.8 |
| `crowd-200-geometry-low` | 14.427 | 69.3 | 13.463 / 17.455 | 13.472 / 18.347 | 14.007 / 18.004 | 72.8 |

Validation scope on this workstation: all Go unit tests (`go test ./...`), Go
`go vet ./...`, Python server tool tests (`server/tools`), Windows OpenGL 4.5
render/scene integration tests (11 `internal/render` tests and 3
`internal/scene` tests), and non-CGO Linux `go vet` for pure-Go packages passed
on Windows 10; native Linux runtime execution was not run in this environment.
In `go test -tags integration ./server/... -count=1`, untouched `HEAD`
(`de39442`) has a pre-existing failure in `server/internal/world`
(`TestClassicTeleportRuneDestinations`).
