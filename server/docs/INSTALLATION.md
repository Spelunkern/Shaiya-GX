# GX Server installation

GX Server is now part of the parent Shaiya GX repository. Follow the maintained guides:

- [First run on Windows and Linux](../../docs/quickstart.md): downloads, hashes, account creation and local play.
- [Building and testing](../../docs/building.md): native Windows/Linux toolchains and cross-compilation.
- [Operation](../../docs/server.md): micro app, console, LAN, logs, persistence, backup and updates.
- [Portable archives](distribution.md): package layout and reproduction.
- [Development database](sqlite.md): snapshots and the future database change.

The micro app says **GX Server**; its functionality is unchanged. The server ships the requested development database, not an empty production database. SQLite's embedded setup is used for development convenience and will be replaced by another backend later.
