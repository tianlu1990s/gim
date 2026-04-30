# GIM 实现指南

本文档为 PLAN.md 中每个 TODO 项提供具体实现方案，包括代码结构、接口定义、Redis Key 设计、数据流、Go 代码大纲。

> **零基础？** 先看 [GETTING_STARTED.md](GETTING_STARTED.md) 搭建环境和理解核心概念，再回来读本文档。
>
> **每个代码块前的 `💡` 注释是为新手写的"为什么这样做"的解释。** 有经验的开发者可跳过。

---

## 目录

1. [项目骨架与基础设施](#1-项目骨架与基础设施)
2. [公共组件规划与封装](#2-公共组件规划与封装)
3. [认证模块实现](#3-认证模块实现)
4. [用户模块实现](#4-用户模块实现)
5. [好友模块实现](#5-好友模块实现)
6. [会话模块实现](#6-会话模块实现)
7. [消息模块实现](#7-消息模块实现)
8. [WebSocket 网关实现](#8-websocket-网关实现)
9. [在线状态管理](#9-在线状态管理)
10. [统一响应与错误码](#10-统一响应与错误码)
11. [中间件实现](#11-中间件实现)
12. [Makefile 与构建](#12-makefile-与构建)
13. [Docker 与开发环境](#13-docker-与开发环境)
14. [数据库迁移](#14-数据库迁移)
15. [第二阶段实现要点](#15-第二阶段实现要点)
16. [第三阶段实现要点](#16-第三阶段实现要点)
17. [第四阶段实现要点](#17-第四阶段实现要点)

---

## 1. 项目骨架与基础设施

### 1.1 Go Module 初始化

💡 **什么是 Go Module？** 类似 Node.js 的 `package.json`，Go 用 `go.mod` 文件管理项目依赖。`go get xxx` 会自动把依赖记录到 go.mod 里，别人拿到你的代码后 `go mod download` 就能安装所有依赖。

```bash
go mod init github.com/yourname/gim
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/mysql
go get github.com/redis/go-redis/v9
go get github.com/gorilla/websocket
go get github.com/spf13/viper
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
go get github.com/bwmarrin/snowflake
go get github.com/golang-migrate/migrate/v4
go get gopkg.in/natefinch/lumberjack.v2  # 日志轮转
```

### 1.2 main.go 结构

💡 **为什么 main.go 看起来这么长？** 这是第一阶段的单体入口，所有组件（HTTP、WS、MySQL、Redis）都在这里初始化。第二阶段拆成微服务后，每个服务各自有简短的 main.go。先把所有东西连起来跑通，再考虑拆分。

💡 **执行顺序很重要：** 配置 → 日志 → 数据库 → Redis → 业务层（Repository → Service → Handler）→ 路由 → 启动服务。后面的组件依赖前面的，不能反过来。

```go
// cmd/gim/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/yourname/gim/internal/config"
	"github.com/yourname/gim/internal/handler"
	"github.com/yourname/gim/internal/middleware"
	"github.com/yourname/gim/internal/repository"
	"github.com/yourname/gim/internal/service"
	"github.com/yourname/gim/internal/ws"
	"github.com/yourname/gim/pkg/jwt"
	"github.com/yourname/gim/pkg/slog"
	"github.com/yourname/gim/pkg/snowflake"
)

func main() {
	// 1. 加载配置
	cfg := config.Load()

	// 2. 初始化日志
	logger := slog.New(cfg.Log)

	// 3. 初始化 Snowflake 节点
	snowflake.Init(cfg.Snowflake.NodeID)

	// 4. 初始化 MySQL
	db := repository.InitMySQL(cfg.MySQL, logger)

	// 5. 初始化 Redis
	rdb := repository.InitRedis(cfg.Redis)

	// 6. 初始化 JWT Manager
	jwtMgr := jwt.NewJWTManager(cfg.JWT.PrivateKeyPath, cfg.JWT.PublicKeyPath,
		cfg.JWT.AccessTokenExpire, cfg.JWT.RefreshTokenExpire)

	// 7. 初始化各层
	repos := repository.NewRepositories(db, rdb)
	services := service.NewServices(repos, cfg, jwtMgr, rdb)
	handlers := handler.NewHandlers(services)

	// 8. 初始化 Gin 路由
	r := gin.New()
	r.Use(middleware.CORS())
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestLogger(logger))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 公开路由（无需鉴权）
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", handlers.Auth.Register)
		auth.POST("/login", handlers.Auth.Login)
		auth.POST("/refresh", handlers.Auth.Refresh)
	}

	// 鉴权路由
	api := r.Group("/api/v1")
	api.Use(middleware.JWTAuth(jwtMgr, rdb))
	{
		api.POST("/auth/logout", handlers.Auth.Logout)

		user := api.Group("/user")
		{
			user.GET("/profile", handlers.User.GetProfile)
			user.PUT("/profile", handlers.User.UpdateProfile)
			user.GET("/profile/:userId", handlers.User.GetOtherProfile)
			user.POST("/search", handlers.User.Search)
		}

		friend := api.Group("/friend")
		{
			friend.POST("/request", handlers.Friend.SendRequest)
			friend.GET("/request/incoming", handlers.Friend.ListRequests)
			friend.POST("/request/:id/accept", handlers.Friend.AcceptRequest)
			friend.POST("/request/:id/reject", handlers.Friend.RejectRequest)
			friend.DELETE("/:userId", handlers.Friend.Delete)
			friend.GET("/list", handlers.Friend.List)
			friend.PUT("/:userId/remark", handlers.Friend.SetRemark)
		}

		msg := api.Group("/msg")
		{
			msg.GET("/history", handlers.Message.History)
			msg.POST("/read", handlers.Message.MarkRead)
			msg.POST("/revoke", handlers.Message.Revoke)
		}

		conv := api.Group("/conversation")
		{
			conv.GET("/list", handlers.Conversation.List)
			conv.PUT("/:id/pin", handlers.Conversation.Pin)
			conv.DELETE("/:id", handlers.Conversation.Delete)
		}
	}

	// 9. 启动 WebSocket 服务（独立端口）
	hub := ws.NewHub(services.Message, services.Conversation, rdb)
	go hub.Run()

	wsServer := ws.NewServer(cfg.WebSocket, hub, jwtMgr)

	// 10. 并发启动 HTTP 和 WS
	go func() {
		logger.Info("WebSocket server starting", "port", cfg.Server.WSPort)
		if err := wsServer.Start(); err != nil {
			logger.Error("WS server error", "error", err)
		}
	}()

	logger.Info("HTTP server starting", "port", cfg.Server.HTTPPort)
	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.HTTPPort)); err != nil {
		logger.Fatal("HTTP server error", "error", err)
	}
}
```

### 1.3 配置结构

💡 **为什么不把 MySQL 密码直接写在代码里？** 配置和代码分离是基本原则：不同环境（开发/测试/生产）密码不同，写死在代码里换环境就要改代码。用配置文件（config.yaml）只需改配置，代码不用动。

💡 **mapstructure 标签是什么？** Viper 库用来把 YAML 里的键名映射到 Go 结构体的字段名。比如 YAML 里的 `httpPort` 会自动赋值给 `ServerConfig.HTTPPort`。

💡 **日志配置说明**：使用 Go 1.21+ 标准库 `log/slog`，支持：
- 日志等级：debug, info, warn, error
- 短文件名：只显示文件名不含完整路径
- 颜色支持：开发环境开启彩色输出
- 日志轮转：支持按大小和时间自动压缩轮转

```go
// internal/config/config.go
package config

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

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
	WSPort       int           `mapstructure:"wsPort"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
}

type MySQLConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"poolSize"`
}

type JWTConfig struct {
	AccessTokenExpire  time.Duration `mapstructure:"accessTokenExpire"`
	RefreshTokenExpire time.Duration `mapstructure:"refreshTokenExpire"`
	PrivateKeyPath     string        `mapstructure:"privateKeyPath"`
	PublicKeyPath      string        `mapstructure:"publicKeyPath"`
}

type WebSocketConfig struct {
	MaxConnPerUser int           `mapstructure:"maxConnPerUser"`
	MaxMessageSize int64         `mapstructure:"maxMessageSize"`
	WriteWait      time.Duration `mapstructure:"writeWait"`
	PongWait       time.Duration `mapstructure:"pongWait"`
	PingPeriod     time.Duration `mapstructure:"pingPeriod"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Format     string `mapstructure:"format"`      // json, text
	Output     string `mapstructure:"output"`      // stdout, file
	FilePath   string `mapstructure:"filePath"`    // 日志文件路径
	MaxSize    int    `mapstructure:"maxSize"`     // 单个文件最大MB
	MaxBackups int    `mapstructure:"maxBackups"`  // 保留旧文件最大个数
	MaxAge     int    `mapstructure:"maxAge"`      // 保留旧文件最大天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩
	ShortFile  bool   `mapstructure:"shortFile"`   // 短文件名
	Color      bool   `mapstructure:"color"`       // 颜色输出
}

type SnowflakeConfig struct {
	NodeID int64 `mapstructure:"nodeID"`
}

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
```

对应配置文件示例：

```yaml
# configs/config.yaml
server:
  httpPort: 8080
  wsPort: 8081
  readTimeout: 10s
  writeTimeout: 10s

mysql:
  host: localhost
  port: 3306
  user: gim
  password: gim_pass
  dbname: gim
  maxOpenConns: 100
  maxIdleConns: 10
  connMaxLifetime: 3600s

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  poolSize: 10

jwt:
  accessTokenExpire: 24h
  refreshTokenExpire: 168h
  privateKeyPath: configs/jwt/private.pem
  publicKeyPath: configs/jwt/public.pem

websocket:
  maxConnPerUser: 5
  maxMessageSize: 4096
  writeWait: 10s
  pongWait: 60s
  pingPeriod: 30s

log:
  level: debug          # 开发环境用debug，生产环境用info
  format: text          # text格式方便阅读，json格式方便日志聚合
  output: file          # stdout 控制台输出，file 文件输出
  filePath: logs/gim.log
  maxSize: 100          # 单个文件最大100MB
  maxBackups: 10        # 保留最近10个旧文件
  maxAge: 30            # 保留最近30天
  compress: true        # 压缩旧文件
  shortFile: true       # 短文件名（只显示 main.go，不含完整路径）
  color: true           # 彩色输出

snowflake:
  nodeID: 1
```

---

## 2. 公共组件规划与封装

### 2.1 为什么要规划公共组件

在项目初期就识别和封装公共组件，能带来以下好处：

| 好处 | 说明 |
|------|------|
| 代码复用 | 避免各模块重复造轮子，保持代码一致性 |
| 统一规范 | 所有模块使用相同的工具，降低学习成本 |
| 易于测试 | 公共组件独立测试，业务层依赖抽象 |
| 易于升级 | 修改一处即可全局生效 |

### 2.2 本项目的公共组件清单

| 组件 | 路径 | 用途 | 说明 |
|------|------|------|------|
| 日志模块 | `pkg/slog/` | 统一日志输出 | 基于标准库 log/slog，支持等级、短文件名、颜色、轮转 |
| JWT 工具 | `pkg/jwt/` | Token 生成与验证 | RS256 非对称加密，支持 JTI 黑名单 |
| Snowflake ID | `pkg/snowflake/` | 分布式唯一 ID | 时钟回拨防护，单机版即可用 |
| 统一响应 | `pkg/resp/` | HTTP 响应格式 | Success/Fail 统一封装 |
| 错误码 | `pkg/errcode/` | 错误码体系 | 分层错误码，支持详情追加 |
| Redis Key 前缀 | `pkg/rediskey/` | Redis Key 管理 | 统一 Key 命名，避免冲突 |
| 会话 ID 生成 | `pkg/convid/` | 会话 ID 生成 | 单聊按字典序排列，群聊固定格式 |

### 2.3 日志模块封装（基于 slog）

💡 **为什么用 slog 而不是 Zap？** Go 1.21+ 引入了结构化日志标准库 `log/slog`，提供了与 Zap 类似的结构化日志能力，但无需额外依赖。对于本项目，slog 足够用且更轻量。

#### 功能特性

- 日志等级：Debug, Info, Warn, Error
- 短文件名：`main.go:123` 而非 `/home/admin/gim/cmd/gim/main.go:123`
- 颜色输出：开发环境开启彩色，提升可读性
- 日志轮转：按大小/天数自动轮转，支持 gzip 压缩
- 格式切换：JSON 格式（生产环境）或 TEXT 格式（开发环境）

#### 代码实现

```go
// pkg/slog/slog.go
package slog

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourname/gim/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*slog.Logger
}

// New 创建新的日志实例
func New(cfg config.LogConfig) *Logger {
	var writer io.Writer

	// 日志轮转配置
	if cfg.Output == "file" {
		// 确保日志目录存在
		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create log directory: %v", err)
		}

		writer = &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,    // 单个文件最大MB
			MaxBackups: cfg.MaxBackups, // 保留旧文件最大个数
			MaxAge:     cfg.MaxAge,     // 保留旧文件最大天数
			Compress:   cfg.Compress,   // 是否压缩
		}
	} else {
		// 标准输出
		writer = os.Stdout
	}

	// 日志等级映射
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// 短文件名处理器（仅显示文件名不含路径）
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.ShortFile {
		// 添加短文件名支持
		opts.AddSource = true
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					src.File = filepath.Base(src.File)
				}
			}
			return a
		}
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		// TEXT 格式支持颜色
		if cfg.Color && cfg.Output != "file" {
			handler = NewColoredTextHandler(writer, opts)
		} else {
			handler = slog.NewTextHandler(writer, opts)
		}
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// ColoredTextHandler 支持颜色的文本处理器
type ColoredTextHandler struct {
	*slog.TextHandler
}

func NewColoredTextHandler(w io.Writer, opts *slog.HandlerOptions) *ColoredTextHandler {
	return &ColoredTextHandler{
		TextHandler: slog.NewTextHandler(w, opts),
	}
}

// 颜色定义
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
)

// Handle 实现带颜色的日志输出
func (h *ColoredTextHandler) Handle(ctx context.Context, r slog.Record) error {
	var color string
	switch r.Level {
	case slog.LevelDebug:
		color = colorPurple
	case slog.LevelInfo:
		color = colorBlue
	case slog.LevelWarn:
		color = colorYellow
	case slog.LevelError:
		color = colorRed
	default:
		color = colorReset
	}

	// 复用父类处理
	if err := h.TextHandler.Handle(ctx, r); err != nil {
		return err
	}

	// 注意：实际实现需要更复杂的处理来精确控制颜色输出
	// 这里简化处理，实际项目可能需要自定义格式化逻辑

	return nil
}
```

#### 使用示例

```go
// 在 main.go 中初始化
logger := slog.New(cfg.Log)

// 使用日志
logger.Debug("debug message", "key", "value")
logger.Info("server starting", "port", 8080)
logger.Warn("slow query detected", "duration", 500)
logger.Error("database connection failed", "error", err)

// 输出示例（TEXT 格式 + 短文件名 + 颜色）
// [DEBUG] main.go:42 debug message key=value
// [INFO ] main.go:45 server starting port=8080
// [WARN ] main.go:48 slow query detected duration=500
// [ERROR] main.go:51 database connection failed error=connection refused
```

### 2.4 Redis Key 管理

💡 **为什么需要统一管理 Redis Key？** 多个模块都访问 Redis，Key 命名不一致会导致重复或遗漏。统一管理可以：

1. 避免拼写错误
2. 便于全局搜索和修改
3. 支持 Key 前缀统一配置（比如环境隔离）

```go
// pkg/rediskey/rediskey.go
package rediskey

const (
	// Token 黑名单
	BlacklistToken = "blacklist:token:%s"

	// Refresh Token 存储
	Refresh = "refresh:%s:%s"

	// 消息 Seq
	SeqConv = "seq:conv:%s"

	// 消息去重
	DedupMsg = "dedup:msg:%s"

	// 用户已读位置
	ReadSeq = "readseq:%s:%s"

	// 消息缓存
	MsgCache = "msg:cache:%s:%d"

	// 在线状态
	Online = "online:%s"

	// 连接映射
	ConnMap = "conn_map:%s"

	// 限流
	RateLimit = "ratelimit:%s"
)

// BlacklistTokenKey 生成 Token 黑名单 Key
func BlacklistTokenKey(jti string) string {
	return fmt.Sprintf(BlacklistToken, jti)
}

// RefreshKey 生成 Refresh Token Key
func RefreshKey(userID, platform string) string {
	return fmt.Sprintf(Refresh, userID, platform)
}

// SeqConvKey 生成会话 Seq Key
func SeqConvKey(convID string) string {
	return fmt.Sprintf(SeqConv, convID)
}

// DedupMsgKey 生成消息去重 Key
func DedupMsgKey(clientMsgID string) string {
	return fmt.Sprintf(DedupMsg, clientMsgID)
}

// ReadSeqKey 生成用户已读位置 Key
func ReadSeqKey(userID, convID string) string {
	return fmt.Sprintf(ReadSeq, userID, convID)
}

// OnlineKey 生成在线状态 Key
func OnlineKey(userID string) string {
	return fmt.Sprintf(Online, userID)
}

// ConnMapKey 生成连接映射 Key
func ConnMapKey(userID string) string {
	return fmt.Sprintf(ConnMap, userID)
}
```

---

## 3. 认证模块实现

### 3.1 Redis Key 设计

```
# Token 黑名单（logout 时加入）
blacklist:token:{accessTokenJTI}    -> "1"    TTL = accessToken 剩余有效期

# Refresh Token 存储（用于刷新和吊销）
refresh:{userId}:{platform}         -> refreshToken    TTL = 7天
```

💡 **Key 生成使用统一的 rediskey 包**：

```go
import "github.com/yourname/gim/pkg/rediskey"

// Token 黑名单
rdb.Set(ctx, rediskey.BlacklistTokenKey(claims.ID), "1", ttl)

// Refresh Token 存储
rdb.Set(ctx, rediskey.RefreshKey(userID, platform), refreshToken, 7*24*time.Hour)
```

### 3.2 JWT 工具包

💡 **为什么用 RS256 而不是 HS256？** HS256 用同一个密钥签名和验证（对称加密），密钥泄露就完了。RS256 用私钥签名、公钥验证（非对称加密），私钥只存在服务端，公钥可以分发给其他服务验证 Token。第二阶段微服务拆分后，各服务只需公钥即可验证，无需私钥。

💡 **什么是 JTI？** JWT Token 的唯一 ID，用于 Token 黑名单。用户退出登录后，把 JTI 存入 Redis 黑名单，即使 Token 还没过期，鉴权时检查黑名单也会拒绝。

```go
// pkg/jwt/jwt.go
package jwt

import (
    "crypto/rsa"
    "os"
    "time"

    jwtv5 "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID   string `json:"userId"`
    Platform string `json:"platform"`
    jwtv5.RegisteredClaims
}

type JWTManager struct {
    privateKey    *rsa.PrivateKey
    publicKey     *rsa.PublicKey
    accessExpire  time.Duration
    refreshExpire time.Duration
}

func NewJWTManager(privateKeyPath, publicKeyPath string, accessExp, refreshExp time.Duration) *JWTManager {
    privBytes, _ := os.ReadFile(privateKeyPath)
    pubBytes, _ := os.ReadFile(publicKeyPath)
    privKey, _ := jwtv5.ParseRSAPrivateKeyFromPEM(privBytes)
    pubKey, _ := jwtv5.ParseRSAPublicKeyFromPEM(pubBytes)
    return &JWTManager{privKey, pubKey, accessExp, refreshExp}
}

func (m *JWTManager) GenerateAccessToken(userID, platform string) (string, int64, error) {
    now := time.Now()
    expireAt := now.Add(m.accessExpire).Unix()
    claims := &Claims{
        UserID:   userID,
        Platform: platform,
        RegisteredClaims: jwtv5.RegisteredClaims{
            ID:        snowflake.Generate().String(), // JTI 用于黑名单
            IssuedAt:  jwtv5.NewNumericDate(now),
            ExpiresAt: jwtv5.NewNumericDate(now.Add(m.accessExpire)),
        },
    }
    token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims)
    tokenStr, err := token.SignedString(m.privateKey)
    return tokenStr, expireAt, err
}

func (m *JWTManager) GenerateRefreshToken(userID, platform string) (string, int64, error) {
    // 同 GenerateAccessToken，但使用 refreshExpire
    ...
}

func (m *JWTManager) ParseToken(tokenStr string) (*Claims, error) {
    token, err := jwtv5.ParseWithClaims(tokenStr, &Claims{}, func(t *jwtv5.Token) (interface{}, error) {
        return m.publicKey, nil
    })
    if err != nil {
        return nil, err
    }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }
    return claims, nil
}
```

### 3.3 认证 Service

💡 **bcrypt 是什么？为什么不用 MD5/SHA256 存密码？** MD5/SHA256 是"哈希"不是"加密"——相同的密码永远产生相同的结果，攻击者可以用"彩虹表"反查。bcrypt 有两个关键特性：(1) 每次哈希自动加盐（加随机字符串），同样密码每次结果不同；(2) 可调整计算成本（cost 参数），让暴力破解变慢。bcrypt 是存密码的行业标准。

💡 **为什么要"先查是否存在再创建"而不是直接 INSERT 等数据库报错？** 两种方式都可以，但先查可以返回更友好的错误消息（"用户已存在"），而数据库报错是底层技术信息，不适合直接暴露给用户。

```go
// internal/service/auth.go
package service

type AuthService interface {
    Register(ctx context.Context, req *RegisterReq) (*User, error)
    Login(ctx context.Context, req *LoginReq) (*TokenPair, error)
    Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
    Logout(ctx context.Context, userID, platform string, accessToken string) error
}

type authService struct {
    userRepo    repository.UserRepo
    jwtMgr      *jwt.JWTManager
    rdb         *redis.Client
    cfg         *config.Config
}

func (s *authService) Register(ctx context.Context, req *RegisterReq) (*User, error) {
    // 1. 参数校验（userId 格式、密码强度、手机号/邮箱格式）
    if err := validateRegisterReq(req); err != nil {
        return nil, errcode.ErrInvalidParam.WithDetail(err.Error())
    }
    // 2. 检查 userId 是否已存在
    exists, _ := s.userRepo.ExistsByID(ctx, req.UserID)
    if exists {
        return nil, errcode.ErrUserAlreadyExists
    }
    // 3. 密码 bcrypt 哈希
    hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    // 4. 写入数据库
    user := &model.User{
        UserID:   req.UserID,
        Nickname: req.Nickname,
        Password: string(hashedPwd),
        Phone:    req.Phone,
        Email:    req.Email,
    }
    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, errcode.ErrInternal.WithDetail(err.Error())
    }
    return user, nil
}

func (s *authService) Login(ctx context.Context, req *LoginReq) (*TokenPair, error) {
    // 1. 查询用户
    user, err := s.userRepo.GetByID(ctx, req.UserID)
    if err != nil {
        return nil, errcode.ErrUserOrPasswordWrong
    }
    // 2. 校验密码
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        return nil, errcode.ErrUserOrPasswordWrong
    }
    // 3. 检查状态
    if user.Status != 1 {
        return nil, errcode.ErrUserDisabled
    }
    // 4. 生成 Token
    accessToken, accessExp, _ := s.jwtMgr.GenerateAccessToken(user.UserID, req.Platform)
    refreshToken, refreshExp, _ := s.jwtMgr.GenerateRefreshToken(user.UserID, req.Platform)
    // 5. 存储 refreshToken 到 Redis（用于刷新和吊销）
    s.rdb.Set(ctx, fmt.Sprintf("refresh:%s:%s", user.UserID, req.Platform),
        refreshToken, s.cfg.JWT.RefreshTokenExpire)
    // 6. 踢掉同平台旧连接（通过 WS Hub）
    // ...
    return &TokenPair{
        AccessToken:      accessToken,
        RefreshToken:     refreshToken,
        AccessExpireAt:   accessExp,
        RefreshExpireAt:  refreshExp,
        UserID:           user.UserID,
    }, nil
}

func (s *authService) Logout(ctx context.Context, userID, platform, accessToken string) error {
    // 1. 解析 accessToken 获取 JTI 和剩余有效期
    claims, _ := s.jwtMgr.ParseToken(accessToken)
    ttl := time.Until(claims.ExpiresAt.Time)
    // 2. 加入黑名单
    if ttl > 0 {
        s.rdb.Set(ctx, fmt.Sprintf("blacklist:token:%s", claims.ID), "1", ttl)
    }
    // 3. 删除 refresh token
    s.rdb.Del(ctx, fmt.Sprintf("refresh:%s:%s", userID, platform))
    // 4. 清除在线状态
    s.rdb.Del(ctx, fmt.Sprintf("online:%s", userID))
    s.rdb.Del(ctx, fmt.Sprintf("conn_map:%s", userID))
    return nil
}
```

### 3.4 认证 Handler

```go
// internal/handler/auth.go
package handler

type AuthHandler struct {
    svc service.AuthService
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    user, err := h.svc.Register(c.Request.Context(), &req)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, user.ToVO())
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    token, err := h.svc.Login(c.Request.Context(), &req)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, token)
}
```

---

## 4. 用户模块实现

### 3.1 User Repository

```go
// internal/repository/user.go
package repository

type UserRepo interface {
    Create(ctx context.Context, user *model.User) error
    GetByID(ctx context.Context, userID string) (*model.User, error)
    ExistsByID(ctx context.Context, userID string) (bool, error)
    Update(ctx context.Context, userID string, updates map[string]interface{}) error
    Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error)
    ExistsByPhone(ctx context.Context, phone string, excludeUserID string) (bool, error)
    ExistsByEmail(ctx context.Context, email string, excludeUserID string) (bool, error)
}

type userRepo struct {
    db *gorm.DB
}

func (r *userRepo) GetByID(ctx context.Context, userID string) (*model.User, error) {
    var user model.User
    err := r.db.WithContext(ctx).Where("user_id = ? AND status = 1", userID).First(&user).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, errcode.ErrUserNotFound
    }
    return &user, err
}

func (r *userRepo) Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error) {
    var users []*model.User
    var total int64
    query := r.db.WithContext(ctx).Model(&model.User{}).Where("status = 1")
    // 纯数字 -> 手机号精确匹配，否则 -> 昵称模糊匹配
    if isDigit(keyword) {
        query = query.Where("phone = ?", keyword)
    } else {
        query = query.Where("nickname LIKE ?", "%"+keyword+"%")
    }
    query.Count(&total)
    err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error
    return users, total, err
}
```

### 4.2 User Service

```go
// internal/service/user.go
package service

type UserService interface {
    GetProfile(ctx context.Context, userID string) (*UserVO, error)
    UpdateProfile(ctx context.Context, userID string, req *UpdateProfileReq) (*UserVO, error)
    GetOtherProfile(ctx context.Context, currentUserID, targetUserID string) (*OtherUserVO, error)
    Search(ctx context.Context, userID string, req *SearchReq) (*PageResult[*SearchUserVO], error)
}

func (s *userService) GetOtherProfile(ctx context.Context, currentUserID, targetUserID string) (*OtherUserVO, error) {
    user, err := s.userRepo.GetByID(ctx, targetUserID)
    if err != nil {
        return nil, err
    }
    vo := &OtherUserVO{
        UserID:     user.UserID,
        Nickname:   user.Nickname,
        AvatarURL:  user.AvatarURL,
    }
    // 检查好友关系
    isFriend, _ := s.friendRepo.IsFriend(ctx, currentUserID, targetUserID)
    if isFriend {
        friend, _ := s.friendRepo.GetFriend(ctx, currentUserID, targetUserID)
        vo.IsFriend = true
        vo.Remark = friend.Remark
    }
    return vo, nil
}
```

---

## 5. 好友模块实现

### 5.1 Friend Repository

```go
// internal/repository/friend.go
package repository

type FriendRepo interface {
    Create(ctx context.Context, ownerID, friendID, remark string) error
    Delete(ctx context.Context, ownerID, friendID string) error
    IsFriend(ctx context.Context, ownerID, friendID string) (bool, error)
    GetFriend(ctx context.Context, ownerID, friendID string) (*model.Friend, error)
    List(ctx context.Context, ownerID string, page, pageSize int) ([]*FriendVO, int64, error)
    SetRemark(ctx context.Context, ownerID, friendID, remark string) error
}

type FriendRequestRepo interface {
    Create(ctx context.Context, fromID, toID, message string) (int64, error)
    GetByID(ctx context.Context, id int64) (*model.FriendRequest, error)
    ListIncoming(ctx context.Context, toID string, page, pageSize int) ([]*FriendRequestVO, int64, error)
    UpdateStatus(ctx context.Context, id int64, status int) error
    HasPendingRequest(ctx context.Context, fromID, toID string) (bool, error)
}
```

### 5.2 好友申请 — 同意流程（事务）

💡 **什么是事务？** 事务是数据库操作的"打包执行"：要么全部成功，要么全部回滚。好友同意涉及 3 张表写入（申请状态 + 双向好友 + 双方会话），如果第 2 步写入失败但不回滚第 1 步，就会数据不一致（申请已同意但好友关系没建）。事务保证"要么全有，要么全无"。

💡 **为什么好友关系要"双向"写入？** Alice 加 Bob 为好友，意味着 Alice 的好友列表有 Bob，Bob 的好友列表也有 Alice。这是两条独立的记录，不是一条。这样每个人查自己的好友列表时只需查 `WHERE owner_id = 我`，简单高效。

```go
func (s *friendService) AcceptRequest(ctx context.Context, userID string, requestID int64) error {
    // 1. 查询申请，验证 toUserID 是当前用户
    req, err := s.requestRepo.GetByID(ctx, requestID)
    if err != nil {
        return errcode.ErrResourceNotFound
    }
    if req.ToUserID != userID {
        return errcode.ErrForbidden
    }
    if req.Status != 0 {
        return errcode.ErrAlreadyProcessed
    }

    // 2. 事务：更新申请状态 + 双向好友关系 + 双方会话
    err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
        // 更新申请状态
        if err := s.requestRepo.UpdateStatusTx(ctx, tx, requestID, 1); err != nil {
            return err
        }
        // 双向好友关系
        if err := s.friendRepo.CreateTx(ctx, tx, req.FromUserID, req.ToUserID, ""); err != nil {
            return err
        }
        if err := s.friendRepo.CreateTx(ctx, tx, req.ToUserID, req.FromUserID, ""); err != nil {
            return err
        }
        // 创建双方会话
        convID := GenSingleConvID(req.FromUserID, req.ToUserID)
        if err := s.convRepo.CreateIfNotExistTx(ctx, tx, req.FromUserID, convID, 1, req.ToUserID); err != nil {
            return err
        }
        if err := s.convRepo.CreateIfNotExistTx(ctx, tx, req.ToUserID, convID, 1, req.FromUserID); err != nil {
            return err
        }
        return nil
    })

    // 3. WS 推送通知申请方
    if err == nil {
        s.hub.PushToUser(req.FromUserID, &ws.Message{
            Type: 107,
            Data: map[string]interface{}{
                "type":   "friend_accepted",
                "userId": userID,
            },
        })
    }
    return err
}
```

### 5.3 会话 ID 生成

```go
// pkg/convid/convid.go
package convid

