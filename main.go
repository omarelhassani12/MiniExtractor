//go:build windows

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	uxtheme  = syscall.NewLazyDLL("uxtheme.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procMessageBoxW          = user32.NewProc("MessageBoxW")
	procSetWindowTextW       = user32.NewProc("SetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procSendMessageW         = user32.NewProc("SendMessageW")
	procGetDC                = user32.NewProc("GetDC")
	procReleaseDC            = user32.NewProc("ReleaseDC")
	procGetCursorPos         = user32.NewProc("GetCursorPos")
	procGetAsyncKeyState     = user32.NewProc("GetAsyncKeyState")
	procDrawFocusRect        = user32.NewProc("DrawFocusRect")
	procLoadCursorW          = user32.NewProc("LoadCursorW")
	procLoadCursorFromFileW  = user32.NewProc("LoadCursorFromFileW")
	procSetCursor            = user32.NewProc("SetCursor")
	procSetProcessDPIAware   = user32.NewProc("SetProcessDPIAware")
	procSetLayeredWindowAttr = user32.NewProc("SetLayeredWindowAttributes")
	procSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	procGetWindowLongPtrW   = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW   = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procSetFocus             = user32.NewProc("SetFocus")
	procFindWindowW          = user32.NewProc("FindWindowW")
	procIsWindowVisible      = user32.NewProc("IsWindowVisible")
	procLoadIconW            = user32.NewProc("LoadIconW")
	procLoadImageW           = user32.NewProc("LoadImageW")
	procCreatePopupMenu      = user32.NewProc("CreatePopupMenu")
	procAppendMenuW          = user32.NewProc("AppendMenuW")
	procTrackPopupMenu       = user32.NewProc("TrackPopupMenu")
	procDestroyMenu          = user32.NewProc("DestroyMenu")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procInvalidateRect       = user32.NewProc("InvalidateRect")
	procSetCapture           = user32.NewProc("SetCapture")
	procReleaseCapture       = user32.NewProc("ReleaseCapture")
	procBeginPaint           = user32.NewProc("BeginPaint")
	procEndPaint             = user32.NewProc("EndPaint")
	procGetClientRect        = user32.NewProc("GetClientRect")
	procFillRect             = user32.NewProc("FillRect")
	procSetBkMode           = gdi32.NewProc("SetBkMode")
	procSetBkColor          = gdi32.NewProc("SetBkColor")
	procSetTextColor        = gdi32.NewProc("SetTextColor")
	procCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
	procCreateFontW         = gdi32.NewProc("CreateFontW")
	procRegisterHotKey       = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey     = user32.NewProc("UnregisterHotKey")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	procOpenClipboard        = user32.NewProc("OpenClipboard")
	procEmptyClipboard       = user32.NewProc("EmptyClipboard")
	procSetClipboardData     = user32.NewProc("SetClipboardData")
	procCloseClipboard       = user32.NewProc("CloseClipboard")
	procGetPixel             = gdi32.NewProc("GetPixel")
	procGetStockObject       = gdi32.NewProc("GetStockObject")
	procCreateCompatibleDC   = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBmp  = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject         = gdi32.NewProc("SelectObject")
	procBitBlt               = gdi32.NewProc("BitBlt")
	procGetDIBits            = gdi32.NewProc("GetDIBits")
	procStretchDIBits        = gdi32.NewProc("StretchDIBits")
	procDeleteObject         = gdi32.NewProc("DeleteObject")
	procDeleteDC             = gdi32.NewProc("DeleteDC")
	procGlobalAlloc          = kernel32.NewProc("GlobalAlloc")
	procGlobalLock           = kernel32.NewProc("GlobalLock")
	procGlobalUnlock         = kernel32.NewProc("GlobalUnlock")
	procGlobalFree           = kernel32.NewProc("GlobalFree")
	procGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
	procCreateMutexW         = kernel32.NewProc("CreateMutexW")
	procGetLastError         = kernel32.NewProc("GetLastError")
	procCloseHandle          = kernel32.NewProc("CloseHandle")
	procGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	procCommDlgExtendedErr  = comdlg32.NewProc("CommDlgExtendedError")
	procSetWindowTheme      = uxtheme.NewProc("SetWindowTheme")
	procShellNotifyIconW    = shell32.NewProc("Shell_NotifyIconW")
)

const (
	className = "MiniExtractorGoWindow"

	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_POPUP            = 0x80000000
	WS_EX_TOPMOST       = 0x00000008
	WS_EX_TOOLWINDOW    = 0x00000080
	WS_EX_LAYERED       = 0x00080000
	WS_VISIBLE          = 0x10000000
	WS_EX_RTLREADING    = 0x00002000
	WS_EX_LEFTSCROLLBAR = 0x00004000
	GWL_STYLE           = -16
	GWL_EXSTYLE         = -20
	SWP_NOSIZE          = 0x0001
	SWP_NOMOVE          = 0x0002
	SWP_NOZORDER        = 0x0004
	SWP_FRAMECHANGED    = 0x0020
	WS_CHILD            = 0x40000000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	ES_MULTILINE        = 0x0004
	ES_AUTOVSCROLL      = 0x0040
	ES_READONLY         = 0x0800
	ES_RIGHT            = 0x0002
	ES_AUTOHSCROLL      = 0x0080
	BS_PUSHBUTTON       = 0
	BS_GROUPBOX         = 0x00000007

	CW_USEDEFAULT = 0x80000000
	SW_SHOW       = 5
	SW_HIDE       = 0
	SW_RESTORE    = 9
	LWA_ALPHA     = 0x00000002

	WM_DESTROY     = 0x0002
	WM_SIZE        = 0x0005
	WM_APP         = 0x8000
	WM_PAINT       = 0x000F
	WM_SETCURSOR   = 0x0020
	WM_KEYDOWN     = 0x0100
	WM_SYSKEYDOWN  = 0x0104
	WM_CLOSE       = 0x0010
	WM_MOUSEMOVE   = 0x0200
	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205
	WM_COMMAND     = 0x0111
	WM_HOTKEY      = 0x0312
	WM_SETFONT        = 0x0030
	WM_CTLCOLORSTATIC = 0x0138
	WM_CTLCOLOREDIT   = 0x0133
	EM_GETSEL         = 0x00B0
	WM_CTLCOLORBTN    = 0x0135
	EM_SETSEL      = 0x00B1

	IDC_ARROW = 32512
	IDI_APPLICATION = 32512
	IMAGE_ICON      = 1
	LR_LOADFROMFILE = 0x00000010
	IDC_CROSS = 32515

	ID_AREA_OCR   = 1001
	ID_IMAGE_OCR  = 1002
	ID_PICK_COLOR = 1003
	ID_COPY       = 1004
	ID_EXIT          = 1005
	ID_APPLY_HOTKEYS = 1006
	ID_RESET_HOTKEYS = 1007
	ID_CHANGE_AREA    = 1008
	ID_CHANGE_IMAGE   = 1009
	ID_CHANGE_COLOR   = 1010
	ID_CLEAR_RESULT    = 1011
	ID_COPY_SELECTION  = 1012
	ID_DIRECTION_AUTO  = 1013
	ID_DIRECTION_LTR   = 1014
	ID_DIRECTION_RTL   = 1015
	ID_TRAY_OPEN       = 1101
	ID_TRAY_AREA       = 1102
	ID_TRAY_IMAGE      = 1103
	ID_TRAY_COLOR      = 1104
	ID_TRAY_EXIT       = 1105

	HOTKEY_AREA  = 2001
	HOTKEY_COLOR = 2002
	HOTKEY_IMAGE = 2003

	MOD_ALT      = 0x0001
	MOD_CONTROL  = 0x0002
	MOD_SHIFT    = 0x0004
	MOD_WIN      = 0x0008
	MOD_NOREPEAT = 0x4000

	VK_LBUTTON = 0x01
	VK_RETURN  = 0x0D
	VK_SHIFT   = 0x10
	VK_CONTROL = 0x11
	VK_MENU    = 0x12
	VK_LWIN    = 0x5B
	VK_RWIN    = 0x5C
	VK_ESCAPE  = 0x1B
	VK_T       = 0x54
	VK_C       = 0x43
	VK_I       = 0x49

	MB_OK              = 0x00000000
	MB_ICONERROR       = 0x00000010
	MB_ICONINFORMATION = 0x00000040

	SRCCOPY        = 0x00CC0020
	DIB_RGB_COLORS = 0
	BI_RGB         = 0
	BLACK_BRUSH    = 4
	TRANSPARENT    = 1
	FW_NORMAL      = 400
	FW_SEMIBOLD    = 600
	FW_BOLD        = 700

	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002

	SM_XVIRTUALSCREEN  = 76
	SM_YVIRTUALSCREEN  = 77
	SM_CXVIRTUALSCREEN = 78
	SM_CYVIRTUALSCREEN = 79

	OFN_FILEMUSTEXIST = 0x00001000
	OFN_PATHMUSTEXIST = 0x00000800
	OFN_NOCHANGEDIR   = 0x00000008
	OFN_EXPLORER      = 0x00080000

	WM_TRAYICON       = WM_APP + 1
	SIZE_MINIMIZED    = 1
	MF_STRING         = 0x00000000
	TPM_RIGHTBUTTON   = 0x0002
	TPM_RETURNCMD     = 0x0100
	NIM_ADD           = 0x00000000
	NIM_DELETE        = 0x00000002
	NIF_MESSAGE       = 0x00000001
	NIF_ICON          = 0x00000002
	NIF_TIP           = 0x00000004
	ERROR_ALREADY_EXISTS = 183
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type MSG struct {
	Hwnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
	Private        uint32
}
type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}
type PAINTSTRUCT struct {
	Hdc         uintptr
	Erase       int32
	RcPaint     RECT
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}
type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}
type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	Colors    [1]uint32
}

type HotkeyConfig struct {
	Area  string `json:"area"`
	Image string `json:"image"`
	Color string `json:"color"`
}

type ParsedHotkey struct {
	Text      string
	Modifiers uint32
	Key       uint32
}

type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

type OPENFILENAME struct {
	LStructSize       uint32
	HwndOwner         uintptr
	HInstance         uintptr
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        unsafe.Pointer
	DwReserved        uint32
	FlagsEx           uint32
}

var (
	mainHwnd      uintptr
	editHwnd      uintptr
	statusHwnd    uintptr
	areaHotkeyLabelHwnd  uintptr
	imageHotkeyLabelHwnd uintptr
	colorHotkeyLabelHwnd uintptr
	currentHotkeys       HotkeyConfig
	recordingShortcut    int

	uiBackgroundBrush uintptr
	uiEditBrush       uintptr
	uiBodyFont        uintptr
	uiSmallFont       uintptr
	uiSectionFont     uintptr
	uiTitleFont       uintptr
	textDirectionMode string
	trayData          NOTIFYICONDATAW
	trayReady         bool
	mutexHandle       uintptr
	exitRequested     bool
	logger        *log.Logger
	appDir        string
	controlErrors []string

	overlayClassName = "MiniExtractorGoOverlay"
	overlayMode      int32
	overlayHwnd      uintptr
	overlayCursor    uintptr
	overlayStart     POINT
	overlayCurrent   POINT
	overlayRect      RECT
	overlayPoint     POINT
	overlayOK        bool
	overlaySelecting bool
	toolActive       int32

	previewClassName   = "MiniExtractorGoImagePreview"
	previewHwnd        uintptr
	previewSource      image.Image
	previewPixels      []byte
	previewBMI         BITMAPINFO
	previewDisplayRect RECT
	previewSelection   RECT
	previewSelecting   bool
	previewOK          bool
	previewFull        bool
	previewResult      image.Image
)

