#!/usr/bin/env bash
# Builds the Linux binaries into build/. GOOS=windows is not enabled yet.
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$HOME/.local/opt/go-1.27.3/bin:$PATH" GOPATH="${GOPATH:-$HOME/.local/share/go}" GOTOOLCHAIN=local
go vet ./...
for c in shaiya shaiyacli assets; do
  nice go build -o "build/$c" "./cmd/$c"
done
go test ./...
