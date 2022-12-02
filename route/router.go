package route

import (
	"encoding/binary"
	"math"
	"reflect"
	"strings"

	"github.com/pyihe/gogame/chanrpc"
	"github.com/pyihe/gogame/internal/gopool"
	"github.com/pyihe/gogame/pkg"
)

const idLen = 2

type Codec interface {
	Name() string
	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
}

var (
	codecMap = make(map[string]Codec)
)

func RegisterCodec(c Codec) {
	if c != nil {
		codecMap[strings.ToLower(c.Name())] = c
	}
}

func GetCodec(name string) Codec {
	return codecMap[strings.ToLower(name)]
}

type Processor interface {
	// Registered 消息是否已经注册
	Registered(msgId uint16) bool

	// Register 注册消息
	Register(msg *Message)

	// SetRouter 设置消息路由
	SetRouter(msgID uint16, router *chanrpc.Server)

	// SetHandler 设置消息handler
	// handler和router同时设置了的话，只执行handler
	SetHandler(msgID uint16, handler MessageHandler)

	// Route must goroutine safe
	Route(msg interface{}, userData interface{}) error

	// Marshal must goroutine safe
	// 序列化消息，序列化后的格式为
	//  --------------------------
	// | mid (2byte) | msg length |
	//  --------------------------
	Marshal(msg interface{}) ([]byte, error)

	// Unmarshal must goroutine safe
	Unmarshal(data []byte) (interface{}, error)
}

type processor struct {
	littleEndian bool
	codec        Codec
	msgMap       *pkg.Map
	typeMap      *pkg.Map
}

func NewProcessor(littleEndian bool, codec Codec) Processor {
	if codec == nil {
		panic(pkg.ErrCodecRequired)
	}
	return &processor{
		codec:        codec,
		littleEndian: littleEndian,
		msgMap:       &pkg.Map{},
		typeMap:      &pkg.Map{},
	}
}

func (p *processor) byteOrder() binary.ByteOrder {
	switch p.littleEndian {
	case false:
		return binary.BigEndian
	default:
		return binary.LittleEndian
	}
}

func (p *processor) isRegistered(mId uint16) (*Message, bool) {
	m, ok := p.msgMap.Get(mId).(*Message)
	return m, ok
}

func (p *processor) Registered(mId uint16) bool {
	_, ok := p.isRegistered(mId)
	return ok
}

func (p *processor) Register(msg *Message) {
	// 不能为空
	if msg == nil {
		panic(pkg.ErrPointerRequired)
	}
	msg.assert()

	// 是否已经注册
	if _, ok := p.isRegistered(msg.id); ok {
		panic(pkg.ErrRepeatedRegister)
	}

	// 消息数量达到上限
	if p.msgMap.Len() > math.MaxUint16 {
		panic(pkg.ErrMessageTooMany)
	}
	p.msgMap.Set(msg.id, msg)
	p.typeMap.Set(msg.mType, msg)
}

func (p *processor) SetRouter(messageID uint16, router *chanrpc.Server) {
	// 是否以注册
	m, ok := p.isRegistered(messageID)
	if !ok {
		panic(pkg.ErrNotRegistered)
	}
	m.router = router
}

func (p *processor) SetHandler(messageID uint16, handler MessageHandler) {
	m, ok := p.isRegistered(messageID)
	if !ok {
		panic(pkg.ErrNotRegistered)
	}
	m.handler = handler
}

func (p *processor) Route(msg interface{}, userData interface{}) (err error) {
	mType := reflect.TypeOf(msg)
	m, ok := p.typeMap.Get(mType).(*Message)
	if !ok {
		err = pkg.ErrNotRegistered
		return
	}
	if m.handler != nil {
		gopool.AddTask(func() {
			m.handler(msg, userData)
		})
	}
	if m.router != nil {
		m.router.Go(mType, msg, userData)
	}
	return
}

func (p *processor) Marshal(msg interface{}) ([]byte, error) {
	mType := reflect.TypeOf(msg)
	m, ok := p.typeMap.Get(mType).(*Message)
	if !ok {
		return nil, pkg.ErrNotRegistered
	}

	mBytes, err := p.codec.Encode(msg)
	if err != nil {
		return nil, err
	}

	mData := make([]byte, len(mBytes)+idLen)
	p.byteOrder().PutUint16(mData[:idLen], m.id)
	copy(mData[2:], mBytes)

	return mData, nil
}

func (p *processor) Unmarshal(data []byte) (interface{}, error) {
	mId := p.byteOrder().Uint16(data[:2])

	m, ok := p.isRegistered(mId)
	if !ok {
		return nil, pkg.ErrNotRegistered
	}
	msg := reflect.New(m.mType.Elem()).Interface()
	err := p.codec.Decode(data[2:], msg)

	return msg, err
}
