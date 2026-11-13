#!/bin/sh
# Opens the GX Server Console ("start server" / "shutdown server"), building
# the binaries first if needed.
set -e
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"
[ -x build/gx_console ] || "$repo_root/tools/build.sh"
exec "$repo_root/build/gx_console"