import "sort"

// GenSingleConvID 生成单聊会话ID，两个 userId 按字典序排列
func GenSingleConvID(uid1, uid2 string) string {
    ids := []string{uid1, uid2}
    sort.Strings(ids)
    return fmt.Sprintf("single_%s_%s", ids[0], ids[1])
}
```

---

## 6. 会话模块实现

### 6.1 Conversation Repository

```go
// internal/repository/conversation.go
package repository

type ConversationRepo interface {
    CreateIfNotExistTx(ctx context.Context, tx *gorm.DB, ownerID, convID string, convType int, targetID string) error
    List(ctx context.Context, ownerID string, page, pageSize int) ([]*ConversationVO, int64, error)
    UpdatePin(ctx context.Context, ownerID, convID string, isPinned bool) error
    Delete(ctx context.Context, ownerID, convID string) error
    UpdateMaxSeq(ctx context.Context, convID string, seq int64) error
    GetByID(ctx context.Context, ownerID, convID string) (*model.Conversation, error)
}
```

### 6.2 会话列表（含未读计数）

```go
func (s *convService) List(ctx context.Context, userID string, page, pageSize int) (*PageResult[*ConversationVO], error) {
    convs, total, err := s.convRepo.List(ctx, userID, page, pageSize)
    if err != nil {
        return nil, err
    }

    // 批量获取 readSeq（从 Redis 或数据库）
    for _, conv := range convs {
        readSeq, _ := s.msgRepo.GetUserReadSeq(ctx, userID, conv.ConversationID)
        conv.UnreadCount = conv.MaxSeq - readSeq
        if conv.UnreadCount < 0 {
            conv.UnreadCount = 0
        }
        conv.ReadSeq = readSeq

        // 获取最后一条消息
        lastMsg, _ := s.msgRepo.GetLastMsg(ctx, conv.ConversationID)
        conv.LastMsg = lastMsg
    }

    // 排序：置顶优先，其次按最后消息时间倒序
    sort.Slice(convs, func(i, j int) bool {
        if convs[i].IsPinned != convs[j].IsPinned {
            return convs[i].IsPinned
        }
        return convs[i].UpdatedAt.After(convs[j].UpdatedAt)
    })

    return &PageResult[*ConversationVO]{
        List:     convs,
        Total:    total,
        Page:     page,
        PageSize: pageSize,
    }, nil
}
```

---

## 7. 消息模块实现

### 7.1 Redis Key 设计

```
# 消息 Seq（会话维度递增）
seq:conv:{conversationId}          -> int64 (当前 maxSeq)

