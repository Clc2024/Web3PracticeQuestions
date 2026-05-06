// config 包负责管理应用程序的配置信息
// 使用 mapstructure 标签支持从配置文件加载配置
package config

import (
	"log"

	"github.com/spf13/viper"
)

// Config 应用程序的总配置结构
// 包含服务器、数据库和JWT相关配置
//
// 字段:
//   Server: 服务器配置
//   Database: 数据库配置
//   JWT: JWT令牌配置
//
// mapstructure标签用于从配置文件加载对应字段
// 例如，配置文件中的server部分会映射到Server字段

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`   // 服务器配置
	Database DatabaseConfig `mapstructure:"database"` // 数据库配置
	JWT      JWTConfig      `mapstructure:"jwt"`      // JWT配置
}

// ServerConfig 服务器相关配置
//
// 字段:
//   Port: 服务器监听端口
//   Host: 服务器监听地址
//   Mode: 服务器运行模式（如debug、release等）

type ServerConfig struct {
	Port string `mapstructure:"port"` // 服务器监听端口
	Host string `mapstructure:"host"` // 服务器监听地址
	Mode string `mapstructure:"mode"` // 服务器运行模式
}

// DatabaseConfig 数据库相关配置
//
// 字段:
//   Host: 数据库主机地址
//   Port: 数据库端口
//   Username: 数据库用户名
//   Password: 数据库密码
//   DBName: 数据库名称

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`     // 数据库主机地址
	Port     int    `mapstructure:"port"`     // 数据库端口
	Username string `mapstructure:"username"` // 数据库用户名
	Password string `mapstructure:"password"` // 数据库密码
	DBName   string `mapstructure:"dbname"`   // 数据库名称
}

// JWTConfig JWT令牌相关配置
//
// 字段:
//   Secret: JWT签名密钥
//   Expire: JWT令牌过期时间

type JWTConfig struct {
	Secret string `mapstructure:"secret"` // JWT签名密钥
	Expire string `mapstructure:"expire"` // JWT令牌过期时间
}

func init() {
	//设置配置文件名称（不含扩展名）
	viper.SetConfigName("config")
	//设置配置文件类型
	viper.SetConfigType("yaml")
	//添加配置文件搜索路径
	viper.AddConfigPath("D:/GoProjects/learning/lesson-03/examples/project/config/")
	viper.AddConfigPath("$HOME/.app")
	//读取环境变量
	viper.AutomaticEnv()
	viper.SetEnvPrefix("APP")
	//设置默认值
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "debug")

	//读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在或读取失败时，使用默认值Q
		log.Printf("Warng: Error reading config file: %v", err)
		log.Println("Using default configuration values and environment variables")
	} else {
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}
}

func LoadConfig() (*Config, error) {
	var cfg Config
	// 使用 viper.Unmarshal 将配置数据解析到 config 结构体中
	// 如果解析失败，返回错误
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Printf("Error unmarshalling config: %v", err)
		return nil, err
	}
	return &cfg, nil
}

// Load 加载应用程序配置
// 注意：这是一个简化的配置加载实现
// 实际项目中应该使用Viper等配置管理库从配置文件加载
//
// 返回值:
//
//	*Config - 配置结构体指针，包含默认配置值
//
// 默认配置:
//
//	服务器: 端口8080，地址0.0.0.0，模式debug
//	数据库: 本地主机，端口3306，用户root，密码password，数据库名mydb
//	JWT: 密钥your-secret-key-change-in-production，过期时间24h
// func Load() *Config {
// 	// 简化配置加载，实际应该使用 Viper
// 	return &Config{
// 		Server: ServerConfig{
// 			Port: "8080",    // 服务器默认端口
// 			Host: "0.0.0.0", // 服务器默认地址，允许所有网络接口访问
// 			Mode: "debug",   // 服务器默认运行模式
// 		},
// 		Database: DatabaseConfig{
// 			Host:     "localhost", // 数据库默认主机
// 			Port:     3306,        // 数据库默认端口
// 			Username: "root",      // 数据库默认用户名
// 			Password: "password",  // 数据库默认密码
// 			DBName:   "mydb",      // 数据库默认名称
// 		},
// 		JWT: JWTConfig{
// 			Secret: "your-secret-key-change-in-production", // JWT默认密钥，生产环境应更改
// 			Expire: "24h",                                  // JWT默认过期时间
// 		},
// 	}
// }
