# GX documentation

Start with [first run](quickstart.md), then [building and testing](building.md).

| Guide | Contents |
|---|---|
| [Portable client](client-portable.md) | Complete Drive download for Linux/Windows, runtime requirements and executable updates |
| [Data formats](data-formats.md) | Loose data layout, PNG and Ogg conversion |
| [First run](quickstart.md) | Windows/Linux downloads, verification, accounts and local play |
| [Build and test](building.md) | Toolchains, native/cross builds, CI, integration and packaging |
| [Server operations](server.md) | Micro app, console, LAN, logs, backup and upgrades |
| [Architecture](architecture.md) | Single Go module, client/server boundaries and state ownership |
| [Interface](interface.md) | Implemented behavior, parameters and remaining work |
| [Protocol](protocol.md) | Wire compatibility and server authority |
| [Weapon enchantments](weapon-enchantments.md) | Local/remote equipment presentation and validation |
| [Server storage](../server/docs/sqlite.md) | Embedded development DB and future replacement |
| [Portable distribution](../server/docs/distribution.md) | Archive contents, verification and publishing |
| [Server parity](../server/docs/original-parity.md) | Recovered rules and known differences |
| [Social service](../server/docs/social.md) | Friends, blocks and party search |
| [Contributing](../CONTRIBUTING.md) | Scope, review, evidence and data policy |
| [Agents](../AGENTS.md) | Automation and repository instructions |

Server data and the explicitly requested test database are distributed fixtures. Client archives and private workstation research/captures remain outside the source tree. Rapture is incorporated as GX Server, not an external project dependency.
