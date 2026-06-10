@echo off
setlocal
cd /d "%~dp0"

echo ============================================================
echo Mini Extractor Go - Installer with Custom Icon
echo ============================================================
echo.

set "TARGET=%LOCALAPPDATA%\Programs\MiniExtractorGo"
if exist "%TARGET%" rmdir /s /q "%TARGET%"
mkdir "%TARGET%"
mkdir "%TARGET%\assets"
copy /y "%~dp0MiniExtractor.exe" "%TARGET%\MiniExtractor.exe" >nul
copy /y "%~dp0ocr.ps1" "%TARGET%\ocr.ps1" >nul
copy /y "%~dp0assets\eyedropper.cur" "%TARGET%\assets\eyedropper.cur" >nul
copy /y "%~dp0assets\app_icon.ico" "%TARGET%\assets\app_icon.ico" >nul
copy /y "%~dp0assets\app_icon.png" "%TARGET%\assets\app_icon.png" >nul

powershell.exe -NoProfile -ExecutionPolicy Bypass -Command ^
  "$shell=New-Object -ComObject WScript.Shell;" ^
  "$desktop=[Environment]::GetFolderPath('Desktop');" ^
  "$programs=Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs';" ^
  "$startup=Join-Path $programs 'Startup';" ^
  "$exe=Join-Path $env:LOCALAPPDATA 'Programs\MiniExtractorGo\MiniExtractor.exe';" ^
  "$icon=Join-Path $env:LOCALAPPDATA 'Programs\MiniExtractorGo\assets\app_icon.ico';" ^
  "$s=$shell.CreateShortcut((Join-Path $desktop 'Mini Extractor Go.lnk'));$s.TargetPath=$exe;$s.Arguments='--show';$s.WorkingDirectory=(Split-Path $exe);$s.IconLocation=$icon;$s.Save();" ^
  "$s=$shell.CreateShortcut((Join-Path $programs 'Mini Extractor Go.lnk'));$s.TargetPath=$exe;$s.Arguments='--show';$s.WorkingDirectory=(Split-Path $exe);$s.IconLocation=$icon;$s.Save();" ^
  "$s=$shell.CreateShortcut((Join-Path $startup 'Mini Extractor Go Background.lnk'));$s.TargetPath=$exe;$s.WorkingDirectory=(Split-Path $exe);$s.IconLocation=$icon;$s.Save();"

if errorlevel 1 (
  echo Installation failed while creating shortcuts.
  pause
  exit /b 1
)

taskkill /f /im MiniExtractor.exe >nul 2>nul
start "" "%TARGET%\MiniExtractor.exe"
echo.
echo Installation complete. Mini Extractor is running silently in the tray.
echo The custom icon is now used for the installed shortcuts and tray icon.
pause
