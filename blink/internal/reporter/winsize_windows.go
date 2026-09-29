//go:build windows

package reporter

import (
	"os"
	"syscall"
	"unsafe"
)

var procGetConsoleScreenBufferInfo = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleScreenBufferInfo")

// consoleScreenBufferInfo mirrors the Win32 CONSOLE_SCREEN_BUFFER_INFO struct.
type consoleScreenBufferInfo struct {
	SizeX, SizeY                                     int16
	CursorX, CursorY                                 int16
	Attributes                                       uint16
	WindowLeft, WindowTop, WindowRight, WindowBottom int16
	MaxWindowSizeX, MaxWindowSizeY                   int16
}

func winsize(file *os.File) (int, bool) {
	var info consoleScreenBufferInfo
	ok, _, _ := procGetConsoleScreenBufferInfo.Call(file.Fd(), uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return 0, false
	}
	return int(info.WindowRight-info.WindowLeft) + 1, true
}
