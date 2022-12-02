package pkg

import (
	"math"
	"time"
)

const (
	// StackSize 用于获取堆栈信息
	StackSize = 4096

	// StatusRunning 状态标志： 开（运行）
	StatusRunning = 1

	// StatusInitial 初始运行状态
	StatusInitial = 0

	// StatusClosed 状态标志： 关
	StatusClosed = -1

	// DefaultLockTimeout 分布式锁默认获取锁的超时时间，此期间仍未获取到锁，则返回超时
	DefaultLockTimeout = math.MaxInt32 * time.Second
)
