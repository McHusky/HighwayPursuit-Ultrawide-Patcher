@echo off
setlocal

where go >nul 2>nul
if errorlevel 1 (
  echo Go was not found. Install Go 1.21 or newer to build the patcher.
  exit /b 1
)

if not exist dist mkdir dist

echo Running tests...
go test ./...
if errorlevel 1 exit /b 1

echo Building portable Windows patcher...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=386
go build -trimpath -ldflags="-s -w -H=windowsgui" -o dist\HighwayPursuit-Ultrawide-Patcher.exe .
if errorlevel 1 exit /b 1

certutil -hashfile dist\HighwayPursuit-Ultrawide-Patcher.exe SHA256

echo.
echo Done: dist\HighwayPursuit-Ultrawide-Patcher.exe
