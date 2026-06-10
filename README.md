<div align="center">
  <img src="assets/app_icon.png" alt="Mini Extractor icon" width="180" />

  # Mini Extractor

  **A lightweight Windows OCR and desktop color-picker utility that runs quietly in the system tray.**

  Extract text from any desktop area or image, copy only the text you need, sample screen colors safely, and keep global shortcuts available in the background.
</div>

---

## Overview

Mini Extractor is a compact Windows desktop utility built in Go. It is designed as a lightweight alternative to installing a larger utility suite when you only need OCR, image text extraction, and a desktop eyedropper.

The app starts silently in the Windows system tray. Its global shortcuts remain available while the main window is hidden.

## Features

### OCR extraction

- Select a precise rectangular area anywhere on the desktop.
- Open an image preview and drag over a specific region.
- Press `Enter` in the image preview to OCR the entire image.
- Copy OCR output automatically to the clipboard.
- Copy the entire OCR output or only a highlighted substring.
- Use multiple preprocessing passes for screenshots, scans, small text, faint text, and dark backgrounds.

### Multilingual and Arabic-aware text

- Tries all OCR language packs installed in Windows and selects the strongest result.
- Supports Arabic-script OCR with reconstructed right-to-left word order.
- Switches the text workspace automatically between RTL and LTR display modes.
- Includes manual **Auto**, **LTR**, and **RTL** controls.

### Click-safe desktop eyedropper

- Activate the eyedropper from the UI or a global shortcut.
- Click a pixel anywhere on the desktop.
- Copy the HEX value automatically.
- Prevent the underlying app from receiving the picker click.

### Background tray mode

- Runs silently in the Windows system tray.
- Keeps shortcuts active while the interface is hidden.
- Hides to the tray when you close or minimize the main window.
- Supports startup after Windows login.
- Uses a single-instance guard to prevent duplicate background processes.

## Default shortcuts

| Action | Shortcut |
|---|---|
| Capture a desktop area and run OCR | `Ctrl + Shift + T` |
| Open image-preview OCR | `Ctrl + Shift + I` |
| Pick a desktop color | `Ctrl + Shift + C` |

Shortcuts can be changed from the app interface: click **Change**, then press the new combination.

## Install the app

### Recommended: single-file installer

Download the Windows installer from the repository release assets:

```text
MiniExtractor-Setup.exe
```

Double-click it. The installer will:

1. Copy the app into `%LOCALAPPDATA%\Programs\MiniExtractorGo`.
2. Create Desktop and Start Menu shortcuts.
3. Add a Windows-login startup shortcut.
4. Start Mini Extractor silently in the tray.

### Portable package

A portable ZIP is also provided:

```text
MiniExtractor-Portable-Windows.zip
```

Extract it and keep these files together:

```text
MiniExtractor.exe
ocr.ps1
assets\app_icon.ico
assets\eyedropper.cur
```

## How to use it

### Extract text from the desktop

1. Press `Ctrl + Shift + T`.
2. Drag a rectangle around the text.
3. Release the mouse button.
4. The recognized text is copied automatically.

### Extract text from an image

1. Press `Ctrl + Shift + I` or choose **Open image preview**.
2. Select an image file.
3. Drag around the text region, or press `Enter` to process the entire image.
4. Press `Esc` to cancel.

### Copy only part of the OCR result

1. Open the main interface.
2. Highlight the required text inside the workspace.
3. Select **Copy selection**.

### Pick a desktop color

1. Press `Ctrl + Shift + C`.
2. Click the required pixel.
3. The HEX value is copied automatically.

## OCR languages

Mini Extractor uses the OCR language packs installed in Windows.

To add a language pack:

1. Open **Settings**.
2. Open **Time & language**.
3. Open **Language & region**.
4. Add the required language or open its language options.
5. Install the available language features.
6. Restart Mini Extractor.

For Arabic OCR, ensure that an Arabic Windows OCR language pack is installed.

## Build from source

### Requirements

- Windows 10 or Windows 11
- Go 1.23 or newer
- Windows PowerShell 5.1 for the built-in Windows OCR bridge

### Build the app

From PowerShell:

```powershell
go build -trimpath -ldflags "-H windowsgui -s -w" -o MiniExtractor.exe .
```

Keep the compiled executable next to:

```text
ocr.ps1
assets\app_icon.ico
assets\eyedropper.cur
```

## Repository structure

```text
.
├── .github/workflows/build-windows.yml
├── assets/
│   ├── app_icon.ico
│   ├── app_icon.png
│   └── eyedropper.cur
├── docs/
│   └── PUBLISH_TO_GITHUB.md
├── release-assets/
│   ├── MiniExtractor-Portable-Windows.zip
│   └── MiniExtractor-Setup.exe
├── main.go
├── ocr.ps1
├── go.mod
├── INSTALL-MINI-EXTRACTOR.cmd
├── UNINSTALL-MINI-EXTRACTOR.cmd
├── ENABLE-START-WITH-WINDOWS.cmd
├── DISABLE-START-WITH-WINDOWS.cmd
└── PUSH-TO-GITHUB.cmd
```

## Local files

The app stores logs and shortcut settings under:

```text
%LOCALAPPDATA%\MiniExtractorGo
```

The installed application files are stored under:

```text
%LOCALAPPDATA%\Programs\MiniExtractorGo
```

## Troubleshooting

### OCR is inaccurate

- Select a tighter rectangle around the text.
- Install the matching Windows language pack.
- For Arabic text, make sure an Arabic OCR pack is installed.
- Use the image preview and drag around the relevant region instead of processing the entire image.

### A shortcut does not work

Another application may already use it. Open Mini Extractor, select **Change** beside the tool, and record a different combination.

### The interface is closed but shortcuts still work

This is expected. Mini Extractor continues running in the system tray. Right-click the tray icon and select **Exit Mini Extractor** to stop it fully.

## License

No open-source license has been selected yet. Add a license file before publishing the project as an open-source repository. An MIT template is included in `docs/LICENSE-MIT-TEMPLATE.txt` for convenience.
