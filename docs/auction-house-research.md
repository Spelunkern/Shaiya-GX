# Auction House research and implementation design

For the live GX implementation and operating rules, see [Auction House](auction-house.md).

GX can open one realm-wide, cross-faction market from any configured auction NPC. The client data already identifies the Notice Board. The original interface separates browsing, active business, settlement and watched listings. Gold and items can stay in the market after a listing closes, until their owner claims them.

## Evidence and versions

- Compared the installed GX loose data, the original US PS0107 client and the original EP5.4 client. Read `42.wld` with GX's WLD parser and `NpcQuest.SData` with its native NPC reader.
- Inspected and compared the auction textures in both originals. The decoded RGBA pixels of all sixteen matching sources are identical. Transparent padding on the four large panels leaves approximately 738 × 606 useful pixels in their 1024 × 1024 textures. These are texture measurements, while screen placement and hitboxes need a live reference pass.
- Inspected EP5.4 UI send sites, the local legacy market handler and market function disassembly, DB Agent procedure names and the existing DB Agent concern-list oracle. Function addresses below refer to the local reverse-engineering build, which has SHA-256 `d5c770a70375fc49bf6980f7eb6b87a07fb69763685a4f757a7fb5eda7742401`. A second local server reference corroborates the constants at the cited data addresses. Different executable revisions can change addresses and category IDs.
- Queried the GX fixture database read-only. No server, client or database migration was run for this research.

## NPC and access

All three copies of map 42 place NPC key `1/240` at the same four positions. The NPC definition is **Notice Board**, model 55, faction field 0, no shop products and **`MerchantType = 117`**. Type 1 is the merchant record group. The subtype is the service identifier. The later client/server reference headers also name 117 `AuctionBoard`.

| X | Y | Z |
|---:|---:|---:|
| 87.811 | 1.613 | 53.446 |
| 20.550 | 1.283 | 60.605 |
| 65.904 | 11.531 | 33.079 |
| 40.274 | 11.552 | 33.079 |

The legacy server configuration sets the market zone flag for map 42 at `0x42DB4B`. `PacketMarket` at `0x4862B0` checks that flag and then an independent market availability flag. GX should use `PS_GameDefs_NpcDefinitions.MerchantType` to identify the provider. Validate subtype 117 server-side on interaction and issue a session tied to that NPC entity, player's instance and distance. The current `useNpc` tracks NPC use and checks map membership. The auction route needs its own proximity, liveness and instance checks. Close the UI and invalidate the session when the player leaves range, changes instance, dies, logs off, or starts an incompatible interaction. Use the same market data for every valid session, with no market partition by map, instance or faction. Keep equipment usability as an optional search filter.

Use the existing field in GX Studio if this capability is exposed there. Show the human label **Auction Board** and the exact storage field `MerchantType = 117`; preserve the rest of the NPC definition.

## Interface inventory

| Surface | Controls and states | Local reference assets |
|---|---|---|
| Search Items | Expandable category tree, criteria, Search, paginated result list, item name/level/time remaining/price, Bid, Buy Now, Add to Interest, Search Item | `auction_objectsearch`, `auction_objectsearch_button`, `auction_buy_search_botton` |
| Auction Progress | Separate selling and bidding lists, Setup Auction, Cancel Auction, current bid/status and time remaining | `auction_progress`, `auction_progress_button` |
| Ended Auctions | Separate item and gold lists, result reason and claim expiry, Get Item, Get All, Collect Gold, Collect All | `auction_completio`, `auction_completion_button` |
| Interest List | Watched auctions, Refresh, View Details, Delete, Delete All | `auction_interest_list`, `auction_interestlist_button` |
| Shared tabs and category headers | Four tab states, Weapons, Armor, Accessories, Lapis and Misc group headings | `auction_top_manual_button`, `auction_completion`, `auction_completion ` |
| Setup Auction | Inventory drop/selection, item and quantity, starting price, buyout, duration, fee preview, submit and close | `auction_commission`, `auction_commission_button` |
| Bid | Current bid, buyout, minimum next bid, numeric input and confirmation | `auction_tender`, `auction_tender_enter_button` |
| Details | Item icon, properties and comparisons as applicable, equipped gems and craft data, listing terms | `auction_detailview` |

