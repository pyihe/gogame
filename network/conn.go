package network

import (
	"net"
	"time"
)

type Agent interface {
	// Run 运行Agent
	Run()
	// OnConnect 连接建立时执行，根据实际需要实现对应的功能
	// 比如对于有状态的模块，可能需要通知该模块有新连接上来了
	OnConnect()
	// OnClose 连接断开时调用，根据实际需要实现对应的功能
	// 比如对于需要维持有状态长连接的模块，可能需要通知该模块，连接断开了
	OnClose()
}

type Conn interface {
	// Close 关闭底层连接
	Close()
	// LocalAddr 获取本地主机网络地址
	LocalAddr() net.Addr
	// RemoteAddr 获取远端主机网络地址
	RemoteAddr() net.Addr
	// Read 从底层连接中读取数据
	Read(b []byte) (int, error)
	// Write 向底层连接写入数据
	Write(b []byte) (int, error)
	// ReadMsg 读取消息
	ReadMsg() ([]byte, error)
	// WriteMsg 写入消息
	WriteMsg(args ...[]byte) error
	// SetReadDeadline 设置读超时时间点
	SetReadDeadline(t time.Time) error
	// SetWriteDeadline 设置写超时时间点
	SetWriteDeadline(t time.Time) error
}
