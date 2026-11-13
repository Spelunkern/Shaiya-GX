# Opens the GX Server Console ("start server" / "shutdown server"), building
# the binaries first if needed.
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
if (-not (Test-Path "build\gx_console.exe")) { & "$PSScriptRoot\build.ps1" }
& "build\gx_console.exe"
