package main

import (
	"os"
	"syscall"
	"unsafe"
)

// isTerminal asks the kernel for terminal attributes, which only a real
// terminal has. A character-device check is not enough: /dev/null, which agent
// harnesses hand mur as stdin, is a character device too.
func isTerminal(f *os.File) bool {
	var t syscall.Termios

	//nolint:gosec // the ioctl needs the termios address
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(getTermios), uintptr(unsafe.Pointer(&t)))

	return errno == 0
}
