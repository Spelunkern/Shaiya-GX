#!/bin/sh
# Builds gx_login, gx_world and gx_console into build/, then
# the GX Server window, which needs a C compiler and graphics headers.
set -e
repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"
if ! command -v go >/dev/null 2>&1; then
    for candidate in "$HOME"/.local/opt/go-*/bin /usr/local/go/bin; do
        [ -x "$candidate/go" ] && PATH="$candidate:$PATH" && break
    done
fi
CGO_ENABLED=0 go build -o build/ ./cmd/gx_login ./cmd/gx_world ./cmd/gx_console
CGO_ENABLED=1 go build -o build/ ./cmd/gx_server