func main() {
	runtime.LockOSThread()
	exe, _ := os.Executable()
	appDir = filepath.Dir(exe)
	initLog()

	if !ensureSingleInstance() {
		return
	}
	defer releaseSingleInstance()

	defer func() {
		if r := recover(); r != nil {
			logger.Printf("panic: %v", r)
			messageBox(0, fmt.Sprintf("Mini Extractor crashed:\n\n%v\n\nLog: %s", r, logPath()), "Mini Extractor", MB_OK|MB_ICONERROR)
		}
	}()
	if err := run(); err != nil {
		logger.Printf("startup error: %v", err)
		messageBox(0, fmt.Sprintf("Mini Extractor could not start:\n\n%v\n\nLog: %s", err, logPath()), "Mini Extractor", MB_OK|MB_ICONERROR)
	}
}

func initLog() {
	_ = os.MkdirAll(filepath.Dir(logPath()), 0755)
	f, err := os.OpenFile(logPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
		return
	}
	logger = log.New(f, "", log.LstdFlags)
	logger.Println("Mini Extractor Go starting")
}
func logPath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "MiniExtractorGo", "MiniExtractor.log")
}
func p(s string) *uint16                                   { ptr, _ := syscall.UTF16PtrFromString(s); return ptr }
func call(proc *syscall.LazyProc, args ...uintptr) uintptr { r, _, _ := proc.Call(args...); return r }
func loword(v uintptr) uint16                              { return uint16(v & 0xffff) }

func run() error {
	call(procSetProcessDPIAware)
	initUIResources()
	defer disposeUIResources()
	hInstance := call(procGetModuleHandleW, 0)
	cursor := call(procLoadCursorW, 0, IDC_ARROW)
	appIcon := loadAppIcon()

	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInstance,
		HCursor:       cursor,
		HIcon:         appIcon,
		HIconSm:       appIcon,
		HbrBackground: uiBackgroundBrush,
		LpszClassName: p(className),
	}

	if call(procRegisterClassExW, uintptr(unsafe.Pointer(&wc))) == 0 {
		return errors.New("RegisterClassExW failed")
	}

	overlayWC := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(overlayWndProc),
		HInstance:     hInstance,
		HCursor:       call(procLoadCursorW, 0, IDC_CROSS),
		HbrBackground: call(procGetStockObject, BLACK_BRUSH),
		LpszClassName: p(overlayClassName),
	}

	if call(procRegisterClassExW, uintptr(unsafe.Pointer(&overlayWC))) == 0 {
		return errors.New("RegisterClassExW overlay failed")
	}

	previewWC := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(previewWndProc),
		HInstance:     hInstance,
		HCursor:       call(procLoadCursorW, 0, IDC_CROSS),
		HbrBackground: call(procGetStockObject, BLACK_BRUSH),
		LpszClassName: p(previewClassName),
	}

	if call(procRegisterClassExW, uintptr(unsafe.Pointer(&previewWC))) == 0 {
		return errors.New("RegisterClassExW image preview failed")
	}

	mainHwnd = call(
		procCreateWindowExW,
		0,
		uintptr(unsafe.Pointer(p(className))),
		uintptr(unsafe.Pointer(p("Mini Extractor · Background OCR Toolkit"))),
		WS_OVERLAPPEDWINDOW,
		110, 30, 1080, 875,
		0, 0, hInstance, 0,
	)

	if mainHwnd == 0 {
		return errors.New("CreateWindowExW failed")
	}

	createControls(mainHwnd)
	textDirectionMode = "auto"
	applyTextDirection("")

	if err := initTrayIcon(); err != nil {
		return err
	}
	defer removeTrayIcon()

	currentHotkeys = loadHotkeyConfig()
	setHotkeyLabels(currentHotkeys)

	if err := registerConfiguredHotkeys(currentHotkeys); err != nil {
		logger.Printf("configured shortcut registration failed: %v", err)

		defaults := defaultHotkeyConfig()
		setHotkeyLabels(defaults)

		if fallbackErr := registerConfiguredHotkeys(defaults); fallbackErr != nil {
			logger.Printf("default shortcut registration failed: %v", fallbackErr)
			setStatus("Shortcut warning: shortcuts could not be registered. Use the buttons or choose different keys.")
		} else {
			currentHotkeys = defaults
			setStatus("Saved shortcuts conflicted with another app. Default shortcuts were restored.")
		}
	}

	if shouldShowWindow() {
		showMainWindow()
		setStatus("Ready. Closing or minimizing this window keeps Mini Extractor running in the tray.")
	} else {
		hideMainWindow()
		setStatus("Running in the background. Use global shortcuts or the tray icon.")
	}

	var msg MSG
	for call(procGetMessageW, uintptr(unsafe.Pointer(&msg)), 0, 0, 0) > 0 {
		call(procTranslateMessage, uintptr(unsafe.Pointer(&msg)))
		call(procDispatchMessageW, uintptr(unsafe.Pointer(&msg)))
	}

	unregisterConfiguredHotkeys()
	return nil
}

