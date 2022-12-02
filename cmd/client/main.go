package main

import (
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/pyihe/gogame/cmd/protocol"
	"github.com/pyihe/gogame/network"
	"github.com/pyihe/gogame/pkg/log"
	"github.com/pyihe/gogame/route"
	jsonc "github.com/pyihe/gogame/route/json"
)

var (
	mu        sync.RWMutex
	agentMap  = make(map[*client]struct{})
	processor = route.NewProcessor(true, route.GetCodec(jsonc.Name))
)

func init() {
	processor.Register(route.NewMessage(1, &protocol.Hello{}).SetHandler(handler))
}

func handler(args ...interface{}) {
	m := args[0].(*protocol.Hello)
	agent := args[1].(*client)
	agent.handleHello(m)
}

type client struct {
	cType     string
	processor route.Processor
	conn      network.Conn
}

func newTCPAgent(conn *network.TCPConn) network.Agent {
	a := &client{}
	a.cType = "tcp"
	a.conn = conn
	a.processor = processor

	mu.Lock()
	agentMap[a] = struct{}{}
	mu.Unlock()

	return a
}

func newWSAgent(conn *network.WSConn) network.Agent {
	a := &client{}
	a.cType = "ws"
	a.conn = conn
	a.processor = processor

	mu.Lock()
	agentMap[a] = struct{}{}
	mu.Unlock()

	return a
}

func (a *client) handleHello(ms ...interface{}) {
	time.Sleep(1 * time.Second)
	m := ms[0].(*protocol.Hello)
	m.Name = "server"
	a.writeMsg(m)
}

func (a *client) writeMsg(msg interface{}) {
	var (
		data []byte
		err  error
	)

	switch a.cType {
	case "tcp", "ws":
		data, err = a.processor.Marshal(msg)
		if err != nil {
			log.Printf("marshal message [%v] error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = a.conn.WriteMsg(data)
		if err != nil {
			log.Printf("write message [%v] error: %v", reflect.TypeOf(msg), err)
		}
	}
}

func (a *client) Run() {
	switch a.cType {
	case "tcp", "ws":
		for {
			data, err := a.conn.ReadMsg()
			if err != nil {
				if err != io.EOF {
					log.Printf("read message: %v", err)
				}
				break
			}
			log.Printf("client recv: %v", string(data))
			if a.processor != nil {
				msg, err := a.processor.Unmarshal(data)
				if err != nil {
					log.Printf("unmarshal message error: %v", err)
					break
				}
				err = a.processor.Route(msg, a)
				if err != nil {
					log.Printf("route err: %v", err)
					break
				}
			}
		}
	}
}

func (a *client) Close() {
	a.conn.Close()
}

func (a *client) OnClose() {}

func (a *client) OnConnect() {
	switch a.cType {
	case "tcp", "ws":
		a.writeMsg(&protocol.Hello{Name: "server"})
	}
}

func main() {
	// TCP客户端
	//tcpClient := initTCP()
	//defer tcpClient.Close()

	// websocket 客户端
	wsClient := initWS()
	defer wsClient.Close()

	ticker := time.NewTimer(3600 * time.Second)
	select {
	case <-ticker.C:
		ticker.Stop()
		mu.RLock()
		for a := range agentMap {
			a.Close()
		}
		mu.RUnlock()
	}
}

func initTCP() *network.TCPClient {
	opts := network.TCPClientOption{
		Addr:            ":5555",
		ConnNum:         1,
		AutoReconnect:   true,
		WriteBuffer:     100,
		ConnectInterval: 3 * time.Second,
		TLSOption:       nil,
		MsgOption: &network.TCPMsgOption{
			MsgHeaderLen: 2,
			MsgMinLen:    1,
			MsgMaxLen:    4096,
			LittleEndian: true,
		},
	}
	c, _ := network.NewTCPClient(opts, newTCPAgent)
	c.Start()
	return c
}

func initWS() *network.WSClient {
	opts := network.WSClientOption{
		Addr:             "ws://192.168.1.192:6666",
		ConnNum:          1,
		AutoReconnect:    true,
		ConnectInterval:  3 * time.Second,
		MsgMaxLen:        4096,
		WriteBuffer:      10000,
		HandshakeTimeout: 10 * time.Second,
	}
	c, _ := network.NewWSClient(opts, newWSAgent)
	c.Start()
	return c
}
