package route

import (
	"reflect"

	"github.com/pyihe/gogame/chanrpc"
	"github.com/pyihe/gogame/pkg"
)

type MessageHandler func(...interface{})

type Message struct {
	id      uint16
	mType   reflect.Type
	router  *chanrpc.Server
	handler MessageHandler
	initial bool
}

func NewMessage(id uint16, m interface{}) *Message {
	mType := reflect.TypeOf(m)
	if mType == nil || mType.Kind() != reflect.Ptr {
		panic(pkg.ErrPointerRequired)
	}
	return &Message{
		id:      id,
		mType:   mType,
		initial: true,
	}
}

func (m *Message) assert() {
	if !m.initial {
		panic("please initialize message by NewMessage")
	}
}

func (m *Message) SetRouter(router *chanrpc.Server) *Message {
	m.assert()
	m.router = router
	return m
}

func (m *Message) SetHandler(handler MessageHandler) *Message {
	m.assert()
	m.handler = handler
	return m
}
