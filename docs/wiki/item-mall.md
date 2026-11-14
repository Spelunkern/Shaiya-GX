# Item Mall catalog

The client reads `data/Cash.SData`; the world server reads
`PS_GameDefs_ProductList` from its configured database at startup. SQL determines
prices and the items actually purchased or gifted. Keep product codes, prices
and all 24 bundle slots aligned when editing the two catalogs. Restart the client
and world after changing their respective data. CSV is no longer runtime input;
there is no catalog hash handshake.

Cash.SData retains its native binary layout: product count (uint32), followed by
index/category/icon/price (four uint32 values), 24 item ID (uint32) and quantity
(uint8) pairs, and name/code/description (each byte length uint32 plus text).
Encrypted original SData and plaintext binary tables are supported. Categories
1–4 are visible tabs; category 0 preserves hidden legacy products. Codes fit
20 ASCII bytes. Packed item IDs use type * 1000 + type ID.

The local testing catalog retains all 300 products, including the added supplies
and kits; 216 are visible and every price is zero. Stackable supplies deliver up
to 255 per occupied bundle slot; non-stackable items remain single. Purchase,
gift and claim storage remain authoritative in SQL. Native inventory icons are
used when a testing product has no authored mall icon.

The maximum item stack is 255. The server definition export
`item-stack-limits.csv` preserves which items stack on the client. Quantity
packets use native opcodes and one-byte counts. Internal arithmetic may use
wider integers to avoid overflow; login revision `0x11122202` requires matching
updated client and server binaries.

`tools/restore_native_cash.py` is a one-time importer for a former CSV catalog.
Use explicit stopped database paths and a client data directory. It writes the
native Cash.SData table and matching SQL ProductList, caps catalog entries at
255, exports stack definitions, and splits existing carried stacks above 255
without losing items. It aborts when there is insufficient room or unsupported
oversized storage. Keep backups before running it. It never replaces a personal
database with the distributed seed. The normal asset importer copies Cash.SData
without converting it to CSV.

Lapis and lapisias all have a 255 definition stack limit. Every occupied mall
bundle slot containing either family delivers 255. The catalog also contains
18 targeted recreation runes, each delivered as 255; see the recreation guide.
