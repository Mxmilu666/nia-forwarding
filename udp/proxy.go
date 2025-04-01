package udp

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Proxy 表示UDP代理
type Proxy struct {
	proxyID    string
	listenAddr string
	targetAddr string
	bufferSize int
	timeout    time.Duration
}

// NewProxy 创建一个新的UDP代理
func NewProxy(proxyID, listenAddr, targetAddr string, bufferSize int, timeout time.Duration) *Proxy {
	return &Proxy{
		proxyID:    proxyID,
		listenAddr: listenAddr,
		targetAddr: targetAddr,
		bufferSize: bufferSize,
		timeout:    timeout,
	}
}

// Start 启动UDP代理服务
func (p *Proxy) Start(ctx context.Context) error {
	// 监听IPv4 UDP
	addr, err := net.ResolveUDPAddr("udp4", p.listenAddr)
	if err != nil {
		return fmt.Errorf("无法解析UDP监听地址: %w", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("无法监听UDP: %w", err)
	}
	defer conn.Close()

	log.Printf("[%s] UDP转发已启动: %s -> %s\n", p.proxyID, p.listenAddr, p.targetAddr)

	// 使用map存储UDP会话
	sessions := &sync.Map{}

	// 监听上下文取消
	go func() {
		<-ctx.Done()
		conn.Close()
		// 关闭所有会话
		sessions.Range(func(key, value interface{}) bool {
			if s, ok := value.(*Session); ok {
				s.Close()
			}
			return true
		})
	}()

	buffer := make([]byte, p.bufferSize)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("[%s] UDP读取错误: %v", p.proxyID, err)
				continue
			}
		}

		data := make([]byte, n)
		copy(data, buffer[:n])

		clientAddrStr := clientAddr.String()
		var session *Session

		// 查找或创建会话
		v, ok := sessions.Load(clientAddrStr)
		if !ok {
			// 使用客户端地址作为会话 ID
			newSession, err := NewSession(ctx, conn, clientAddr, p.targetAddr, sessions, clientAddrStr, p.bufferSize, p.timeout)
			if err != nil {
				log.Printf("[%s] 创建UDP会话失败: %v", p.proxyID, err)
				continue
			}
			sessions.Store(clientAddrStr, newSession)
			session = newSession
		} else {
			session = v.(*Session)
			session.Refresh() // 刷新超时
		}

		// 发送数据到目标
		session.Send(data)
	}
}
