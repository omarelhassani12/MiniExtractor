# Publish Mini Extractor to GitHub

## 1. Create an empty repository on GitHub

Create a repository such as:

```text
mini-extractor-windows
```

Do not add a README, `.gitignore`, or license from the GitHub website because this folder already contains those files.

## 2. Push with the included helper

Run:

```text
PUSH-TO-GITHUB.cmd
```

Paste the repository URL when requested. Example:

```text
https://github.com/YOUR_USERNAME/mini-extractor-windows.git
```

## 3. Manual Git commands

```bash
git init
git add .
git commit -m "Initial release: Mini Extractor"
git branch -M main
git remote add origin https://github.com/YOUR_USERNAME/mini-extractor-windows.git
git push -u origin main
```

## 4. Add a GitHub Release

After pushing the repository:

1. Open the repository on GitHub.
2. Open **Releases**.
3. Select **Create a new release**.
4. Use a tag such as `v2.5.0`.
5. Upload:
   - `release-assets/MiniExtractor-Setup.exe`
   - `release-assets/MiniExtractor-Portable-Windows.zip`
6. Publish the release.
