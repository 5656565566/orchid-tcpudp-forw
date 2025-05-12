package mapping

import (
	"context"
	"net"
	"sync"
)

type TcpPortMapping struct {
	ListenAddr  string
	ForwardAddr string
	Listener    net.Listener
}

type UdpPortMapping struct {
	ListenAddr  string
	ForwardAddr string
	Listener    net.PacketConn
}

var (
	MappingsTcp = &sync.Map{}
	MappingsUdp = &sync.Map{}
)

type PipeHandler interface {
	CopyAndHandle(ctx context.Context, src, dst net.Conn, wg *sync.WaitGroup)
}
