package pkg

import (
	"sync"
)

type Semaphore struct {
	queue  chan struct{}
	waiter *sync.WaitGroup
}

func NewLimiter(size int) *Semaphore {
	if size < 0 {
		size = 0
	}
	return &Semaphore{
		queue:  make(chan struct{}, size),
		waiter: &sync.WaitGroup{},
	}
}

func (lim *Semaphore) Add(delta int) {
	switch {
	case delta > 0:
		for i := 0; i < delta; i++ {
			lim.queue <- struct{}{}
		}
	default:
		for i := 0; i > delta; i-- {
			<-lim.queue
		}
	}

	lim.waiter.Add(delta)
}

func (lim *Semaphore) Done() {
	<-lim.queue
	lim.waiter.Done()
}

func (lim *Semaphore) Wait() {
	lim.waiter.Wait()
}
