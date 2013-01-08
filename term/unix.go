// +build darwin freebsd linux netbsd openbsd

package term

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// IsTerminal returns true if f is a terminal.
func IsTerminal(fd int) bool {
	cmd := exec.Command("test", "-t", "0")
	cmd.Stdin = os.NewFile(uintptr(fd), "")
	return cmd.Run() == nil
}

func MakeRaw(fd int) (uint32, error) {
	return 0, stty(fd, "-icanon", "-echo").Run()
}

func Restore(fd int, mode uint32) error {
	return stty(fd, "icanon", "echo").Run()
}

func Cols(fd int) (int, error) {
	cols, err := tput("cols")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(cols)
}

func Lines(fd int) (int, error) {
	cols, err := tput("lines")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(cols)
}

// helpers

func stty(fd int, args ...string) *exec.Cmd {
	c := exec.Command("stty", args...)
	c.Stdin = os.NewFile(uintptr(fd), "")
	return c
}

func tput(what string) (string, error) {
	c := exec.Command("tput", what)
	c.Stderr = os.Stderr
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
