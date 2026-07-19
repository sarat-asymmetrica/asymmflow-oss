@echo off
setlocal
cd /d "%~dp0"

rem Mission A2 Band 3, deliverable 4: "status without a console" (I3,
rem phone-readable) — prints the last heartbeat and recent log lines, so
rem SPOC can check on the anchor over a phone call without touching the
rem scheduled task or opening a black window.

set "LOGFILE=%~dp0data\keys\anchor.log"

echo ANCHOR STATUS
echo -------------
if not exist "%LOGFILE%" (
  echo No heartbeat log yet at data\keys\anchor.log
  echo Either the anchor has never run, or it was just installed and hasn't
  echo started yet. Run install_anchor.cmd to set it up, or run_anchor.cmd
  echo to test it in the foreground right now.
  echo.
  pause
  exit /b 1
)

echo Recent activity ^(last 10 lines^):
powershell -NoProfile -Command "Get-Content -LiteralPath '%LOGFILE%' -Tail 10"

echo.
powershell -NoProfile -Command "$last = Get-Content -LiteralPath '%LOGFILE%' -Tail 1; if ($last -match '^(\S+)') { try { $ts = [DateTime]::Parse($Matches[1]).ToUniversalTime(); $age = [int]((Get-Date).ToUniversalTime() - $ts).TotalSeconds; if ($age -gt 150) { Write-Host ('ANCHOR MAY BE DOWN - the last heartbeat was ' + $age + ' seconds ago.') } else { Write-Host ('Anchor looks alive - last heartbeat ' + $age + ' seconds ago.') } } catch { Write-Host 'Could not read the last heartbeat time.' } } else { Write-Host 'Could not read the last heartbeat time.' }"

echo.
pause
