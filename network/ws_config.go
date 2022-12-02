package network

import (
	"math"
	"time"
)

type WSServerOption struct {
	Addr        string
	MaxConnNum  int
	WriteBuff   int
	MsgMaxLen   uint32
	HTTPTimeout time.Duration
	TLSOption   *TLSOption
}

func (opt *WSServerOption) setDefault() {
	if opt.MaxConnNum <= 0 {
		opt.MaxConnNum = math.MaxInt
	}
	if opt.WriteBuff <= 0 {
		opt.WriteBuff = 10000
	}

	if opt.MsgMaxLen == 0 {
		opt.MsgMaxLen = math.MaxUint32
	}

	if opt.HTTPTimeout <= 0 {
		opt.HTTPTimeout = 10 * time.Second
	}
}

type WSClientOption struct {
	Addr             string
	ConnNum          int
	AutoReconnect    bool
	ConnectInterval  time.Duration
	MsgMaxLen        uint32
	WriteBuffer      int
	HandshakeTimeout time.Duration

	TLSOption *TLSOption
}

func (opt *WSClientOption) setDefault() {
	if opt.ConnNum <= 0 {
		opt.ConnNum = 1
	}
	if opt.ConnectInterval <= 0 {
		opt.ConnectInterval = 3 * time.Second
	}
	if opt.WriteBuffer <= 0 {
		opt.WriteBuffer = 100
	}
	if opt.MsgMaxLen == 0 {
		opt.MsgMaxLen = 4096
	}
	if opt.HandshakeTimeout <= 0 {
		opt.HandshakeTimeout = 10 * time.Second
	}
}
