# Sector culling, asynchronous streaming and static LOD

These are local development changes. Published releases and Drive packages have
not been updated automatically.

## Visibility sectors

Static placements have small integer resource handles and cached world bounds.
64-unit spatial sectors contain the **complete bounds** of their assigned objects,
including the full swept sphere of animated rotors. Whole sectors can be rejected
before individual object tests. Large objects crossing sector borders remain
covered; native placement/material ordering is retained in the fine pass.

In a map-1 comparison, 16,917 resident placements required 895 individual tests
for one view. Across 32 sampled views on maps 1, 2 and 19, sector culling and
asynchronous loading preserved full-detail images byte for byte.

## Asynchronous field loading

Live network gameplay enables two CPU workers for file reads, texture decoding,
SMOD/VANI parsing, texture-name indexing and LOD preparation. At most four jobs
are in flight; new model requests pause while decoded models await dependency
uploads. GPU operations and resource-cache ownership stay on the render thread.
Deterministic offline previews/imports retain synchronous loading by default.

The render thread aims to spend 2 ms per update on asset uploads/preparation,
then updates terrain batches and residency caches. This is a **soft budget**:
a single OpenGL operation cannot be preempted. Generating missing mipmaps for
some 1024×1024 DXT3 lightmaps still takes about 20 ms during initial loading.
No authored mip levels are replaced or discarded to hide this cost.

Camera-cell changes invalidate stale requests by generation. Map close cancels
workers and waits for them before the archive/resources can be released.
Resources needed by pending cells are pinned; missing-texture negative cache
entries are preserved so missing optional data cannot permanently stall a cell.

Residency rebuilding uses reusable placement/resource marks and integer handles,
rather than allocating and hashing a file path for every placement. Returning
along a route restores the same GPU residency and closing releases its storage.

A map-1 round trip measured GL-thread streaming work:

| Metric | Previous synchronous traversal | Async plus residency improvements |
| --- | ---: | ---: |
| Mean update | 2.918 ms | 0.792 ms |
| Maximum update | 12.532 ms | 3.226 ms |
| Updates needed | 85 | 224 |
| GPU residency at start / after returning | 45.5 MiB / same | 45.5 MiB / same |
| Peak GPU residency | 49.8 MiB | 49.9 MiB |

The additional updates distribute work; this is a frame-latency improvement,
not a claim of faster total initial loading. Worker/temporary allocations can
increase. The benchmark's initial texture load is excluded from this route.
A stationary map-1 frame benchmark measured 0.718 → 0.495 ms at 640×360 with
159 draw calls in both cases. These local RTX 3050 measurements are not universal
FPS guarantees or isolated GPU timings.

## Conservative distant-object LOD

Parsed SMOD meshes can produce two clustered variants (grids 32 and 16) on a CPU
worker. Only meshes with at least 64 triangles and at least 10% reduction keep
a variant. Position displacement is measured against original vertices; normal
and UV buckets limit merging across attribute changes. Original mesh data and
colliders remain intact.

Selection uses projected error rather than a fixed world distance. A variant
enters below 0.35 pixels of conservative position error and is retained up to
0.45 pixels to provide hysteresis. This state belongs to the placement and
survives residency-cache rebuilds. Near geometry uses full detail. Characters,
terrain, animated rotors and VANI are excluded; shadows and reflection passes
use original geometry. Original parts remain available when a part has no LOD.

This first trial is deliberately modest: 32 sampled views reduced a total of
3,607 submitted triangles over 58 LOD placements, with no changed pixels in those
captures. The original low-poly resources offer limited reduction under this
error budget; no substantial FPS benefit is attributed to LOD. It is not a
billboard/impostor system or a general-purpose asset optimizer.

## Validation

- `TestAsyncSectorsAndDistantLODGL`: synchronous/async image equality, sector
  tests, LOD geometry/image comparisons, moving residency and cancellation.
- `TestTerrainStreamingRouteGL` with `SHAIYA_TEST_ASYNC_STREAM=1`: round-trip
  timings, bounded GPU residency and release on close.
- Race tests: worker decoding without GL, generation identity, shutdown under
  backpressure and pinned missing-texture readiness.
- Math tests: LOD reduction/error, original-data preservation and pass selection.
- Full render/scene/UI graphics tests and isolated N2, including a paid map change.

Private measurements and captures are retained under `work/stream-lod-*`.
