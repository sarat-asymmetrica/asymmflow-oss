param(
    [switch]$SkipWailsBuild,
    [int]$GoTestTimeoutSeconds = 300
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

function Step($Name) {
    Write-Host ""
    Write-Host "== $Name =="
}

$started = Get-Date

Step "Go build"
go build ./...

Step "Go tests"
go test ./... -count=1 -timeout "${GoTestTimeoutSeconds}s"

Step "Frontend build"
Push-Location frontend
npm run build

Step "Frontend check"
npm run check
Pop-Location

if (-not $SkipWailsBuild) {
    Step "Wails build"
    wails build
}

$elapsed = (Get-Date) - $started
Write-Host ""
Write-Host ("Release verification passed in " + $elapsed.ToString("hh\:mm\:ss"))
