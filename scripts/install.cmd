@echo off
rem Wrapper so the installer can be launched from a classic cmd.exe prompt
rem or by double-clicking. Forwards all args to install.ps1.

setlocal
set "SCRIPT_DIR=%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%install.ps1" %*
exit /b %ERRORLEVEL%
