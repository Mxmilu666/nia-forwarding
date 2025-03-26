package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

// 默认配置文件名
const DefaultConfigFile = "config.yaml"

// Config 包含应用程序的所有配置
type Config struct {
	TCP TCPConfig `yaml:"tcp"`
	UDP UDPConfig `yaml:"udp"`
}

// TCPConfig TCP代理的配置
type TCPConfig struct {
	Enabled    bool   `yaml:"enabled"`
	ListenAddr string `yaml:"listen_addr"`
	TargetAddr string `yaml:"target_addr"`
}

// UDPConfig UDP代理的配置
type UDPConfig struct {
	Enabled    bool          `yaml:"enabled"`
	ListenAddr string        `yaml:"listen_addr"`
	TargetAddr string        `yaml:"target_addr"`
	BufferSize int           `yaml:"buffer_size"`
	Timeout    time.Duration `yaml:"timeout"`
}

// LoadConfig 从指定文件路径加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 默认配置
	config := &Config{
		TCP: TCPConfig{
			Enabled:    true,
			ListenAddr: "0.0.0.0:8080",
			TargetAddr: "[::1]:8081",
		},
		UDP: UDPConfig{
			Enabled:    true,
			ListenAddr: "0.0.0.0:8080",
			TargetAddr: "[::1]:8081",
			BufferSize: 4096,
			Timeout:    3 * time.Minute,
		},
	}

	// 确定配置文件路径
	var finalConfigPath string
	if configPath != "" {
		finalConfigPath = configPath
	} else {
		// 尝试从当前目录读取默认配置文件
		currentDir, err := os.Getwd()
		if err == nil {
			finalConfigPath = filepath.Join(currentDir, DefaultConfigFile)
		}
	}

	// 检查配置文件是否存在
	if finalConfigPath != "" {
		if _, err := os.Stat(finalConfigPath); os.IsNotExist(err) {
			// 配置文件不存在，生成一个
			fmt.Printf("未找到配置文件 %s，正在生成默认配置...\n", finalConfigPath)
			if err := SaveDefaultConfig(finalConfigPath); err != nil {
				return nil, fmt.Errorf("无法创建默认配置文件: %w", err)
			}
			fmt.Printf("已生成默认配置文件: %s，请编辑后重新运行程序\n", finalConfigPath)
			return nil, fmt.Errorf("请编辑配置文件后重新运行")
		} else if err == nil {
			// 配置文件存在，加载它
			data, err := ioutil.ReadFile(finalConfigPath)
			if err != nil {
				return nil, fmt.Errorf("无法读取配置文件: %w", err)
			}

			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("无法解析配置文件: %w", err)
			}

			fmt.Printf("已加载配置文件: %s\n", finalConfigPath)
		} else {
			// 其他错误
			return nil, fmt.Errorf("检查配置文件时出错: %w", err)
		}
	} else {
		fmt.Println("使用默认配置")
	}

	return config, nil
}

// SaveDefaultConfig 保存默认配置到文件
func SaveDefaultConfig(filePath string) error {
	config := &Config{
		TCP: TCPConfig{
			Enabled:    true,
			ListenAddr: "0.0.0.0:8080",
			TargetAddr: "[::1]:8081",
		},
		UDP: UDPConfig{
			Enabled:    true,
			ListenAddr: "0.0.0.0:8080",
			TargetAddr: "[::1]:8081",
			BufferSize: 4096,
			Timeout:    3 * time.Minute,
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("无法序列化配置: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("无法写入配置文件: %w", err)
	}

	return nil
}