# 消息去重（clientMsgId）
dedup:msg:{clientMsgId}            -> "1"    TTL = 5分钟

# 用户已读位置
readseq:{userId}:{conversationId}  -> int64

# 消息内容缓存（可选，减少数据库查询）
msg:cache:{conversationId}:{seq}   -> JSON   TTL = 10分钟
```

### 7.2 Message Repository

```go
// internal/repository/message.go
package repository

type MessageRepo interface {
    Create(ctx context.Context, msg *model.Message) error
    GetBySeqRange(ctx context.Context, conversationID string, startSeq, endSeq int64, limit int) ([]*model.Message, error)
    GetByClientMsgID(ctx context.Context, clientMsgID string) (*model.Message, error)
    Revoke(ctx context.Context, conversationID, clientMsgID string) error
    GetLastMsg(ctx context.Context, conversationID string) (*model.Message, error)
    GetUserReadSeq(ctx context.Context, userID, conversationID string) (int64, error)
    SetUserReadSeq(ctx context.Context, userID, conversationID string, seq int64) error
    GetMaxSeq(ctx context.Context, conversationID string) (int64, error)
    IncrSeq(ctx context.Context, conversationID string) (int64, error)
}
```

### 7.3 消息发送核心流程

💡 **这是整个 IM 系统最核心的流程，务必理解每一步为什么这样设计：**

1. **去重（SETNX）**：网络抖动可能导致客户端重复发送同一条消息。SETNX（Set if Not eXists）保证同一个 clientMsgId 只处理一次
2. **好友校验**：单聊必须先成为好友，否则不能发消息。这是业务规则，不是技术需求
3. **Redis INCR 分配 Seq**：为什么不直接用 MySQL 自增？因为 MySQL 自增是表级锁，高并发下成为瓶颈；Redis INCR 是原子操作，每秒百万级
4. **先持久化再推送**：消息必须先写入数据库，再通知接收方。如果先推送再写库，写库失败时接收方看到了不存在的消息
5. **推送通知 vs 推送消息全文**：第一阶段推送消息全文（简化），第二阶段改为推送通知（只告诉有新消息，客户端主动拉取）

```go
func (s *msgService) SendMessage(ctx context.Context, senderID string, req *SendMsgReq) (*SendMsgResp, error) {
    convID := req.ConversationID

    // 1. 去重检查
    dedupKey := fmt.Sprintf("dedup:msg:%s", req.ClientMsgID)
    ok, _ := s.rdb.SetNX(ctx, dedupKey, "1", 5*time.Minute).Result()
    if !ok {
        // 消息已存在，返回已有消息的 seq
        existing, _ := s.msgRepo.GetByClientMsgID(ctx, req.ClientMsgID)
        if existing != nil {
            return &SendMsgResp{Seq: existing.Seq, ServerMsgID: existing.ServerMsgID}, nil
        }
    }

    // 2. 好友校验（单聊）
    if strings.HasPrefix(convID, "single_") {
        targetID := extractTargetID(convID, senderID)
        isFriend, _ := s.friendRepo.IsFriend(ctx, senderID, targetID)
        if !isFriend {
            return nil, errcode.ErrNotFriend
        }
    }

    // 3. 分配 Seq（Redis INCR）
    seq, err := s.msgRepo.IncrSeq(ctx, convID)
    if err != nil {
        return nil, errcode.ErrInternal.WithDetail("failed to alloc seq")
    }

    // 4. 生成 serverMsgID
    serverMsgID := snowflake.Generate().String()

    // 5. 持久化消息
    msg := &model.Message{
        ConversationID: convID,
        Seq:            seq,
        SenderID:       senderID,
        MsgType:        req.ContentType,
        Content:        req.Content,
        ClientMsgID:    req.ClientMsgID,
        ServerMsgID:    serverMsgID,
    }
    if err := s.msgRepo.Create(ctx, msg); err != nil {
        return nil, errcode.ErrInternal.WithDetail(err.Error())
    }

    // 6. 更新会话 maxSeq
    s.convRepo.UpdateMaxSeq(ctx, convID, seq)

    // 7. 推送通知给接收方
    now := time.Now().UnixMilli()
    pushMsg := &ws.WSMessage{
        Type: 101,
        Data: map[string]interface{}{
            "conversationId": convID,
            "seq":            seq,
            "senderId":       senderID,
            "contentType":    req.ContentType,
            "content":        req.Content,
            "serverMsgId":    serverMsgID,
            "clientMsgId":    req.ClientMsgID,
            "sendTime":       now,
        },
    }
    // 推送给会话中除了发送者之外的所有用户
    targetIDs := s.getConversationMembers(ctx, convID, senderID)
    for _, targetID := range targetIDs {
        s.hub.PushToUser(targetID, pushMsg)
    }

    return &SendMsgResp{
        Seq:         seq,
        ServerMsgID: serverMsgID,
        SendTime:    now,
    }, nil
}
```

### 7.4 消息拉取

```go
func (s *msgService) History(ctx context.Context, userID string, req *HistoryReq) (*HistoryResp, error) {
    convID := req.ConversationID

    // 获取会话 maxSeq
    maxSeq, _ := s.msgRepo.GetMaxSeq(ctx, convID)
    if maxSeq == 0 {
        return &HistoryResp{List: []*model.Message{}, HasMore: false}, nil
    }

    // 确定 startSeq
    startSeq := req.StartSeq
    if startSeq == 0 {
        startSeq = maxSeq
    }

    // 拉取消息：seq <= startSeq，倒序，limit count
    msgs, err := s.msgRepo.GetBySeqRange(ctx, convID, 0, startSeq, req.Count+1)
    if err != nil {
        return nil, err
    }

    // 判断是否还有更多
    hasMore := len(msgs) > req.Count
    if hasMore {
        msgs = msgs[:req.Count]
    }

    // 获取 minSeq
    minSeq, _ := s.msgRepo.GetMinSeq(ctx, convID)

    return &HistoryResp{
        List:    msgs,
        HasMore: hasMore,
        MinSeq:  minSeq,
        MaxSeq:  maxSeq,
    }, nil
}
```

### 7.5 已读回执

```go
func (s *msgService) MarkRead(ctx context.Context, userID string, req *MarkReadReq) error {
    convID := req.ConversationID

    // 校验 readSeq <= maxSeq
    maxSeq, _ := s.msgRepo.GetMaxSeq(ctx, convID)
    if req.ReadSeq > maxSeq {
        return errcode.ErrInvalidParam.WithDetail("readSeq exceeds maxSeq")
    }

    // 更新 readSeq（Redis + MySQL 双写）
    s.msgRepo.SetUserReadSeq(ctx, userID, convID, req.ReadSeq)
    // MySQL 更新
    s.msgRepo.UpdateUserReadSeqDB(ctx, userID, convID, req.ReadSeq)

    // 通知对方已读
    targetID := extractTargetID(convID, userID)
    s.hub.PushToUser(targetID, &ws.WSMessage{
        Type: 102,
        Data: map[string]interface{}{
            "conversationId": convID,
            "readUserId":     userID,
            "readSeq":        req.ReadSeq,
        },
    })
    return nil
}
```

---

## 8. WebSocket 网关实现

### 8.1 Hub（连接中心）

💡 **Hub 的工作原理类比电话总机**：所有 WebSocket 连接都注册到 Hub，Hub 维护一张"谁在线"的表。当要给某人推送消息时，查 Hub 找到这个人的连接，通过连接发送。没有 Hub，每条消息要遍历所有连接去找目标用户，效率极低。

💡 **为什么用 channel（register/unregister/push）而不是直接操作 map？** Go 的 map 不是线程安全的，多个 goroutine 同时读写 map 会 panic。用 channel 可以保证同一时刻只有一个 goroutine 操作 map（在 Run() 的 for-select 循环中），这是 Go 并发编程的惯用模式——"不要通过共享内存来通信，而要通过通信来共享内存"。

```go
// internal/ws/hub.go
package ws

