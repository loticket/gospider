package gospider

import (
	"runtime"
)

func SprintStack() string {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	return string(buf[:n])
}
