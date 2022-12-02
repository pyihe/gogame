package gogame

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/pyihe/gogame/chanrpc"
	"github.com/pyihe/gogame/internal/gopool"
	g "github.com/pyihe/gogame/internal/goroutine"
	"github.com/pyihe/gogame/internal/gotimer"
	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
	"github.com/pyihe/timer"
)

// Skeleton 模块骨架，用于每个模块消息、任务、模块间的调度
// 每个模块需要包含骨架
type Skeleton struct {
	cancelFunc context.CancelFunc
	g          *g.Go
	dispatcher *gotimer.Dispatcher
	client     *chanrpc.Client
	server     *chanrpc.Server
	status     int32
}

func NewSkeleton() *Skeleton {
	const defaultChanSize = 10000

	s := &Skeleton{
		g:          g.New(defaultChanSize),
		dispatcher: gotimer.NewDispatcher(defaultChanSize),
		client:     chanrpc.NewClient(defaultChanSize),
		server:     chanrpc.NewServer(defaultChanSize),
		status:     pkg.StatusInitial,
	}

	//s.Run()

	return s
}

func (s *Skeleton) isRunning() bool {
	return atomic.LoadInt32(&s.status) == pkg.StatusRunning
}

func (s *Skeleton) isClosed() bool {
	return atomic.LoadInt32(&s.status) == pkg.StatusClosed
}

func (s *Skeleton) ChanRPCServer() *chanrpc.Server {
	return s.server
}

func (s *Skeleton) Close() {
	if !atomic.CompareAndSwapInt32(&s.status, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}
	s.cancelFunc()
}

func (s *Skeleton) Run() {
	if !atomic.CompareAndSwapInt32(&s.status, pkg.StatusInitial, pkg.StatusRunning) {
		return
	}
	var ctx context.Context
	ctx, s.cancelFunc = context.WithCancel(context.Background())

	gopool.AddTask(func() {
		for {
			select {
			case <-ctx.Done():
				s.dispatcher.Close()
				s.server.Close()
				for !s.g.Idle() || !s.client.Idle() {
					s.g.Close()
					s.client.Close()
				}
				return
			case ri := <-s.client.ChanAsynRet:
				s.client.Cb(ri)
			case ci := <-s.server.Chan():
				s.server.Exec(ci)
			case cb := <-s.g.ChanCb:
				s.g.Cb(cb)
			case t := <-s.dispatcher.ChanJob:
				t.Run()
			}
		}
	})
}

func (s *Skeleton) AfterFunc(d time.Duration, cb func()) (timer.TaskID, error) {
	if s.isRunning() {
		return s.dispatcher.AfterFunc(d, cb)
	}
	return timer.EmptyTaskID, nil
}

func (s *Skeleton) CronFunc(desc string, cb func()) (timer.TaskID, error) {
	if s.isRunning() {
		return s.dispatcher.CronFunc(desc, cb)
	}
	return timer.EmptyTaskID, nil
}

func (s *Skeleton) Go(f func(), cb func()) {
	if s.isRunning() {
		s.g.Go(f, cb)
	}
}

func (s *Skeleton) AsynCall(server *chanrpc.Server, id interface{}, args ...interface{}) {
	if s.isRunning() {
		s.client.AttachSever(server)
		s.client.AsynCall(id, args...)
	}
}

func (s *Skeleton) RegisterChanRPC(id interface{}, f interface{}) {
	err := s.server.Register(id, f)
	if err != nil {
		log.Printf("register chan rpc err: %v", err)
	}
}
