package mapping

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

func AddUdpMapping(listenAddr, forwardAddr string) error {
	ln, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		return err
	}

	mapping := &UdpPortMapping{
		ListenAddr:  listenAddr,
		ForwardAddr: forwardAddr,
		Listener:    ln,
	}

	log.Printf("正在监听 UDP: %s 并转发至 %s\n", mapping.ListenAddr, mapping.ForwardAddr)

	MappingsUdp.Store(listenAddr, mapping)
	go handleUdpConnections(mapping)
	return nil
}

func handleUdpConnections(mapping *UdpPortMapping) {
	buffer := make([]byte, 2048)

	for {
		_, srcAddr, err := mapping.Listener.ReadFrom(buffer)
		if err != nil {
			log.Printf("接受数据包失败 %s: %s", mapping.ListenAddr, err)
			return
		}

		go handleUdpRequest(mapping.Listener, mapping.ForwardAddr, srcAddr)
	}
}

func handleUdpRequest(src net.PacketConn, forwardAddr string, srcAddr net.Addr) {
	localAddr := src.LocalAddr().String()
	if localAddr == forwardAddr {
		log.Printf("源和目标地址相同，关闭连接: %s", localAddr)
		return
	}

	_, err := net.ResolveUDPAddr("udp", forwardAddr)
	if err != nil {
		log.Printf("无法解析转发地址: %s", err)
		return
	}

	log.Printf("连接建立: %s -> %s\n", srcAddr, forwardAddr)

	timeout := 30 * time.Second
	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dest, err := net.Dial("udp", forwardAddr)
	if err != nil {
		log.Printf("连接转发地址失败: %s", err)
		return
	}
	defer dest.Close()

	UdpPipe(src, dest.(net.PacketConn), srcAddr, dest.RemoteAddr(), timeout)
}

func UdpPipe(src, dest net.PacketConn, srcAddr net.Addr, destAddr net.Addr, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go copyAndHandleUdp(ctx, src, dest, srcAddr, &wg)
	go copyAndHandleUdp(ctx, dest, src, destAddr, &wg)
	wg.Wait()
}

func copyAndHandleUdp(ctx context.Context, src, dst net.PacketConn, dstAddr net.Addr, wg *sync.WaitGroup) {
	defer wg.Done()
	buffer := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			log.Println("超时")
			return
		default:
			n, _, err := src.ReadFrom(buffer)
			if err != nil {
				log.Printf("读取错误: %s", err)
				return
			}

			if _, err := dst.WriteTo(buffer[:n], dstAddr); err != nil {
				log.Printf("写入错误: %s", err)
				return
			}
		}
	}
}

func DeleteUdpMapping(listenAddr string) error {
	value, ok := MappingsUdp.Load(listenAddr)
	if !ok {
		return errors.New("未找到映射")
	}
	mapping := value.(*UdpPortMapping)
	mapping.Listener.Close()
	MappingsUdp.Delete(listenAddr)
	return nil
}
