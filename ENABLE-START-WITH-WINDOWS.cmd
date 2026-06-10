@echo off
setlocal
set "EXE=%LOCALAPPDATA%\Programs\MiniExtractorGo\MiniExtractor.exe"
if not exist "%EXE%" (echo Mini Extractor is not installed.& pause & exit /b 1)
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$shell=New-Object -ComObject WScript.Shell;$startup=Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Startup';$exe=Join-Path $env:LOCALAPPDATA 'Programs\MiniExtractorGo\MiniExtractor.exe';$s=$shell.CreateShortcut((Join-Path $startup 'Mini Extractor Go Background.lnk'));$s.TargetPath=$exe;$s.WorkingDirectory=(Split-Path $exe);$s.Save();"
echo Mini Extractor will start automatically after Windows login.
pause
