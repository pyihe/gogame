package network

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/pkg"
)

type WSConn struct {
	conn      *websocket.Conn
	writeChan chan []byte
	maxMsgLen uint32
	closeFlag int32
}

func newWSConn(conn *websocket.Conn, writeBuffer int, maxMsgLen uint32) *WSConn {
	wsConn := new(WSConn)
	wsConn.conn = conn
	wsConn.writeChan = make(chan []byte, writeBuffer)
	wsConn.maxMsgLen = maxMsgLen

	gopool.AddTask(func() {
		wsConn.writeLoop()
	})
	return wsConn
}

func (wsConn *WSConn) isClosed() bool {
	return atomic.LoadInt32(&wsConn.closeFlag) == pkg.StatusClosed
}

func (wsConn *WSConn) writeLoop() {
	for b := range wsConn.writeChan {
		if b == nil {
			break
		}

		err := wsConn.conn.WriteMessage(websocket.BinaryMessage, b)
		if err != nil {
			break
		}
	}
}

func (wsConn *WSConn) doDestroy() {
	if !atomic.CompareAndSwapInt32(&wsConn.closeFlag, pkg.StatusRunning, pkg.StatusClosed) {
		return
	}
	wsConn.conn.UnderlyingConn().(*net.TCPConn).SetLinger(0)
	wsConn.conn.Close()
	close(wsConn.writeChan)
}

func (wsConn *WSConn) Close() {
	wsConn.doDestroy()
}

func (wsConn *WSConn) doWrite(b []byte) {
	if b == nil {
		return
	}
	if wsConn.isClosed() {
		return
	}
	if len(wsConn.writeChan) == cap(wsConn.writeChan) {
		wsConn.doDestroy()
		return
	}

	wsConn.writeChan <- b
}

func (wsConn *WSConn) LocalAddr() net.Addr {
	return wsConn.conn.LocalAddr()
}

func (wsConn *WSConn) RemoteAddr() net.Addr {
	return wsConn.conn.RemoteAddr()
}

func (wsConn *WSConn) Read(b []byte) (int, error) {
	return wsConn.conn.UnderlyingConn().Read(b)
}

func (wsConn *WSConn) Write(b []byte) (int, error) {
	return wsConn.conn.UnderlyingConn().Write(b)
}

func (wsConn *WSConn) ReadMsg() ([]byte, error) {
	_, b, err := wsConn.conn.ReadMessage()
	return b, err
}

func (wsConn *WSConn) WriteMsg(args ...[]byte) error {
	if wsConn.isClosed() {
		return pkg.ErrConnClosed
	}

	var mLen uint32
	for i := 0; i < len(args); i++ {
		mLen += uint32(len(args[i]))
	}

	if mLen <= 0 {
		return nil
	} else {
		if mLen > wsConn.maxMsgLen {
			return pkg.ErrMessageTooLong
		} else if mLen < 1 {
			return pkg.ErrMessageTooShort
		}
	}

	var mData []byte

	switch len(args) {
	case 1:
		mData = args[0]
	default:
		mData = make([]byte, mLen)
		at := 0
		for _, m := range args {
			copy(mData[at:], m)
			at += len(m)
		}
	}

	wsConn.doWrite(mData)
	return nil
}

func (wsConn *WSConn) SetReadDeadline(t time.Time) error {
	return wsConn.conn.SetReadDeadline(t)
}

func (wsConn *WSConn) SetWriteDeadline(t time.Time) error {
	return wsConn.conn.SetWriteDeadline(t)
}