type Hub struct {
    // 用户ID -> 该用户的所有连接
    clients    map[string]map[*Client]struct{}
    register   chan *Client
    unregister chan *Client
    push       chan *PushMessage

    msgSvc  service.MessageService
    convSvc service.ConversationService
    rdb     *redis.Client
    mu      sync.RWMutex
}

type PushMessage struct {
    UserID  string
    Message *WSMessage
}

func NewHub(msgSvc service.MessageService, convSvc service.ConversationService, rdb *redis.Client) *Hub {
    return &Hub{
        clients:   make(map[string]map[*Client]struct{}),
        register:  make(chan *Client, 256),
        unregister: make(chan *Client, 256),
        push:      make(chan *PushMessage, 1024),
        msgSvc:    msgSvc,
        convSvc:   convSvc,
        rdb:       rdb,
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            if h.clients[client.userID] == nil {
                h.clients[client.userID] = make(map[*Client]struct{})
            }
            h.clients[client.userID][client] = struct{}{}
            h.mu.Unlock()
            h.setOnline(client.userID, client.connID, client.platform)

        case client := <-h.unregister:
            h.mu.Lock()
            if conns, ok := h.clients[client.userID]; ok {
                delete(conns, client)
                if len(conns) == 0 {
                    delete(h.clients, client.userID)
                }
            }
            h.mu.Unlock()
            h.setOffline(client.userID, client.connID, client.platform)

        case msg := <-h.push:
            h.mu.RLock()
            conns := h.clients[msg.UserID]
            h.mu.RUnlock()
            for client := range conns {
                client.Send(msg.Message)
            }
        }
    }
}

