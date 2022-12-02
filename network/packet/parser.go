package packet

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/pyihe/gogame/pkg"
)

// 消息格式
//  -------------------------------
// ｜mLen|message|....|mLen|message|
//  -------------------------------

// Parser 包解析器
type Parser interface {
	// Packet 封包
	Packet(...[]byte) ([]byte, error)

	// UnPacket 拆包
	UnPacket(io.Reader) ([]byte, error)
}

type parser struct {
	hLen         int    // 消息头长度
	minMsgLen    uint32 // 单次发送的最短消息长度，不是单个消息体的最小长度
	maxMsgLen    uint32 // 单次发送的最大消息长度，不是单个消息体的最大长度
	littleEndian bool   // 大小端
}

type Option func(*parser)

// WithHeader 设置消息头长度
func WithHeader(n int) Option {
	return func(parser *parser) {
		parser.hLen = n
	}
}

// WithMinLen 设置最短消息长度
func WithMinLen(minLen uint32) Option {
	return func(parser *parser) {
		parser.minMsgLen = minLen
	}
}

// WithMaxLen 设置最长消息长度
func WithMaxLen(maxLen uint32) Option {
	return func(parser *parser) {
		parser.maxMsgLen = maxLen
	}
}

// WithLittleEndian 设置大小端
func WithLittleEndian(b bool) Option {
	return func(parser *parser) {
		parser.littleEndian = b
	}
}

func NewParser(opts ...Option) Parser {
	p := &parser{}

	for _, op := range opts {
		op(p)
	}

	// 完善参数的初始化
	p.setDefault()

	return p
}

func (p *parser) setDefault() {
	if p.hLen != 1 && p.hLen != 2 && p.hLen != 4 {
		p.hLen = 2
	}
	if p.minMsgLen == 0 {
		p.minMsgLen = 1
	}
	if p.maxMsgLen == 0 {
		p.maxMsgLen = 4096
	}
	var max uint32
	switch p.hLen {
	case 1:
		max = math.MaxUint8
	case 2:
		max = math.MaxUint16
	case 4:
		max = math.MaxUint32
	}
	if p.minMsgLen > max {
		p.minMsgLen = max
	}
	if p.maxMsgLen > max {
		p.maxMsgLen = max
	}
}

func (p *parser) byteOrder() binary.ByteOrder {
	var byteOrder binary.ByteOrder = binary.LittleEndian
	if !p.littleEndian {
		byteOrder = binary.BigEndian
	}
	return byteOrder
}

func (p *parser) Packet(msgs ...[]byte) ([]byte, error) {
	// 1. 获取消息实际长度
	var mLen uint32
	for _, m := range msgs {
		mLen += uint32(len(m))
	}

	// 2. 判断消息长度是否合法
	if mLen < p.minMsgLen {
		return nil, pkg.ErrMessageTooShort
	}
	if mLen > p.maxMsgLen {
		return nil, pkg.ErrMessageTooLong
	}

	// 3. 根据大小端将消息长度写入消息头
	var mData = make([]byte, mLen+uint32(p.hLen))
	var byteOrder = p.byteOrder()

	switch p.hLen {
	case 1:
		mData[0] = byte(mLen)
	case 2:
		byteOrder.PutUint16(mData, uint16(mLen))
	case 4:
		byteOrder.PutUint32(mData, mLen)
	}

	// 4. 汇聚所有消息
	at := p.hLen
	for _, m := range msgs {
		copy(mData[at:], m)
		at += len(m)
	}

	return mData, nil
}

func (p *parser) UnPacket(reader io.Reader) ([]byte, error) {
	// 1. 读取消息头字节流
	hb := make([]byte, p.hLen)
	if _, err := io.ReadFull(reader, hb); err != nil {
		return nil, err
	}

	// 2. 根据消息头长度参数以及大小端参数解析消息头的长度值
	var mLen uint32
	var byteOrder = p.byteOrder()

	switch p.hLen {
	case 1:
		mLen = uint32(hb[0])
	case 2:
		mLen = uint32(byteOrder.Uint16(hb))
	case 4:
		mLen = byteOrder.Uint32(hb)
	}

	// 3. 判断消息长度是否符合要求
	if mLen < p.minMsgLen {
		return nil, pkg.ErrMessageTooShort
	}
	if mLen > p.maxMsgLen {
		return nil, pkg.ErrMessageTooLong
	}

	// 4. 读取实际的消息
	m := make([]byte, mLen)
	if _, err := io.ReadFull(reader, m); err != nil {
		return nil, err
	}
	return m, nil
}
