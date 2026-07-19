@echo off
setlocal
cd /d "%~dp0"

where powershell >nul 2>nul
if errorlevel 1 (
  echo.
  echo PowerShell was not found on this computer - it is required to install
  echo the anchor as a background task. See run_anchor.cmd for a foreground
  echo alternative that does not need PowerShell.
  echo.
  pause
  exit /b 1
)

rem Translate the CLI's --print-only convention (used everywhere else in
rem this kit, e.g. build-kit.mjs's own gate mode) to PowerShell's -PrintOnly
rem switch syntax, so the SAME flag name works at this command line.
set "PSARGS="
if /i "%~1"=="--print-only" set "PSARGS=-PrintOnly"

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0install_anchor.ps1" %PSARGS%
set "RC=%errorlevel%"
if not "%PSARGS%"=="-PrintOnly" pause
exit /b %RC%
