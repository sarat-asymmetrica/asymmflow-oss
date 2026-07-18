# Stage the installer PAYLOAD (seed + identity) that the NSIS script Files into
# the code / identity planes. Run BEFORE `wails build -nsis`.
#
# The payload lives in build/windows/installer/payload/ -- a sibling of
# project.nsi that the wails executable build never touches, so it survives the
# build->package sequence. It is gitignored (build output; the .db carries zero
# client data by construction but stays out of the repo per invariant 4.6).
#
# Contents produced:
#   payload/ph_holdings.db  - fresh SYNTHETIC seed (schema-complete, 0 business
#                             rows) from the seedgen provisioner. Installed to
#                             the code plane $INSTDIR\data; first boot copies it
#                             into an absent data plane (-> seeded_fresh).
#   payload/overlay.json    - deployment identity overlay, seeded IF-ABSENT into
#                             the identity plane by the installer. REQUIRED (the
#                             NSIS File for it is not /nonfatal).
#   payload/ssot/*          - letterhead artwork, seeded IF-ABSENT into the
#                             identity plane. OPTIONAL (substrate ships none; the
#                             fork ships PH letterheads). NSIS uses /nonfatal.
#
# The fork overrides overlay.json + ssot with PH identity via its own
# data/overlay.json + data/ssot before running this script; nothing else differs.

param(
    [string]$OverlaySource = "data/overlay.json",
    [string]$SsotSource    = "data/ssot"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$payload = Join-Path $repoRoot "build/windows/installer/payload"
New-Item -ItemType Directory -Force -Path $payload | Out-Null

Write-Host "== Staging installer payload =="

# 1) Fresh synthetic seed via the seedgen provisioner (writes payload/ph_holdings.db).
Write-Host "-- Provisioning fresh synthetic seed"
$seedPath = Join-Path $payload "ph_holdings.db"
$env:INSTALLER_SEED_PATH = $seedPath
go test -tags seedgen -run TestProvisionFreshInstallerSeed -count=1 .
if ($LASTEXITCODE -ne 0) { throw "seed provisioning failed" }
if (-not (Test-Path $seedPath)) { throw "seed not produced at $seedPath" }

# 2) Identity overlay (REQUIRED). Substrate ships the synthetic data/overlay.json;
#    the fork ships its PH overlay at the same path.
if (-not (Test-Path $OverlaySource)) {
    throw "overlay source '$OverlaySource' not found; the installer requires an overlay.json to seed the identity plane"
}
Copy-Item $OverlaySource (Join-Path $payload "overlay.json") -Force
Write-Host "-- Staged overlay.json from $OverlaySource"

# 3) Letterhead artwork (OPTIONAL). Present in the fork, absent in the substrate.
$ssotDest = Join-Path $payload "ssot"
if (Test-Path $ssotDest) { Remove-Item $ssotDest -Recurse -Force }
if (Test-Path $SsotSource) {
    Copy-Item $SsotSource $ssotDest -Recurse -Force
    $n = (Get-ChildItem $ssotDest -File -Recurse | Measure-Object).Count
    Write-Host "-- Staged $n ssot letterhead file(s) from $SsotSource"
} else {
    # Keep an empty dir so the NSIS `File /nonfatal /r ssot\*` has a valid path.
    New-Item -ItemType Directory -Force -Path $ssotDest | Out-Null
    Write-Host "-- No ssot source ($SsotSource); staged empty ssot dir (substrate)"
}

$seedSize = (Get-Item $seedPath).Length
Write-Host ""
Write-Host ("Payload staged at {0} (seed {1} bytes)" -f $payload, $seedSize)
