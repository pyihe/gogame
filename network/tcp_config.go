package network

import (
	"math"
	"time"
)

type TLSOption struct {
	// 证书路径
	TLSCert string
	// 密钥路径
	TLSKey string
	// tcp客户端验证策略
	TLSClientAuthPolicy string
	// ca
	TLSRootCAFile string
	// 可接受的TLS最低版本号
	TLSMinVersion uint16
}

type TCPMsgOption struct {
	// 消息头长度
	MsgHeaderLen int
	// 单次发送的最小消息体长度
	MsgMinLen uint32
	// 单次发送的最大消息体长度
	MsgMaxLen uint32
	// 封包/拆包大小端
	LittleEndian bool
}

type TCPServerOptions struct {
	// 服务相关属性配置
	// TCP地址
	Addr string
	// 最大重连次数
	MaxRetry int
	// 最大连接数
	MaxConnNum int
	// 写缓冲区大小
	WriteBuffer int

	// TLS相关配置
	TLSOption *TLSOption

	// 消息相关配置
	MsgOption *TCPMsgOption
}

func (opt *TCPServerOptions) setDefault() {
	if opt.MaxConnNum <= 0 {
		opt.MaxConnNum = math.MaxInt
	}
	if opt.WriteBuffer <= 0 {
		opt.WriteBuffer = 100
	}
	if opt.MsgOption == nil {
		opt.MsgOption = &TCPMsgOption{}
	}
	if l := opt.MsgOption.MsgHeaderLen; l != 1 && l != 2 && l != 4 {
		opt.MsgOption.MsgHeaderLen = 2
	}
	if opt.MsgOption.MsgMinLen == 0 {
		opt.MsgOption.MsgMinLen = 1
	}
	if opt.MsgOption.MsgMaxLen == 0 {
		opt.MsgOption.MsgMaxLen = math.MaxUint32
	}
	if opt.MaxRetry <= 0 {
		opt.MaxRetry = 7
	}
}

type TCPClientOption struct {
	Addr            string
	ConnNum         int
	AutoReconnect   bool
	WriteBuffer     int
	ConnectInterval time.Duration

	TLSOption *TLSOption

	MsgOption *TCPMsgOption
}

func (opt *TCPClientOption) setDefault() {
	if opt.ConnNum <= 0 {
		opt.ConnNum = 1
	}
	if opt.ConnectInterval <= 0 {
		opt.ConnectInterval = 3 * time.Second
	}
	if opt.WriteBuffer <= 0 {
		opt.WriteBuffer = 100
	}

	if opt.MsgOption == nil {
		opt.MsgOption = &TCPMsgOption{}
	}
	if l := opt.MsgOption.MsgHeaderLen; l != 1 && l != 2 && l != 4 {
		opt.MsgOption.MsgHeaderLen = 2
	}
	if opt.MsgOption.MsgMinLen == 0 {
		opt.MsgOption.MsgMinLen = 1
	}
	if opt.MsgOption.MsgMaxLen == 0 {
		opt.MsgOption.MsgMaxLen = 4096
	}
}
