package network

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/network/packet"
	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
)

type TCPClient struct {
	opts      atomic.Value
	tlsConfig *tls.Config
	newAgent  func(*TCPConn) Agent
	msgParser packet.Parser

	mu    sync.RWMutex
	conns tcpConnSet

	wg     sync.WaitGroup
	closed int32
}

func NewTCPClient(opts TCPClientOption, newAgent func(*TCPConn) Agent) (*TCPClient, error) {
	if newAgent == nil {
		return nil, pkg.ErrNilNewAgent
	}
	opts.setDefault()

	var err error
	var msgOptions = []packet.Option{
		packet.WithHeader(opts.MsgOption.MsgHeaderLen),
		packet.WithMaxLen(opts.MsgOption.MsgMaxLen),
		packet.WithMinLen(opts.MsgOption.MsgMinLen),
		packet.WithLittleEndian(opts.MsgOption.LittleEndian),
	}

	c := &TCPClient{
		newAgent:  newAgent,
		conns:     make(tcpConnSet),
		msgParser: packet.NewParser(msgOptions...),
	}

	c.swapOpts(&opts)

	c.tlsConfig, err = buildTLSConfig(opts.TLSOption)
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %s", err)
	}

	return c, nil
}

func (client *TCPClient) getOpts() *TCPClientOption {
	return client.opts.Load().(*TCPClientOption)
}

func (client *TCPClient) swapOpts(opts *TCPClientOption) {
	client.opts.Store(opts)
}

func (client *TCPClient) isClosed() bool {
	return atomic.LoadInt32(&client.closed) == pkg.StatusClosed
}

func (client *TCPClient) Start() {
	if !atomic.CompareAndSwapInt32(&client.closed, pkg.StatusInitial, pkg.StatusRunning) {
		return
	}

	opts := client.getOpts()

	for i := 0; i < opts.ConnNum; i++ {
		client.wg.Add(1)
		gopool.AddTask(client.connect)
	}
}

func (client *TCPClient) dial() net.Conn {
	const maxRetry = 3
	var err error
	var conn net.Conn
	var retry int

	for {
		if client.tlsConfig != nil {
			conn, err = tls.Dial("tcp", client.getOpts().Addr, client.tlsConfig)
		} else {
			conn, err = net.Dial("tcp", client.getOpts().Addr)
		}
		if err == nil || client.isClosed() {
			return conn
		}
		if retry >= maxRetry {
			log.Printf("tcp connect(%v): retry timeout", client.getOpts().Addr)
			return conn
		}
		retry += 1
		log.Printf("failed to connect TCP(%s) error: %v, retry: %d, maxRetry: %d", client.getOpts().Addr, err, retry, maxRetry)
		time.Sleep(pkg.Get(nil, retry))
		continue
	}
}

func (client *TCPClient) connect() {
	defer client.wg.Done()

reconnect:
	conn := client.dial()
	if conn == nil {
		return
	}

	if client.isClosed() {
		conn.Close()
		return
	}

	client.mu.Lock()
	client.conns[conn] = struct{}{}
	client.mu.Unlock()

	tcpConn := newTCPConn(conn, client.getOpts().WriteBuffer, client.msgParser)
	agent := client.newAgent(tcpConn)
	agent.OnConnect()
	agent.Run()

	// cleanup
	tcpConn.Close()
	client.mu.Lock()
	delete(client.conns, conn)
	client.mu.Unlock()
	agent.OnClose()

	if client.getOpts().AutoReconnect {
		time.Sleep(client.getOpts().ConnectInterval)
		goto reconnect
	}
}

func (client *TCPClient) Close() {
	if !atomic.CompareAndSwapInt32(&client.closed, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}
	client.mu.RLock()
	for conn := range client.conns {
		conn.Close()
	}
	client.conns = nil
	client.mu.RUnlock()
	client.wg.Wait()
}
