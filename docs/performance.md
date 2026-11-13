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
