@echo off
setlocal
cd /d "%~dp0"

rem Mission A2 Band 3 (mesh\docs\MISSION_A2_CORRIDOR_SPEC.md): foreground
rem run of the anchor, for field testing before install_anchor.cmd registers
rem it as a background scheduled task. Bundled-node-preferring: a node.exe
rem sitting beside this .cmd file (the field-kit build convention, Band 2)
rem wins over PATH so the receptionist machine needs zero installs; PATH
rem node is the fallback for a dev/source checkout that has no bundled copy.

set "NODE_EXE=%~dp0node.exe"
if not exist "%NODE_EXE%" set "NODE_EXE=node"

"%NODE_EXE%" --version >nul 2>nul
if errorlevel 1 (
  echo.
  echo Node.js was not found ^(no bundled node.exe next to this file, and none
  echo on PATH^). Install it from https://nodejs.org/ ^(the LTS installer,
  echo default options are fine^), or use a kit build that bundles node.exe.
  echo.
  pause
  exit /b 1
)

echo Starting the anchor in the foreground. Press Ctrl+C to stop it cleanly.
echo Heartbeat log: data\keys\anchor.log
echo.

"%NODE_EXE%" "%~dp0anchor.mjs" --data "%~dp0data" %*

echo.
echo (the anchor has stopped — press any key to close this window)
pause >nul