func (h *Hub) PushToUser(userID string, msg *WSMessage) {
    h.push <- &PushMessage{UserID: userID, Message: msg}
}
```

### 8.2 Client（连接管理）

```go
// internal/ws/client.go
package ws

type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte
    userID   string
    platform string
    connID   string
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, platform string) *Client {
    return &Client{
        hub:      hub,
        conn:     conn,
        send:     make(chan []byte, 256),
        userID:   userID,
        platform: platform,
        connID:   snowflake.Generate().String(),
    }
}

// ReadPump 从客户端读取消息
func (c *Client) ReadPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(c.hub.maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(c.hub.pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(c.hub.pongWait))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
        c.handleMessage(message)
    }
}

// WritePump 向客户端发送消息
func (c *Client) WritePump() {
    ticker := time.NewTicker(c.hub.pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(c.hub.writeWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(c.hub.writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

func (c *Client) Send(msg *WSMessage) {
    data, _ := json.Marshal(msg)
    select {
    case c.send <- data:
    default:
        // 缓冲区满，关闭连接
        close(c.send)
    }
}
```

### 8.3 消息处理分发

```go
// internal/ws/client.go — handleMessage 方法
func (c *Client) handleMessage(raw []byte) {
    var msg WSMessage
    if err := json.Unmarshal(raw, &msg); err != nil {
        c.Send(&WSMessage{Type: -1, ReqID: "", Data: map[string]interface{}{"error": "invalid format"}})
        return
    }

    ctx := context.Background()

    switch msg.Type {
    case 1: // 发送聊天消息
        var req SendMsgData
        json.Unmarshal(toJSON(msg.Data), &req)
        resp, err := c.hub.msgSvc.SendMessage(ctx, c.userID, &req)
        if err != nil {
            c.Send(&WSMessage{Type: -1, ReqID: msg.ReqID, Data: map[string]interface{}{"error": err.Error()}})
            return
        }
        c.Send(&WSMessage{Type: 101, ReqID: msg.ReqID, Data: resp})

    case 2: // 已读回执
        var req MarkReadData
        json.Unmarshal(toJSON(msg.Data), &req)
        c.hub.msgSvc.MarkRead(ctx, c.userID, &req)

    case 3: // 心跳
        c.conn.SetReadDeadline(time.Now().Add(c.hub.pongWait))
        c.hub.refreshOnline(c.userID, c.connID)
        c.Send(&WSMessage{Type: 103, ReqID: msg.ReqID, Data: map[string]interface{}{}})

    case 4: // 拉取消息
        var req HistoryData
        json.Unmarshal(toJSON(msg.Data), &req)
        resp, _ := c.hub.msgSvc.History(ctx, c.userID, &req)
        c.Send(&WSMessage{Type: 104, ReqID: msg.ReqID, Data: resp})

    case 5: // 输入状态
        var req TypingData
        json.Unmarshal(toJSON(msg.Data), &req)
        convID := req.ConversationID
        targetID := extractTargetID(convID, c.userID)
        c.hub.PushToUser(targetID, &WSMessage{
            Type: 105,
            Data: map[string]interface{}{
                "conversationId": convID,
                "userId":         c.userID,
                "isTyping":       req.IsTyping,
            },
        })
    }
}
```

### 8.4 WS Server

```go
// internal/ws/server.go
package ws

type Server struct {
    cfg   *config.WebSocketConfig
    hub   *Hub
    jwtMgr *jwt.JWTManager
}

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin:     func(r *http.Request) bool { return true }, // 生产环境需限制
}

func (s *Server) Start() error {
    http.HandleFunc("/ws", s.handleWebSocket)
    return http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Port), nil)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // 1. 从查询参数获取 token
    token := r.URL.Query().Get("token")
    platform := r.URL.Query().Get("platform")
    if platform == "" {
        platform = "web"
    }

    // 2. 验证 Token
    claims, err := s.jwtMgr.ParseToken(token)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // 3. 检查黑名单
    ctx := r.Context()
    exists, _ := s.hub.rdb.Exists(ctx, fmt.Sprintf("blacklist:token:%s", claims.ID)).Result()
    if exists > 0 {
        http.Error(w, "Token revoked", http.StatusUnauthorized)
        return
    }

    // 4. 检查连接数限制
    s.hub.mu.RLock()
    conns := s.hub.clients[claims.UserID]
    s.hub.mu.RUnlock()
    if len(conns) >= s.cfg.MaxConnPerUser {
        http.Error(w, "Too many connections", http.StatusTooManyRequests)
        return
    }

    // 5. Upgrade
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }

    // 6. 创建 Client 并注册
    client := NewClient(s.hub, conn, claims.UserID, platform)
    s.hub.register <- client

    // 7. 上线后拉取离线消息
    go client.pullOfflineMessages()

    // 8. 启动读写协程
    go client.WritePump()
    go client.ReadPump()
}