All asset names in the table have a `.tga` extension in the archives. The spelling `auction_completio.tga` is the ended panel. The two `auction_completion` variants with and without a trailing space are pixel-identical category art. Convert asset names and locations through the existing loose-PNG import convention if authorized for the local resource pack. Public source only needs code and evidence. Use existing GX inventory drag, quantity and tooltip components. Implement list scroll, clipping, row selection, input focus, hover/pressed/disabled states, and return to the same tab, search and scroll position after a modal. An inventory selection should remain in place until an explicit drag or confirmed posting. Display server-backed pending states and show changed prices before confirming again.

Group by native item category and equip requirements, rather than deriving categories from item names. The later reference enum includes weapon subclasses, armor slots, shields and cloaks, jewelry, lapis, lapisians, consumables and mounts. The later client added a vehicle market category, so build the EP4/EP5 mapping separately instead of adopting that enum wholesale. The legacy server exposes filtered search, lookup by exact base item key and a seller-name search helper. The regular search sends a short text field. The exact roles of that field, of each filter value, and of the seller-name helper in the EP5.4 UI merit a packet replay before finalizing controls. For GX use a documented filter model with bounded query size, stable ordering and server-side pagination.

## Rules recovered from the legacy server

| Operation | Observed behavior |
|---|---|
| Listing | Selects a bag, slot and nonzero count. The handler has whole-stack and split-stack paths. It checks durability/type and trade restriction fields before removing inventory. Starting price is at least 500 gold and at most 3,800,000,000. Buyout is required by this handler, greater than starting price and at most 4,000,000,000. |
| Duration and deposit | Duration slots 0–3 resolve to **7, 24, 72, 168 hours** at `0x56E5D4`. Deposit is `hours × 5 × ceil(starting price / 500000)`. A 600,000 starting price for 24 hours costs 240 gold. |
| Bid | Rejects the seller and the current high bidder. Checks wallet and debits the full new bid. The current amount starts at the listing's starting price. The minimum equals truncated **110% of the current amount** at `0x4437F8`–`0x443834`; even the first bid goes through this check. Bid must stay below buyout. Previous high bidder gets a gold result. |
| Buy Now | Charges buyout and closes the listing. Buyer receives an item result. Seller receives a gold result. The direct-buy path computes `sale price + deposit − floor(sale price × 2%)` at `0x443128`–`0x443182`. The prior bid also has a refund path. The buyout function checks the full wallet debit and refunds any prior bidder. The timed-sale branch at `0x44033A` also applies the 2% commission and returns the deposit. |
| Cancellation | Seller-owned path at `0x442B20` creates a return-item result for the seller. It explicitly handles an already-bid listing, returning the bidder's held gold as a gold result. A ban on canceling listings with bids would change this behavior. |
| Closing by time | Legacy periodic housekeeping checks listing deadlines. No-bid expiry creates a return-item result. A timed winner gets a claimable item, with a gold result for the seller. |
| Claiming | Items and money have independent queues. The shared results timestamp helper at `0x43FBD0` adds 1,209,600 seconds (**14 days**). Verify the later purge branches and each result's deposit handling before enabling irrecoverable expiry in GX. |
| Watch list | Watch-list state is per character. The legacy DB Agent concern state has ten market IDs and persists through the read/save concern procedures. Keep ten favorites per character for parity, with duplicate adds treated as idempotent by GX. |

An explicit product choice remains for expired unclaimed assets. Preserve deadlines and reasons in SQLite from the first implementation. Keep any discard policy gated until its original behavior is confirmed and cover it in recovery tests. The full inventory flag matrix, seller cancellation/unsold deposit treatment, watch-list behavior after settlement, and bulk-claim ordering are acceptance fixtures. Use integer gold, compute percentages with 64-bit integer intermediates, and check the GX wallet maximum separately from the listing price bounds.

## Wire trace

Addresses in this table are server-side function entry points unless otherwise stated. Meaning is established from dispatcher paths, operations and the EP5.4 sender sequence. This is the functional inventory. Reply shapes, every status value and exact original chunking require more fixtures before claiming legacy wire compatibility.

