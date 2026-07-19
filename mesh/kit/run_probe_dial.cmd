@echo off
setlocal
cd /d "%~dp0"

if "%~1"=="" (
  echo.
  echo Usage: run_probe_dial.cmd ^<z32-key-from-the-other-side^> [--json]
  echo.
  pause
  exit /b 2
)

set NODE_EXE=node
if exist "..\node.exe" set NODE_EXE=..\node.exe
if exist ".\node.exe" set NODE_EXE=.\node.exe

if "%NODE_EXE%"=="node" (
  where node >nul 2>nul
  if errorlevel 1 (
    echo.
    echo Node.js was not found on this computer.
    echo Install it from https://nodejs.org/  ^(the LTS installer, default options are fine^),
    echo then double-click this file again.
    echo.
    pause
    exit /b 1
  )
)

set DIAL_KEY=%~1
shift
"%NODE_EXE%" probe.mjs --dial %DIAL_KEY% %1 %2 %3 %4 %5 %6 %7 %8 %9

echo.
echo (the probe has stopped — press any key to close this window)
pause >nul
