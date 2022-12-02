package gogame

import (
	"math"
	"sync"
	"time"

	"github.com/pyihe/gogame/internal/gopprof"
	"github.com/pyihe/gogame/internal/uuid"
	"github.com/pyihe/gogame/network"
	"github.com/pyihe/gogame/pkg"
	"github.com/pyihe/gogame/pkg/log"
)

var server struct {
	once sync.Once
	opts *Options

	clusterServer  *network.TCPServer
	clusterClients []*network.TCPClient
	mods           []*module // modules
}

func initial(opts *Options, modules ...Module) {
	server.once.Do(func() {
		server.opts = opts
		// 如果启动了pprof
		if opts.ProfileAddr != "" {
			nMods := make([]Module, len(modules)+1)
			copy(nMods[:len(modules)], modules)
			nMods[len(modules)] = gopprof.New(opts.ProfileAddr)
			modules = nMods
		}
		initModule(modules...)
		initCluster()
	})
}

func initModule(modules ...Module) {
	server.mods = make([]*module, len(modules))
	for i, m := range modules {
		server.mods[i] = newModule(m)
		m.Init()
	}
}

func initCluster() {
	if server.opts.ClusterAddr == "" && len(server.opts.ClusterConnAddrs) == 0 {
		return
	}
	msgOption := &network.TCPMsgOption{
		MsgHeaderLen: 4,
		MsgMinLen:    1,
		MsgMaxLen:    math.MaxUint32,
		LittleEndian: false,
	}
	if server.opts.ClusterAddr != "" {
		opts := network.TCPServerOptions{
			Addr:        server.opts.ClusterAddr,
			MaxConnNum:  math.MaxInt,
			WriteBuffer: 100,
			MsgOption:   msgOption,
		}
		server.clusterServer, _ = network.NewTCPServer(opts, newClusterAgent)
	}

	server.clusterClients = make([]*network.TCPClient, 0, len(server.opts.ClusterConnAddrs))
	for _, addr := range server.opts.ClusterConnAddrs {
		opts := network.TCPClientOption{
			Addr:            addr,
			ConnNum:         1,
			WriteBuffer:     100,
			ConnectInterval: 3 * time.Second,
			MsgOption:       msgOption,
		}
		client, _ := network.NewTCPClient(opts, newClusterAgent)
		server.clusterClients = append(server.clusterClients, client)
	}
}

func start() {
	// 运行每个模块
	for _, m := range server.mods {
		m.run()
	}

	// 开启cluster
	if server.clusterServer != nil {
		server.clusterServer.Start()
	}
	for _, client := range server.clusterClients {
		client.Start()
	}
}

func stop() {
	// 关闭cluster
	if server.clusterServer != nil {
		server.clusterServer.Close()
	}
	for _, client := range server.clusterClients {
		client.Close()
	}
	// 关闭每个模块
	for i := len(server.mods) - 1; i >= 0; i-- {
		m := server.mods[i]
		m.destroy()
	}
}

// Run 模块注册入口，必须提供Options与Module
func Run(opts *Options, modules ...Module) {
	if len(modules) == 0 {
		panic("modules required")
	}
	if opts == nil {
		panic("options required")
	}

	// 初始化ID生成器
	uuid.New(opts.ServeId)

	// 初始化
	initial(opts, modules...)

	// 开始运行
	start()

	log.Printf("server start running...")

	// wait to be close
	pkg.Wait(stop)
}