func createControls(parent uintptr) {
	controlErrors = nil

	title := create("STATIC", "Mini Extractor", WS_CHILD|WS_VISIBLE, 28, 20, 520, 42, 0, parent)
	setControlFont(title, uiTitleFont)

	subtitle := create("STATIC", "Fast desktop OCR, bidirectional text, and a click-safe color picker for Windows", WS_CHILD|WS_VISIBLE, 30, 64, 760, 24, 0, parent)
	setControlFont(subtitle, uiBodyFont)

	hint := create("STATIC", "Arabic OCR now reconstructs word order from detected positions. RTL/LTR display direction remains automatic.", WS_CHILD|WS_VISIBLE, 30, 87, 920, 22, 0, parent)
	setControlFont(hint, uiSmallFont)

	workspaceGroup := create("BUTTON", "  Text workspace  ", WS_CHILD|WS_VISIBLE|BS_GROUPBOX, 24, 126, 690, 570, 0, parent)
	setControlFont(workspaceGroup, uiSectionFont)

	resultLabel := create("STATIC", "Extracted text", WS_CHILD|WS_VISIBLE, 44, 166, 220, 24, 0, parent)
	setControlFont(resultLabel, uiSectionFont)

	editHwnd = create(
		"EDIT",
		"",
		WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY,
		44, 198, 650, 418,
		0,
		parent,
	)
	setControlFont(editHwnd, uiBodyFont)

	copyButton := create("BUTTON", "Copy all", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 44, 636, 100, 36, ID_COPY, parent)
	styleActionButton(copyButton)

	copySelectionButton := create("BUTTON", "Copy selection", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 154, 636, 130, 36, ID_COPY_SELECTION, parent)
	styleActionButton(copySelectionButton)

	clearButton := create("BUTTON", "Clear", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 294, 636, 82, 36, ID_CLEAR_RESULT, parent)
	styleActionButton(clearButton)

	directionLabel := create("STATIC", "Direction:", WS_CHILD|WS_VISIBLE, 402, 644, 62, 22, 0, parent)
	setControlFont(directionLabel, uiSmallFont)

	autoDirectionButton := create("BUTTON", "Auto", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 470, 636, 58, 36, ID_DIRECTION_AUTO, parent)
	styleSmallButton(autoDirectionButton)

	ltrDirectionButton := create("BUTTON", "LTR", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 536, 636, 52, 36, ID_DIRECTION_LTR, parent)
	styleSmallButton(ltrDirectionButton)

	rtlDirectionButton := create("BUTTON", "RTL", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 596, 636, 52, 36, ID_DIRECTION_RTL, parent)
	styleSmallButton(rtlDirectionButton)

	autoCopy := create("STATIC", "Auto direction detects Arabic-script OCR output.", WS_CHILD|WS_VISIBLE, 44, 682, 610, 20, 0, parent)
	setControlFont(autoCopy, uiSmallFont)

	actionGroup := create("BUTTON", "  Quick actions  ", WS_CHILD|WS_VISIBLE|BS_GROUPBOX, 734, 126, 310, 320, 0, parent)
	setControlFont(actionGroup, uiSectionFont)

	actionHint := create("STATIC", "Choose a tool or use its global shortcut.", WS_CHILD|WS_VISIBLE, 754, 160, 270, 20, 0, parent)
	setControlFont(actionHint, uiSmallFont)

	areaButton := create("BUTTON", "Capture desktop area", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 754, 194, 270, 42, ID_AREA_OCR, parent)
	styleActionButton(areaButton)

	areaShortcut := create("STATIC", "Drag a rectangle around text  ·  Ctrl+Shift+T", WS_CHILD|WS_VISIBLE, 758, 240, 264, 20, 0, parent)
	setControlFont(areaShortcut, uiSmallFont)

	imageButton := create("BUTTON", "Open image preview", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 754, 272, 270, 42, ID_IMAGE_OCR, parent)
	styleActionButton(imageButton)

	imageShortcut := create("STATIC", "Drag a region or press Enter  ·  Ctrl+Shift+I", WS_CHILD|WS_VISIBLE, 758, 318, 264, 20, 0, parent)
	setControlFont(imageShortcut, uiSmallFont)

	colorButton := create("BUTTON", "Pick desktop color", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 754, 340, 270, 42, ID_PICK_COLOR, parent)
	styleActionButton(colorButton)

	colorShortcut := create("STATIC", "Click-safe picker  ·  Ctrl+Shift+C", WS_CHILD|WS_VISIBLE, 758, 386, 264, 20, 0, parent)
	setControlFont(colorShortcut, uiSmallFont)

	settingsGroup := create("BUTTON", "  Keyboard shortcuts  ", WS_CHILD|WS_VISIBLE|BS_GROUPBOX, 734, 464, 310, 286, 0, parent)
	setControlFont(settingsGroup, uiSectionFont)

	settingsHint := create("STATIC", "Click Change, then press a new combination.", WS_CHILD|WS_VISIBLE, 754, 498, 270, 20, 0, parent)
	setControlFont(settingsHint, uiSmallFont)

	areaLabel := create("STATIC", "Area OCR", WS_CHILD|WS_VISIBLE, 754, 536, 90, 22, 0, parent)
	setControlFont(areaLabel, uiBodyFont)
	areaHotkeyLabelHwnd = create("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER, 846, 532, 112, 28, 0, parent)
	setControlFont(areaHotkeyLabelHwnd, uiSmallFont)
	areaChange := create("BUTTON", "Change", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 964, 532, 64, 28, ID_CHANGE_AREA, parent)
	styleSmallButton(areaChange)

	imageLabel := create("STATIC", "Image OCR", WS_CHILD|WS_VISIBLE, 754, 580, 90, 22, 0, parent)
	setControlFont(imageLabel, uiBodyFont)
	imageHotkeyLabelHwnd = create("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER, 846, 576, 112, 28, 0, parent)
	setControlFont(imageHotkeyLabelHwnd, uiSmallFont)
	imageChange := create("BUTTON", "Change", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 964, 576, 64, 28, ID_CHANGE_IMAGE, parent)
	styleSmallButton(imageChange)

	colorLabel := create("STATIC", "Color picker", WS_CHILD|WS_VISIBLE, 754, 624, 90, 22, 0, parent)
	setControlFont(colorLabel, uiBodyFont)
	colorHotkeyLabelHwnd = create("STATIC", "", WS_CHILD|WS_VISIBLE|WS_BORDER, 846, 620, 112, 28, 0, parent)
	setControlFont(colorHotkeyLabelHwnd, uiSmallFont)
	colorChange := create("BUTTON", "Change", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 964, 620, 64, 28, ID_CHANGE_COLOR, parent)
	styleSmallButton(colorChange)

	resetButton := create("BUTTON", "Restore defaults", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 754, 670, 150, 34, ID_RESET_HOTKEYS, parent)
	styleActionButton(resetButton)

	exitButton := create("BUTTON", "Exit app", WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON, 914, 670, 112, 34, ID_EXIT, parent)
	styleActionButton(exitButton)

	shortcutHelp := create("STATIC", "Supported: Ctrl, Shift, Alt or Win + A-Z, 0-9 or F1-F12", WS_CHILD|WS_VISIBLE, 754, 712, 272, 24, 0, parent)
	setControlFont(shortcutHelp, uiSmallFont)

	statusTitle := create("STATIC", "STATUS", WS_CHILD|WS_VISIBLE, 28, 804, 62, 20, 0, parent)
	setControlFont(statusTitle, uiSmallFont)

	statusHwnd = create("STATIC", "Ready. Use a quick action or a global shortcut.", WS_CHILD|WS_VISIBLE, 94, 802, 936, 24, 0, parent)
	setControlFont(statusHwnd, uiBodyFont)

	if len(controlErrors) > 0 {
		messageBox(
			mainHwnd,
			"Some interface controls could not be created:\n\n"+strings.Join(controlErrors, "\n"),
			"Mini Extractor UI error",
			MB_OK|MB_ICONERROR,
		)
	}
}

func create(class, text string, style uint32, x, y, w, h int32, id uintptr, parent uintptr) uintptr {
	hwnd := call(procCreateWindowExW, 0, uintptr(unsafe.Pointer(p(class))), uintptr(unsafe.Pointer(p(text))), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, id, 0, 0)

	if hwnd == 0 {
		entry := fmt.Sprintf("%s control failed: %q", class, text)
		controlErrors = append(controlErrors, entry)
		logger.Println(entry)
		return 0
	}

	setControlFont(hwnd, uiBodyFont)

	if class == "BUTTON" || class == "EDIT" {
		call(
			procSetWindowTheme,
			hwnd,
			uintptr(unsafe.Pointer(p("Explorer"))),
			0,
		)
	}

	return hwnd
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_COMMAND:
		switch int(loword(wParam)) {
		case ID_AREA_OCR:
			go areaOCR()

		case ID_IMAGE_OCR:
			startImageOCRDialog()

		case ID_PICK_COLOR:
			go pickColor()

		case ID_COPY:
			copyEdit()

		case ID_COPY_SELECTION:
			copySelectedEditText()

		case ID_CLEAR_RESULT:
			setEdit("")
			setStatus("Text workspace cleared.")

		case ID_DIRECTION_AUTO:
			textDirectionMode = "auto"
			applyTextDirection(getEdit())
			setStatus("Text direction: automatic detection.")

		case ID_DIRECTION_LTR:
			textDirectionMode = "ltr"
			applyTextDirection(getEdit())
			setStatus("Text direction: left-to-right.")

		case ID_DIRECTION_RTL:
			textDirectionMode = "rtl"
			applyTextDirection(getEdit())
			setStatus("Text direction: right-to-left.")

		case ID_CHANGE_AREA:
			startShortcutRecording(ID_CHANGE_AREA)

		case ID_CHANGE_IMAGE:
			startShortcutRecording(ID_CHANGE_IMAGE)

		case ID_CHANGE_COLOR:
			startShortcutRecording(ID_CHANGE_COLOR)

		case ID_RESET_HOTKEYS:
			defaults := defaultHotkeyConfig()

			if err := registerConfiguredHotkeys(defaults); err != nil {
				setStatus("Could not restore defaults: " + err.Error())
				messageBox(mainHwnd, err.Error(), "Shortcut error", MB_OK|MB_ICONERROR)
				return 0
			}

			currentHotkeys = defaults
			_ = saveHotkeyConfig(defaults)
			setHotkeyLabels(defaults)
			setStatus("Default shortcuts restored.")

		case ID_EXIT:
			exitRequested = true
			call(procDestroyWindow, mainHwnd)
		}

		return 0

	case WM_CLOSE:
		if exitRequested {
			call(procDestroyWindow, hwnd)
			return 0
		}
		hideMainWindow()
		setStatus("Running in the background. Use shortcuts or the tray icon to reopen the interface.")
		return 0

	case WM_SIZE:
		if int(wParam) == SIZE_MINIMIZED {
			hideMainWindow()
			setStatus("Minimized to the tray. Global shortcuts remain active.")
			return 0
		}

	case WM_TRAYICON:
		switch uint32(lParam) {
		case WM_LBUTTONDBLCLK:
			showMainWindow()
		case WM_RBUTTONUP:
			showTrayMenu()
		}
		return 0

	case WM_KEYDOWN, WM_SYSKEYDOWN:
		if recordingShortcut != 0 {
			handleRecordedShortcut(uint32(wParam))
			return 0
		}

	case WM_HOTKEY:
		switch int(wParam) {
		case HOTKEY_AREA:
			go areaOCR()

		case HOTKEY_IMAGE:
			startImageOCRDialog()

		case HOTKEY_COLOR:
			go pickColor()
		}

		return 0

	case WM_CTLCOLORSTATIC:
		call(procSetBkMode, wParam, TRANSPARENT)
		call(procSetTextColor, wParam, uintptr(rgb(52, 65, 85)))
		return uiBackgroundBrush

	case WM_CTLCOLOREDIT:
		call(procSetBkColor, wParam, uintptr(rgb(255, 255, 255)))
		call(procSetTextColor, wParam, uintptr(rgb(15, 23, 42)))
		return uiEditBrush

	case WM_CTLCOLORBTN:
		return uiBackgroundBrush

	case WM_DESTROY:
		call(procPostQuitMessage, 0)
		return 0
	}

	return call(procDefWindowProcW, hwnd, uintptr(msg), wParam, lParam)
}

func ensureSingleInstance() bool {
	mutexName := p("Local\\MiniExtractorGoBackgroundService")
	mutexHandle = call(procCreateMutexW, 0, 0, uintptr(unsafe.Pointer(mutexName)))
	if mutexHandle == 0 { logger.Printf("CreateMutexW failed; continuing without guard"); return true }
	if call(procGetLastError) != ERROR_ALREADY_EXISTS { return true }
	existing := call(procFindWindowW, uintptr(unsafe.Pointer(p(className))), 0)
	if existing != 0 && shouldShowWindow() { call(procShowWindow, existing, SW_RESTORE); call(procSetForegroundWindow, existing) }
	return false
}
func releaseSingleInstance() { if mutexHandle != 0 { call(procCloseHandle, mutexHandle); mutexHandle = 0 } }
func shouldShowWindow() bool { for _, argument := range os.Args[1:] { switch strings.ToLower(strings.TrimSpace(argument)) { case "--show", "/show", "-show": return true } }; return false }
func isMainWindowVisible() bool { return mainHwnd != 0 && call(procIsWindowVisible, mainHwnd) != 0 }
func showMainWindow() { if mainHwnd != 0 { call(procShowWindow, mainHwnd, SW_RESTORE); call(procUpdateWindow, mainHwnd); call(procSetForegroundWindow, mainHwnd) } }
func hideMainWindow() { if mainHwnd != 0 { call(procShowWindow, mainHwnd, SW_HIDE) } }
func restoreMainWindowVisibility(wasVisible bool) { if wasVisible { showMainWindow() } else { hideMainWindow() } }
func initTrayIcon() error {
	trayData = NOTIFYICONDATAW{ CbSize:uint32(unsafe.Sizeof(NOTIFYICONDATAW{})), HWnd:mainHwnd, UID:1, UFlags:NIF_MESSAGE|NIF_ICON|NIF_TIP, UCallbackMessage:WM_TRAYICON, HIcon:loadAppIcon() }
	copyUTF16(trayData.SzTip[:], "Mini Extractor · shortcuts active in background")
	if call(procShellNotifyIconW, NIM_ADD, uintptr(unsafe.Pointer(&trayData))) == 0 { return errors.New("could not add system-tray icon") }
	trayReady=true; return nil
}
func removeTrayIcon() { if trayReady { call(procShellNotifyIconW,NIM_DELETE,uintptr(unsafe.Pointer(&trayData))); trayReady=false } }
func copyUTF16(destination []uint16, value string) { encoded,err:=syscall.UTF16FromString(value); if err!=nil{return}; if len(encoded)>len(destination){encoded=encoded[:len(destination)]}; copy(destination,encoded) }
func showTrayMenu() {
	menu:=call(procCreatePopupMenu); if menu==0{return}; defer call(procDestroyMenu,menu)
	appendTrayMenuItem(menu,ID_TRAY_OPEN,"Open Mini Extractor"); appendTrayMenuItem(menu,ID_TRAY_AREA,"Capture desktop area OCR"); appendTrayMenuItem(menu,ID_TRAY_IMAGE,"Open image OCR"); appendTrayMenuItem(menu,ID_TRAY_COLOR,"Pick desktop color"); appendTrayMenuItem(menu,ID_TRAY_EXIT,"Exit Mini Extractor")
	var point POINT; call(procGetCursorPos,uintptr(unsafe.Pointer(&point))); call(procSetForegroundWindow,mainHwnd)
	command:=call(procTrackPopupMenu,menu,TPM_RIGHTBUTTON|TPM_RETURNCMD,uintptr(int64(point.X)),uintptr(int64(point.Y)),0,mainHwnd,0)
	switch command { case ID_TRAY_OPEN: showMainWindow(); case ID_TRAY_AREA: go areaOCR(); case ID_TRAY_IMAGE: startImageOCRDialog(); case ID_TRAY_COLOR: go pickColor(); case ID_TRAY_EXIT: exitRequested=true; call(procDestroyWindow,mainHwnd) }
}
func appendTrayMenuItem(menu uintptr, identifier uintptr, label string) { call(procAppendMenuW,menu,MF_STRING,identifier,uintptr(unsafe.Pointer(p(label)))) }
func dialogOwner() uintptr { if isMainWindowVisible(){return mainHwnd}; return 0 }


func appIconPath() string {
	return filepath.Join(appDir, "assets", "app_icon.ico")
}

func loadAppIcon() uintptr {
	path := appIconPath()
	if _, err := os.Stat(path); err == nil {
		if handle := call(procLoadImageW, 0, uintptr(unsafe.Pointer(p(path))), IMAGE_ICON, 0, 0, LR_LOADFROMFILE); handle != 0 {
			return handle
		}
	}
	return call(procLoadIconW, 0, IDI_APPLICATION)
}

func initUIResources() {
	uiBackgroundBrush = call(procCreateSolidBrush, uintptr(rgb(245, 247, 250)))
	uiEditBrush = call(procCreateSolidBrush, uintptr(rgb(255, 255, 255)))

	uiBodyFont = createUIFont(-16, FW_NORMAL)
	uiSmallFont = createUIFont(-14, FW_NORMAL)
	uiSectionFont = createUIFont(-17, FW_SEMIBOLD)
	uiTitleFont = createUIFont(-31, FW_BOLD)
}

func disposeUIResources() {
	for _, handle := range []uintptr{
		uiBackgroundBrush,
		uiEditBrush,
		uiBodyFont,
		uiSmallFont,
		uiSectionFont,
		uiTitleFont,
	} {
		if handle != 0 {
			call(procDeleteObject, handle)
		}
	}
}

func createUIFont(height int32, weight int32) uintptr {
	return call(
		procCreateFontW,
		uintptr(int64(height)),
		0,
		0,
		0,
		uintptr(weight),
		0,
		0,
		0,
		1,
		0,
		0,
		5,
		0,
		uintptr(unsafe.Pointer(p("Segoe UI"))),
	)
}

func setControlFont(hwnd uintptr, font uintptr) {
	if hwnd == 0 || font == 0 {
		return
	}

	call(procSendMessageW, hwnd, WM_SETFONT, font, 1)
}

func styleActionButton(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	call(procSetWindowTheme, hwnd, uintptr(unsafe.Pointer(p("Explorer"))), 0)
	setControlFont(hwnd, uiBodyFont)
}

func styleSmallButton(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	call(procSetWindowTheme, hwnd, uintptr(unsafe.Pointer(p("Explorer"))), 0)
	setControlFont(hwnd, uiSmallFont)
}

func rgb(red, green, blue byte) uint32 {
	return uint32(red) | uint32(green)<<8 | uint32(blue)<<16
}

func setStatus(s string) { setWindowText(statusHwnd, s) }

func setEdit(s string) {
	setWindowText(editHwnd, s)
	applyTextDirection(s)
}

func getEdit() string { return getWindowText(editHwnd) }
func copySelectedEditText() {
	text := getEdit()
	if text == "" {
		setStatus("There is no extracted text to copy.")
		return
	}

	start, end := getEditSelection(editHwnd)
	if end <= start {
		setStatus("Select part of the extracted text first, then click Copy selection.")
		return
	}

	utf16Text, err := syscall.UTF16FromString(text)
	if err != nil || len(utf16Text) == 0 {
		setStatus("The extracted text could not be encoded for partial copy.")
		return
	}

	// UTF16FromString includes a trailing NUL.
	utf16Text = utf16Text[:len(utf16Text)-1]

	if start < 0 {
		start = 0
	}
	if end > len(utf16Text) {
		end = len(utf16Text)
	}
	if end <= start {
		setStatus("The current selection is empty.")
		return
	}

	selected := syscall.UTF16ToString(utf16Text[start:end])
	if selected == "" {
		setStatus("The current selection is empty.")
		return
	}

	if err := setClipboard(selected); err != nil {
		setStatus("Copy selection failed: " + err.Error())
		return
	}

	setStatus(fmt.Sprintf("Copied %d selected characters.", len([]rune(selected))))
}

func getEditSelection(hwnd uintptr) (int, int) {
	var start uint32
	var end uint32

	call(
		procSendMessageW,
		hwnd,
		EM_GETSEL,
		uintptr(unsafe.Pointer(&start)),
		uintptr(unsafe.Pointer(&end)),
	)

	return int(start), int(end)
}

func applyTextDirection(text string) {
	if editHwnd == 0 {
		return
	}

	mode := textDirectionMode
	if mode == "" {
		mode = "auto"
	}

	rtl := false
	switch mode {
	case "rtl":
		rtl = true
	case "ltr":
		rtl = false
	default:
		rtl = isMostlyRTL(text)
	}

	style := call(procGetWindowLongPtrW, editHwnd, signedIndex(GWL_STYLE))
	exStyle := call(procGetWindowLongPtrW, editHwnd, signedIndex(GWL_EXSTYLE))

	style &^= uintptr(ES_RIGHT)
	exStyle &^= uintptr(WS_EX_RTLREADING | WS_EX_LEFTSCROLLBAR)

	if rtl {
		style |= uintptr(ES_RIGHT)
		exStyle |= uintptr(WS_EX_RTLREADING | WS_EX_LEFTSCROLLBAR)
	}

	call(procSetWindowLongPtrW, editHwnd, signedIndex(GWL_STYLE), style)
	call(procSetWindowLongPtrW, editHwnd, signedIndex(GWL_EXSTYLE), exStyle)

	call(
		procSetWindowPos,
		editHwnd,
		0,
		0,
		0,
		0,
		0,
		SWP_NOMOVE|SWP_NOSIZE|SWP_NOZORDER|SWP_FRAMECHANGED,
	)

	call(procInvalidateRect, editHwnd, 0, 1)
}

func signedIndex(value int32) uintptr {
	return uintptr(int64(value))
}

func isMostlyRTL(text string) bool {
	rtlCount := 0
	ltrCount := 0

	for _, r := range text {
		switch {
		case isRTLScriptRune(r):
			rtlCount++
		case isLTRScriptRune(r):
			ltrCount++
		}
	}

	if rtlCount == 0 {
		return false
	}

	return rtlCount >= ltrCount
}

func isRTLScriptRune(r rune) bool {
	return (r >= 0x0590 && r <= 0x08FF) ||
		(r >= 0xFB1D && r <= 0xFDFF) ||
		(r >= 0xFE70 && r <= 0xFEFF) ||
		(r >= 0x1EE00 && r <= 0x1EEFF)
}

func isLTRScriptRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') ||
		(r >= 'a' && r <= 'z') ||
		(r >= 0x00C0 && r <= 0x02AF) ||
		(r >= 0x0370 && r <= 0x052F)
}