| Opcode | Observed operation |
|---|---|
| `0x1C01` | Filtered search, five selector bytes and a fixed-size text region. Calls `0x441CA0`. |
| `0x1C02` | Search by base item, with byte Type and TypeID combined as `Type × 1000 + TypeID`. Calls `0x441E20`. |
| `0x1C03` | Page/follow-up on a server-side search context. Byte selector at `0x442060`. |
| `0x1C04–05` | Active selling and bidding views. Calls `0x442140` and `0x4423D0`. |
| `0x1C06–07` | Ended item and gold views. Calls `0x442620` and `0x4427F0`. |
| `0x1C08` | Post listing with bag, slot, count, duration selector and two prices. |
| `0x1C09` | Seller cancellation by MarketID. |
| `0x1C0A` | Buy Now by MarketID. |
| `0x1C0B` | Bid by MarketID and amount. The low-bid error includes a refreshed minimum. |
| `0x1C0C–0D` | Claim item by MarketID or gold by MoneyID. EP5.4 submits row claims for bulk controls. |
| `0x1C0E–12` | Server result/update family. Direct seller payout uses `0x1C0E`, expiry branches use `0x1C0F` and `0x1C11`, and the refund/result paths emit `0x1C12`. A bid update uses `0x1C10`. |
| `0x1C13` | Server market-unavailable response, status 1 on the global availability guard. |
| `0x1C14–17` | Watch list, add, delete one, delete all. |

Keep legacy opcode meanings where tested. For independently developed GX client/server behavior, use the existing GX extension namespace for fields or actions that need a new shape, including a market-session acknowledgement, request IDs and explicit partial-bulk results. Validate packet sizes and selector bounds before decoding and ignore stale list replies when the tab/search context changes.

## SQLite design

The distributed fixture already includes five empty legacy tables: `PS_GameData_Market`, `PS_GameData_MarketItems`, `PS_GameData_MarketCharConcern`, `PS_GameData_MarketCharResultItems` and `PS_GameData_MarketCharResultMoney`, plus three empty market deletion logs. The legacy DB Agent loads market, result items and result money separately, with `usp_Market_ADD`, `usp_Market_Tender_UPDATE`, `usp_Market_DEL` and result `ADD/DEL/READ` procedures. Its watch list uses `usp_Read_Char_ConcernMarket_R` and the `usp_Save_Char_ConcernMarket_*` procedures. `PS_Billing_AuctionProductList` is a different table and belongs outside this migration.

Evolve the existing market namespace through a versioned SQLite migration. The fixture being empty does not establish the contents of another deployment. Migrate verified historical rows or fail without mutation if their ownership or state cannot be mapped.

| Table | Proposed responsibilities |
|---|---|
| `PS_GameData_Market` | MarketID, seller, immutable listing terms, current bidder and held amount, UTC timestamps, state and version. Preserve terminal state and settlement reason. |
| `PS_GameData_MarketItems` | Escrow snapshot of the full GX inventory instance. Preserve ItemUID, type, quantity, quality, gems, craft and any GX-specific instance fields. Carry a unique ownership key and transition it to claimant ownership only in the successful claim transaction. |
| `PS_GameData_MarketCharConcern` | Unique character/listing pair. Ownership lookup and cap check within the insert transaction. |
| `PS_GameData_MarketCharResultItems` | Owned item claim with reason, creation, deadline, resolution and escrow identity. One claim per item settlement. |
| `PS_GameData_MarketCharResultMoney` | Owned gold claim and ledger entry materialized from the corresponding event. Distinguish sale proceeds and bid refunds. Separate successive refunds to the same character on the same listing. |
| Market operation record | Unique character/request key for retry safety on writes, persisted with the result. Retain sale, deposit, fee and refunds in an auditable transaction history. |

Keep ownership and money changes behind `server/internal/db`. Extend the guarantees already used by `CommitGameplay` and player trade by committing wallet/inventory and the market effect together. Publish runtime wallet and inventory deltas only after commit and fence session save snapshots against the committed version. Reuse inventory stack-split/merge rules, assigning a fresh instance ID only where splitting requires it. A packet retry can read the prior result without repeating the economic effect.

Each listing mutation validates session, owner, state, deadline and expected version at commit. Opening two clients on the same listing must produce exactly one winning transition when one buys and the other bids, cancels or expires it. Replace the held bid and create its refund in the same transaction. On buyout or expiry create the claimable item and all credits atomically. On claiming, check bag fit or wallet capacity before marking the result claimed; leave failures retriable and give separate successes/failures for bulk operations. Prices and ownership always come from the persisted listing.

