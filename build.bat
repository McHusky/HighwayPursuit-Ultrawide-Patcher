@echo off
setlocal EnableExtensions EnableDelayedExpansion
cd /d "%~dp0"

rem ---------------------------------------------------------------------------
rem Highway Pursuit Ultrawide Patcher - one-click source build for Windows
rem
rem No Go installation is required. On the first run this script downloads the
rem official portable Go toolchain from go.dev, verifies its SHA-256 checksum,
rem extracts it into .\tools\go, runs the tests, and builds the patcher.
rem Nothing is installed system-wide and no administrator rights are required.
rem ---------------------------------------------------------------------------

set "GO_VERSION=1.27.1"
set "GO_ARCH=amd64"
set "GO_ZIP_NAME=go%GO_VERSION%.windows-%GO_ARCH%.zip"
set "GO_URL=https://go.dev/dl/%GO_ZIP_NAME%"
set "GO_SHA256=a3911b5e0e1b1053f25ed0675f4c1c6aad1e2bfcf253df2b9be4caabd2edd95d"
set "TOOLS_DIR=%~dp0tools"
set "GO_DIR=%TOOLS_DIR%\go"
set "GO_ZIP=%TOOLS_DIR%\%GO_ZIP_NAME%"
set "GO_EXE=%GO_DIR%\bin\go.exe"
set "DIST_DIR=%~dp0dist"
set "OUTPUT_EXE=%DIST_DIR%\HighwayPursuit-Ultrawide-Patcher.exe"

where powershell.exe >nul 2>nul
if errorlevel 1 (
    echo.
    echo ERROR: Windows PowerShell was not found.
    echo This build script uses the PowerShell included with Windows to download
    echo and verify the official portable Go toolchain.
    echo.
    pause
    exit /b 1
)

if not exist "%GO_EXE%" (
    echo.
    echo Portable Go %GO_VERSION% was not found.
    echo It will be downloaded once from the official Go website:
    echo %GO_URL%
    echo.

    if not exist "%TOOLS_DIR%" mkdir "%TOOLS_DIR%"
    if exist "%GO_DIR%" rmdir /s /q "%GO_DIR%"
    if exist "%GO_ZIP%" del /q "%GO_ZIP%"

    echo Downloading Go %GO_VERSION%...
    powershell.exe -NoProfile -Command "$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue'; Invoke-WebRequest -UseBasicParsing -Uri '%GO_URL%' -OutFile '%GO_ZIP%'"
    if errorlevel 1 (
        echo.
        echo ERROR: Download failed.
        if exist "%GO_ZIP%" del /q "%GO_ZIP%"
        echo.
        pause
        exit /b 1
    )

    echo Verifying downloaded Go archive...
    set "ACTUAL_SHA256="
    for /f "usebackq delims=" %%H in (`powershell.exe -NoProfile -Command "(Get-FileHash -Algorithm SHA256 -LiteralPath '%GO_ZIP%').Hash.ToLowerInvariant()"`) do set "ACTUAL_SHA256=%%H"

    if not defined ACTUAL_SHA256 (
        echo.
        echo ERROR: Could not calculate the SHA-256 checksum.
        del /q "%GO_ZIP%" >nul 2>nul
        echo.
        pause
        exit /b 1
    )

    if /I not "!ACTUAL_SHA256!"=="%GO_SHA256%" (
        echo.
        echo ERROR: The downloaded Go archive failed SHA-256 verification.
        echo Expected: %GO_SHA256%
        echo Actual:   !ACTUAL_SHA256!
        echo.
        echo The archive will not be extracted.
        del /q "%GO_ZIP%" >nul 2>nul
        echo.
        pause
        exit /b 1
    )

    echo Checksum OK.
    echo Extracting portable Go toolchain...
    powershell.exe -NoProfile -Command "$ErrorActionPreference='Stop'; Expand-Archive -LiteralPath '%GO_ZIP%' -DestinationPath '%TOOLS_DIR%' -Force"
    if errorlevel 1 (
        echo.
        echo ERROR: Could not extract the Go toolchain.
        echo.
        pause
        exit /b 1
    )

    del /q "%GO_ZIP%" >nul 2>nul
)

if not exist "%GO_EXE%" (
    echo.
    echo ERROR: Portable Go is incomplete or could not be found at:
    echo %GO_EXE%
    echo.
    pause
    exit /b 1
)

if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"

rem Prevent Go from fetching a different toolchain automatically. The exact
rem toolchain used by this script is intentionally pinned and verified above.
set "GOTOOLCHAIN=local"
set "CGO_ENABLED=0"

 echo.
echo Using:
"%GO_EXE%" version
if errorlevel 1 goto :build_failed

 echo.
echo Running patcher tests...
rem Test only this module root. The portable Go SDK lives under tools\go, and
rem using ./... here would incorrectly include Go's own compiler test tree.
"%GO_EXE%" test .
if errorlevel 1 goto :build_failed

 echo.
echo Building Windows patcher...
set "GOOS=windows"
set "GOARCH=386"
"%GO_EXE%" build -trimpath -ldflags="-H=windowsgui" -o "%OUTPUT_EXE%" .
if errorlevel 1 goto :build_failed

 echo.
echo Build successful.
echo.
echo Output:
echo %OUTPUT_EXE%
echo.
echo Next step:
echo Copy HighwayPursuit-Ultrawide-Patcher.exe next to your original
echo HighwayPursuit.exe and run the patcher.
echo.
echo The portable Go toolchain remains only in this source folder under tools\
echo and can be deleted at any time.
echo.
pause
exit /b 0

:build_failed
echo.
echo ERROR: Build failed. No game files were modified.
echo.
pause
exit /b 1
