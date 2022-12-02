package gogame

import (
	"io"
	"net"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/pyihe/gogame/network"
	"github.com/pyihe/gogame/pkg/log"
)

type AgentHook interface {
	OnConnect(Agent)
	OnClose(Agent)
}

type Agent interface {
	WriteMsg(msg interface{})
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	UserData() interface{}
	SetUserData(data interface{})
}

type gateAgent struct {
	conn     network.Conn // 底层连接
	gate     *Gate        // 所属gate
	userData atomic.Value // 附加数据
}

func (a *gateAgent) Run() {
	for {
		data, err := a.conn.ReadMsg()
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				log.Printf("read message: %v", err)
			}
			break
		}
		if a.gate.Processor != nil {
			msg, err := a.gate.Processor.Unmarshal(data)
			if err != nil {
				log.Printf("unmarshal message error: %v", err)
				break
			}
			err = a.gate.Processor.Route(msg, a)
			if err != nil {
				log.Printf("route message error: %v", err)
				break
			}
		}
	}
}

func (a *gateAgent) OnClose() {
	if handler := a.gate.AgentHandler; handler != nil {
		handler.OnClose(a)
	}
}

func (a *gateAgent) OnConnect() {
	if handler := a.gate.AgentHandler; handler != nil {
		handler.OnConnect(a)
	}
}

func (a *gateAgent) WriteMsg(msg interface{}) {
	if a.gate.Processor != nil {
		data, err := a.gate.Processor.Marshal(msg)
		if err != nil {
			log.Printf("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = a.conn.WriteMsg(data)
		if err != nil {
			log.Printf("write message %v error: %v", reflect.TypeOf(msg), err)
		}
	}
}

func (a *gateAgent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *gateAgent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *gateAgent) Close() {
	a.conn.Close()
}

func (a *gateAgent) UserData() interface{} {
	return a.userData.Load()
}

func (a *gateAgent) SetUserData(data interface{}) {
	a.userData.Store(data)
}
