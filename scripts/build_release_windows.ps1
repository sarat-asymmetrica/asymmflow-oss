param(
    [string]$Version = "0.1.0-alpha.1",
    [string]$Channel = "alpha",
    [string]$OutputRoot = "release_artifacts"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$commit = (git rev-parse --short=12 HEAD).Trim()
$dirty = if ((git status --porcelain).Trim()) { "true" } else { "false" }
$buildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$artifactName = "AsymmFlow-$Version-windows-amd64"
$artifactDir = Join-Path $OutputRoot $artifactName

New-Item -ItemType Directory -Force -Path $artifactDir | Out-Null

Write-Host "== AsymmFlow Windows Release Build =="
Write-Host "Version: $Version"
Write-Host "Channel: $Channel"
Write-Host "Commit:  $commit"
Write-Host "Dirty:   $dirty"

go test ./pkg/infra/release -count=1
go build ./...
Push-Location frontend
npm run build
npm run check
Pop-Location

$ldflags = @(
    "-X ph_holdings_app/pkg/infra/release.Version=$Version",
    "-X ph_holdings_app/pkg/infra/release.Commit=$commit",
    "-X ph_holdings_app/pkg/infra/release.BuildTime=$buildTime",
    "-X ph_holdings_app/pkg/infra/release.Dirty=$dirty"
) -join " "

wails build -platform windows/amd64 -ldflags $ldflags

Copy-Item build/bin/AsymmFlow.exe $artifactDir/
Copy-Item pkg/infra/release/manifest.json $artifactDir/
Copy-Item docs/V0_1_RELEASE_ROADMAP_2026_05_08.md $artifactDir/
Copy-Item docs/RELEASE_CHECKLIST_V0_1.md $artifactDir/
Copy-Item docs/BACKUP_RESTORE_PREFLIGHT_V0_1.md $artifactDir/
Copy-Item scripts/preflight_backup_restore.ps1 $artifactDir/

@{
    version = $Version
    channel = $Channel
    commit = $commit
    dirty = [bool]::Parse($dirty)
    build_time = $buildTime
    artifact = $artifactName
} | ConvertTo-Json -Depth 4 | Set-Content (Join-Path $artifactDir "build-info.json")

Compress-Archive -Path (Join-Path $artifactDir "*") -DestinationPath (Join-Path $OutputRoot "$artifactName.zip") -Force
Write-Host "Release artifact: $(Join-Path $OutputRoot "$artifactName.zip")"
