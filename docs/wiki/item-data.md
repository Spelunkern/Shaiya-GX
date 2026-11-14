# Item data and compiled rules

## Sources of authority

| Data | Authoritative source | Applying a change |
|---|---|---|
| Enchant cap, weapon/armor rates and stat bonuses | `server/internal/world/enchant_rules.go` | Rebuild and restart world |
| Native forge gold table | `lapisiaMoney` in `server/internal/world/crafting.go` | Rebuild and restart world |
| Linking/extraction formulas | `server/internal/world/forge_quote.go` | Rebuild and restart world |
| Item requirements, hammer ReqVg, stack limits | SQL `PS_GameDefs_Items` in the selected server DB | Restart world |
| Item names, descriptions, icons and native flags shown by client | External `data/Item/Item.SData` | Restart client |
| Client stackability matching SQL | `data/item-stack-limits.csv` | Restart client |
| Mall display / actual products | Client Cash.SData / SQL ProductList | Restart affected component |

The retired `ItemEnchant.ini` is not read or required. A historical copy lives
under `server/docs/reference/ItemEnchant.legacy.ini`; changing it has no gameplay
effect. There is no text or environment override for the compiled enchant rules.
The old free-cost environment setting is also ignored.

The server quote is read-only. It uses the same functions as execution; the
client checks request serials and inventory snapshots before accepting a reply.
Selecting a gray material hint never grants an item or authorizes an operation.

## Overloaded native fields

- **ReqIg:** selects the native gold table for lapis and lapisias. For lapis it
  also participates in the linking formula; it does not control lapisia chance.
- **ReqVg on Special 36:** added percentage points for a linking hammer.
- **ReqVg on Special 58:** direct extraction success percentage.
- **ReqVg on lapisias:** no equipment-destruction behavior. Current content sets
  it to zero and describes the safe behavior.
- **Slot on equipment:** total lapis capacity, with six physical storage entries.
- **Slot on a lapis:** capacity consumed by that gem. It is not a socket owned by
  the lapis; lapis/lapisia tooltips do not draw their own socket rows.
- **ReqLuc on equipment:** packed enhancement material ID, type*1000+type ID.
- **ReqWis / Server:** recreation maximum / number of distinct rolled stats.

Native target flags on the lapis/lapisia determine compatible equipment families:
weapon families require nonzero ReqLevel; family16 uses Country; families17–22
use their respective class flag; family23 uses Grow; family24 is accepted;
family25 uses ReqStr. These overloaded fields do not introduce character-level
or character-mode requirements for forge attempts.

## Current hammer values and definition migration

The exact shipped changes, including item keys and matching English descriptions,
are in [`tools/data/forge-item-patches.csv`](../../tools/data/forge-item-patches.csv).
It includes both Type44 and Type100 variants of numbered linking hammers,
common/operator hammers, extraction hammers and lapisias. Lapisias now stack to
255; existing player quantities are not rewritten or automatically merged.

`cmd/itempatch` applies that CSV to a native Item.SData, preserving every unrelated
record byte. It supports encrypted input and writes a plaintext native table
accepted by the client. Use a distinct output file, verify it, then install it
atomically with the client stopped. Original archives and client assets are not
part of the source repository.

`tools/apply_forge_rules.py` applies only the specified SQL ReqVg and Count fields,
creates SQLite backups in an explicit directory, updates a seed manifest when
present and exports matching stack limits. Stop the affected services first.
It never replaces a personal runtime with the seed. The distributed seed and an
existing personal runtime are separate targets and must be named explicitly.

## Verification

From the repository root with the documented Go toolchain:

```sh
go test ./...
go vet ./...
go test -tags integration ./server/... -count=1
```

The forge regressions cover exact costs, level/Vg independence, additive hammers,
compiled cap/rates, safe enchant failures, selective extraction, repeated
consumption, original SysMsg refusals and valid drop rolls surviving pickup and
redrop. `tools/forge_smoke.py` additionally runs the continuous client workflow
against isolated processes and a disposable DB. It creates temporary fixture
accounts; it does not log into a personal character.

For the isolated client workflow, first build native login/world binaries, then
provide the external client data explicitly:

```sh
python3 tools/forge_smoke.py --data "/path/to/client/data" --item-data "/path/to/client/data/Item/Item.SData"
```

It uses ports 41800/41810 and refuses to start if either is occupied. Fixture
accounts and all test mutations belong only to a temporary copy of the seed.


Targeted recreation properties are implemented in
`server/internal/world/recreation_runes.go`. ReqVg 101–109 selects individual
random stats; 201–209 selects perfect stats. Type101 IDs1–18 use those properties.
The client now retains native Server to determine eligibility for its Max OJ
footer. Keep it and ReqWis aligned with SQL. See [Recreation](recreation-and-loot.md).
