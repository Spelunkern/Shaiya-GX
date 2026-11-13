# Builds gx_login, gx_world and gx_console into build\, then
# the gx_server window, which needs cgo (gcc from MSYS2/TDM-GCC on
# PATH). A missing GUI compiler is a build failure.
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
$previousCGO = $env:CGO_ENABLED
try {
    $env:CGO_ENABLED = "0"
    go build -o build\ ./cmd/gx_login ./cmd/gx_world ./cmd/gx_console
} finally {
    if ($null -eq $previousCGO) { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue }
    else { $env:CGO_ENABLED = $previousCGO }
}
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$env:CGO_ENABLED = "1"
go build -ldflags "-H windowsgui" -o build\ ./cmd/gx_server
if ($LASTEXITCODE -ne 0) { throw "GX Server micro app build failed" }
