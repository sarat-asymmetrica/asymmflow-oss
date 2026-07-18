param(
    [switch]$SkipWailsBuild,
    [switch]$SkipNSIS,
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
Push-Location frontend-lab
npm run build

Step "Frontend check"
npm run check
Pop-Location

if (-not $SkipWailsBuild) {
    if ($SkipNSIS) {
        Step "Wails build"
        wails build
    } else {
        # DP2 G5: build the full NSIS installer. Stage the payload (seed +
        # identity) into build/windows/installer/payload/ first, then build with
        # -nsis. makensis must be on PATH; add the default NSIS location if the
        # caller has not already.
        Step "Stage installer payload"
        & (Join-Path $PSScriptRoot "stage_installer_payload.ps1")

        Step "Wails build (NSIS installer)"
        $nsisDir = "C:\Program Files (x86)\NSIS"
        if ((Get-Command makensis -ErrorAction SilentlyContinue) -eq $null -and (Test-Path (Join-Path $nsisDir "makensis.exe"))) {
            $env:Path = "$nsisDir;$env:Path"
        }
        if ((Get-Command makensis -ErrorAction SilentlyContinue) -eq $null) {
            throw "makensis (NSIS) not found on PATH; install NSIS or pass -SkipNSIS"
        }
        wails build -nsis
    }
}

$elapsed = (Get-Date) - $started
Write-Host ""
Write-Host ("Release verification passed in " + $elapsed.ToString("hh\:mm\:ss"))
