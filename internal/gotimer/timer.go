package gotimer

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
	"github.com/pyihe/timer"
	"github.com/pyihe/timer/timewheel"
)

const (
	timeWheelSlots = 300
)

type Task func()

func (task Task) Run() {
	if err := recover(); err != nil {
		buf := make([]byte, pkg.StackSize)
		n := runtime.Stack(buf, false)
		log.Printf("%s", buf[:n])
	}
	task()
}

// Dispatcher 定时任务调度器
type Dispatcher struct {
	timer   timer.Timer // 基于时间轮的内存定时器
	status  int32       // 状态
	ChanJob chan Task   // 调度定时任务
}

func NewDispatcher(jobCap int) *Dispatcher {
	dispatcher := new(Dispatcher)
	dispatcher.ChanJob = make(chan Task, jobCap)
	dispatcher.timer = timewheel.New(500*time.Millisecond, timeWheelSlots, 1000)
	dispatcher.status = pkg.StatusRunning
	return dispatcher
}

func (dis *Dispatcher) isClosed() bool {
	return atomic.LoadInt32(&dis.status) == pkg.StatusClosed
}

func (dis *Dispatcher) Close() {
	if !atomic.CompareAndSwapInt32(&dis.status, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}

	for i := 0; i < len(dis.ChanJob); i++ {
		job := <-dis.ChanJob
		if job != nil {
			job.Run()
		}
	}

	dis.timer.Stop()
}

func (dis *Dispatcher) DeleteFunc(jobId timer.TaskID) error {
	if dis.isClosed() {
		return pkg.ErrTimerClosed
	}
	return dis.timer.Delete(jobId)
}

func (dis *Dispatcher) AfterFunc(d time.Duration, cb func()) (jobID timer.TaskID, err error) {
	if dis.isClosed() {
		return timer.EmptyTaskID, pkg.ErrTimerClosed
	}
	jobID, err = dis.timer.After(d, func() {
		dis.ChanJob <- cb
	})
	return
}

func (dis *Dispatcher) EveryFunc(interval time.Duration, cb func()) (jobId timer.TaskID, err error) {
	if dis.isClosed() {
		err = pkg.ErrTimerClosed
		return
	}
	jobId, err = dis.timer.Every(interval, func() {
		dis.ChanJob <- cb
	})
	return
}

func (dis *Dispatcher) CronFunc(cronDesc string, cb func()) (jobId timer.TaskID, err error) {
	if dis.isClosed() {
		err = pkg.ErrTimerClosed
		return
	}
	return dis.timer.Cron(cronDesc, cb)
}