func copyEdit() {
	s := getEdit()
	if s == "" {
		setStatus("No extracted text to copy.")
		return
	}
	if err := setClipboard(s); err != nil {
		setStatus("Clipboard error: " + err.Error())
		return
	}
	setStatus("Result copied to clipboard.")
}

func areaOCR() {
	wasVisible := isMainWindowVisible()
	setStatus("Area selector active. Drag a precise rectangle around the text. Esc cancels. Your drag will not affect the underlying app.")
	hideMainWindow()
	time.Sleep(140 * time.Millisecond)
	rect, ok := selectRect()
	if !ok {
		restoreMainWindowVisibility(wasVisible)
		setStatus("Area OCR cancelled.")
		return
	}
	img, err := captureRect(expandOCRRect(rect, 7))
	restoreMainWindowVisibility(wasVisible)
	if err != nil {
		setStatus("Capture failed: " + err.Error())
		return
	}
	setStatus("Running OCR passes...")
	text, mode, lang, err := bestOCR(img)
	if err != nil {
		setStatus("OCR failed: " + err.Error())
		messageBox(mainHwnd, err.Error(), "OCR error", MB_OK|MB_ICONERROR)
		return
	}
	setEdit(text)
	_ = setClipboard(text)
	setStatus(fmt.Sprintf("Copied %d OCR characters · language: %s · best pass: %s", len([]rune(text)), lang, mode))
}
func startImageOCRDialog() {
	path, ok := chooseImage()
	if !ok {
		return
	}

	setStatus("Opening image preview...")
	img, err := decodeImageForPreview(path)
	if err != nil {
		setStatus("Preview unavailable. Running full-image OCR...")
		go runFullImageOCR(path)
		return
	}

	selected, full, ok, err := previewImageSelection(img)
	if err != nil {
		setStatus("Image preview failed: " + err.Error())
		messageBox(mainHwnd, err.Error(), "Image preview error", MB_OK|MB_ICONERROR)
		return
	}

	if !ok {
		setStatus("Image OCR cancelled.")
		return
	}

	if full {
		setStatus("Running full-image OCR passes...")
		go runFullImageOCR(path)
		return
	}

	setStatus("Running selected image-region OCR passes...")
	go runSelectedImageOCR(selected)
}

func runFullImageOCR(path string) {
	text, mode, lang, err := bestOCRFile(path)
	if err != nil {
		setStatus("Image OCR failed: " + err.Error())
		messageBox(mainHwnd, err.Error(), "Image OCR error", MB_OK|MB_ICONERROR)
		return
	}

	setEdit(text)
	_ = setClipboard(text)
	setStatus(fmt.Sprintf("Copied %d image OCR characters · language: %s · best pass: %s", len([]rune(text)), lang, mode))
}

func runSelectedImageOCR(img image.Image) {
	text, mode, lang, err := bestOCR(img)
	if err != nil {
		setStatus("Selected image-region OCR failed: " + err.Error())
		messageBox(mainHwnd, err.Error(), "Image OCR error", MB_OK|MB_ICONERROR)
		return
	}

	setEdit(text)
	_ = setClipboard(text)
	setStatus(fmt.Sprintf("Copied %d selected-region OCR characters · language: %s · best pass: %s", len([]rune(text)), lang, mode))
}

func decodeImageForPreview(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func bestOCRFile(path string) (string, string, string, error) {
	bestText := ""
	bestMode := ""
	bestLanguage := ""
	bestScore := -1e9

	// First let Windows decode the original file directly. This supports formats
	// that Go's standard image decoder does not handle, including BMP and TIFF.
	if rawText, rawLanguage, rawErr := invokeWindowsOCR(path); rawErr == nil {
		rawScore := scoreText(rawText)

		if strings.TrimSpace(rawText) != "" && rawScore > bestScore {
			bestText = rawText
			bestMode = "original file via Windows decoder"
			bestLanguage = rawLanguage
			bestScore = rawScore
		}
	} else {
		logger.Printf("direct Windows file OCR failed for %s: %v", path, rawErr)
	}

	file, openErr := os.Open(path)
	if openErr == nil {
		defer file.Close()

		decoded, _, decodeErr := image.Decode(file)
		if decodeErr == nil {
			processedText, processedMode, processedLanguage, processedErr := bestOCR(decoded)

			if processedErr == nil {
				processedScore := scoreText(processedText)

				if strings.TrimSpace(processedText) != "" && processedScore > bestScore {
					bestText = processedText
					bestMode = processedMode
					bestLanguage = processedLanguage
					bestScore = processedScore
				}
			} else {
				logger.Printf("processed image OCR failed for %s: %v", path, processedErr)
			}
		} else {
			// This is expected for some Windows-supported formats such as TIFF.
			logger.Printf("Go image preprocessing skipped for %s: %v", path, decodeErr)
		}
	} else {
		logger.Printf("image open failed for %s: %v", path, openErr)
	}

	if strings.TrimSpace(bestText) == "" {
		return "", "", "", errors.New(
			"Windows could not extract text from the selected image. Supported file choices include PNG, JPG, JPEG, BMP, GIF, TIF and TIFF. Try a clearer image, a tighter crop, or install the required Windows OCR language pack.",
		)
	}

	if strings.TrimSpace(bestLanguage) == "" {
		bestLanguage = "automatic"
	}

	return strings.TrimSpace(bestText), bestMode, bestLanguage, nil
}

func previewImageSelection(src image.Image) (image.Image, bool, bool, error) {
	if src == nil {
		return nil, false, false, errors.New("the selected image could not be decoded for preview")
	}

	bounds := src.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil, false, false, errors.New("the selected image has invalid dimensions")
	}

	if !atomic.CompareAndSwapInt32(&toolActive, 0, 1) {
		return nil, false, false, errors.New("another Mini Extractor tool is already active")
	}
	defer atomic.StoreInt32(&toolActive, 0)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	previewSource = src
	previewPixels, previewBMI = imageToBGRA(src)
	previewSelection = RECT{}
	previewDisplayRect = RECT{}
	previewSelecting = false
	previewOK = false
	previewFull = false
	previewResult = nil

	hInstance := call(procGetModuleHandleW, 0)
	screenW := int32(call(procGetSystemMetrics, SM_CXVIRTUALSCREEN))
	screenH := int32(call(procGetSystemMetrics, SM_CYVIRTUALSCREEN))

	windowW := min32(1200, max32(760, screenW-140))
	windowH := min32(900, max32(560, screenH-140))

	previewHwnd = call(
		procCreateWindowExW,
		WS_EX_TOPMOST|WS_EX_TOOLWINDOW,
		uintptr(unsafe.Pointer(p(previewClassName))),
		uintptr(unsafe.Pointer(p("Image OCR preview · Drag a text region · Enter extracts the full image · Esc cancels"))),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		uintptr(int64(70)),
		uintptr(int64(50)),
		uintptr(windowW),
		uintptr(windowH),
		mainHwnd,
		0,
		hInstance,
		0,
	)

	if previewHwnd == 0 {
		return nil, false, false, errors.New("could not create the image preview window")
	}

	call(procSetForegroundWindow, previewHwnd)
	call(procSetFocus, previewHwnd)
	call(procShowWindow, previewHwnd, SW_SHOW)
	call(procUpdateWindow, previewHwnd)

	var msg MSG
	for previewHwnd != 0 && call(procGetMessageW, uintptr(unsafe.Pointer(&msg)), 0, 0, 0) > 0 {
		call(procTranslateMessage, uintptr(unsafe.Pointer(&msg)))
		call(procDispatchMessageW, uintptr(unsafe.Pointer(&msg)))
	}

	result := previewResult
	full := previewFull
	ok := previewOK

	previewSource = nil
	previewPixels = nil
	previewResult = nil
	previewSelection = RECT{}
	previewDisplayRect = RECT{}

	return result, full, ok, nil
}

func previewWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_SETCURSOR:
		call(procSetCursor, call(procLoadCursorW, 0, IDC_CROSS))
		return 1

	case WM_KEYDOWN:
		switch int(wParam) {
		case VK_ESCAPE:
			previewOK = false
			previewFull = false
			call(procDestroyWindow, hwnd)
			return 0

		case VK_RETURN:
			previewOK = true
			previewFull = true
			previewResult = previewSource
			call(procDestroyWindow, hwnd)
			return 0
		}

	case WM_LBUTTONDOWN:
		pt := pointFromLParam(lParam)
		if pointInsideRect(pt, previewDisplayRect) {
			previewSelecting = true
			previewSelection = RECT{Left: pt.X, Top: pt.Y, Right: pt.X, Bottom: pt.Y}
			call(procSetCapture, hwnd)
			call(procInvalidateRect, hwnd, 0, 1)
		}
		return 0

	case WM_MOUSEMOVE:
		if previewSelecting {
			pt := clampPointToRect(pointFromLParam(lParam), previewDisplayRect)
			previewSelection = normRect(POINT{X: previewSelection.Left, Y: previewSelection.Top}, pt)
			call(procInvalidateRect, hwnd, 0, 1)
		}
		return 0

	case WM_LBUTTONUP:
		if previewSelecting {
			previewSelecting = false
			call(procReleaseCapture)

			pt := clampPointToRect(pointFromLParam(lParam), previewDisplayRect)
			previewSelection = normRect(POINT{X: previewSelection.Left, Y: previewSelection.Top}, pt)

			if previewSelection.Right-previewSelection.Left > 5 && previewSelection.Bottom-previewSelection.Top > 5 {
				previewResult = cropPreviewSelection(previewSource, previewSelection, previewDisplayRect)
				previewOK = previewResult != nil
				previewFull = false
			}

			call(procDestroyWindow, hwnd)
		}
		return 0

	case WM_PAINT:
		var paint PAINTSTRUCT
		dc := call(procBeginPaint, hwnd, uintptr(unsafe.Pointer(&paint)))

		if dc != 0 {
			var client RECT
			call(procGetClientRect, hwnd, uintptr(unsafe.Pointer(&client)))
			call(procFillRect, dc, uintptr(unsafe.Pointer(&client)), call(procGetStockObject, BLACK_BRUSH))

			previewDisplayRect = fitImageRect(previewSource, client, 18, 18, 18, 18)

			if len(previewPixels) > 0 && previewDisplayRect.Right > previewDisplayRect.Left && previewDisplayRect.Bottom > previewDisplayRect.Top {
				srcBounds := previewSource.Bounds()

				call(
					procStretchDIBits,
					dc,
					uintptr(previewDisplayRect.Left),
					uintptr(previewDisplayRect.Top),
					uintptr(previewDisplayRect.Right-previewDisplayRect.Left),
					uintptr(previewDisplayRect.Bottom-previewDisplayRect.Top),
					0,
					0,
					uintptr(srcBounds.Dx()),
					uintptr(srcBounds.Dy()),
					uintptr(unsafe.Pointer(&previewPixels[0])),
					uintptr(unsafe.Pointer(&previewBMI)),
					DIB_RGB_COLORS,
					SRCCOPY,
				)
			}

			if previewSelecting && previewSelection.Right > previewSelection.Left && previewSelection.Bottom > previewSelection.Top {
				selection := previewSelection
				call(procDrawFocusRect, dc, uintptr(unsafe.Pointer(&selection)))
			}
		}

		call(procEndPaint, hwnd, uintptr(unsafe.Pointer(&paint)))
		return 0

	case WM_CLOSE:
		previewOK = false
		previewFull = false
		call(procDestroyWindow, hwnd)
		return 0

	case WM_DESTROY:
		previewHwnd = 0
		return 0
	}

	return call(procDefWindowProcW, hwnd, uintptr(msg), wParam, lParam)
}

func imageToBGRA(src image.Image) ([]byte, BITMAPINFO) {
	if src == nil {
		return nil, BITMAPINFO{}
	}

	b := src.Bounds()
	width := b.Dx()
	height := b.Dy()

	if width <= 0 || height <= 0 {
		return nil, BITMAPINFO{}
	}

	pixels := make([]byte, width*height*4)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, blue, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			index := (y*width + x) * 4
			pixels[index] = byte(blue >> 8)
			pixels[index+1] = byte(g >> 8)
			pixels[index+2] = byte(r >> 8)
			pixels[index+3] = 255
		}
	}

	bmi := BITMAPINFO{
		BmiHeader: BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      -int32(height),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: BI_RGB,
		},
	}

	return pixels, bmi
}

func fitImageRect(src image.Image, client RECT, leftMargin, topMargin, rightMargin, bottomMargin int32) RECT {
	if src == nil {
		return RECT{}
	}

	b := src.Bounds()
	availableW := max32(1, client.Right-client.Left-leftMargin-rightMargin)
	availableH := max32(1, client.Bottom-client.Top-topMargin-bottomMargin)

	scaleX := float64(availableW) / float64(b.Dx())
	scaleY := float64(availableH) / float64(b.Dy())
	scale := math.Min(scaleX, scaleY)

	width := int32(math.Max(1, math.Round(float64(b.Dx())*scale)))
	height := int32(math.Max(1, math.Round(float64(b.Dy())*scale)))

	x := leftMargin + (availableW-width)/2
	y := topMargin + (availableH-height)/2

	return RECT{Left: x, Top: y, Right: x + width, Bottom: y + height}
}

func pointInsideRect(point POINT, rectangle RECT) bool {
	return point.X >= rectangle.Left &&
		point.X <= rectangle.Right &&
		point.Y >= rectangle.Top &&
		point.Y <= rectangle.Bottom
}

func clampPointToRect(point POINT, rectangle RECT) POINT {
	return POINT{
		X: max32(rectangle.Left, min32(rectangle.Right, point.X)),
		Y: max32(rectangle.Top, min32(rectangle.Bottom, point.Y)),
	}
}

func cropPreviewSelection(src image.Image, selected RECT, displayed RECT) image.Image {
	if src == nil || displayed.Right <= displayed.Left || displayed.Bottom <= displayed.Top {
		return nil
	}

	b := src.Bounds()
	scaleX := float64(b.Dx()) / float64(displayed.Right-displayed.Left)
	scaleY := float64(b.Dy()) / float64(displayed.Bottom-displayed.Top)

	left := int(math.Floor(float64(selected.Left-displayed.Left) * scaleX))
	top := int(math.Floor(float64(selected.Top-displayed.Top) * scaleY))
	right := int(math.Ceil(float64(selected.Right-displayed.Left) * scaleX))
	bottom := int(math.Ceil(float64(selected.Bottom-displayed.Top) * scaleY))

	left = maxInt(0, left-4)
	top = maxInt(0, top-4)
	right = minInt(b.Dx(), right+4)
	bottom = minInt(b.Dy(), bottom+4)

	if right <= left || bottom <= top {
		return nil
	}

	out := image.NewRGBA(image.Rect(0, 0, right-left, bottom-top))
	draw.Draw(out, out.Bounds(), src, image.Pt(b.Min.X+left, b.Min.Y+top), draw.Src)
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func selectRect() (RECT, bool) {
	if !atomic.CompareAndSwapInt32(&toolActive, 0, 1) {
		return RECT{}, false
	}
	defer atomic.StoreInt32(&toolActive, 0)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	overlayMode = 1
	overlayOK = false
	overlaySelecting = false
	overlayRect = RECT{}
	overlayCursor = call(procLoadCursorW, 0, IDC_CROSS)

	if err := runOverlay(58); err != nil {
		logger.Printf("region overlay failed: %v", err)
		return RECT{}, false
	}
	return overlayRect, overlayOK && overlayRect.Right-overlayRect.Left > 5 && overlayRect.Bottom-overlayRect.Top > 5
}

func pickColor() {
	if !atomic.CompareAndSwapInt32(&toolActive, 0, 1) {
		setStatus("Another Mini Extractor tool is already active.")
		return
	}
	defer atomic.StoreInt32(&toolActive, 0)

	setStatus("Eyedropper active. Click a pixel to copy its HEX value. Esc cancels. The click will not reach the underlying app.")
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	overlayMode = 2
	overlayOK = false
	overlayPoint = POINT{}
	curFile := filepath.Join(appDir, "assets", "eyedropper.cur")
	overlayCursor = call(procLoadCursorFromFileW, uintptr(unsafe.Pointer(p(curFile))))
	if overlayCursor == 0 {
		overlayCursor = call(procLoadCursorW, 0, IDC_CROSS)
	}

	if err := runOverlay(1); err != nil {
		setStatus("Color picker failed: " + err.Error())
		return
	}
	if !overlayOK {
		setStatus("Color picker cancelled.")
		return
	}

	// The overlay consumed the mouse click. Hide it before sampling so the sampled pixel
	// comes from the real desktop/application underneath it.
	time.Sleep(35 * time.Millisecond)
	dc := call(procGetDC, 0)
	if dc == 0 {
		setStatus("Color picker failed: GetDC failed")
		return
	}
	defer call(procReleaseDC, 0, dc)

	raw := call(procGetPixel, dc, uintptr(int64(overlayPoint.X)), uintptr(int64(overlayPoint.Y)))
	r := byte(raw & 0xff)
	g := byte((raw >> 8) & 0xff)
	b := byte((raw >> 16) & 0xff)
	hex := fmt.Sprintf("#%02X%02X%02X", r, g, b)
	_ = setClipboard(hex)
	setStatus(fmt.Sprintf("%s copied · rgb(%d, %d, %d)", hex, r, g, b))
}

func runOverlay(alpha byte) error {
	hInstance := call(procGetModuleHandleW, 0)
	x := int32(call(procGetSystemMetrics, SM_XVIRTUALSCREEN))
	y := int32(call(procGetSystemMetrics, SM_YVIRTUALSCREEN))
	w := int32(call(procGetSystemMetrics, SM_CXVIRTUALSCREEN))
	h := int32(call(procGetSystemMetrics, SM_CYVIRTUALSCREEN))
	if w <= 0 || h <= 0 {
		return errors.New("virtual screen size is invalid")
	}

	overlayHwnd = call(procCreateWindowExW,
		WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_LAYERED,
		uintptr(unsafe.Pointer(p(overlayClassName))),
		uintptr(unsafe.Pointer(p("Mini Extractor Overlay"))),
		WS_POPUP|WS_VISIBLE,
		uintptr(int64(x)), uintptr(int64(y)), uintptr(w), uintptr(h),
		0, 0, hInstance, 0,
	)
	if overlayHwnd == 0 {
		return errors.New("CreateWindowExW overlay failed")
	}
	call(procSetLayeredWindowAttr, overlayHwnd, 0, uintptr(alpha), LWA_ALPHA)
	call(procSetForegroundWindow, overlayHwnd)
	call(procSetCursor, overlayCursor)
	call(procShowWindow, overlayHwnd, SW_SHOW)
	call(procUpdateWindow, overlayHwnd)

	var msg MSG
	for call(procGetMessageW, uintptr(unsafe.Pointer(&msg)), 0, 0, 0) > 0 {
		call(procTranslateMessage, uintptr(unsafe.Pointer(&msg)))
		call(procDispatchMessageW, uintptr(unsafe.Pointer(&msg)))
	}
	overlayHwnd = 0
	return nil
}

func overlayWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_SETCURSOR:
		call(procSetCursor, overlayCursor)
		return 1
	case WM_KEYDOWN:
		if int(wParam) == VK_ESCAPE {
			overlayOK = false
			call(procDestroyWindow, hwnd)
			return 0
		}
	case WM_LBUTTONDOWN:
		pt := pointFromLParam(lParam)
		if overlayMode == 2 {
			overlayPoint = clientToScreenPoint(pt)
			overlayOK = true
			call(procShowWindow, hwnd, SW_HIDE)
			call(procDestroyWindow, hwnd)
			return 0
		}
		overlaySelecting = true
		overlayStart = clientToScreenPoint(pt)
		overlayCurrent = overlayStart
		overlayRect = normRect(overlayStart, overlayCurrent)
		call(procSetCapture, hwnd)
		call(procInvalidateRect, hwnd, 0, 1)
		return 0
	case WM_MOUSEMOVE:
		if overlayMode == 1 && overlaySelecting {
			overlayCurrent = clientToScreenPoint(pointFromLParam(lParam))
			overlayRect = normRect(overlayStart, overlayCurrent)
			call(procInvalidateRect, hwnd, 0, 1)
			return 0
		}
	case WM_LBUTTONUP:
		if overlayMode == 1 && overlaySelecting {
			overlaySelecting = false
			overlayCurrent = clientToScreenPoint(pointFromLParam(lParam))
			overlayRect = normRect(overlayStart, overlayCurrent)
			overlayOK = overlayRect.Right-overlayRect.Left > 5 && overlayRect.Bottom-overlayRect.Top > 5
			call(procReleaseCapture)
			call(procDestroyWindow, hwnd)
			return 0
		}
	case WM_PAINT:
		var ps PAINTSTRUCT
		dc := call(procBeginPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		if overlayMode == 1 && overlaySelecting && dc != 0 {
			r := screenToClientRect(overlayRect)
			call(procDrawFocusRect, dc, uintptr(unsafe.Pointer(&r)))
		}
		call(procEndPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_DESTROY:
		call(procPostQuitMessage, 0)
		return 0
	}
	return call(procDefWindowProcW, hwnd, uintptr(msg), wParam, lParam)
}

func pointFromLParam(lParam uintptr) POINT {
	return POINT{X: int32(int16(lParam & 0xffff)), Y: int32(int16((lParam >> 16) & 0xffff))}
}
func clientToScreenPoint(pt POINT) POINT {
	return POINT{X: pt.X + int32(call(procGetSystemMetrics, SM_XVIRTUALSCREEN)), Y: pt.Y + int32(call(procGetSystemMetrics, SM_YVIRTUALSCREEN))}
}
func screenToClientRect(r RECT) RECT {
	x := int32(call(procGetSystemMetrics, SM_XVIRTUALSCREEN))
	y := int32(call(procGetSystemMetrics, SM_YVIRTUALSCREEN))
	return RECT{Left: r.Left - x, Top: r.Top - y, Right: r.Right - x, Bottom: r.Bottom - y}
}
func normRect(a, b POINT) RECT {
	return RECT{Left: min32(a.X, b.X), Top: min32(a.Y, b.Y), Right: max32(a.X, b.X), Bottom: max32(a.Y, b.Y)}
}
func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}


