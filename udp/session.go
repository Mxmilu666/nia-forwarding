package udp

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Session 表示UDP会话
type Session struct {
	clientAddr     *net.UDPAddr
	targetConn     *net.UDPConn
	targetAddr     *net.UDPAddr
	sourceConn     *net.UDPConn
	sessions       *sync.Map
	sessionKey     string
	lastActiveTime time.Time
	done           chan struct{}
	mu             sync.Mutex
	bufferSize     int
	timeout        time.Duration
}

// NewSession 创建一个新的UDP会话
func NewSession(ctx context.Context, sourceConn *net.UDPConn, clientAddr *net.UDPAddr,
	targetAddrStr string, sessions *sync.Map, sessionKey string,
	bufferSize int, timeout time.Duration) (*Session, error) {

	targetAddr, err := net.ResolveUDPAddr("udp6", targetAddrStr)
	if err != nil {
		return nil, fmt.Errorf("无法解析目标UDP地址: %w", err)
	}

	targetConn, err := net.ListenUDP("udp6", &net.UDPAddr{IP: net.IPv6zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("无法创建UDP会话: %w", err)
	}

	session := &Session{
		clientAddr:     clientAddr,
		targetConn:     targetConn,
		targetAddr:     targetAddr,
		sourceConn:     sourceConn,
		sessions:       sessions,
		sessionKey:     sessionKey,
		lastActiveTime: time.Now(),
		done:           make(chan struct{}),
		bufferSize:     bufferSize,
		timeout:        timeout,
	}

	log.Printf("UDP会话创建: %s -> %s", clientAddr.String(), targetAddrStr)

	// 处理从目标返回的数据
	go session.handleTargetData(ctx)

	// 启动超时检查
	go session.checkTimeout(ctx)

	return session, nil
}

// Refresh 刷新会话活动时间
func (s *Session) Refresh() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastActiveTime = time.Now()
}

// Send 发送数据到目标
func (s *Session) Send(data []byte) {
	s.Refresh()
	if _, err := s.targetConn.WriteToUDP(data, s.targetAddr); err != nil {
		log.Printf("UDP发送到目标错误: %v", err)
	}
}

// 处理从目标返回的数据
func (s *Session) handleTargetData(ctx context.Context) {
	buffer := make([]byte, s.bufferSize)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		default:
			// 设置超时以便能检查上下文取消
			s.targetConn.SetReadDeadline(time.Now().Add(time.Second))
			n, _, err := s.targetConn.ReadFromUDP(buffer)

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// 超时，继续循环
					continue
				}

				// 其他网络错误，关闭会话
				s.Close()
				return
			}

			s.Refresh()

			// 将数据返回给客户端
			if _, err := s.sourceConn.WriteToUDP(buffer[:n], s.clientAddr); err != nil {
				log.Printf("UDP返回到客户端错误: %v", err)
				s.Close()
				return
			}
		}
	}
}

// 检查会话是否超时
func (s *Session) checkTimeout(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.Close()
			return
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			inactive := time.Since(s.lastActiveTime) > s.timeout
			s.mu.Unlock()

			if inactive {
				log.Printf("UDP会话超时: %s", s.sessionKey)
				s.Close()
				return
			}
		}
	}
}

// Close 关闭会话
func (s *Session) Close() {
	select {
	case <-s.done:
		// 已经关闭
		return
	default:
		close(s.done)
		s.targetConn.Close()
		s.sessions.Delete(s.sessionKey)
		log.Printf("UDP会话关闭: %s", s.sessionKey)
	}
}
