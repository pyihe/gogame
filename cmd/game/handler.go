package game

import (
	"github.com/pyihe/gogame"
	"github.com/pyihe/gogame/cmd/protocol"
	"github.com/pyihe/gogame/pkg/log"
)

func handleHello(args ...interface{}) {
	m := args[0].(*protocol.Hello)
	a := args[1].(gogame.Agent)
	m.Name = "client"
	log.Printf("hello %v, %+v", a.GetId(), m)
	a.WriteMsg(m)
}