func expandOCRRect(r RECT, padding int32) RECT {
	x := int32(call(procGetSystemMetrics, SM_XVIRTUALSCREEN))
	y := int32(call(procGetSystemMetrics, SM_YVIRTUALSCREEN))
	w := int32(call(procGetSystemMetrics, SM_CXVIRTUALSCREEN))
	h := int32(call(procGetSystemMetrics, SM_CYVIRTUALSCREEN))

	left := max32(x, r.Left-padding)
	top := max32(y, r.Top-padding)
	right := min32(x+w, r.Right+padding)
	bottom := min32(y+h, r.Bottom+padding)

	if right <= left || bottom <= top {
		return r
	}

	return RECT{Left: left, Top: top, Right: right, Bottom: bottom}
}

func captureRect(r RECT) (*image.RGBA, error) {
	w := int(r.Right - r.Left)
	h := int(r.Bottom - r.Top)
	if w <= 0 || h <= 0 {
		return nil, errors.New("invalid capture rectangle")
	}
	src := call(procGetDC, 0)
	if src == 0 {
		return nil, errors.New("GetDC failed")
	}
	defer call(procReleaseDC, 0, src)
	mem := call(procCreateCompatibleDC, src)
	if mem == 0 {
		return nil, errors.New("CreateCompatibleDC failed")
	}
	defer call(procDeleteDC, mem)
	bmp := call(procCreateCompatibleBmp, src, uintptr(w), uintptr(h))
	if bmp == 0 {
		return nil, errors.New("CreateCompatibleBitmap failed")
	}
	defer call(procDeleteObject, bmp)
	old := call(procSelectObject, mem, bmp)
	defer call(procSelectObject, mem, old)
	if call(procBitBlt, mem, 0, 0, uintptr(w), uintptr(h), src, uintptr(r.Left), uintptr(r.Top), SRCCOPY) == 0 {
		return nil, errors.New("BitBlt failed")
	}
	bmi := BITMAPINFO{BmiHeader: BITMAPINFOHEADER{BiSize: uint32(unsafe.Sizeof(BITMAPINFOHEADER{})), BiWidth: int32(w), BiHeight: -int32(h), BiPlanes: 1, BiBitCount: 32, BiCompression: BI_RGB}}
	buf := make([]byte, w*h*4)
	if call(procGetDIBits, mem, bmp, 0, uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bmi)), DIB_RGB_COLORS) == 0 {
		return nil, errors.New("GetDIBits failed")
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		out.Pix[i*4] = buf[i*4+2]
		out.Pix[i*4+1] = buf[i*4+1]
		out.Pix[i*4+2] = buf[i*4]
		out.Pix[i*4+3] = 255
	}
	return out, nil
}

func bestOCR(src image.Image) (string, string, string, error) {
	tmp, err := os.MkdirTemp("", "mini-extractor-")
	if err != nil {
		return "", "", "", err
	}
	defer os.RemoveAll(tmp)

	normalized := fitWithinMax(src, 2400)
	enlarged := resizeForOCR(normalized, 2200)
	gray := autoContrastGray(enlarged)
	sharp := sharpenGray(gray)
	adaptiveSmall := adaptiveThreshold(sharp, 13, 8)
	adaptiveLarge := adaptiveThreshold(sharp, 23, 12)

	passes := []struct {
		Name string
		Img  image.Image
	}{
		{"original normalized crop", normalized},
		{"OCR-sized enlarged crop", enlarged},
		{"sharpened grayscale", sharp},
		{"adaptive threshold · fine", adaptiveSmall},
		{"adaptive threshold · wide", adaptiveLarge},
		{"inverted adaptive threshold", invertGray(adaptiveSmall)},
	}

	bestText := ""
	bestMode := ""
	bestLang := ""
	bestScore := -1e9

	for i, pass := range passes {
		setStatus("OCR pass: " + pass.Name + " · automatic language detection...")

		path := filepath.Join(tmp, fmt.Sprintf("pass-%d.png", i))
		f, createErr := os.Create(path)
		if createErr != nil {
			logger.Printf("OCR pass %s could not create file: %v", pass.Name, createErr)
			continue
		}

		encodeErr := png.Encode(f, pass.Img)
		_ = f.Close()

		if encodeErr != nil {
			logger.Printf("OCR pass %s could not encode PNG: %v", pass.Name, encodeErr)
			continue
		}

		text, language, ocrErr := invokeWindowsOCR(path)
		if ocrErr != nil {
			logger.Printf("OCR pass %s failed: %v", pass.Name, ocrErr)
			continue
		}

		sc := scoreText(text)
		if sc > bestScore {
			bestScore = sc
			bestText = text
			bestMode = pass.Name
			bestLang = language
		}
	}

	if strings.TrimSpace(bestText) == "" {
		return "", "", "", errors.New("Windows OCR returned no text. Select a tighter text region and install the required Windows language pack in Settings > Time & language > Language & region.")
	}

	if strings.TrimSpace(bestLang) == "" {
		bestLang = "automatic"
	}

	return strings.TrimSpace(bestText), bestMode, bestLang, nil
}

func invokeWindowsOCR(path string) (string, string, error) {
	helper := filepath.Join(appDir, "ocr.ps1")
	if _, err := os.Stat(helper); err != nil {
		return "", "", errors.New("ocr.ps1 is missing from the installed folder")
	}

	outFile := filepath.Join(os.TempDir(), fmt.Sprintf("mini-extractor-ocr-%d.txt", time.Now().UnixNano()))
	defer os.Remove(outFile)

	cmd := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-STA",
		"-File", helper,
		"-ImagePath", path,
		"-OutputPath", outFile,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("Windows OCR bridge failed: %v %s", err, strings.TrimSpace(string(output)))
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		return "", "", err
	}

	raw := strings.TrimSpace(string(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})))
	const marker = "__MINIEXTRACTOR_LANGUAGE__="

	language := "automatic"
	text := raw

	if strings.HasPrefix(raw, marker) {
		parts := strings.SplitN(raw, "\n", 2)
		language = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(parts[0]), marker))

		if len(parts) == 2 {
			text = strings.TrimSpace(parts[1])
		} else {
			text = ""
		}
	}

	return text, language, nil
}
func scoreText(s string) float64 {
	var score float64
	for _, r := range s {
		if unicode.IsLetter(r) {
			score += 2.2
		} else if unicode.IsDigit(r) {
			score += 1.4
		} else if unicode.IsSpace(r) {
			score += 0.05
		} else if unicode.IsPunct(r) {
			score += 0.2
		} else {
			score -= 1.5
		}
	}
	return score
}

func fitWithinMax(src image.Image, maxSide int) image.Image {
	b := src.Bounds()
	longest := b.Dx()
	if b.Dy() > longest {
		longest = b.Dy()
	}

	if longest <= maxSide {
		return src
	}

	scale := float64(maxSide) / float64(longest)
	width := maxInt(1, int(math.Round(float64(b.Dx())*scale)))
	height := maxInt(1, int(math.Round(float64(b.Dy())*scale)))

	return resizeBilinearTo(src, width, height)
}

func resizeForOCR(src image.Image, targetLongest int) image.Image {
	b := src.Bounds()
	longest := b.Dx()
	if b.Dy() > longest {
		longest = b.Dy()
	}

	if longest <= 0 {
		return src
	}

	// Avoid tiny OCR input, but stay below Windows OCR dimension limits.
	scale := float64(targetLongest) / float64(longest)
	if scale < 1.0 {
		scale = 1.0
	}
	if scale > 4.0 {
		scale = 4.0
	}

	width := maxInt(1, int(math.Round(float64(b.Dx())*scale)))
	height := maxInt(1, int(math.Round(float64(b.Dy())*scale)))

	longestOutput := width
	if height > longestOutput {
		longestOutput = height
	}

	if longestOutput > 2400 {
		limitScale := 2400.0 / float64(longestOutput)
		width = maxInt(1, int(math.Round(float64(width)*limitScale)))
		height = maxInt(1, int(math.Round(float64(height)*limitScale)))
	}

	if width == b.Dx() && height == b.Dy() {
		return src
	}

	return resizeBilinearTo(src, width, height)
}