Index active state/deadline, seller, current bidder, category and exact item key, plus recipient/claim state and watch-list owner. Use server-side pagination with a bounded page size and a stable MarketID tie-breaker. Process due listings in bounded batches on startup and on a timer. Recheck the deadline at each mutation so timer lag creates no late-purchase window. Use targeted notifications for sellers and affected bidders, refresh inactive views when requested and keep database work off the frame loop.

## Implementation and acceptance

1. Add migration and repository operations in a disposable SQLite copy. Build fixture coverage for constants and status transitions, escrow ownership, gold conservation, and rollback with injected failures at each write.
2. Route NPC subtype 117 through a proximity-checked server interaction and build the four GX screens. Ship read-path and mutation code together with posting, browsing, bidding, buyout, cancellation, settlement, claims and favorites.
3. Exercise Windows/Linux input and layout behavior with small windows, dragging a stack, focus transitions, sorting/paging, keyboard confirmation and live result updates. Inspect original live geometry and message copy with an authorized reference environment if exact placement is needed.
4. Run two opposite-faction clients at auction NPCs on separate maps. Confirm same market and visibility from either NPC; reject a closed, distant, invalid or different-instance NPC. Confirm scope persists through logout and restart and survives competing requests.
5. Test no bid, many bids, a returning bidder, buyout with an active bid, timed settlement, cancel after bid, expired/current boundary, full bags, wallet cap, partial bulk claim, retry after disconnect, bound items, modified item attributes and stack splitting. Run a large populated market and verify indexed query plans, expiry batch bounds and restart recovery.

The statically recovered behaviors are sufficient to design the market and UI. The unverified result codes, purge behavior, category/filter mapping and live layout details should be tracked as specific parity checks. Preserve GX economic guarantees across those choices.

## Client control follow-up

The EP5.4 executable loads Search at `0x458935` with a 1024 × 1024 atlas,
Bid at `0x45BFBC` with 256 × 256, Setup at `0x45CA2E` with 256 × 256,
and Details at `0x45CFD6` with 256 × 512. These dimensions describe
textures or native window bounds at the load sites. The art uses separate
button state rows and transparent padding. The Input control has its own
texture, used from both Bid (`0x45C352`) and Setup (`0x45CB6A`,
`0x45CC6F`).

GX separates numeric entry from submission, offers the four durations
directly and uses separate visible selectors for the paired market lists.
Action controls share their fitted rectangles between drawing and input,
including at widths where the inventory reduces the available space.
Native list state, geometry and split-pane behavior still require further
comparison. The GX layout is visually distinct.

## EP5.4 market screens and GX list behavior

The original executable references the Progress, Ended and Interest atlases
at `0x4589CA`, `0x458A92` and `0x458B5E`. Progress has separate action
references at `0x453184` and `0x45329B`. Ended has four at `0x4536E4`,
`0x4537FF`, `0x453891` and `0x453945`, consistent with the two independent
claim queues and the individual and bulk actions. Cross-checking the extracted
EP5.4 manifest confirms separate backgrounds and state atlases for each screen.

GX displays Selling and Bidding together, and Item Claims and Gold Claims
together. Each list keeps its own selection and page. Requests still use the
existing view and sequence fields; selecting or paging one side does not
change the other side. Search keeps the GX category, name or seller query,
and level-range filters visible at narrow widths. Fine-grained legacy
category selectors and live screen measurements remain unverified.

## Dialog placement and presentation

The original Bid initializer stores 256 in both window size fields and
subtracts 128 from each viewport midpoint (`0x45BFAF–0x45BFE5`). Details
stores 256 × 512 and uses corresponding half-size offsets
(`0x45CFCB–0x45D000`). The Setup atlas is loaded as 256 × 256 at
`0x45CA2E`, while the adjoining code derives an X coordinate from a related
panel's X plus 256 and copies its Y. Texture dimensions alone therefore
cannot establish every dialog's position.

GX keeps the market visible behind separate forms for setup, bid, buyout,
cancellation and details. Its forms size to their content and share the
same rectangles for rendering, numeric input and inventory drops. Gold
values, secondary labels and warnings have distinct colors. The form
colors and coordinates are GX choices.

