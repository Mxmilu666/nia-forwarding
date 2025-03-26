package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

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
	if cfg.TCP.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tcpProxy := tcp.NewProxy(cfg.TCP.ListenAddr, cfg.TCP.TargetAddr)
			if err := tcpProxy.Start(ctx); err != nil {
				log.Printf("TCP代理错误: %v", err)
			}
		}()
	}

	// 启动UDP转发服务
	if cfg.UDP.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			udpProxy := udp.NewProxy(
				cfg.UDP.ListenAddr,
				cfg.UDP.TargetAddr,
				cfg.UDP.BufferSize,
				cfg.UDP.Timeout,
			)
			if err := udpProxy.Start(ctx); err != nil {
				log.Printf("UDP代理错误: %v", err)
			}
		}()
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
