@echo off
setlocal
cd /d "%~dp0"

echo ============================================================
echo Mini Extractor - Push Source to GitHub
echo ============================================================
echo.

where git >nul 2>nul
if errorlevel 1 (
  echo ERROR: Git is not installed or is not available in PATH.
  echo Install Git for Windows, reopen this folder, and run this file again.
  echo.
  pause
  exit /b 1
)

echo Create an empty GitHub repository first.
echo Example: https://github.com/YOUR_USERNAME/mini-extractor-windows.git
echo.
set /p REPO_URL=Paste the GitHub repository URL: 

if "%REPO_URL%"=="" (
  echo ERROR: A repository URL is required.
  pause
  exit /b 1
)

if not exist .git (
  git init
  if errorlevel 1 goto :error
)

git add .
if errorlevel 1 goto :error

git commit -m "Initial release: Mini Extractor"
if errorlevel 1 (
  echo.
  echo Git could not create the commit. If this is your first Git commit,
  echo configure your name and email, then run this script again:
  echo.
  echo   git config --global user.name "YOUR NAME"
  echo   git config --global user.email "YOUR_EMAIL@example.com"
  echo.
  pause
  exit /b 1
)

git branch -M main

git remote remove origin >nul 2>nul
git remote add origin "%REPO_URL%"
if errorlevel 1 goto :error

git push -u origin main
if errorlevel 1 (
  echo.
  echo Push failed. Check the repository URL and your GitHub authentication.
  echo GitHub may ask you to sign in through the browser or use a token.
  echo.
  pause
  exit /b 1
)

echo.
echo Repository pushed successfully.
echo.
pause
exit /b 0

:error
echo.
echo Git command failed. Review the output above.
echo.
pause
exit /b 1
