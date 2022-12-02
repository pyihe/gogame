package network

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
)

type websocketConnSet map[*websocket.Conn]struct{}

type WSServer struct {
	opts      atomic.Value
	newAgent  func(*WSConn) Agent
	ln        net.Listener
	tlsConfig *tls.Config
	upgrader  websocket.Upgrader
	waiter    sync.WaitGroup
	connsMu   sync.RWMutex
	conns     websocketConnSet
	closed    int32
}

func NewWSServer(opts WSServerOption, newAgent func(*WSConn) Agent) (*WSServer, error) {
	if newAgent == nil {
		return nil, pkg.ErrNilNewAgent
	}

	opts.setDefault()

	var err error
	var s = &WSServer{
		newAgent: newAgent,
		conns:    make(websocketConnSet),
		closed:   pkg.StatusRunning,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: opts.HTTPTimeout,
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
	}

	s.swapOpts(&opts)

	s.tlsConfig, err = buildTLSConfig(opts.TLSOption)
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %s", err)
	}

	return s, nil
}

func (server *WSServer) getOpts() *WSServerOption {
	return server.opts.Load().(*WSServerOption)
}

func (server *WSServer) swapOpts(opts *WSServerOption) {
	server.opts.Store(opts)
}

func (server *WSServer) isClosed() bool {
	return atomic.LoadInt32(&server.closed) == pkg.StatusClosed
}

func (server *WSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 服务器是否已关闭
	if server.isClosed() {
		http.Error(w, "server closed", http.StatusServiceUnavailable)
		return
	}

	opts := server.getOpts()
	// 协议升级
	conn, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade err: %v", err)
		return
	}

	conn.SetReadLimit(int64(opts.MsgMaxLen))

	server.connsMu.Lock()
	if existCount := len(server.conns); existCount >= opts.MaxConnNum {
		server.connsMu.Unlock()
		http.Error(w, "too many connection", http.StatusTooManyRequests)
		conn.Close()
		log.Printf("too many connections")
		return
	}

	server.conns[conn] = struct{}{}
	server.connsMu.Unlock()

	server.waiter.Add(1)
	// 新建WSConn
	wsConn := newWSConn(conn, opts.WriteBuff, opts.MsgMaxLen)
	agent := server.newAgent(wsConn)
	agent.OnConnect()
	agent.Run()

	server.waiter.Done()
	wsConn.Close()
	server.connsMu.Lock()
	delete(server.conns, conn)
	server.connsMu.Unlock()
	agent.OnClose()
}

func (server *WSServer) Start() {
	if server.isClosed() {
		return
	}

	var err error
	var opts = server.getOpts()

	switch {
	case server.tlsConfig != nil:
		server.ln, err = tls.Listen("tcp", opts.Addr, server.tlsConfig)
	default:
		server.ln, err = net.Listen("tcp", opts.Addr)
	}

	if err != nil {
		log.Fatalf("failed to listen address: %v", err)
		return
	}

	httpServer := &http.Server{
		Addr:           opts.Addr,
		Handler:        server,
		ReadTimeout:    opts.HTTPTimeout,
		WriteTimeout:   opts.HTTPTimeout,
		MaxHeaderBytes: 1024,
	}

	gopool.AddTask(func() {
		httpServer.Serve(server.ln)
	})
}

func (server *WSServer) Close() {
	if !atomic.CompareAndSwapInt32(&server.closed, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}

	server.ln.Close()

	server.connsMu.Lock()
	for conn := range server.conns {
		conn.Close()
	}
	server.conns = nil
	server.connsMu.Unlock()

	server.waiter.Wait()
}
