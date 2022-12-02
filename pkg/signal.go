package pkg

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pyihe/gogame/pkg/log"
)

func Wait(callbacks ...func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-ch
		log.Printf("server closing down: (signal: %v)", s)
		switch s {
		case os.Kill, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP:
			for _, fn := range callbacks {
				if fn != nil {
					fn()
				}
			}
			return
		}
	}
}
