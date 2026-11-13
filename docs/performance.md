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

Displayed values refresh approximately every 250 ms. FPS and timings are averaged over that interval; memory and draw-call counters are sampled when the display refreshes. GPU timestamps are read only when available, avoiding a forced GPU wait. CPU and GPU overlap, so adding their times does not give total frame time.

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

DUN worlds disable world fog. Their WLD distance is retained for the projection
far plane, so disabling fog does not silently shorten the previous draw distance.
Field and underwater rendering retain their existing settings.

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
