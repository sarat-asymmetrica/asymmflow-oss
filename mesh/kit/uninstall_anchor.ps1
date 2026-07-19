# uninstall_anchor.ps1 — Mission A2 Band 3: reverses install_anchor.ps1.
# Idempotent both directions: removing an already-absent task is a normal,
# quiet success (not an error) — see install_anchor.ps1's header for the
# same self-elevation-as-needed and --print-only doctrine, mirrored here.

param(
  [switch]$PrintOnly
)

$ErrorActionPreference = 'Stop'
$taskName = 'AsymmFlowMeshAnchor'
$deleteCmd = "schtasks /Delete /TN `"$taskName`" /F"

if ($PrintOnly) {
  Write-Host "ANCHOR UNINSTALL - print-only (nothing executed, no elevation requested)"
  Write-Host "would run: $deleteCmd"
  exit 0
}

$existing = & schtasks /Query /TN $taskName 2>$null
if (-not $existing) {
  Write-Host "ANCHOR NOT INSTALLED - nothing to remove."
  exit 0
}

& cmd /c $deleteCmd 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
  Write-Host ""
  Write-Host "Could not remove the task directly (this can happen without administrator rights)."
  Write-Host "Retrying with administrator privileges..."
  $selfArgs = "-NoProfile -ExecutionPolicy Bypass -File `"$($MyInvocation.MyCommand.Path)`""
  Start-Process -FilePath 'powershell' -ArgumentList $selfArgs -Verb RunAs
  exit 0
}

Write-Host "ANCHOR REMOVED - it will no longer start automatically at logon."
