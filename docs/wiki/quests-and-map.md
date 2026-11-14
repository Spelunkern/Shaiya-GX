# Quests and map markers

Available quest givers and ready turn-in locations are shown across the entire
current map. They do not depend on which NPCs are inside the player's 3D view.
The small minimap still clips to its displayed area; the expanded map can show
locations across the map, subject to its current zoom and pan.

## Lightweight map catalog

On map entry, GX Server sends quest-bearing NPC locations as metadata. It does
not spawn distant NPC entities, load their models, extend visibility, or allow
remote quest interaction. Multiple placements of the same NPC definition remain
separate locations. Patrolling NPCs use their authored first waypoint as a stable
map anchor, rather than streaming distant patrol positions continuously.

The server shares one immutable catalog per map. Normal movement and animation
never rebuild or resend it. Adding quest-bearing NPCs invalidates the catalog;
adding a non-quest NPC does not resend an identical result. A session checks only
the map's revision during its normal loop. No periodic client polling is used.

## When availability changes

The client caches the filtered quest markers and reuses them while drawing.
Availability is rebuilt after changes to:

- The character, map, level, or received map catalog.
- Active quests, counters, completion, abandonment or prerequisites.
- Inventory readiness or relevant quest-item quantities in carried bags.
- A known quest minimum-duration deadline, once that deadline is reached.

Picking up an unrelated item does not rebuild quest availability. Moving or
splitting a quest-item stack between carried bags also does not rebuild it if
the aggregate quantities are unchanged. Equipment changes only matter if they
change the carried quantity of an item required by a quest. Cached rendering
reuses marker storage and does not scan all NPC quests every frame.

A deadline is scheduled at the exact next relevant minimum-time transition.
There is no repeated eligibility scan while waiting, and an expired deadline
does not keep triggering rebuilds. Missing timing snapshots retain the existing
presentation hint; the server still validates actual turn-in conditions.

Quest eligibility remains a client presentation hint based on quest definitions
and confirmed character/progress/inventory state. Starting and finishing quests
still use the native server requests and the real NPC interaction requirements.
The map marker is not an authorization to accept a quest remotely.

Party positions and nearby creature-objective indicators remain live overlays.
Their movement does not invalidate the cached map-wide NPC quest availability.
Quest markers remain individually present with stable paint order, including
nearby overlapping markers.

## GX protocol

`0x7F04` is a map quest-NPC catalog chunk:

| Field | Type |
|---|---|
| Map ID | `u16`, rendered/base map ID |
| Flags | `u8`: begin = 1, end = 2 |
| Record count | `u16` |
| Each record | NPC type `u8`, definition ID `u16`, X `f32`, Z `f32` |

All numeric fields are little-endian. Chunks stay within native frame limits.
The client assembles a replacement snapshot and publishes it only when the end
flag arrives, so a large catalog does not flicker into view in partial batches.
A delayed catalog from a previous map is ignored. Nonfinite positions, malformed
lengths, invalid flags and unexpected fragments are rejected. The metadata does
not contain interactive entity IDs and never creates 3D entities.

Matching GX client/world builds provide the map-wide catalog. During initial
loading, or with an older world without the extension, the client retains its
previous nearby-only presentation until catalog metadata is available.

## Verification

Regression tests cover distant quest givers without visible NPC entities,
atomic multi-chunk catalogs, map changes, progress and level invalidation,
relevant versus irrelevant inventory changes, and a minimum-time transition
without sleeping. A steady cached marker query performs no allocations in the
benchmark fixture. The encrypted login/world integration checks a real map's
catalog against its much smaller visible NPC set and verifies that movement
neither resends the catalog nor rebuilds the eligibility cache.
