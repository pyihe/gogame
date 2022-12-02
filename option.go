package gogame

// Options 服务器选项
type Options struct {
	ServeId uint16 // 服务器ID

	// cluster option
	ClusterAddr      string
	ClusterConnAddrs []string

	// pprof port
	ProfileAddr string
}
