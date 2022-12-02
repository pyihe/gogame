package gogame

import (
	"runtime"
	"sync"
	"time"

	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
)

type Module interface {
	// Init 初始化工作
	// 每个模块初始化工作需要做的内容至少包括：
	// 1. 注册自己需要被其他模块直接调用的RPC函数
	// 2. 给自己模块的消息注册路由
	Init()

	// Run 运行模块
	// 每个模块必须自己Run Skeleton
	Run()

	// Running 模块是否已经运行起来了
	Running() bool

	// Destroy 处理模块销毁工作
	// 每个模块必须自己Destroy Skeleton
	Destroy()
}

type module struct {
	mi Module
	wg sync.WaitGroup
}

func newModule(m Module) *module {
	mod := &module{}
	mod.mi = m
	return mod
}

func (m *module) run() {
	m.wg.Add(1)

	gopool.AddTask(func() {
		m.mi.Run()
		m.wg.Done()
	})

	retry := 0
	for {
		if m.mi.Running() || retry >= 10 {
			break
		}
		retry += 1
		time.Sleep(200 * time.Microsecond)
	}
}

func (m *module) destroy() {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, pkg.StackSize)
			n := runtime.Stack(buf, false)
			log.Printf("%v: %s", r, buf[:n])
		}
	}()

	m.mi.Destroy()
	m.wg.Wait()
}
