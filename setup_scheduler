@echo off
echo Setting up LTO Monitor Windows Scheduled Tasks...
echo.

REM Get the current directory where the batch file is located
set "SCRIPT_DIR=%~dp0"
set "EXE_PATH=%SCRIPT_DIR%lto-monitor.exe"

echo Executable path: %EXE_PATH%
echo.

REM Check if the executable exists
if not exist "%EXE_PATH%" (
    echo ERROR: lto-monitor.exe not found in %SCRIPT_DIR%
    echo Please ensure lto-monitor.exe is in the same directory as this script.
    pause
    exit /b 1
)

echo Creating morning task (8:00 AM)...
schtasks /create /tn "LTO Monitor - Morning Check" /tr "%EXE_PATH%" /sc daily /st 08:00 /ru "SYSTEM" /f
if %errorlevel% neq 0 (
    echo Failed to create morning task. You may need to run as Administrator.
    pause
    exit /b 1
)

echo Creating evening task (6:00 PM)...
schtasks /create /tn "LTO Monitor - Evening Check" /tr "%EXE_PATH%" /sc daily /st 18:00 /ru "SYSTEM" /f
if %errorlevel% neq 0 (
    echo Failed to create evening task. You may need to run as Administrator.
    pause
    exit /b 1
)

echo.
echo SUCCESS: Both scheduled tasks have been created!
echo.
echo Tasks created:
echo - "LTO Monitor - Morning Check" runs daily at 8:00 AM
echo - "LTO Monitor - Evening Check" runs daily at 6:00 PM
echo.
echo You can view and modify these tasks in Windows Task Scheduler.
echo To open Task Scheduler: Press Win+R, type "taskschd.msc", press Enter
echo.
echo To test the task manually:
echo   schtasks /run /tn "LTO Monitor - Morning Check"
echo.
echo To delete the tasks:
echo   schtasks /delete /tn "LTO Monitor - Morning Check" /f
echo   schtasks /delete /tn "LTO Monitor - Evening Check" /f
echo.
pause
