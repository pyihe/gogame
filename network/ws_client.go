package network

import (
	"crypto/tls"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
)

type WSClient struct {
	opts      atomic.Value
	tlsConfig *tls.Config

	newAgent func(*WSConn) Agent
	dialer   websocket.Dialer

	waiter sync.WaitGroup
	mu     sync.RWMutex
	conns  websocketConnSet

	closed int32
}

func NewWSClient(opts WSClientOption, newAgent func(*WSConn) Agent) (*WSClient, error) {
	if newAgent == nil {
		return nil, pkg.ErrNilNewAgent
	}
	opts.setDefault()

	var err error
	var tlsConfig *tls.Config
	tlsConfig, err = buildTLSConfig(opts.TLSOption)
	if err != nil {
		return nil, err
	}

	c := &WSClient{
		newAgent: newAgent,
		conns:    make(websocketConnSet),
		closed:   pkg.StatusInitial,
		dialer: websocket.Dialer{
			HandshakeTimeout: opts.HandshakeTimeout,
			TLSClientConfig:  tlsConfig,
		},
	}
	c.swapOpts(&opts)

	return c, nil
}

func (client *WSClient) getOpts() *WSClientOption {
	return client.opts.Load().(*WSClientOption)
}

func (client *WSClient) swapOpts(opts *WSClientOption) {
	client.opts.Store(opts)
}

func (client *WSClient) isClosed() bool {
	return atomic.LoadInt32(&client.closed) == pkg.StatusClosed
}

func (client *WSClient) Start() {
	if !atomic.CompareAndSwapInt32(&client.closed, pkg.StatusInitial, pkg.StatusRunning) {
		return
	}

	for i := 0; i < client.getOpts().ConnNum; i++ {
		client.waiter.Add(1)
		gopool.AddTask(client.connect)
	}
}

func (client *WSClient) dial() *websocket.Conn {
	const maxRetries = 3
	var retry int
	for {
		conn, _, err := client.dialer.Dial(client.getOpts().Addr, nil)
		if err == nil || client.isClosed() {
			return conn
		}

		if retry >= maxRetries {
			log.Printf("websocket connect(%v): retry timeout", client.getOpts().Addr)
			return conn
		}

		retry += 1
		log.Printf("failed to connect Websocket(%s) error: %v, retry: %d, maxRetry: %d", client.getOpts().Addr, err, retry, maxRetries)
		time.Sleep(pkg.Get(nil, retry))
		continue
	}
}

func (client *WSClient) connect() {
	defer client.waiter.Done()

reconnect:
	conn := client.dial()
	if conn == nil {
		return
	}
	conn.SetReadLimit(int64(client.getOpts().MsgMaxLen))

	client.mu.Lock()
	if client.isClosed() {
		client.mu.Unlock()
		conn.Close()
		return
	}
	client.conns[conn] = struct{}{}
	client.mu.Unlock()

	wsConn := newWSConn(conn, client.getOpts().WriteBuffer, client.getOpts().MsgMaxLen)
	agent := client.newAgent(wsConn)
	agent.OnConnect()
	agent.Run()

	// cleanup
	wsConn.Close()
	client.mu.Lock()
	delete(client.conns, conn)
	client.mu.Unlock()
	agent.OnClose()

	if client.getOpts().AutoReconnect {
		time.Sleep(client.getOpts().ConnectInterval)
		goto reconnect
	}
}

func (client *WSClient) Close() {
	if !atomic.CompareAndSwapInt32(&client.closed, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}

	client.mu.RLock()
	for conn := range client.conns {
		conn.Close()
	}
	client.conns = nil
	client.mu.RUnlock()

	client.waiter.Wait()
}
