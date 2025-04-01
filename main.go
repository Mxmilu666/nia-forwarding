package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Mxmilu666/nia-forwarding/config"
	"github.com/Mxmilu666/nia-forwarding/tcp"
	"github.com/Mxmilu666/nia-forwarding/udp"
)

var (
	configPath   string
	generateConf string
)

func init() {
	flag.StringVar(&configPath, "config", "", "配置文件路径 (默认为当前目录下的config.yaml)")
	flag.StringVar(&generateConf, "gen-config", "", "生成默认配置文件到指定路径")
	flag.Parse()
}

// 解析端口范围/列表字符串，返回所有端口的切片
func parsePorts(portsStr string) ([]int, error) {
	var ports []int

	// 先按逗号分割，处理可能的多个区间或单端口
	parts := strings.Split(portsStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 检查是否为端口范围 (例如 "8080-8085")
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("端口范围格式无效: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("无效的起始端口: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("无效的结束端口: %s", rangeParts[1])
			}

			if start > end {
				return nil, fmt.Errorf("端口范围无效，起始端口大于结束端口: %d > %d", start, end)
			}

			// 添加范围内的所有端口
			for port := start; port <= end; port++ {
				ports = append(ports, port)
			}
		} else {
			// 单个端口
			port, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("无效的端口号: %s", part)
			}
			ports = append(ports, port)
		}
	}

	return ports, nil
}

func main() {
	// 如果指定了生成配置文件
	if generateConf != "" {
		if err := config.SaveDefaultConfig(generateConf); err != nil {
			log.Fatalf("生成配置文件失败: %v", err)
		}
		log.Printf("默认配置已保存到: %s", generateConf)
		return
	}

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// 启动TCP转发服务
	for i, tcpCfg := range cfg.TCP {
		if !tcpCfg.Enabled {
			continue
		}

		groupID := fmt.Sprintf("tcp-%d", i+1)

		listenPorts, err := parsePorts(tcpCfg.ListenPorts)
		if err != nil {
			log.Printf("TCP配置[%s]监听端口解析错误: %v", groupID, err)
			continue
		}

		targetPorts, err := parsePorts(tcpCfg.TargetPorts)
		if err != nil {
			log.Printf("TCP配置[%s]目标端口解析错误: %v", groupID, err)
			continue
		}

		// 检查端口数量是否匹配
		if len(listenPorts) != len(targetPorts) {
			log.Printf("TCP配置[%s]错误: 监听端口数量(%d)与目标端口数量(%d)不匹配",
				groupID, len(listenPorts), len(targetPorts))
			continue
		}

		// 为每对端口创建一个代理
		for j := 0; j < len(listenPorts); j++ {
			wg.Add(1)
			listenAddr := fmt.Sprintf("%s:%d", tcpCfg.ListenIP, listenPorts[j])
			targetAddr := fmt.Sprintf("%s:%d", tcpCfg.TargetIP, targetPorts[j])
			proxyID := fmt.Sprintf("%s-p%d", groupID, j+1)

			go func(listenAddr, targetAddr, proxyID string) {
				defer wg.Done()
				tcpProxy := tcp.NewProxy(proxyID, listenAddr, targetAddr)
				if err := tcpProxy.Start(ctx); err != nil {
					log.Printf("TCP代理[%s]错误: %v", proxyID, err)
				}
			}(listenAddr, targetAddr, proxyID)
		}

		log.Printf("已启动TCP端口组[%s]: %s:%s -> %s:%s, 共%d个端口对",
			groupID, tcpCfg.ListenIP, tcpCfg.ListenPorts, tcpCfg.TargetIP, tcpCfg.TargetPorts, len(listenPorts))
	}

	// 启动UDP转发服务
	for i, udpCfg := range cfg.UDP {
		if !udpCfg.Enabled {
			continue
		}

		groupID := fmt.Sprintf("udp-%d", i+1)

		listenPorts, err := parsePorts(udpCfg.ListenPorts)
		if err != nil {
			log.Printf("UDP配置[%s]监听端口解析错误: %v", groupID, err)
			continue
		}

		targetPorts, err := parsePorts(udpCfg.TargetPorts)
		if err != nil {
			log.Printf("UDP配置[%s]目标端口解析错误: %v", groupID, err)
			continue
		}

		// 检查端口数量是否匹配
		if len(listenPorts) != len(targetPorts) {
			log.Printf("UDP配置[%s]错误: 监听端口数量(%d)与目标端口数量(%d)不匹配",
				groupID, len(listenPorts), len(targetPorts))
			continue
		}

		// 为每对端口创建一个代理
		for j := 0; j < len(listenPorts); j++ {
			wg.Add(1)
			listenAddr := fmt.Sprintf("%s:%d", udpCfg.ListenIP, listenPorts[j])
			targetAddr := fmt.Sprintf("%s:%d", udpCfg.TargetIP, targetPorts[j])
			proxyID := fmt.Sprintf("%s-p%d", groupID, j+1)

			go func(listenAddr, targetAddr, proxyID string, bufferSize int, timeout time.Duration) {
				defer wg.Done()
				udpProxy := udp.NewProxy(
					proxyID,
					listenAddr,
					targetAddr,
					bufferSize,
					timeout,
				)
				if err := udpProxy.Start(ctx); err != nil {
					log.Printf("UDP代理[%s]错误: %v", proxyID, err)
				}
			}(listenAddr, targetAddr, proxyID, udpCfg.BufferSize, udpCfg.Timeout)
		}

		log.Printf("已启动UDP端口组[%s]: %s:%s -> %s:%s, 共%d个端口对",
			groupID, udpCfg.ListenIP, udpCfg.ListenPorts, udpCfg.TargetIP, udpCfg.TargetPorts, len(listenPorts))
	}

	// 优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("正在关闭服务...")
	cancel()
	wg.Wait()
	log.Println("服务已关闭")
}
