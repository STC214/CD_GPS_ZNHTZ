//go:build windows

package runtime

import "syscall"

const utf8CodePage = 65001

var (
	consoleKernel32      = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleCP     = consoleKernel32.NewProc("SetConsoleCP")
	procSetConsoleOutput = consoleKernel32.NewProc("SetConsoleOutputCP")
)

func init() {
	_, _, _ = procSetConsoleCP.Call(utf8CodePage)
	_, _, _ = procSetConsoleOutput.Call(utf8CodePage)
}
