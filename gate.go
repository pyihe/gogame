package gogame

import (
	"time"

	"github.com/pyihe/gogame/network"
	"github.com/pyihe/gogame/pkg/log"
	"github.com/pyihe/gogame/route"
)

// Gate 基础模块：网关，用于端口监听以及消息转发，一般需要自己实现一个包含本Gate的组合Gate或者直接套用
type Gate struct {
	MsgMaxLen    uint32          // 最大消息长度
	MsgMinLen    uint32          // 最小消息长度
	MaxConnNum   int             // 最大连接数
	WriteBuffer  int             // 发送消息时的写缓冲区大小
	Processor    route.Processor // 消息处理
	AgentHandler AgentHook       // agent handler

	// websocket
	WSAddr      string
	CertFile    string
	KeyFile     string
	RootCAFile  string
	HTTPTimeout time.Duration

	// tcp
	TCPAddr      string
	MsgHeaderLen int
	LittleEndian bool

	wsServer  *network.WSServer
	tcpServer *network.TCPServer
}

func (gate *Gate) Start() {
	if gate.WSAddr == "" && gate.TCPAddr == "" {
		log.Fatalf("no addr to listen")
	}
	if gate.Processor == nil {
		log.Fatalf("no route")
	}

	err := gate.newWSServer()
	if err != nil {
		log.Fatalf("new ws server err: %v", err)
	}
	err = gate.newTCPServer()
	if err != nil {
		log.Fatalf("new tcp server err: %v", err)
	}
}

func (gate *Gate) Close() {
	if gate.wsServer != nil {
		gate.wsServer.Close()
	}
	if gate.tcpServer != nil {
		gate.tcpServer.Close()
	}
}

func (gate *Gate) newWSServer() (err error) {
	if gate.WSAddr == "" {
		return
	}
	newAgentFunc := func(conn *network.WSConn) network.Agent {
		agt := &gateAgent{
			conn: conn,
			gate: gate,
		}
		return agt
	}

	opts := network.WSServerOption{
		Addr:        gate.WSAddr,
		MaxConnNum:  gate.MaxConnNum,
		WriteBuff:   gate.WriteBuffer,
		MsgMaxLen:   gate.MsgMaxLen,
		HTTPTimeout: gate.HTTPTimeout,
		TLSOption: &network.TLSOption{
			TLSCert:       gate.CertFile,
			TLSKey:        gate.KeyFile,
			TLSRootCAFile: gate.RootCAFile,
		},
	}
	gate.wsServer, err = network.NewWSServer(opts, newAgentFunc)
	if err != nil {
		return
	}
	// 启动服务
	gate.wsServer.Start()
	return
}

func (gate *Gate) newTCPServer() (err error) {
	if gate.TCPAddr == "" {
		return
	}
	newAgentFunc := func(conn *network.TCPConn) network.Agent {
		agt := &gateAgent{
			conn: conn,
			gate: gate,
		}
		return agt
	}
	opts := network.TCPServerOptions{
		Addr:        gate.TCPAddr,
		MaxConnNum:  gate.MaxConnNum,
		WriteBuffer: gate.WriteBuffer,
		MsgOption: &network.TCPMsgOption{
			MsgHeaderLen: gate.MsgHeaderLen,
			MsgMinLen:    gate.MsgMinLen,
			MsgMaxLen:    gate.MsgMaxLen,
			LittleEndian: gate.LittleEndian,
		},
	}
	gate.tcpServer, err = network.NewTCPServer(opts, newAgentFunc)
	if err != nil {
		return
	}
	// 启动服务
	gate.tcpServer.Start()
	return
}