## Search screenshot comparison

The original reference shows the category rail and listing grid as independent
regions, separate faction and quality selectors, and a player gold display.
The original search background also has baked row rules. Scaling a narrow,
off-center sample of that image across the whole GX panel visibly stretched
those rules and the illustration behind them.

GX now preserves frame corners at source size, keeps the same scale on both
axes for button art, and draws the list rules in the same geometry used for
selection and wheel paging. The category rail uses the supplied category
atlas. Search and setup reuse their supplied state atlases. The panel grows
vertically when the viewport permits. The detailed hierarchy still needs verification.

## Item family filtering in GX

The GX category rail expands the selected group into its supported native
item families. Each child maps to catalog item types, including variant type
IDs, and is applied before search pagination. The optional GX request field
does not require a SQLite migration. The labels and family list are GX choices
based on the current catalog. They do not establish the complete EP5.4 menu.
## EP5.4 FR auction requests

The [OpenShaiya EP5.4 FR client](https://archive.openshaiya.org/unpacked_clients.html)
has SHA-256 `f8aba8ca504735e6cedaeac02f870ce0990306f83b6ae4cfcc658246c2c33319`.
The addresses below are virtual addresses for that image, loaded at
`0x400000`. The 740 × 600 size is written at `0x4594FD` and `0x459504`.
Search initializes the faction checkboxes at relative coordinates
`(0x23, 0x43)` and `(0x56, 0x43)`; both are checked at `0x45431D–0x45432A`.
The opcode sites below call the common sender at `0x596830`. Lengths are
payload lengths, excluding the two-byte frame length and two-byte opcode.
Multi-byte fields are little-endian.

| Request | FR client opcode site | Payload |
|---|---|---|
| `0x1C01` | `0x459972` | 37 bytes. Faction, minimum level, maximum level, category and name grade (one byte each), followed by 32 bytes of NUL-terminated text. |
| `0x1C02` | `0x459A24` | 2 bytes. Item Type and TypeID. |
| `0x1C03` | `0x45999C`, `0x4599D9` | 1 byte. Page selector 1 or 2 on the existing search. |
| `0x1C04–07` | `0x458215`, `0x458239`, `0x4582E1`, `0x458305` | Empty. Selling, bidding, ended items and ended gold, respectively. |
| `0x1C08` | `0x45A472` | 12 bytes. Bag, slot, count and duration (one byte each), then starting price and buyout (u32 each). |
| `0x1C09–0A` | `0x45A2B2`, `0x45A2D8` | Market ID (u32). |
| `0x1C0B` | `0x45A3C3` | Market ID and bid amount (u32 each). |
| `0x1C0C–0D` | `0x459D20`, `0x459DF3` | Item result's market ID or gold result's money ID (u32). |
| `0x1C14–17` | `0x459E0F`, `0x459BFF`, `0x459E4C`, `0x45A28C` | Watch view and clear have empty payloads. Add and remove each send a Market ID (u32). |

Search assembles the faction selector at `0x4598D1–0x459904`. Equal
checkbox states send 0, Light alone sends 1, and Fury alone sends 2.
Level selections are multiplied by ten, with an unset maximum encoded
as `0xFF`. The category lookup at `0x4527D0` returns 0 without a selected
family. Grade selections 0–5 send their index, index 6 sends grade 7,
and index 7 sends `0xFF` for an unrestricted grade search.

A local legacy `ps_game.exe` with SHA-256
`48e8688c24b0d8fde241015a87342ebde14b4e8b48c793152530ffe214581f8e`
corroborates the field order at `0x486315–0x486335`. The search matcher at
`0x440E6B–0x440EEC` reads the item definition. Light excludes country
values 3–5, Fury excludes 0–2, and Type 30 bypasses the faction check.
An explicit name grade compares the byte at item definition offset
`0x38`, the low byte of `ReqDex`. The initial matching search reply
at `0x440FCD–0x440FF9` uses a status byte, a more-results byte and a
row count followed by 48-byte rows. Further result codes and layouts
still need their own verification.

GX carries the confirmed faction and name-grade selectors as optional
fields in `0x7F1F`. It reads both values from the item catalog before
paginating listings, including existing listings. The legacy layouts
above were recovered statically.
