package gate

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/pyihe/gogame"
	"github.com/pyihe/gogame/cmd/game"
	"github.com/pyihe/gogame/cmd/protocol"
	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/route"
	jsonc "github.com/pyihe/gogame/route/json"
)

var (
	module = new(Server)
)

func NewModule() gogame.Module {
	processor := route.NewProcessor(true, route.GetCodec(jsonc.Name))

	module = &Server{
		Gate: &gogame.Gate{
			MsgMaxLen:    4096,
			MsgMinLen:    1,
			MaxConnNum:   math.MaxInt,
			WriteBuffer:  10000,
			Processor:    processor,
			AgentHandler: module,
			WSAddr:       "192.168.1.192:6666",
			HTTPTimeout:  3 * time.Second,
		},
		status: pkg.StatusInitial,
	}
	return module
}

type Server struct {
	*gogame.Gate

	status int32
}

func (m *Server) Init() {
	// 注册消息的router
	m.Processor.Register(route.NewMessage(1, &protocol.Hello{}).SetRouter(game.ChanRPC()))
}

func (m *Server) Run() {
	if !atomic.CompareAndSwapInt32(&m.status, pkg.StatusInitial, pkg.StatusRunning) {
		return
	}
	m.Gate.Start()
}

func (m *Server) Running() bool {
	return atomic.LoadInt32(&m.status) == pkg.StatusRunning
}

func (m *Server) Destroy() {
	if !atomic.CompareAndSwapInt32(&m.status, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}
	m.Gate.Close()
}

func (m *Server) OnConnect(agent gogame.Agent) {
	game.ChanRPC().Go("NewAgent", agent)
}

func (m *Server) OnClose(agent gogame.Agent) {
	game.ChanRPC().Go("CloseAgent", agent)
}
