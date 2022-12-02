package main

import (
	"github.com/pyihe/gogame"
	"github.com/pyihe/gogame/cmd/game"
	"github.com/pyihe/gogame/cmd/gate"
	"github.com/pyihe/gogame/cmd/login"
)

func main() {
	opts := &gogame.Options{
		ServeId:     1,
		ProfileAddr: ":8080",
	}
	gogame.Run(
		opts,
		game.NewModule(),
		login.NewModule(),
		gate.NewModule(),
	)
}
