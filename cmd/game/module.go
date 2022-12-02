package game

import (
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/pyihe/gogame"
	"github.com/pyihe/gogame/chanrpc"
	"github.com/pyihe/gogame/cmd/protocol"
	"github.com/pyihe/gogame/pkg"
)

var (
	module *Server
)

func NewModule() gogame.Module {
	module = &Server{}
	module.status = pkg.StatusInitial
	module.skeleton = gogame.NewSkeleton()
	module.chanRPC = module.skeleton.ChanRPCServer()
	module.agentSet = make(map[uint64]*player)

	return module
}

// ChanRPC 导出Game模块的ChanRPC
func ChanRPC() *chanrpc.Server {
	return module.chanRPC
}

type Server struct {
	chanRPC *chanrpc.Server

	skeleton *gogame.Skeleton
	status   int32

	// 管理玩家信息
	mu       sync.RWMutex
	agentSet map[uint64]*player
}

func (m *Server) Init() {
	// 注册需要其他模块直接调用的RPC Function
	m.skeleton.RegisterChanRPC("NewAgent", newPlayer)
	m.skeleton.RegisterChanRPC("CloseAgent", closePlayer)

	// 注册需要本模块处理的消息
	m.skeleton.RegisterChanRPC(reflect.TypeOf(&protocol.Hello{}), handleHello)

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
