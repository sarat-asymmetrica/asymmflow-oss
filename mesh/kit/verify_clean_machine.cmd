@echo off
rem verify_clean_machine.cmd -- double-click wrapper for the sealed-kit
rem verification. Windows PowerShell ships with Windows itself, so this
rem adds NO dependency beyond what the sealed kit already assumes.
rem ASCII-only, CRLF -- same batch discipline as run_bare_mesh.cmd.
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0verify-clean-machine.ps1" %*
set RC=%errorlevel%
echo.
echo (verification finished -- evidence in VERIFY_EVIDENCE.txt and verify-logs\)
if not defined ASYMMFLOW_KIT_NONINTERACTIVE pause >nul
exit /b %RC%
