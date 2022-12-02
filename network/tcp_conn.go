package network

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/network/packet"
	"github.com/pyihe/gogame/pkg"
)

type TCPConn struct {
	msgParser packet.Parser
	conn      net.Conn
	writeChan chan []byte
	closeFlag int32
}

func newTCPConn(conn net.Conn, writeBuffer int, msgParser packet.Parser) *TCPConn {
	tcpConn := new(TCPConn)
	tcpConn.conn = conn
	tcpConn.writeChan = make(chan []byte, writeBuffer)
	tcpConn.msgParser = msgParser

	gopool.AddTask(func() {
		tcpConn.writeLoop()
	})

	return tcpConn
}

func (tcpConn *TCPConn) isClosed() bool {
	return atomic.LoadInt32(&tcpConn.closeFlag) == pkg.StatusClosed
}

func (tcpConn *TCPConn) writeLoop() {
	for b := range tcpConn.writeChan {
		if b == nil {
			break
		}
		_, err := tcpConn.conn.Write(b)
		if err != nil {
			break
		}
	}
}

func (tcpConn *TCPConn) doDestroy() {
	if !atomic.CompareAndSwapInt32(&tcpConn.closeFlag, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}
	tcpConn.conn.(*net.TCPConn).SetLinger(0)
	tcpConn.conn.Close()
	close(tcpConn.writeChan)
}

func (tcpConn *TCPConn) WriteBytes(b []byte) {
	if b == nil {
		return
	}
	// conn已经关闭
	if tcpConn.isClosed() {
		return
	}

	if len(tcpConn.writeChan) == cap(tcpConn.writeChan) {
		tcpConn.doDestroy()
		return
	}
	tcpConn.writeChan <- b
}

func (tcpConn *TCPConn) SetReadDeadline(t time.Time) error {
	return tcpConn.conn.SetReadDeadline(t)
}

func (tcpConn *TCPConn) SetWriteDeadline(t time.Time) error {
	return tcpConn.conn.SetWriteDeadline(t)
}

func (tcpConn *TCPConn) Close() {
	tcpConn.doDestroy()
}

func (tcpConn *TCPConn) Read(b []byte) (int, error) {
	return tcpConn.conn.Read(b)
}

func (tcpConn *TCPConn) Write(b []byte) (int, error) {
	return tcpConn.conn.Write(b)
}

func (tcpConn *TCPConn) LocalAddr() net.Addr {
	return tcpConn.conn.LocalAddr()
}

func (tcpConn *TCPConn) RemoteAddr() net.Addr {
	return tcpConn.conn.RemoteAddr()
}

func (tcpConn *TCPConn) ReadMsg() ([]byte, error) {
	if tcpConn.isClosed() {
		return nil, pkg.ErrConnClosed
	}
	return tcpConn.msgParser.UnPacket(tcpConn.conn)
}

func (tcpConn *TCPConn) WriteMsg(args ...[]byte) error {
	if tcpConn.isClosed() {
		return pkg.ErrConnClosed
	}
	mData, err := tcpConn.msgParser.Packet(args...)
	if err != nil {
		return err
	}

	tcpConn.WriteBytes(mData)
	return nil
}
