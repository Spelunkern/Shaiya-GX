#!/usr/bin/env bash
# Build and check the integrated Linux client and GX Server.
set -euo pipefail
cd "$(dirname "$0")/.."
if ! command -v go >/dev/null 2>&1; then
  for candidate in "$HOME"/.local/opt/go-*/bin /usr/local/go/bin; do
    if [ -x "$candidate/go" ]; then export PATH="$candidate:$PATH"; break; fi
  done
fi
export GOTOOLCHAIN=local
mkdir -p build server/build
go vet ./...
for c in shaiya shaiyacli assets; do
  if [ "$c" = shaiya ]; then target="Shaiya GX"; else target="$c"; fi
  go build -o "build/$target" "./cmd/$c"
done
./server/tools/build.sh
go test ./...
