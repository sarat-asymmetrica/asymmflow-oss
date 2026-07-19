@echo off
setlocal
cd /d "%~dp0"

where powershell >nul 2>nul
if errorlevel 1 (
  echo.
  echo PowerShell was not found on this computer - it is required to remove
  echo the anchor's background task.
  echo.
  pause
  exit /b 1
)

set "PSARGS="
if /i "%~1"=="--print-only" set "PSARGS=-PrintOnly"

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall_anchor.ps1" %PSARGS%
set "RC=%errorlevel%"
if not "%PSARGS%"=="-PrintOnly" pause
exit /b %RC%
