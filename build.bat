@echo off
title VAULT build
cd /d "%~dp0"
color 0a

echo.
echo   VAULT  build script
echo   ================================
echo.

where go >nul 2>nul
if errorlevel 1 (
  echo   [!] go not found
  echo       install from https://go.dev then run again
  echo.
  pause
  exit /b 1
)

echo   building vault.exe...
go build -ldflags="-s -w" -o vault.exe .
if errorlevel 1 (
  echo   [!] build failed
  pause
  exit /b 1
)

echo   done. output: vault.exe
echo.
pause
