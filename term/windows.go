// +build windows

package term

const flag_ENABLE_ECHO_INPUT = 0x0004

func IsTerminal(console syscall.Handle) bool {
	return true
}

var (
	procSetConsoleMode = modkernel32.NewProc("SetConsoleMode")
)

func MakeRaw(console syscall.Handle) (oldState uint32, err error) {
	err = syscall.GetConsoleMode(console, &oldState)
	if err != nil {
		return
	}
	return setConsoleMode(console, oldState & ~flag_ENABLE_ECHO_INPUT)
}

func Restore(console syscall.Handle, mode uint32) error {
	return setConsoleMode(console, mode)
}

func Cols() int {
	return 100
}

func Lines() int {
	return 100
}

func setConsoleMode(console syscall.Handle, mode unit32) (err error) {
	r1, _, e1 := Syscall(procSetConsoleMode.Addr(), 2, uintptr(console), mode, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = EINVAL
		}
	}
	return
}
