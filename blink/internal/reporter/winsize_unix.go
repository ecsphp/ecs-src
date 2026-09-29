//go:build !windows

package reporter

import (
	"os"
	"syscall"
	"unsafe"
)

const tiocgwinsz = 0x5413 // linux TIOCGWINSZ

func winsize(file *os.File) (int, bool) {
	ws := struct{ Row, Col, X, Y uint16 }{}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(tiocgwinsz),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0, false
	}
	return int(ws.Col), true
}