func (c *Client) pullOfflineMessages() {
    // 拉取所有会话的离线消息
    convs, _ := c.hub.convSvc.ListConversations(c.userID)
    for _, conv := range convs {
        readSeq, _ := c.hub.msgRepo.GetUserReadSeq(context.Background(), c.userID, conv.ConversationID)
        if conv.MaxSeq > readSeq {
            msgs, _ := c.hub.msgSvc.History(context.Background(), c.userID, &service.HistoryReq{
                ConversationID: conv.ConversationID,
                StartSeq:       conv.MaxSeq,
                Count:          50,
            })
            for _, msg := range msgs.List {
                c.Send(&WSMessage{Type: 101, Data: msg})
            }
        }
    }
}
```

---

## 9. 在线状态管理

### 9.1 Redis 数据结构

```
# 用户在线状态（Hash，存平台和连接信息）
Key:   online:{userId}
Value: {
    "platform": "web",
    "connCount": 2,
    "lastActive": 1714000050
}
TTL:   60s（心跳续期）

# 用户连接映射（Set，存 connID）
Key:   conn_map:{userId}
Value: {"conn-001", "conn-002"}
TTL:   60s（心跳续期）
```

### 9.2 上线/下线/续期

```go
func (h *Hub) setOnline(userID, connID, platform string) {
    ctx := context.Background()
    key := fmt.Sprintf("online:%s", userID)
    mapKey := fmt.Sprintf("conn_map:%s", userID)
    ttl := 60 * time.Second

    h.rdb.SAdd(ctx, mapKey, connID)
    h.rdb.Expire(ctx, mapKey, ttl)
    h.rdb.HSet(ctx, key, "platform", platform, "lastActive", time.Now().Unix())
    h.rdb.Expire(ctx, key, ttl)
}

func (h *Hub) setOffline(userID, connID, platform string) {
    ctx := context.Background()
    mapKey := fmt.Sprintf("conn_map:%s", userID)

    h.rdb.SRem(ctx, mapKey, connID)
    count := h.rdb.SCard(ctx, mapKey).Val()
    if count == 0 {
        h.rdb.Del(ctx, mapKey)
        h.rdb.Del(ctx, fmt.Sprintf("online:%s", userID))
    }
}

func (h *Hub) refreshOnline(userID, connID string) {
    ctx := context.Background()
    key := fmt.Sprintf("online:%s", userID)
    mapKey := fmt.Sprintf("conn_map:%s", userID)
    ttl := 60 * time.Second

    h.rdb.Expire(ctx, key, ttl)
    h.rdb.Expire(ctx, mapKey, ttl)
    h.rdb.HSet(ctx, key, "lastActive", time.Now().Unix())
}

func (h *Hub) IsOnline(userID string) bool {
    ctx := context.Background()
    exists, _ := h.rdb.Exists(ctx, fmt.Sprintf("online:%s", userID)).Result()
    return exists > 0
}
```

---

## 10. 统一响应与错误码

### 10.1 错误码体系

```go
// pkg/errcode/errcode.go
package errcode

type Code struct {
    Code    int    `json:"code"`
    Message string `json:"msg"`
}

func (c *Code) WithDetail(detail string) *Code {
    return &Code{Code: c.Code, Message: c.Message + ": " + detail}
}

var (
    ErrSuccess         = &Code{0, "success"}
    ErrInvalidParam    = &Code{10001, "参数错误"}
    ErrUnauthorized    = &Code{10002, "未授权"}
    ErrForbidden       = &Code{10003, "禁止访问"}
    ErrResourceNotFound = &Code{10004, "资源不存在"}
    ErrAlreadyExists   = &Code{10005, "资源已存在"}
    ErrInternal        = &Code{10006, "服务器内部错误"}

    ErrUserOrPasswordWrong = &Code{20001, "用户名或密码错误"}
    ErrUserAlreadyExists   = &Code{20002, "用户已存在"}
    ErrUserNotFound        = &Code{20003, "用户不存在"}
    ErrUserDisabled        = &Code{20004, "用户被禁用"}
    ErrFriendExists        = &Code{20005, "好友关系已存在"}
    ErrFriendReqExists     = &Code{20006, "好友申请已存在"}
    ErrNotFriend           = &Code{20007, "非好友关系"}
    ErrSelfOperation       = &Code{20008, "不能对自己操作"}

    ErrConvNotFound    = &Code{30001, "会话不存在"}
    ErrMsgNotFound     = &Code{30002, "消息不存在"}
    ErrMsgRevoked      = &Code{30003, "消息已撤回"}
    ErrMsgRevokeExpire = &Code{30004, "消息超过可撤回时间"}
    ErrNotMsgSender    = &Code{30005, "非消息发送者"}
    ErrNotFriendSend   = &Code{30006, "非好友不能发消息"}
)
```

### 10.2 统一响应

```go
// pkg/resp/resp.go
package resp

func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "msg":  "success",
        "data": data,
    })
}

func Fail(c *gin.Context, err *errcode.Code) {
    c.JSON(http.StatusOK, gin.H{
        "code": err.Code,
        "msg":  err.Message,
        "data": nil,
    })
}

func FailWithStatus(c *gin.Context, httpStatus int, err *errcode.Code) {
    c.JSON(httpStatus, gin.H{
        "code": err.Code,
        "msg":  err.Message,
        "data": nil,
    })
}
```

---

## 11. 中间件实现

### 11.1 JWT 鉴权中间件

```go
// internal/middleware/auth.go
package middleware

func JWTAuth(jwtCfg config.JWTConfig) gin.HandlerFunc {
    jwtMgr := jwt.NewJWTManager(jwtCfg.PrivateKeyPath, jwtCfg.PublicKeyPath,
        jwtCfg.AccessTokenExpire, jwtCfg.RefreshTokenExpire)

    return func(c *gin.Context) {
        tokenStr := c.GetHeader("Authorization")
        if tokenStr == "" || !strings.HasPrefix(tokenStr, "Bearer ") {
            resp.FailWithStatus(c, http.StatusUnauthorized, errcode.ErrUnauthorized)
            c.Abort()
            return
        }
        tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

        claims, err := jwtMgr.ParseToken(tokenStr)
        if err != nil {
            resp.FailWithStatus(c, http.StatusUnauthorized, errcode.ErrUnauthorized)
            c.Abort()
            return
        }

        // 检查黑名单
        rdb := c.MustGet("redis").(*redis.Client)
        exists, _ := rdb.Exists(c, fmt.Sprintf("blacklist:token:%s", claims.ID)).Result()
        if exists > 0 {
            resp.FailWithStatus(c, http.StatusUnauthorized, errcode.ErrUnauthorized)
            c.Abort()
            return
        }

        // 注入用户信息到上下文
        c.Set("userID", claims.UserID)
        c.Set("platform", claims.Platform)
        c.Next()
    }
}
```

### 11.2 限流中间件

```go
// internal/middleware/ratelimit.go
package middleware

