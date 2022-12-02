package goroutine

import (
	"runtime"

	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
)

//

type Go struct {
	ChanCb    chan func()
	pendingGo *pkg.AtomicInt32
}

type LinearGo struct {
	f  func()
	cb func()
}

func New(size int) *Go {
	return &Go{
		ChanCb: make(chan func(), size),
	}
}

func (g *Go) Go(f func(), cb func()) {
	g.pendingGo.Incr(1)

	gopool.AddTask(func() {
		if f != nil {
			f()
		}
		g.ChanCb <- cb
	})
}

func (g *Go) Cb(cb func()) {
	if cb == nil {
		g.pendingGo.Incr(-1)
		return
	}
	defer func() {
		g.pendingGo.Incr(-1)
		if r := recover(); r != nil {
			buf := make([]byte, pkg.StackSize)
			n := runtime.Stack(buf, false)
			log.Printf("%v: %s", r, buf[:n])
		}
	}()

	cb()
}

func (g *Go) Close() {
	for g.pendingGo.Value() > 0 {
		g.Cb(<-g.ChanCb)
	}
}

func (g *Go) Idle() bool {
	return g.pendingGo.Value() == 0
}
