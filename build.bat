@echo off
echo ========================================
echo   Comfy-Swap Build Script
echo ========================================
echo.

echo [1/2] Cleaning build cache...
go clean -cache 2>nul

echo [2/2] Building comfy-swap.exe...
go build -a -o comfy-swap.exe .

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo   Build successful!
    echo   Output: comfy-swap.exe
    echo ========================================
) else (
    echo.
    echo Build failed!
    exit /b 1
)