func RateLimit(rdb *redis.Client, rate int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetString("userID")
        if userID == "" {
            userID = c.ClientIP()
        }
        key := fmt.Sprintf("ratelimit:%s", userID)

        count, _ := rdb.Incr(c, key).Result()
        if count == 1 {
            rdb.Expire(c, key, window)
        }
        if count > int64(rate) {
            resp.Fail(c, errcode.ErrTooManyRequests)
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 11.3 CORS 中间件

```go
// internal/middleware/cors.go
package middleware

func CORS() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
        c.Header("Access-Control-Max-Age", "86400")
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        c.Next()
    }
}
```

---

## 12. Makefile 与构建

```makefile
# Makefile
.PHONY: build run test lint migrate clean docker

APP_NAME := gim
BUILD_DIR := bin
GO ?= go
MAIN := cmd/gim/main.go

build:
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)

run: build
	$(BUILD_DIR)/$(APP_NAME)

test:
	$(GO) test -v -count=1 ./...

test-single:
	@echo "Usage: make test-single TEST=TestName PKG=./path/to/package"
	$(GO) test -v -count=1 -run $(TEST) $(PKG)

lint:
	golangci-lint run ./...

migrate-up:
	migrate -path migrations -database "mysql://$(DB_DSN)" up

migrate-down:
	migrate -path migrations -database "mysql://$(DB_DSN)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

docker:
	docker compose -f deploy/docker-compose.yaml up -d

docker-down:
	docker compose -f deploy/docker-compose.yaml down

docker-build:
	docker build -f deploy/docker/Dockerfile -t $(APP_NAME) .

clean:
	rm -rf $(BUILD_DIR)

deps:
	$(GO) mod tidy
	$(GO) mod download

gen:
	protoc --go_out=. --go-grpc_out=. api/**/*.proto

swagger:
	swag init -g cmd/gim/main.go -o docs/swagger
```

---

## 13. Docker 与开发环境

### 12.1 Dockerfile

```dockerfile
# deploy/docker/Dockerfile

# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/gim cmd/gim/main.go

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/bin/gim /usr/local/bin/gim
COPY --from=builder /app/configs /etc/gim/configs
EXPOSE 8080 8081
ENTRYPOINT ["gim"]
```

### 13.2 docker-compose.yaml（开发环境）

💡 **为什么用 Docker Compose 开发？** Docker Compose 可以一键启动开发所需的所有依赖（MySQL、Redis），无需手动安装，保证所有开发者环境一致。同时也可以方便地清空数据重新测试。

```yaml
# deploy/docker-compose.yaml
version: "3.8"

services:
  mysql:
    image: mysql:8.0
    container_name: gim-mysql
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: gim
      MYSQL_USER: gim
      MYSQL_PASSWORD: gim_pass
      TZ: Asia/Shanghai
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ./deploy/mysql/init.sql:/docker-entrypoint-initdb.d/init.sql
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci --default-authentication-plugin=mysql_native_password
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-uroot", "-proot"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - gim-network

  redis:
    image: redis:7-alpine
    container_name: gim-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes --requirepass ""
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - gim-network

  # 本地开发时可选：启动应用服务
  gim:
    build:
      context: ..
      dockerfile: deploy/docker/Dockerfile
    container_name: gim-app
    ports:
      - "8080:8080"
      - "8081:8081"
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      GIM_SERVER_HTTPPORT: 8080
      GIM_SERVER_WSPORT: 8081
      GIM_MYSQL_HOST: mysql
      GIM_MYSQL_PORT: 3306
      GIM_MYSQL_USER: gim
      GIM_MYSQL_PASSWORD: gim_pass
      GIM_MYSQL_DBNAME: gim
      GIM_REDIS_HOST: redis
      GIM_REDIS_PORT: 6379
      GIM_REDIS_DB: 0
    networks:
      - gim-network
    profiles:
      - app  # 使用 profile 区分，默认不启动应用

volumes:
  mysql_data:
  redis_data:

networks:
  gim-network:
    driver: bridge
```

#### Docker Compose 使用方法

💡 **开发环境两种使用方式：**

1. **仅启动依赖服务**（推荐用于本地开发）
   - 只启动 MySQL 和 Redis，应用在本地运行
   - 优势：修改代码立即生效，调试方便

   ```bash
   # 启动依赖服务
   make docker-up   # 或 docker compose -f deploy/docker-compose.yaml up -d mysql redis

   # 查看服务状态
   docker compose ps

   # 查看日志
   docker compose logs mysql
   docker compose logs redis

   # 停止服务
   make docker-down  # 或 docker compose -f deploy/docker-compose.yaml down

   # 清空数据（重新初始化）
   docker compose down -v  # 删除 volumes，会清空所有数据
   ```

2. **启动完整服务**（用于部署测试）
   - 同时启动 MySQL、Redis 和应用服务
   - 优势：环境一致，适合部署测试

   ```bash
   # 启动所有服务（包括应用）
   docker compose --profile app up -d

   # 查看应用日志
   docker compose logs -f gim

   # 重启应用
   docker compose restart gim

   # 停止所有服务
   docker compose --profile app down
   ```

#### MySQL 初始化脚本

```sql
-- deploy/mysql/init.sql
-- 数据库初始化脚本（容器启动时自动执行）
CREATE DATABASE IF NOT EXISTS gim CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE gim;

-- 创建基础表结构
-- 注意：实际建表使用 golang-migrate 迁移工具
-- 这里只创建数据库，表结构通过迁移创建
```

#### 本地开发配置

```yaml
# configs/config.yaml（本地开发配置）
server:
  httpPort: 8080
  wsPort: 8081
  readTimeout: 10s
  writeTimeout: 10s

mysql:
  host: localhost  # Docker Compose 启动后映射到本地
  port: 3306
  user: gim
  password: gim_pass
  dbname: gim
  maxOpenConns: 100
  maxIdleConns: 10
  connMaxLifetime: 3600s

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  poolSize: 10

# ... 其他配置
```

```yaml
# deploy/docker-compose.yaml
version: "3.8"

services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: gim
      MYSQL_USER: gim
      MYSQL_PASSWORD: gim_pass
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  gim:
    build:
      context: ..
      dockerfile: deploy/docker/Dockerfile
    ports:
      - "8080:8080"
      - "8081:8081"
    depends_on:
      - mysql
      - redis
    environment:
      GIM_MYSQL_HOST: mysql
      GIM_REDIS_HOST: redis

volumes:
  mysql_data:
  redis_data:
```

---

## 14. 数据库迁移

### 14.1 迁移文件命名规范

```
migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_friends_tables.up.sql
├── 000002_create_friends_tables.down.sql
├── 000003_create_conversations_table.up.sql
├── 000003_create_conversations_table.down.sql
├── 000004_create_messages_table.up.sql
├── 000004_create_messages_table.down.sql
└── 000005_create_user_conversation_seq_table.up.sql
    000005_create_user_conversation_seq_table.down.sql
```

### 14.2 迁移示例

```sql
-- migrations/000001_create_users_table.up.sql
CREATE TABLE users (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id    VARCHAR(64)  NOT NULL UNIQUE,
    nickname   VARCHAR(64)  NOT NULL DEFAULT '',
    avatar_url VARCHAR(512) NOT NULL DEFAULT '',
    password   VARCHAR(128) NOT NULL,
    phone      VARCHAR(20)  NOT NULL DEFAULT '',
    email      VARCHAR(128) NOT NULL DEFAULT '',
    status     TINYINT      NOT NULL DEFAULT 1,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_phone (phone),
    INDEX idx_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

```sql
-- migrations/000001_create_users_table.down.sql
DROP TABLE IF EXISTS users;
```

---

## 15. 第二阶段实现要点

### 15.1 gRPC Protobuf 定义示例

```protobuf
// api/msg/msg.proto
syntax = "proto3";
package msg;
option go_package = "github.com/yourname/gim/api/msg";

service MsgService {
    rpc SendMessage(SendMsgReq) returns (SendMsgResp);
    rpc GetHistory(GetHistoryReq) returns (GetHistoryResp);
    rpc MarkRead(MarkReadReq) returns (MarkReadResp);
    rpc RevokeMsg(RevokeMsgReq) returns (RevokeMsgResp);
}

message SendMsgReq {
    string conversation_id = 1;
    string sender_id = 2;
    string client_msg_id = 3;
    int32  content_type = 4;
    string content = 5;
}

message SendMsgResp {
    int64  seq = 1;
    string server_msg_id = 2;
    int64  send_time = 3;
}
```

### 15.2 Kafka 消息流转改造

第二阶段核心改造点：消息发送从同步写库变为异步写 Kafka。

```
原流程: WS -> rpc-msg -> 写MySQL -> 返回Seq
新流程: WS -> rpc-msg -> 分配Seq(Redis) -> 写Kafka -> 返回Seq
                                              |
                                     MsgTransfer消费
                                              |
                                        写MongoDB(批量)
```

Kafka Topic 设计：

| Topic | 分区数 | 生产者 | 消费者 | 用途 |
|-------|--------|--------|--------|------|
| toMongo | 8 | rpc-msg | MsgTransfer | 消息持久化到 MongoDB |
| toPush | 8 | rpc-msg | Push 服务 | 在线推送通知 |
| toOfflinePush | 4 | Push 服务 | OfflinePush 服务 | 离线推送 |

### 15.3 MongoDB 文档分片写入

```go
// internal/mongo/message.go
func (r *msgMongoRepo) BatchInsert(ctx context.Context, msgs []*MsgDoc) error {
    // 按 DocID 分组
    groups := make(map[string][]*MsgInfo)
    for _, msg := range msgs {
        seqSuffix := (msg.Seq - 1) / 100
        docID := fmt.Sprintf("%s:%d", msg.ConversationID, seqSuffix)
        groups[docID] = append(groups[docID], msg)
    }

    // 批量 Upsert
    for docID, msgList := range groups {
        filter := bson.M{"_id": docID}
        update := bson.M{"$push": bson.M{"msgs": bson.M{"$each": msgList}}}
        opts := options.Update().SetUpsert(true)
        _, err := r.collection.UpdateOne(ctx, filter, update, opts)
        if err != nil {
            return err
        }
    }
    return nil
}
```

### 15.4 WS Gateway 改造为 WS+gRPC 双协议

```go
// 第二阶段 gim-ws 增加 gRPC 服务器
func (s *WSServer) Start() error {
    // 启动 WebSocket 服务
    go s.startWebSocket()

    // 启动 gRPC 服务（供 Push 服务调用）
    lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GRPCPort))
    grpcServer := grpc.NewServer()
    pb.RegisterMsgGatewayServer(grpcServer, s)
    return grpcServer.Serve(lis)
}

