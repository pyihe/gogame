package login

import (
	"sync/atomic"

	"github.com/pyihe/gogame"
	"github.com/pyihe/gogame/chanrpc"
	"github.com/pyihe/gogame/pkg"
)

var (
	module *Server
)

func NewModule() gogame.Module {
	module = new(Server)
	module.skeleton = gogame.NewSkeleton()
	module.chanRPC = module.skeleton.ChanRPCServer()
	module.status = pkg.StatusInitial
	return module
}

func ChanRPC() *chanrpc.Server {
	return module.chanRPC
}

type Server struct {
	skeleton *gogame.Skeleton
	chanRPC  *chanrpc.Server

	status int32
}

func (m *Server) Init() {
	// 注册需要被其他模块直接调用的RPC Function

	// 注册需要本模块处理的消息
}

func (m *Server) Run() {
	if !atomic.CompareAndSwapInt32(&m.status, pkg.StatusInitial, pkg.StatusRunning) {
		return
	}
	m.skeleton.Run()
}

func (m *Server) Running() bool {
	return atomic.LoadInt32(&m.status) == pkg.StatusRunning
}

func (m *Server) Destroy() {
	if !atomic.CompareAndSwapInt32(&m.status, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}
	m.skeleton.Close()
}