func resizeBilinearTo(src image.Image, width, height int) image.Image {
	b := src.Bounds()
	sourceWidth := b.Dx()
	sourceHeight := b.Dy()

	if width <= 0 || height <= 0 || sourceWidth <= 0 || sourceHeight <= 0 {
		return src
	}

	out := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		sy := (float64(y)+0.5)*float64(sourceHeight)/float64(height) - 0.5
		y0 := int(math.Floor(sy))
		fy := sy - float64(y0)

		if y0 < 0 {
			y0 = 0
			fy = 0
		}

		y1 := y0 + 1
		if y1 >= sourceHeight {
			y1 = sourceHeight - 1
		}

		for x := 0; x < width; x++ {
			sx := (float64(x)+0.5)*float64(sourceWidth)/float64(width) - 0.5
			x0 := int(math.Floor(sx))
			fx := sx - float64(x0)

			if x0 < 0 {
				x0 = 0
				fx = 0
			}

			x1 := x0 + 1
			if x1 >= sourceWidth {
				x1 = sourceWidth - 1
			}

			c00 := color.RGBAModel.Convert(src.At(b.Min.X+x0, b.Min.Y+y0)).(color.RGBA)
			c10 := color.RGBAModel.Convert(src.At(b.Min.X+x1, b.Min.Y+y0)).(color.RGBA)
			c01 := color.RGBAModel.Convert(src.At(b.Min.X+x0, b.Min.Y+y1)).(color.RGBA)
			c11 := color.RGBAModel.Convert(src.At(b.Min.X+x1, b.Min.Y+y1)).(color.RGBA)

			blend := func(a, b, c, d uint8) uint8 {
				value := (1-fx)*(1-fy)*float64(a) +
					fx*(1-fy)*float64(b) +
					(1-fx)*fy*float64(c) +
					fx*fy*float64(d)

				if value < 0 {
					value = 0
				}
				if value > 255 {
					value = 255
				}

				return uint8(math.Round(value))
			}

			out.SetRGBA(x, y, color.RGBA{
				R: blend(c00.R, c10.R, c01.R, c11.R),
				G: blend(c00.G, c10.G, c01.G, c11.G),
				B: blend(c00.B, c10.B, c01.B, c11.B),
				A: 255,
			})
		}
	}

	return out
}

func sharpenGray(src *image.Gray) *image.Gray {
	b := src.Bounds()
	width := b.Dx()
	height := b.Dy()
	out := image.NewGray(image.Rect(0, 0, width, height))

	if width < 3 || height < 3 {
		copy(out.Pix, src.Pix)
		return out
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x == 0 || y == 0 || x == width-1 || y == height-1 {
				out.SetGray(x, y, src.GrayAt(x, y))
				continue
			}

			center := int(src.GrayAt(x, y).Y) * 5
			left := int(src.GrayAt(x-1, y).Y)
			right := int(src.GrayAt(x+1, y).Y)
			up := int(src.GrayAt(x, y-1).Y)
			down := int(src.GrayAt(x, y+1).Y)

			value := center - left - right - up - down
			if value < 0 {
				value = 0
			}
			if value > 255 {
				value = 255
			}

			out.SetGray(x, y, color.Gray{Y: uint8(value)})
		}
	}

	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func resizeBilinear(src image.Image, factor int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if factor < 1 {
		factor = 1
	}
	out := image.NewRGBA(image.Rect(0, 0, w*factor, h*factor))
	for y := 0; y < out.Bounds().Dy(); y++ {
		sy := float64(y) / float64(factor)
		y0 := int(math.Floor(sy))
		if y0 >= h {
			y0 = h - 1
		}
		y1 := y0 + 1
		if y1 >= h {
			y1 = h - 1
		}
		fy := sy - float64(y0)
		for x := 0; x < out.Bounds().Dx(); x++ {
			sx := float64(x) / float64(factor)
			x0 := int(math.Floor(sx))
			if x0 >= w {
				x0 = w - 1
			}
			x1 := x0 + 1
			if x1 >= w {
				x1 = w - 1
			}
			fx := sx - float64(x0)
			c00 := color.RGBAModel.Convert(src.At(b.Min.X+x0, b.Min.Y+y0)).(color.RGBA)
			c10 := color.RGBAModel.Convert(src.At(b.Min.X+x1, b.Min.Y+y0)).(color.RGBA)
			c01 := color.RGBAModel.Convert(src.At(b.Min.X+x0, b.Min.Y+y1)).(color.RGBA)
			c11 := color.RGBAModel.Convert(src.At(b.Min.X+x1, b.Min.Y+y1)).(color.RGBA)
			blend := func(a, b, c, d uint8) uint8 {
				return uint8((1-fx)*(1-fy)*float64(a) + fx*(1-fy)*float64(b) + (1-fx)*fy*float64(c) + fx*fy*float64(d))
			}
			out.SetRGBA(x, y, color.RGBA{blend(c00.R, c10.R, c01.R, c11.R), blend(c00.G, c10.G, c01.G, c11.G), blend(c00.B, c10.B, c01.B, c11.B), 255})
		}
	}
	return out
}
func autoContrastGray(src image.Image) *image.Gray {
	b := src.Bounds()
	out := image.NewGray(image.Rect(0, 0, b.Dx(), b.Dy()))
	minV, maxV := uint8(255), uint8(0)
	vals := make([]uint8, b.Dx()*b.Dy())
	i := 0
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			g := color.GrayModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.Gray).Y
			vals[i] = g
			i++
			if g < minV {
				minV = g
			}
			if g > maxV {
				maxV = g
			}
		}
	}
	if maxV <= minV {
		copy(out.Pix, vals)
		return out
	}
	for i, g := range vals {
		out.Pix[i] = uint8((int(g) - int(minV)) * 255 / (int(maxV) - int(minV)))
	}
	return out
}
func adaptiveThreshold(src *image.Gray, radius int, offset int) *image.Gray {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewGray(image.Rect(0, 0, w, h))
	integral := make([]int, (w+1)*(h+1))
	for y := 1; y <= h; y++ {
		row := 0
		for x := 1; x <= w; x++ {
			row += int(src.GrayAt(x-1, y-1).Y)
			integral[y*(w+1)+x] = integral[(y-1)*(w+1)+x] + row
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			x0 := x - radius
			if x0 < 0 {
				x0 = 0
			}
			y0 := y - radius
			if y0 < 0 {
				y0 = 0
			}
			x1 := x + radius
			if x1 >= w {
				x1 = w - 1
			}
			y1 := y + radius
			if y1 >= h {
				y1 = h - 1
			}
			area := (x1 - x0 + 1) * (y1 - y0 + 1)
			sum := integral[(y1+1)*(w+1)+(x1+1)] - integral[y0*(w+1)+(x1+1)] - integral[(y1+1)*(w+1)+x0] + integral[y0*(w+1)+x0]
			mean := sum / area
			if int(src.GrayAt(x, y).Y) < mean-offset {
				out.SetGray(x, y, color.Gray{0})
			} else {
				out.SetGray(x, y, color.Gray{255})
			}
		}
	}
	return out
}
func invertGray(src *image.Gray) *image.Gray {
	b := src.Bounds()
	out := image.NewGray(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.SetGray(x, y, color.Gray{255 - src.GrayAt(x, y).Y})
		}
	}
	return out
}

func resizeNN(src image.Image, factor int) image.Image {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*factor, b.Dy()*factor))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.Set(x, y, src.At(b.Min.X+x/factor, b.Min.Y+y/factor))
		}
	}
	return out
}
func contrastGray(src image.Image, factor float64) image.Image {
	b := src.Bounds()
	out := image.NewGray(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			g := color.GrayModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.Gray).Y
			v := int(math.Round((float64(g)-128)*factor + 128))
			if v < 0 {
				v = 0
			}
			if v > 255 {
				v = 255
			}
			out.SetGray(x, y, color.Gray{uint8(v)})
		}
	}
	return out
}
func threshold(src image.Image, t uint8) image.Image {
	b := src.Bounds()
	out := image.NewGray(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			g := color.GrayModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.Gray).Y
			if g >= t {
				g = 255
			} else {
				g = 0
			}
			out.SetGray(x, y, color.Gray{g})
		}
	}
	return out
}

func defaultHotkeyConfig() HotkeyConfig {
	return HotkeyConfig{
		Area:  "Ctrl+Shift+T",
		Image: "Ctrl+Shift+I",
		Color: "Ctrl+Shift+C",
	}
}

func shortcutConfigPath() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "MiniExtractorGo", "shortcuts.txt")
}

func loadHotkeyConfig() HotkeyConfig {
	config := defaultHotkeyConfig()

	data, err := os.ReadFile(shortcutConfigPath())
	if err != nil {
		return config
	}

	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if value == "" {
			continue
		}

		switch key {
		case "area":
			config.Area = value
		case "image":
			config.Image = value
		case "color":
			config.Color = value
		}
	}

	return config
}

func saveHotkeyConfig(config HotkeyConfig) error {
	path := shortcutConfigPath()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data := strings.Join([]string{
		"Area=" + config.Area,
		"Image=" + config.Image,
		"Color=" + config.Color,
		"",
	}, "\n")

	return os.WriteFile(path, []byte(data), 0644)
}

func setHotkeyLabels(config HotkeyConfig) {
	setWindowText(areaHotkeyLabelHwnd, "  "+config.Area)
	setWindowText(imageHotkeyLabelHwnd, "  "+config.Image)
	setWindowText(colorHotkeyLabelHwnd, "  "+config.Color)
}

func startShortcutRecording(action int) {
	unregisterConfiguredHotkeys()
	recordingShortcut = action

	actionLabel := "shortcut"
	switch action {
	case ID_CHANGE_AREA:
		actionLabel = "Area OCR shortcut"
	case ID_CHANGE_IMAGE:
		actionLabel = "Image OCR shortcut"
	case ID_CHANGE_COLOR:
		actionLabel = "Color picker shortcut"
	}

	setStatus(actionLabel + ": press Ctrl, Shift, Alt or Win together with A-Z, 0-9 or F1-F12. Esc cancels.")
	call(procSetForegroundWindow, mainHwnd)
	call(procSetFocus, mainHwnd)
}

func cancelShortcutRecording() {
	recordingShortcut = 0

	if err := registerConfiguredHotkeys(currentHotkeys); err != nil {
		logger.Printf("shortcut restore after cancellation failed: %v", err)
	}

	setHotkeyLabels(currentHotkeys)
	setStatus("Shortcut change cancelled.")
}

func handleRecordedShortcut(key uint32) {
	if key == VK_ESCAPE {
		cancelShortcutRecording()
		return
	}

	if key == VK_SHIFT || key == VK_CONTROL || key == VK_MENU || key == VK_LWIN || key == VK_RWIN {
		return
	}

	shortcut, err := shortcutFromPressedKey(key)
	if err != nil {
		setStatus("Shortcut not accepted: " + err.Error())
		return
	}

	updated := currentHotkeys

	switch recordingShortcut {
	case ID_CHANGE_AREA:
		updated.Area = shortcut.Text
	case ID_CHANGE_IMAGE:
		updated.Image = shortcut.Text
	case ID_CHANGE_COLOR:
		updated.Color = shortcut.Text
	default:
		cancelShortcutRecording()
		return
	}

	recordingShortcut = 0

	if err := registerConfiguredHotkeys(updated); err != nil {
		logger.Printf("shortcut recording failed: %v", err)
		_ = registerConfiguredHotkeys(currentHotkeys)
		setHotkeyLabels(currentHotkeys)
		setStatus("Shortcut update failed: " + err.Error())
		messageBox(mainHwnd, err.Error(), "Shortcut error", MB_OK|MB_ICONERROR)
		return
	}

	currentHotkeys = updated
	_ = saveHotkeyConfig(updated)
	setHotkeyLabels(updated)
	setStatus("Shortcut saved: " + shortcut.Text)
}

