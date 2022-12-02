package gogame

import (
	"github.com/pyihe/gogame/network"
)

type clusterAgent struct {
	conn *network.TCPConn
}

func newClusterAgent(conn *network.TCPConn) network.Agent {
	a := new(clusterAgent)
	a.conn = conn
	return a
}

func (a *clusterAgent) Run() {}

func (a *clusterAgent) OnConnect() {}

func (a *clusterAgent) OnClose() {}
