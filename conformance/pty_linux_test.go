package conformance

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func openTerminal() (ptmx, pts *os.File, err error) {
	ptmx, err = os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	var unlock, n int32
	if err := ioctl(ptmx, syscall.TIOCSPTLCK, unsafe.Pointer(&unlock)); err != nil {
		_ = ptmx.Close()
		return nil, nil, err
	}
	if err := ioctl(ptmx, syscall.TIOCGPTN, unsafe.Pointer(&n)); err != nil {
		_ = ptmx.Close()
		return nil, nil, err
	}
	pts, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY|syscall.O_CLOEXEC, 0)
	if err != nil {
		_ = ptmx.Close()
		return nil, nil, err
	}
	return ptmx, pts, nil
}

func ioctl(f *os.File, req uintptr, arg unsafe.Pointer) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), req, uintptr(arg)); errno != 0 {
		return errno
	}
	return nil
}
