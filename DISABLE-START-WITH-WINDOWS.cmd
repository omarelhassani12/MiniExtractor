@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$startup=Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Startup\Mini Extractor Go Background.lnk';Remove-Item -LiteralPath $startup -Force -ErrorAction SilentlyContinue;"
echo Mini Extractor automatic startup has been disabled.
pause
