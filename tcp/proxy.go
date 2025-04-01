package tcp

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// Proxy 表示TCP代理
type Proxy struct {
	listenAddr string
	targetAddr string
	proxyID    string
}

// NewProxy 创建一个新的TCP代理
func NewProxy(proxyID, listenAddr, targetAddr string) *Proxy {
	return &Proxy{
		proxyID:    proxyID,
		listenAddr: listenAddr,
		targetAddr: targetAddr,
	}
}

// Start 启动TCP代理服务
func (p *Proxy) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp4", p.listenAddr)
	if err != nil {
		return fmt.Errorf("无法监听TCP: %w", err)
	}
	defer listener.Close()

	log.Printf("[%s]TCP转发已启动: %s -> %s\n", p.proxyID, p.listenAddr, p.targetAddr)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("[%s]TCP接受连接错误: %v", p.proxyID, err)
				continue
			}
		}

		go p.handleConnection(ctx, conn)
	}
}

func (p *Proxy) handleConnection(ctx context.Context, clientConn net.Conn) {
	defer clientConn.Close()

	targetConn, err := net.Dial("tcp6", p.targetAddr)
	if err != nil {
		log.Printf("[%s]无法连接到TCP目标 %s: %v", p.proxyID, p.targetAddr, err)
		return
	}
	defer targetConn.Close()

	log.Printf("[%s]TCP转发: %s -> %s", p.proxyID, clientConn.RemoteAddr(), p.targetAddr)

	// 创建一个新的上下文，在连接关闭时取消
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	// 客户端 -> 目标
	go func() {
		defer wg.Done()
		defer cancel() // 任一方向出错都会取消整个连接
		if _, err := io.Copy(targetConn, clientConn); err != nil {
			if !isClosedConnError(err) {
				log.Printf("[%s]TCP客户端->目标错误: %v", p.proxyID, err)
			}
		}
	}()

	// 目标 -> 客户端
	go func() {
		defer wg.Done()
		defer cancel() // 任一方向出错都会取消整个连接
		if _, err := io.Copy(clientConn, targetConn); err != nil {
			if !isClosedConnError(err) {
				log.Printf("[%s]TCP目标->客户端错误: %v", p.proxyID, err)
			}
		}
	}()

	// 等待连接结束或上下文被取消
	select {
	case <-connCtx.Done():
		// 连接已取消，关闭连接并等待goroutine完成
		clientConn.Close()
		targetConn.Close()
	}

	wg.Wait()
}

// 判断是否为连接关闭错误
func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		return true
	}
	if opErr, ok := err.(*net.OpError); ok {
		return opErr.Err.Error() == "use of closed network connection"
	}
	return false
}
