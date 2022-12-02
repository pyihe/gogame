package gopool

import (
	"math"

	"github.com/panjf2000/ants/v2"
	"github.com/pyihe/gogame/pkg/log"
)

// goroutine pool from github.com/panjf2000/ants/v2
var pool *ants.Pool

func init() {
	pool, _ = ants.NewPool(math.MaxInt, ants.WithNonblocking(true))
}

func AddTask(fn func()) {
	err := pool.Submit(fn)
	if err != nil {
		log.Printf("gopool add task err: %v", err)
	}
}
