# Run in PowerShell with Go and an x64 MinGW-w64 gcc on PATH.
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
$env:CGO_ENABLED = "1"
New-Item -ItemType Directory -Force build,server/build | Out-Null
function Check-Go { if ($LASTEXITCODE -ne 0) { throw "Go command failed: $LASTEXITCODE" } }
go vet ./...
Check-Go
foreach ($program in @("shaiya", "shaiyacli", "assets")) {
    go build -o "build/$program.exe" "./cmd/$program"
    Check-Go
}
Push-Location .
try {
    & ./server/tools/build.ps1
    Check-Go
} finally {
    Pop-Location
}
go test ./...
Check-Go
