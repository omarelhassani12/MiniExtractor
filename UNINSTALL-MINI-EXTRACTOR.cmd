@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$desktop=[Environment]::GetFolderPath('Desktop');$programs=Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs';$startup=Join-Path $programs 'Startup';Remove-Item -LiteralPath (Join-Path $desktop 'Mini Extractor Go.lnk') -Force -ErrorAction SilentlyContinue;Remove-Item -LiteralPath (Join-Path $programs 'Mini Extractor Go.lnk') -Force -ErrorAction SilentlyContinue;Remove-Item -LiteralPath (Join-Path $startup 'Mini Extractor Go Background.lnk') -Force -ErrorAction SilentlyContinue;"
taskkill /f /im MiniExtractor.exe >nul 2>nul
rmdir /s /q "%LOCALAPPDATA%\Programs\MiniExtractorGo"
echo Mini Extractor Go removed.
pause