func shortcutFromPressedKey(key uint32) (ParsedHotkey, error) {
	modifiers := uint32(0)

	if keyPressed(VK_CONTROL) {
		modifiers |= MOD_CONTROL
	}

	if keyPressed(VK_SHIFT) {
		modifiers |= MOD_SHIFT
	}

	if keyPressed(VK_MENU) {
		modifiers |= MOD_ALT
	}

	if keyPressed(VK_LWIN) || keyPressed(VK_RWIN) {
		modifiers |= MOD_WIN
	}

	if modifiers == 0 {
		return ParsedHotkey{}, errors.New("include Ctrl, Shift, Alt or Win")
	}

	keyText, err := displayShortcutKey(key)
	if err != nil {
		return ParsedHotkey{}, err
	}

	parts := []string{}

	if modifiers&MOD_CONTROL != 0 {
		parts = append(parts, "Ctrl")
	}

	if modifiers&MOD_SHIFT != 0 {
		parts = append(parts, "Shift")
	}

	if modifiers&MOD_ALT != 0 {
		parts = append(parts, "Alt")
	}

	if modifiers&MOD_WIN != 0 {
		parts = append(parts, "Win")
	}

	parts = append(parts, keyText)

	return ParsedHotkey{
		Text:      strings.Join(parts, "+"),
		Modifiers: modifiers,
		Key:       key,
	}, nil
}

func displayShortcutKey(key uint32) (string, error) {
	if (key >= 'A' && key <= 'Z') || (key >= '0' && key <= '9') {
		return string(rune(key)), nil
	}

	if key >= 0x70 && key <= 0x7B {
		return fmt.Sprintf("F%d", key-0x70+1), nil
	}

	return "", fmt.Errorf("unsupported final key; use A-Z, 0-9 or F1-F12")
}

func keyPressed(key uint32) bool {
	return call(procGetAsyncKeyState, uintptr(key))&0x8000 != 0
}

func registerConfiguredHotkeys(config HotkeyConfig) error {
	area, err := parseShortcut(config.Area)
	if err != nil {
		return fmt.Errorf("area OCR shortcut: %w", err)
	}

	imageShortcut, err := parseShortcut(config.Image)
	if err != nil {
		return fmt.Errorf("image OCR shortcut: %w", err)
	}

	colorShortcut, err := parseShortcut(config.Color)
	if err != nil {
		return fmt.Errorf("color picker shortcut: %w", err)
	}

	if shortcutsEqual(area, imageShortcut) ||
		shortcutsEqual(area, colorShortcut) ||
		shortcutsEqual(imageShortcut, colorShortcut) {
		return errors.New("each shortcut must be different")
	}

	unregisterConfiguredHotkeys()

	registered := []int{}

	register := func(id int, shortcut ParsedHotkey, label string) error {
		if call(
			procRegisterHotKey,
			mainHwnd,
			uintptr(id),
			uintptr(shortcut.Modifiers|MOD_NOREPEAT),
			uintptr(shortcut.Key),
		) == 0 {
			return fmt.Errorf("%s shortcut %q is already used by another application", label, shortcut.Text)
		}

		registered = append(registered, id)
		return nil
	}

	if err := register(HOTKEY_AREA, area, "area OCR"); err != nil {
		for _, id := range registered {
			call(procUnregisterHotKey, mainHwnd, uintptr(id))
		}
		return err
	}

	if err := register(HOTKEY_IMAGE, imageShortcut, "image OCR"); err != nil {
		for _, id := range registered {
			call(procUnregisterHotKey, mainHwnd, uintptr(id))
		}
		return err
	}

	if err := register(HOTKEY_COLOR, colorShortcut, "color picker"); err != nil {
		for _, id := range registered {
			call(procUnregisterHotKey, mainHwnd, uintptr(id))
		}
		return err
	}

	currentHotkeys = HotkeyConfig{
		Area:  area.Text,
		Image: imageShortcut.Text,
		Color: colorShortcut.Text,
	}

	setHotkeyLabels(currentHotkeys)
	return nil
}

func unregisterConfiguredHotkeys() {
	call(procUnregisterHotKey, mainHwnd, HOTKEY_AREA)
	call(procUnregisterHotKey, mainHwnd, HOTKEY_IMAGE)
	call(procUnregisterHotKey, mainHwnd, HOTKEY_COLOR)
}

func shortcutsEqual(first, second ParsedHotkey) bool {
	return first.Modifiers == second.Modifiers && first.Key == second.Key
}

func parseShortcut(value string) (ParsedHotkey, error) {
	parts := strings.Split(strings.ReplaceAll(strings.TrimSpace(value), " ", ""), "+")

	if len(parts) < 2 {
		return ParsedHotkey{}, errors.New("include at least one modifier and one key")
	}

	var modifiers uint32
	key := uint32(0)
	keyText := ""

	for _, part := range parts {
		token := strings.ToUpper(strings.TrimSpace(part))

		switch token {
		case "CTRL", "CONTROL":
			modifiers |= MOD_CONTROL

		case "SHIFT":
			modifiers |= MOD_SHIFT

		case "ALT":
			modifiers |= MOD_ALT

		case "WIN", "WINDOWS":
			modifiers |= MOD_WIN

		default:
			if key != 0 {
				return ParsedHotkey{}, errors.New("include only one non-modifier key")
			}

			parsedKey, parsedText, err := parseShortcutKey(token)
			if err != nil {
				return ParsedHotkey{}, err
			}

			key = parsedKey
			keyText = parsedText
		}
	}

	if modifiers == 0 {
		return ParsedHotkey{}, errors.New("include Ctrl, Shift, Alt or Win")
	}

	if key == 0 {
		return ParsedHotkey{}, errors.New("include a final key such as T, I, C or F8")
	}

	displayParts := []string{}

	if modifiers&MOD_CONTROL != 0 {
		displayParts = append(displayParts, "Ctrl")
	}

	if modifiers&MOD_SHIFT != 0 {
		displayParts = append(displayParts, "Shift")
	}

	if modifiers&MOD_ALT != 0 {
		displayParts = append(displayParts, "Alt")
	}

	if modifiers&MOD_WIN != 0 {
		displayParts = append(displayParts, "Win")
	}

	displayParts = append(displayParts, keyText)

	return ParsedHotkey{
		Text:      strings.Join(displayParts, "+"),
		Modifiers: modifiers,
		Key:       key,
	}, nil
}

func parseShortcutKey(token string) (uint32, string, error) {
	if len(token) == 1 {
		character := token[0]

		if (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') {
			return uint32(character), string(character), nil
		}
	}

	if strings.HasPrefix(token, "F") {
		number, err := strconv.Atoi(strings.TrimPrefix(token, "F"))

		if err == nil && number >= 1 && number <= 12 {
			return uint32(0x70 + number - 1), fmt.Sprintf("F%d", number), nil
		}
	}

	return 0, "", fmt.Errorf("unsupported key %q; use A-Z, 0-9 or F1-F12", token)
}

func setWindowText(hwnd uintptr, value string) {
	if hwnd == 0 {
		return
	}

	call(procSetWindowTextW, hwnd, uintptr(unsafe.Pointer(p(value))))
}

func getWindowText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}

	length := int(call(procGetWindowTextLengthW, hwnd))
	if length == 0 {
		return ""
	}

	buffer := make([]uint16, length+1)
	call(procGetWindowTextW, hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))

	return syscall.UTF16ToString(buffer)
}

func windowsFilter(parts ...string) []uint16 {
	filter := make([]uint16, 0, 512)

	for _, part := range parts {
		encoded, err := syscall.UTF16FromString(part)
		if err != nil {
			logger.Printf("Windows filter segment encoding failed for %q: %v", part, err)
			continue
		}

		filter = append(filter, encoded...)
	}

	// Windows OPENFILENAME filters require an extra trailing NUL.
	filter = append(filter, 0)

	return filter
}

func chooseImage() (string, bool) {
	buf := make([]uint16, 32768)

	filter := windowsFilter(
		"Supported images (*.png;*.jpg;*.jpeg;*.bmp;*.gif;*.tif;*.tiff)",
		"*.png;*.jpg;*.jpeg;*.bmp;*.gif;*.tif;*.tiff",
		"PNG images (*.png)",
		"*.png",
		"JPEG images (*.jpg;*.jpeg)",
		"*.jpg;*.jpeg",
		"Bitmap images (*.bmp)",
		"*.bmp",
		"GIF images (*.gif)",
		"*.gif",
		"TIFF images (*.tif;*.tiff)",
		"*.tif;*.tiff",
		"All files (*.*)",
		"*.*",
	)

	if len(filter) == 0 {
		setStatus("Image chooser could not initialize its Windows filter.")
		logger.Printf("image chooser filter unexpectedly empty")
		return "", false
	}

	ofn := OPENFILENAME{
		LStructSize: uint32(unsafe.Sizeof(OPENFILENAME{})),
		HwndOwner:   dialogOwner(),
		LpstrFilter: &filter[0],
		LpstrFile:   &buf[0],
		NMaxFile:    uint32(len(buf)),
		LpstrTitle:  p("Choose an image for OCR"),
		Flags:       OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_EXPLORER | OFN_NOCHANGEDIR,
	}

	if call(procGetOpenFileNameW, uintptr(unsafe.Pointer(&ofn))) == 0 {
		if code := call(procCommDlgExtendedErr); code != 0 {
			logger.Printf("GetOpenFileNameW failed with code: 0x%X", code)
			setStatus(fmt.Sprintf("Image chooser failed with Windows dialog code 0x%X.", code))
		}

		return "", false
	}

	path := strings.TrimSpace(syscall.UTF16ToString(buf))
	if path == "" {
		setStatus("No image file was selected.")
		return "", false
	}

	return path, true
}

func setClipboard(s string) error {
	u, err := syscall.UTF16FromString(s)
	if err != nil || len(u) == 0 {
		return errors.New("clipboard text could not be encoded")
	}

	size := uintptr(len(u) * 2)
	if call(procOpenClipboard, 0) == 0 {
		return errors.New("OpenClipboard failed")
	}
	defer call(procCloseClipboard)
	call(procEmptyClipboard)
	h := call(procGlobalAlloc, GMEM_MOVEABLE, size)
	if h == 0 {
		return errors.New("GlobalAlloc failed")
	}
	ptr := call(procGlobalLock, h)
	if ptr == 0 {
		call(procGlobalFree, h)
		return errors.New("GlobalLock failed")
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size)
	var bb bytes.Buffer
	_ = binary.Write(&bb, binary.LittleEndian, u)
	copy(dst, bb.Bytes())
	call(procGlobalUnlock, h)
	if call(procSetClipboardData, CF_UNICODETEXT, h) == 0 {
		call(procGlobalFree, h)
		return errors.New("SetClipboardData failed")
	}
	return nil
}
func messageBox(hwnd uintptr, text, title string, flags uintptr) {
	call(procMessageBoxW, hwnd, uintptr(unsafe.Pointer(p(text))), uintptr(unsafe.Pointer(p(title))), flags)
}

func init() { image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig) }
