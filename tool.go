package gospider

import (
	slog "log"
	"os"
	"runtime"
)

var log *slog.Logger

func init() {
	log = slog.New(os.Stdout, "GoSpider|", slog.Lmicroseconds|slog.Lshortfile)
}

func SprintStack() string {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	return string(buf[:n])
}
