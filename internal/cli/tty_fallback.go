//go:build !linux && !darwin

package cli

import "os"

func isTTY(fd uintptr) bool {
	var file *os.File
	switch fd {
	case os.Stdin.Fd():
		file = os.Stdin
	case os.Stdout.Fd():
		file = os.Stdout
	case os.Stderr.Fd():
		file = os.Stderr
	default:
		return false
	}
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