// 实现 gRPC 接口：Push 服务调用此方法推送消息
func (s *WSServer) OnlineBatchPushOneMsg(ctx context.Context, req *pb.OnlineBatchPushOneMsgReq) (*pb.OnlineBatchPushOneMsgResp, error) {
    results := make(map[string]bool)
    for _, userID := range req.UserIDs {
        // 在本地 Hub 中查找连接
        if clients, ok := s.hub.clients[userID]; ok && len(clients) > 0 {
            for client := range clients {
                client.Send(&WSMessage{Type: 101, Data: req.Msg})
            }
            results[userID] = true
        } else {
            results[userID] = false
        }
    }
    return &pb.OnlineBatchPushOneMsgResp{Results: results}, nil
}
```

### 15.5 群消息扇出

```go
// Push 服务中群消息推送逻辑
func (s *pushService) handleGroupMsg(ctx context.Context, msg *PushMsg) error {
    // 1. 获取群成员列表
    members, _ := s.groupRepo.GetMembers(ctx, msg.GroupID)

    // 2. 批量检查在线状态
    onlineUsers := s.getOnlineUsers(members)

    // 3. 分 Gateway 实例批量推送
    gatewayClients := s.getGatewayClients()
    for gatewayAddr, client := range gatewayClients {
        // 找到该 Gateway 上的在线成员
        usersOnGateway := s.getUsersOnGateway(onlineUsers, gatewayAddr)
        if len(usersOnGateway) == 0 {
            continue
        }
        // gRPC 批量推送
        resp, err := client.OnlineBatchPushOneMsg(ctx, &pb.OnlineBatchPushOneMsgReq{
            UserIDs: usersOnGateway,
            Msg:     msg,
        })
        // 处理推送失败的成员 -> 离线推送
        for userID, ok := range resp.Results {
            if !ok {
                s.offlinePush(ctx, userID, msg)
            }
        }
    }
    return nil
}
```

---

## 16. 第三阶段实现要点

### 16.1 Prometheus 指标埋点

```go
// internal/middleware/metrics.go
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "gim_http_requests_total"},
        []string{"method", "path", "status"},
    )
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "gim_http_request_duration_seconds"},
        []string{"method", "path"},
    )
    wsConnectionsGauge = prometheus.NewGauge(
        prometheus.GaugeOpts{Name: "gim_ws_connections"},
    )
)

func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        duration := time.Since(start).Seconds()
        status := fmt.Sprintf("%d", c.Writer.Status())
        httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
        httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
    }
}
```

### 16.2 OpenTelemetry 链路追踪

```go
// internal/trace/trace.go
func InitTracer(serviceName, jaegerEndpoint string) (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracehttp.New(context.Background(),
        otlptracehttp.WithEndpoint(jaegerEndpoint),
    )
    if err != nil {
        return nil, err
    }
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            attribute.String("service.name", serviceName),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### 16.3 Helm Chart 结构

```
deploy/k8s/helm/gim/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── api-deployment.yaml
│   ├── api-service.yaml
│   ├── ws-statefulset.yaml
│   ├── ws-service.yaml
│   ├── rpc-auth-deployment.yaml
│   ├── rpc-user-deployment.yaml
│   ├── rpc-msg-deployment.yaml
│   ├── push-deployment.yaml
│   ├── msgtransfer-deployment.yaml
│   ├── admin-deployment.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── ingress.yaml
│   └── hpa.yaml
```

### 16.4 HPA 配置

```yaml
# templates/hpa.yaml — WS Gateway 按连接数伸缩
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gim-ws-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: StatefulSet
    name: gim-ws
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Pods
    pods:
      metric:
        name: gim_ws_connections
      target:
        type: AverageValue
        averageValue: "5000"
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

---

## 17. 第四阶段实现要点

> 第四阶段（AI Agent）的完整架构设计、核心代码、数据模型、TODO 详见 [AI_AGENT.md](AI_AGENT.md)。这里只列出与前三阶段代码的关键集成点。

### 17.1 AI Service 接入 WS Gateway

在 WS Client 的 `handleMessage` 方法中增加 AI 消息类型的路由：

```go
// 在 internal/ws/client.go 的 handleMessage switch 中追加

case 10: // AI 智能回复请求
    var req AIReplyData
    json.Unmarshal(toJSON(msg.Data), &req)
    // 异步调用 AI Service（不阻塞 WS 读取）
    go c.hub.aiSvc.HandleReply(c.Request().Context(), c.userID, req.ConversationID, req.Instruction, msg.ReqID)

case 11: // 群 AI 请求
    var req GroupAIRequestData
    json.Unmarshal(toJSON(msg.Data), &req)
    go c.hub.aiSvc.Router.Handle(c.Request().Context(), &req)
```

💡 **为什么用 `go` 异步调用？** LLM API 调用可能耗时数秒，如果在 WS 读取协程中同步等待，该用户的所有消息都会被阻塞。异步调用后立即返回，AI 结果通过 WS 推送回来。

### 17.2 Kafka 集成审核 Agent

在 rpc-msg 的消息发送流程中，追加写 `toModeration` Topic：

```go
// 在 SendMessage 流程最后追加（第二阶段已有 Kafka 生产者）
if s.kafkaProducer != nil {
    s.kafkaProducer.WriteMessage(ctx, "toModeration", msg.ConversationID, &KafkaMsg{
        Type:          "moderation",
        ConversationID: msg.ConversationID,
        ClientMsgID:   msg.ClientMsgID,
        SenderID:      msg.SenderID,
        Content:       msg.Content,
        SendTime:      msg.SendTime,
    })
}
```

### 16.3 Admin API 扩展

```go
// 在 Admin API 路由中追加 AI 对话接口
admin := api.Group("/ai")
{
    admin.POST("/chat", handlers.AdminAI.Chat)           // 管理助手对话
    admin.GET("/violations", handlers.AdminAI.Violations) // 审核日志查询
    admin.GET("/stats", handlers.AdminAI.Stats)           // 统计数据（Agent Tool 用）
}
```

### 16.4 配置扩展

```yaml
# config.yaml 新增 AI 配置段
ai:
  enabled: true
  apiKey: ""                    # 从环境变量 ANTHROPIC_API_KEY 读取，不写配置文件
  model: "claude-sonnet-4-6"
  routerModel: "claude-haiku-4-5-20251001"  # 路由判断用轻量模型
  maxContextMessages: 20        # 回复助手上下文消息数
  maxTokens: 1024               # 单次回复最大 Token
  moderationEnabled: true       # 内容审核开关
  rateLimitPerUser: 100         # 每用户每日 AI 调用上限

# Milvus 配置（RAG 用）
milvus:
  host: 127.0.0.1
  port: 19530
  collection: gim_docs

# Embedding 配置
embedding:
  provider: "openai"            # openai / local
  model: "text-embedding-3-small"
  apiKey: ""                    # 从环境变量读取
```
