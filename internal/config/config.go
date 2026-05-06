package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

// Config 聚合所有配置项，通过 Viper 从 configs/config.yaml 加载。
// 使用 mapstructure 标签将 YAML 的蛇形命名映射到 Go 的驼峰字段。

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	MySQL     MySQLConfig     `mapstructure:"mysql"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
	Log       LogConfig       `mapstructure:"log"`
	Snowflake SnowflakeConfig `mapstructure:"snowflake"`
}

type ServerConfig struct {
	HTTPPort     int           `mapstructure:"httpPort"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
}

type MySQLConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns"`    // 最大连接数
	MaxIdleConns    int           `mapstructure:"maxIdleConns"`    // 最大空闲连接数
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"` // 连接最大存活时间
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"poolSize"`
}

type JWTConfig struct {
	AccessTokenExpire  time.Duration `mapstructure:"accessTokenExpire"`  // 短期 Token 有效期，默认 24h
	RefreshTokenExpire time.Duration `mapstructure:"refreshTokenExpire"` // 长期 Token 有效期，默认 7d
	PrivateKeyPath     string        `mapstructure:"privateKeyPath"`
	PublicKeyPath      string        `mapstructure:"publicKeyPath"`
}

type WebSocketConfig struct {
	Port           int           `mapstructure:"port"`
	MaxConnPerUser int           `mapstructure:"maxConnPerUser"` // 单用户最大连接数，超出踢旧连新
	MaxMessageSize int64         `mapstructure:"maxMessageSize"` // 单条消息最大字节数
	WriteWait      time.Duration `mapstructure:"writeWait"`      // 写超时
	PongWait       time.Duration `mapstructure:"pongWait"`       // 等待 Pong 的超时
	PingPeriod     time.Duration `mapstructure:"pingPeriod"`     // 发送 Ping 的间隔（须小于 PongWait）
}

type LogConfig struct {
	Level      string `mapstructure:"level"`      // debug / info / warn / error
	Format     string `mapstructure:"format"`     // json（生产） / text（开发）
	Output     string `mapstructure:"output"`     // stdout / file
	FilePath   string `mapstructure:"filePath"`
	MaxSize    int    `mapstructure:"maxSize"`    // 单文件最大 MB
	MaxBackups int    `mapstructure:"maxBackups"` // 保留旧文件数
	MaxAge     int    `mapstructure:"maxAge"`     // 保留天数
	Compress   bool   `mapstructure:"compress"`   // 旧文件 gzip 压缩
	ShortFile  bool   `mapstructure:"shortFile"`  // 日志中仅显示文件名，不含完整路径
	Color      bool   `mapstructure:"color"`      // 开发环境彩色输出
}

type SnowflakeConfig struct {
	NodeID int64 `mapstructure:"nodeID"` // 0~1023，K8S 多副本时每 Pod 不同
}

// Load 从 configs/config.yaml 加载配置，失败直接 panic（启动阶段无需优雅处理）。
func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("configs")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}
	return &cfg
}
