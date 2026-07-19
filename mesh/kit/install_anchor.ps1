# install_anchor.ps1 — Mission A2 Band 3: registers the anchor as a Windows
# Scheduled Task (logon trigger, restart-on-failure). Called by
# install_anchor.cmd; not meant to be double-clicked directly.
#
# Idempotent: re-running replaces the existing task definition (schtasks
# /Create /F) rather than erroring or duplicating it.
#
# --print-only: writes the task XML (needed for the printed command to be
# reproducible) but does NOT call schtasks and does NOT elevate — the
# hermetic gate (anchor-spike.mjs) drives this path, so it must never touch
# the real Task Scheduler or pop a UAC prompt.
#
# Self-elevation: a per-user logon-trigger task normally does NOT require
# admin rights to register, so this only elevates (Start-Process -Verb
# RunAs, re-invoking itself) if the non-elevated attempt actually fails —
# "self-elevating as needed", not unconditionally.

param(
  [switch]$PrintOnly,
  [int]$ListenPort = 0
)

$ErrorActionPreference = 'Stop'
$kitDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$taskName = 'AsymmFlowMeshAnchor'

$nodeExe = Join-Path $kitDir 'node.exe'
if (-not (Test-Path $nodeExe)) { $nodeExe = 'node' } # bundled-node-preferring, same convention as run_anchor.cmd

$anchorScript = Join-Path $kitDir 'anchor.mjs'
$dataDir = Join-Path $kitDir 'data'
$xmlPath = Join-Path $kitDir 'anchor_task.xml'

$argsLine = "`"$anchorScript`" --data `"$dataDir`""
if ($ListenPort -gt 0) { $argsLine += " --listen $ListenPort" }

# RestartOnFailure needs the XML task definition — plain `schtasks /Create`
# flags have no equivalent switch for it (only the Task Scheduler's own
# <RestartOnFailure> element does, per the mission's own verified-source
# discipline: checked against schtasks.exe's documented flag set before
# reaching for XML).
$xml = @"
<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <StartWhenAvailable>true</StartWhenAvailable>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>999</Count>
    </RestartOnFailure>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>$nodeExe</Command>
      <Arguments>$argsLine</Arguments>
      <WorkingDirectory>$kitDir</WorkingDirectory>
    </Exec>
  </Actions>
</Task>
"@

Set-Content -Path $xmlPath -Value $xml -Encoding Unicode

$createCmd = "schtasks /Create /TN `"$taskName`" /XML `"$xmlPath`" /F"

if ($PrintOnly) {
  Write-Host "ANCHOR INSTALL - print-only (nothing executed, no elevation requested)"
  Write-Host "task XML written to: $xmlPath"
  Write-Host "would run: $createCmd"
  exit 0
}

$existing = & schtasks /Query /TN $taskName 2>$null
& cmd /c $createCmd 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
  Write-Host ""
  Write-Host "Could not register the scheduled task directly (this can happen without administrator rights)."
  Write-Host "Retrying with administrator privileges..."
  $selfArgs = "-NoProfile -ExecutionPolicy Bypass -File `"$($MyInvocation.MyCommand.Path)`""
  if ($ListenPort -gt 0) { $selfArgs += " -ListenPort $ListenPort" }
  Start-Process -FilePath 'powershell' -ArgumentList $selfArgs -Verb RunAs
  exit 0
}

if ($existing) {
  Write-Host "ANCHOR REINSTALLED - existing task `"$taskName`" was updated."
} else {
  Write-Host "ANCHOR INSTALLED - it will start automatically at next logon, and Windows will restart it (up to 999 times) if it ever crashes."
}
Write-Host "To check on it later, run anchor_status.cmd. To remove it, run uninstall_anchor.cmd."
