@echo off
setlocal
cd /d "%~dp0"

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

"%NODE_EXE%" probe.mjs %*

echo.
echo (the probe has stopped — press any key to close this window)
pause >nul
