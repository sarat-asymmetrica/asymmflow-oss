param(
    [string]$ActiveDb = "ph_holdings.db",
    [string]$BackupDir = "",
    [string]$BackupPath = "",
    [string]$RestoreSandboxRoot = "release_artifacts\restore-preflight",
    [switch]$SkipSandboxRestore
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

function Step($Name) {
    Write-Host ""
    Write-Host "== $Name =="
}

function Resolve-ExistingPath($Path, $Label) {
    $resolved = Resolve-Path -LiteralPath $Path -ErrorAction Stop
    if (-not (Test-Path -LiteralPath $resolved -PathType Leaf)) {
        throw "$Label is not a file: $resolved"
    }
    return $resolved.Path
}

function Invoke-Native($Description, $Command, $Arguments) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Description failed with exit code $LASTEXITCODE"
    }
}

$activeDbPath = Resolve-ExistingPath $ActiveDb "Active database"

if ([string]::IsNullOrWhiteSpace($BackupPath)) {
    if ([string]::IsNullOrWhiteSpace($BackupDir)) {
        $BackupDir = Join-Path (Split-Path -Parent $activeDbPath) "backups"
    }

    $backupDirPath = Resolve-Path -LiteralPath $BackupDir -ErrorAction Stop
    $latestBackup = Get-ChildItem -LiteralPath $backupDirPath -File -Filter "ph_holdings_*_*.db" |
        Sort-Object LastWriteTimeUtc -Descending |
        Select-Object -First 1

    if ($null -eq $latestBackup) {
        throw "No backup files matching ph_holdings_*_*.db found in $backupDirPath"
    }

    $BackupPath = $latestBackup.FullName
}

$backupDbPath = Resolve-ExistingPath $BackupPath "Backup database"

Step "Active database integrity"
Invoke-Native "Active database integrity" "go" @("run", "./cmd/sqlite_integrity", "-db", $activeDbPath)

Step "Backup database integrity"
Invoke-Native "Backup database integrity" "go" @("run", "./cmd/sqlite_integrity", "-db", $backupDbPath)

if (-not $SkipSandboxRestore) {
    Step "Sandbox restore copy"
    $sandboxRootPath = Join-Path $repoRoot $RestoreSandboxRoot
    New-Item -ItemType Directory -Force -Path $sandboxRootPath | Out-Null
    $restoreCopy = Join-Path $sandboxRootPath "restore-test.db"
    Copy-Item -LiteralPath $backupDbPath -Destination $restoreCopy -Force
    Invoke-Native "Sandbox restore integrity" "go" @("run", "./cmd/sqlite_integrity", "-db", $restoreCopy)
    Write-Host "Sandbox restore copy verified: $restoreCopy"
}

Write-Host ""
Write-Host "Backup/restore preflight passed."
