//go:build windows

package clock

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	qpc      = kernel32.NewProc("QueryPerformanceCounter")

	// QueryPerformanceFrequency cannot fail and never returns 0 since
	// Windows XP (Microsoft's documentation), so it is read once, unchecked.
	frequency = func() int64 {
		var f int64
		_, _, _ = kernel32.NewProc("QueryPerformanceFrequency").Call(uintptr(unsafe.Pointer(&f)))
		return f
	}()
)

func ticks() int64 {
	var t int64
	_, _, _ = qpc.Call(uintptr(unsafe.Pointer(&t)))
	return t
}
