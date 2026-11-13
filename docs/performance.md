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
