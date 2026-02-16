$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$goRoot = Join-Path $repoRoot ".go"
$env:GOPATH = $goRoot
$env:GOMODCACHE = Join-Path $goRoot "pkg\mod"
$env:GOCACHE = Join-Path $goRoot "buildcache"

New-Item -ItemType Directory -Force -Path $env:GOPATH | Out-Null
New-Item -ItemType Directory -Force -Path $env:GOMODCACHE | Out-Null
New-Item -ItemType Directory -Force -Path $env:GOCACHE | Out-Null

Set-Location $PSScriptRoot
& 'C:\Program Files\Go\bin\go.exe' build -o bin\mazebot.exe .

