package log

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	once          sync.Once
	defaultLogger Logger = log.New(os.Stderr, "[gogame] ", log.LstdFlags|log.Lmicroseconds|log.Llongfile)
)

type Logger interface {
	Output(calldepth int, s string) error
}

// SetLogger 设置日志记录器
func SetLogger(logger Logger) {
	if logger == nil {
		return
	}
	once.Do(func() {
		defaultLogger = logger
	})
}

func Printf(format string, args ...interface{}) {
	defaultLogger.Output(2, fmt.Sprintf(format, args...))
}

func Fatalf(format string, args ...interface{}) {
	Printf(format, args...)
	os.Exit(1)
}
