package game

import (
	"github.com/pyihe/gogame"
	"github.com/pyihe/gogame/pkg/log"
)

type player struct {
	gogame.Agent
}

func newPlayer(args ...interface{}) {
	a := args[0].(gogame.Agent)

	p := &player{
		Agent: a,
	}

	/*
		处理玩家上线操作
	*/

	module.mu.Lock()
	module.agentSet[a.GetId()] = p
	module.mu.Unlock()
	log.Printf("玩家[%v]上线成功...", a.GetId())
}

func closePlayer(args ...interface{}) {
	a := args[0].(gogame.Agent)

	/*
		处理玩家下线操作
	*/

	module.mu.Lock()
	delete(module.agentSet, a.GetId())
	module.mu.Unlock()
	log.Printf("玩家[%v]下线成功...", a.GetId())
}
