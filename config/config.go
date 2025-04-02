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
	Forwards []ForwardConfig `yaml:"forwards"`
}

// ForwardConfig 转发规则配置
type ForwardConfig struct {
	Name        string        `yaml:"name"`
	Enabled     bool          `yaml:"enabled"`
	Protocol    []string      `yaml:"protocol"`
	ListenIP    string        `yaml:"listen_ip"`
	ListenPorts []string      `yaml:"listen_ports"`
	TargetIP    string        `yaml:"target_ip"`
	TargetPorts []string      `yaml:"target_ports"`
	BufferSize  int           `yaml:"buffer_size"` // 仅用于UDP
	Timeout     time.Duration `yaml:"timeout"`     // 仅用于UDP
}

// LoadConfig 从指定文件路径加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 默认配置
	config := &Config{
		Forwards: []ForwardConfig{
			{
				Name:        "baka",
				Enabled:     true,
				Protocol:    []string{"tcp", "udp"},
				ListenIP:    "0.0.0.0",
				ListenPorts: []string{"8080-8085", "9000"},
				TargetIP:    "::1",
				TargetPorts: []string{"9080-9085", "8000"},
				BufferSize:  4096,
				Timeout:     3 * time.Minute,
			},
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
		Forwards: []ForwardConfig{
			{
				Name:        "baka",
				Enabled:     true,
				Protocol:    []string{"tcp", "udp"},
				ListenIP:    "0.0.0.0",
				ListenPorts: []string{"8080-8085", "9000"},
				TargetIP:    "::1",
				TargetPorts: []string{"9080-9085", "8000"},
				BufferSize:  4096,
				Timeout:     3 * time.Minute,
			},
			{
				Name:        "zako",
				Enabled:     false,
				Protocol:    []string{"tcp", "udp"},
				ListenIP:    "0.0.0.0",
				ListenPorts: []string{"8090", "8091", "8092"},
				TargetIP:    "::1",
				TargetPorts: []string{"9090", "9091", "9092"},
				BufferSize:  4096,
				Timeout:     3 * time.Minute,
			},
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
