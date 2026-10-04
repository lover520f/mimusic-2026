package jsplugin

import "syscall"

func setUDPReuseAddress(c syscall.RawConn) error {
	var optionErr error
	if err := c.Control(func(fd uintptr) {
		optionErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	}); err != nil {
		return err
	}
	return optionErr
}
