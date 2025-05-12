package mapping

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

func AddTcpMapping(listenAddr, forwardAddr string) error {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	mapping := &TcpPortMapping{
		ListenAddr:  listenAddr,
		ForwardAddr: forwardAddr,
		Listener:    ln,
	}

	log.Printf("正在监听 TCP %s 并转发至 %s\n", mapping.ListenAddr, mapping.ForwardAddr)

	MappingsTcp.Store(listenAddr, mapping)
	go handleTcpConnections(mapping)
	return nil
}

func handleTcpConnections(mapping *TcpPortMapping) {
	for {
		conn, err := mapping.Listener.Accept()
		if err != nil {
			log.Printf("接受连接失败 %s: %s", mapping.ListenAddr, err)
			return
		}
		go handleTcpRequest(conn, mapping.ForwardAddr)
	}
}

func handleTcpRequest(src net.Conn, forwardAddr string) {
	localAddr := src.LocalAddr().String()
	if localAddr == forwardAddr {
		log.Printf("源和目标地址相同，关闭连接: %s", localAddr)
		src.Close()
		return
	}

	_, err := net.ResolveTCPAddr("tcp", forwardAddr)
	if err != nil {
		log.Printf("无法解析转发地址: %s", err)
		src.Close()
		return
	}

	log.Printf("连接建立: %s -> %s\n", src.RemoteAddr(), forwardAddr)
	dest, err := net.Dial("tcp", forwardAddr)
	if err != nil {
		log.Printf("连接转发地址失败: %s", err)
		src.Close()
		return
	}

	timeout := 30 * time.Second
	TcpPipe(src, dest, timeout)
}

func TcpPipe(src, dest net.Conn, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go copyAndHandleTcp(ctx, src, dest, &wg)
	go copyAndHandleTcp(ctx, dest, src, &wg)
	wg.Wait()
}

func copyAndHandleTcp(ctx context.Context, src, dst net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	select {
	case <-ctx.Done():
		log.Println("超时")
		return
	default:
		if _, err := io.Copy(dst, src); err != nil {
			log.Printf("管道错误: %s", err)
			return
		}
	}
}

func DeleteTcpMapping(listenAddr string) error {
	value, ok := MappingsTcp.Load(listenAddr)
	if !ok {
		return errors.New("未找到映射")
	}
	mapping := value.(*TcpPortMapping)
	mapping.Listener.Close()
	MappingsTcp.Delete(listenAddr)
	return nil
}
