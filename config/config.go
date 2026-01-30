package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// Config 存储应用配置
type Config struct {
	MySQL struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Database string `json:"database"`
	} `json:"mysql"`
	Server struct {
		Port string `json:"port"`
	} `json:"server"`
	Admin struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"admin"`
	Initialized bool `json:"initialized"`
}

var AppConfig Config

// ConfigPath 获取配置文件路径
func ConfigPath() string {
	execDir, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	execDir = filepath.Dir(execDir)
	return filepath.Join(execDir, "config.json")
}

// LoadConfig 加载配置文件
func LoadConfig() error {
	configPath := ConfigPath()
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，使用默认值
			setDefaultConfig()
			return nil
		}
		return err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&AppConfig)
	if err != nil {
		return err
	}

	return nil
}

// SaveConfig 保存配置文件
func SaveConfig() error {
	configPath := ConfigPath()
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(&AppConfig)
	if err != nil {
		return err
	}

	log.Println("配置已保存到:", configPath)
	return nil
}

// setDefaultConfig 设置默认配置
func setDefaultConfig() {
	AppConfig.MySQL.Host = "localhost"
	AppConfig.MySQL.Port = "3306"
	AppConfig.MySQL.User = "root"
	AppConfig.MySQL.Password = ""
	AppConfig.MySQL.Database = "goblog"
	AppConfig.Server.Port = "8081"
	AppConfig.Admin.Username = "admin"
	AppConfig.Admin.Password = "admin123"
	AppConfig.Initialized = false
}

// IsInitialized 检查是否已初始化
func IsInitialized() bool {
	return AppConfig.Initialized
}

// SetInitialized 设置为已初始化
func SetInitialized() {
	AppConfig.Initialized = true
}
