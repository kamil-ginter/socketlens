@echo off
setlocal
set "SCRIPT=%~dp0start.ps1"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT%"
