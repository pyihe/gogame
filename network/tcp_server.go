package network

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/network/packet"
	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
)

type tcpConnSet map[net.Conn]struct{}

type TCPServer struct {
	opts     atomic.Value
	newAgent func(*TCPConn) Agent

	tlsConfig *tls.Config
	listener  net.Listener

	msgParser packet.Parser

	// guard below
	connsMu sync.RWMutex
	conns   tcpConnSet

	waiter sync.WaitGroup
	closed int32
}

func NewTCPServer(opts TCPServerOptions, newAgent func(*TCPConn) Agent) (*TCPServer, error) {
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

	s := &TCPServer{
		newAgent:  newAgent,
		conns:     make(tcpConnSet),
		msgParser: packet.NewParser(msgOptions...),
		closed:    pkg.StatusRunning,
	}

	s.swapOpts(&opts)

	s.tlsConfig, err = buildTLSConfig(opts.TLSOption)
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %s", err)
	}

	return s, nil
}

func (server *TCPServer) getOpts() *TCPServerOptions {
	return server.opts.Load().(*TCPServerOptions)
}

func (server *TCPServer) swapOpts(opts *TCPServerOptions) {
	server.opts.Store(opts)
}

func (server *TCPServer) isClosed() bool {
	return atomic.LoadInt32(&server.closed) == pkg.StatusClosed
}

func (server *TCPServer) Start() {
	if server.isClosed() {
		return
	}
	var err error
	var opts = server.getOpts()

	switch {
	case server.tlsConfig != nil:
		server.listener, err = tls.Listen("tcp", opts.Addr, server.tlsConfig)
	default:
		server.listener, err = net.Listen("tcp", opts.Addr)
	}
	if err != nil {
		log.Fatalf("failed to listen tcp: %s", err)
	}

	server.waiter.Add(1)

	gopool.AddTask(func() {
		defer server.waiter.Done()
		var retryNum int
		for {
			conn, err := server.listener.Accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					// 重试次数达到最大值，不再重试
					if retryNum >= server.getOpts().MaxRetry {
						return
					}
					retryNum++
					delay := pkg.Get(nil, retryNum)
					log.Printf("accept error: %v; retrying in %v", err, delay)
					time.Sleep(delay)
					continue
				}
				return
			}
			retryNum = 0

			server.connsMu.Lock()
			if existCount := len(server.conns); existCount >= server.getOpts().MaxConnNum {
				server.connsMu.Unlock()
				conn.Close()
				log.Printf("too many connections")
				continue
			}

			server.conns[conn] = struct{}{}
			server.connsMu.Unlock()

			tcpConn := newTCPConn(conn, server.getOpts().WriteBuffer, server.msgParser)
			agent := server.newAgent(tcpConn)
			server.waiter.Add(1)
			gopool.AddTask(func() {
				agent.OnConnect()
				agent.Run()

				tcpConn.Close()
				server.connsMu.Lock()
				delete(server.conns, conn)
				server.connsMu.Unlock()
				agent.OnClose()

				server.waiter.Done()
			})
		}
	})
}

func (server *TCPServer) Close() {
	if !atomic.CompareAndSwapInt32(&server.closed, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}
	server.listener.Close()

	server.connsMu.RLock()
	for conn := range server.conns {
		conn.Close()
	}
	server.conns = nil
	server.connsMu.RUnlock()
	server.waiter.Wait()
}

func buildTLSConfig(opts *TLSOption) (*tls.Config, error) {
	if opts == nil || (opts.TLSCert == "" && opts.TLSKey == "") {
		return nil, nil
	}

	var tlsConfig *tls.Config
	var clientAuthPolicy = tls.VerifyClientCertIfGiven

	cert, err := tls.LoadX509KeyPair(opts.TLSCert, opts.TLSKey)
	if err != nil {
		return nil, err
	}
	switch opts.TLSClientAuthPolicy {
	case "require":
		clientAuthPolicy = tls.RequireAnyClientCert
	case "require-verify":
		clientAuthPolicy = tls.RequireAndVerifyClientCert
	default:
		clientAuthPolicy = tls.NoClientCert
	}

	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   clientAuthPolicy,
		MinVersion:   opts.TLSMinVersion,
	}

	if opts.TLSRootCAFile != "" {
		tlsCertPool := x509.NewCertPool()
		caCertFile, err := os.ReadFile(opts.TLSRootCAFile)
		if err != nil {
			return nil, err
		}
		if !tlsCertPool.AppendCertsFromPEM(caCertFile) {
			return nil, errors.New("failed to append certificate to pool")
		}
		tlsConfig.ClientCAs = tlsCertPool
	}

	return tlsConfig, nil
}
