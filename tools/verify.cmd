@echo off
setlocal
cd /d "%~dp0\.."
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0verify.ps1"
if errorlevel 1 (
  echo.
  echo SocketLens verification failed.
  pause
  exit /b 1
)
echo.
echo SocketLens verification completed successfully.
pause
