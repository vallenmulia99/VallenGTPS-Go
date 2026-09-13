@echo off
echo Building GTPS-VALLEN for Windows...
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
go build -v -o gtps-vallen.exe .
if %ERRORLEVEL% == 0 (
    echo.
    echo BUILD SUCCESS! Output: gtps-vallen.exe
) else (
    echo.
    echo BUILD FAILED with error code %ERRORLEVEL%
)
