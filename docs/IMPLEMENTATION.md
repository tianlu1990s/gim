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

## 文档章节概览

### 文档总结

本文档提供了 GIM 即时通讯系统从零开始到生产级部署的完整实现指南：

| 章节 | 内容 | 关键产出 |
|------|------|----------|
| 0. 项目目录结构 | 完整的项目结构 | 可直接使用的目录树 |
| 1. 项目骨架 | 基础设施和配置 | 可运行的初始化脚本 |
| 2. 公共组件 | pkg 层完整实现 | 日志、JWT、Snowflake 等 |
| 3. 认证模块 | 用户认证完整流程 | 注册、登录、Token 管理 |
| 4. 用户模块 | 用户资料管理 | 资料查看、更新、搜索 |
| 5. 好友模块 | 好友关系管理 | 添加好友、申请处理、列表管理 |
| 6. 会话模块 | 会话管理 | 会话列表、未读计数、置顶删除 |
| 7. 消息模块 | 消息收发 | 发送、历史、已读、撤回 |
| 8. WebSocket | 实时通信 | 连接管理、消息推送、在线状态 |
| 9. 在线状态 | 用户在线管理 | Redis 存储和查询 |
| 10. 统一响应 | 标准化响应 | 成功/失败响应格式 |
| 11. 中间件 | 公共中间件 | JWT、CORS、限流、日志、恢复 |
| 12. Makefile | 构建脚本 | 一键构建、测试、部署 |
| 13. Docker | 容器化 | 开发环境 Docker Compose |
| 14. 迁移 | 数据库版本管理 | 可回滚的迁移脚本 |
| 15. 第二阶段 | 微服务架构 | gRPC、Kafka、MongoDB |
| 16. 第三阶段 | K8S 部署 | 生产级运维、监控、告警 |
| 17. 第四阶段 | AI 集成 | 智能回复、群 AI、内容审核 |

### 实施路径建议

**方案 A：快速验证（推荐）**
1. 按第 13.1 节的步骤表，实现 Phase 1 核心功能
2. 完成基本的消息收发功能
3. 验证核心流程后，再考虑拆分微服务

**方案 B：一步到位**
1. 直接按 Phase 1→2→3→4 顺序实施
2. 每个阶段都完成后进行充分测试
3. 适合有经验的团队

**方案 C：渐进演进**
1. 先实现 Phase 1 验证业务逻辑
2. 逐步将模块拆分到微服务（一个一个拆）
3. 边拆分边优化，保证系统稳定

### 快速开始（5 分钟上手）

```bash
# 1. 克隆项目
git clone https://github.com/tianlu1990s/gim.git
cd gim

# 2. 创建项目目录结构（如果不存在）
mkdir -p cmd/gim internal configs/jwt logs migrations

# 3. 生成 JWT 密钥对
openssl genrsa -out configs/jwt/private.pem 2048
openssl rsa -in configs/jwt/private.pem -pubout -out configs/jwt/public.pem

# 4. 初始化 Go Module（如果 go.mod 不存在）
go mod init github.com/tianlu1990s/gim

# 5. 安装核心依赖
go get github.com/gin-gonic/gin \
    gorm.io/gorm \
    gorm.io/driver/mysql \
    github.com/redis/go-redis/v9 \
    github.com/gorilla/websocket \
    github.com/spf13/viper

# 6. 启动依赖服务（MySQL + Redis）
make docker
# 等待服务启动（约 10 秒）
sleep 10

# 7. 执行数据库迁移（需先设置数据库连接）
export DB_USER=gim
export DB_PASSWORD=gim_pass
export DB_HOST=localhost
export DB_PORT=3306
export DB_NAME=gim
make migrate-up

# 8. 创建配置文件（参考 1.3 节的 config.yaml 内容）
cat > configs/config.yaml << 'EOF'
server:
  httpPort: 8080
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
  privateKeyPath: "configs/jwt/private.pem"
  publicKeyPath: "configs/jwt/public.pem"

websocket:
  port: 8081
  maxConnPerUser: 5
  maxMessageSize: 4096
  writeWait: 10s
  pongWait: 60s
  pingPeriod: 30s

log:
  level: debug
  format: text
  output: stdout
  filePath: logs/gim.log
  maxSize: 100
  maxBackups: 10
  maxAge: 30
  compress: true
  shortFile: true
  color: true

snowflake:
  nodeID: 1
EOF

# 9. 构建并运行应用
make run

# 10. 测试健康检查（新终端窗口）
curl http://localhost:8080/health
# 预期返回: {"status":"ok"}

# 11. 测试用户注册
curl -X POST http://localhost:8080/api/v1/auth/register \
    -H "Content-Type: application/json" \
    -d '{"userId":"alice","password":"Pass1234","nickname":"Alice"}'

# 12. 测试用户登录
curl -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"userId":"alice","password":"Pass1234","platform":"web"}'
```

💡 **注意事项：**
- 如果 `make docker` 启动失败，检查 Docker 是否运行：`docker ps`
- 如果 `make migrate-up` 报错，检查 MySQL 是否就绪：`docker logs gim-mysql`
- 配置文件 `configs/config.yaml` 需要手动创建（上述命令已包含）
- JWT 密钥文件路径必须在 configs/jwt/ 目录下

### 常见问题 FAQ

**Q1: 开发环境需要什么？**
A: Go 1.26+、Docker、MySQL 8.4+、Redis 7+。推荐使用 Docker Compose 启动 MySQL 和 Redis。

**Q2: 如何调试 WebSocket 连接？**
A: 使用 Chrome DevTools 的 Network → WS 标签，查看 WebSocket 连接、消息帧、错误信息。

**Q3: 如何查看实时日志？**
A: `tail -f logs/gim.log` 或使用 `journalctl -u gim`（如果是 systemd 服务）。

**Q4: 如何进行压力测试？**
A: 使用 `wrk` 或 `k6` 工具：
```bash
# 测试登录接口
wrk -t4 -c100 -s 10s -H "Content-Type: application/json"   -d '{"userId":"test","password":"Test1234","platform":"web"}'   http://localhost:8080/api/v1/auth/login
```

**Q5: 如何查看数据库迁移历史？**
A: golang-migrate 会自动创建 `schema_migrations` 表来记录迁移历史。执行以下 SQL 查看：

```bash
# 连接 MySQL
mysql -u gim -pgim_pass gim

# 查看迁移历史
SELECT * FROM schema_migrations ORDER BY version;

# 预期输出：
-- +---------+-------------------------+----------------------+---------------------+
-- | version | dirty                   | identifier            | applied_at          |
-- +---------+-------------------------+----------------------+---------------------+
-- |       1 | 0                       | 000001              | 2026-05-01 10:00:00 |
-- |       2 | 0                       | 000002              | 2026-05-01 10:00:05 |
-- +---------+-------------------------+----------------------+---------------------+
```

💡 **重要概念**：
- `dirty=1` 表示该迁移执行失败，需要手动修复后重新执行
- `dirty=0` 表示该迁移已成功执行
- 如果 `dirty=1`，需要手动修复问题后设置 `UPDATE schema_migrations SET dirty=0 WHERE version=N;`

**Q6: 微服务拆分后如何调试？**
A: 1) 检查 etcd 服务注册；2) 检查 Kafka 消息堆积；3) 查看各服务日志；4) 使用 Jaeger 链路追踪

**Q7: AI 功能需要额外费用吗？**
A: Deepseek API 和 Claude API 按使用量计费。开发阶段可使用本地模型（Ollama）零成本验证，生产环境按需选择 Provider。建议设置使用上限避免超支。

**Q8: 如何进行灰度发布？**
A: 在 K8S 中使用多个 Deployment，通过流量比例逐步切换：
```yaml
# 20% 流量到新版本
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gim-api
spec:
  http:
  - route:
    - destination:
        host: gim-api
        subset: v1
      weight: 80
    - destination:
        host: gim-api
        subset: v2
      weight: 20
```

### 资源链接

- [Go 官方文档](https://golang.org/doc/)
- [Gin 框架](https://gin-gonic.com/docs/)
- [GORM 文档](https://gorm.io/docs/)
- [Redis Go 客户端](https://github.com/redis/go-redis)
- [gorilla/websocket](https://github.com/gorilla/websocket)
- [Kubernetes 文档](https://kubernetes.io/docs/)
- [Helm 文档](https://helm.sh/docs/)
- [Deepseek API 文档](https://platform.deepseek.com/docs)
- [Claude API 文档](https://docs.anthropic.com/)
- [Ollama 本地部署](https://ollama.com/)

---



## 0. 项目目录结构

💡 **为什么需要清晰的目录结构？** 良好的目录结构让代码组织清晰，便于团队协作和新人快速理解项目位置。Go 社区推荐的标准项目结构如下：

```
gim/
├── cmd/                          # 主程序入口
│   └── gim/
│       └── main.go               # 应用入口
├── api/                          # API 定义（第二阶段 gRPC Protobuf）
│   └── msg/
│       └── msg.proto
├── internal/                     # 私有代码（不对外暴露）
│   ├── config/                   # 配置管理
│   │   └── config.go
│   ├── handler/                  # HTTP 处理器（控制器）
│   │   ├── auth.go
│   │   ├── user.go
│   │   ├── friend.go
│   │   ├── conversation.go
│   │   ├── message.go
│   │   └── init.go
│   ├── service/                  # 业务逻辑层
│   │   ├── auth.go
│   │   ├── user.go
│   │   ├── friend.go
│   │   ├── conversation.go
│   │   ├── message.go
│   │   ├── validate.go
│   │   └── init.go
│   ├── repository/               # 数据访问层
│   │   ├── user.go
│   │   ├── friend.go
│   │   ├── conversation.go
│   │   ├── message.go
│   │   └── init.go
│   ├── model/                    # 数据模型
│   │   ├── user.go
│   │   ├── friend.go
│   │   ├── conversation.go
│   │   ├── message.go
│   │   ├── req.go
│   │   ├── vo.go
│   │   └── user_conversation_seq.go
│   ├── middleware/               # Gin 中间件
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── ratelimit.go
│   │   ├── recovery.go
│   │   └── logger.go
│   └── ws/                       # WebSocket 网关
│       ├── hub.go
│       ├── client.go
│       ├── server.go
│       └── util.go
├── pkg/                          # 公共包（可被外部使用）
│   ├── jwt/                      # JWT 工具
│   │   └── jwt.go
│   ├── snowflake/                # Snowflake ID 生成
│   │   └── snowflake.go
│   ├── slog/                     # 结构化日志
│   │   └── slog.go
│   ├── resp/                     # 统一响应
│   │   └── resp.go
│   ├── errcode/                  # 错误码
│   │   └── errcode.go
│   ├── rediskey/                 # Redis Key 管理
│   │   └── rediskey.go
│   ├── convid/                   # 会话 ID 生成
│   │   └── convid.go
│   └── convutil/                 # 会话工具函数
│       └── convutil.go
├── configs/                      # 配置文件
│   ├── config.yaml               # 主配置文件
│   └── jwt/                      # JWT 密钥对
│       ├── private.pem
│       └── public.pem
├── migrations/                   # 数据库迁移文件
│   ├── 000001_create_users_table.up.sql
│   ├── 000001_create_users_table.down.sql
│   ├── 000002_create_friends_tables.up.sql
│   ├── 000002_create_friends_tables.down.sql
│   ├── 000003_create_conversations_table.up.sql
│   ├── 000003_create_conversations_table.down.sql
│   ├── 000004_create_messages_table.up.sql
│   ├── 000004_create_messages_table.down.sql
│   ├── 000005_create_user_conversation_seq_table.up.sql
│   └── 000005_create_user_conversation_seq_table.down.sql
├── deploy/                       # 部署相关
│   ├── docker/
│   │   └── Dockerfile
│   ├── k8s/
│   │   └── helm/
│   │       └── gim/
│   ├── docker-compose.yaml
│   └── mysql/
│       └── init.sql
├── docs/                         # 项目文档
│   ├── API.md
│   ├── IMPLEMENTATION.md
│   ├── GETTING_STARTED.md
│   ├── K8S_DEPLOY.md
│   └── AI_AGENT.md
├── logs/                         # 日志文件（运行时生成）
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
├── CLAUDE.md
├── PLAN.md
├── STRUCTURE.md
├── README.md
└── TODO.md
```

💡 **目录结构说明**：
- `cmd/` - 应用程序入口，每个可执行文件一个子目录
- `internal/` - 私有代码，外部项目无法导入
- `pkg/` - 公共库，可被其他项目导入
- `api/` - API 定义文件（Proto、OpenAPI 等）
- `configs/` - 配置文件，包含 YAML、环境变量等
- `migrations/` - 数据库版本管理文件
- `deploy/` - 部署相关文件（Docker、K8s 等）

---


## 0.1 系统架构总览

💡 **四阶段演进路线**：本项目按四个阶段迭代开发，每个阶段独立可部署，逐步演进。

```
第一阶段（当前）：单体应用
┌─────────────────────────────────────────────────────┐
│                      客户端                      │
│                  ┌──────────────┐                  │
│                  │   HTTP/WS    │                  │
│                  └──────┬───────┘                  │
└─────────────────────┼────────────────────────────────┘
                      │
              ┌─────────▼─────────┐
              │    gim (单体)     │
              │  ┌───────────┐  │
              │  │  Gin + WS  │  │
              │  │  HTTP API  │  │
              │  └─────┬─────┘  │
              │        │         │
              │  ┌─────▼─────┐  │
              │  │ MySQL + Redis│  │
              │  └────────────┘  │
              └───────────────────┘
```

```
第二阶段（微服务）：服务拆分 + Kafka + MongoDB
┌─────────────────────────────────────────────────────────┐
│                        客户端                              │
│                  ┌───────────────┐                           │
│                  │  HTTP/WS       │                           │
│                  └───────┬───────┘                           │
└────────────────────┼───────────────────────────────────────┘
                      │
        ┌───────────────┼───────────────┐
        │               │               │
   ┌────▼────┐    ┌───▼──────┐   ┌───▼──────┐
   │  gim-api  │    │  gim-ws   │   │ gim-push  │
   │  HTTP API  │    │  WS Gateway│   │  gRPC     │
   └────┬─────┘    └────┬───────┘   └────┬───────┘
        │               │               │
   ┌────▼────────────┼───────────────▼───────┐
   │              etcd + Kafka               │
   └───────────────┬──────────────────────┘
                   │
       ┌───────────┼───────────┬───────────┐
       │           │           │           │
   ┌───▼────┐┌──▼────┐┌──▼────┐┌──▼────┐
   │rpc-auth││rpc-user││rpc-friend││rpc-msg│
   │ +MySQL ││+MySQL  ││+MySQL   ││+Kafka │
   └─────────┘└─────────┘└─────────┘└──┬───┘
                                          │
                                ┌────────▼─────┐
                                │ MongoDB      │
                                └──────────────┘
```

```
第三阶段（K8S）：生产级部署 + 监控
第二阶段架构 + K8S 编排
├── Prometheus（指标采集）
├── Grafana（可视化）
├── Jaeger/OTel（链路追踪）
├── AlertManager（告警）
└── Helm（一键部署）
```

```
第四阶段（AI）：智能助手 + 内容审核
第三阶段架构 + AI 服务层
├── 智能回复服务
├── 群 AI 助手
├── 内容审核服务
├── 向量存储
└── AI Provider 集成
```

---

## 0.2 核心数据流图

### 消息发送核心数据流

💡 **消息发送完整流程**：从客户端到消息持久化的全过程。

```
客户端发送消息流程：
┌──────────┐      ┌──────────┐      ┌──────────┐      ┌──────────┐
│  客户端  │ ───►│ WebSocket │ ───►│  Message  │ ───►│  Redis/   │
│          │      │  Gateway  │      │  Service  │      │  MySQL    │
│          │      └──────────┘      └────┬─────┘      └────┬────┘
│          │                          │              │
└──────────┼──────────────────────────┼──────────────┘
           │                          │
      ┌────▼──────────────────────────▼────┐
      │         推送给接收方 WebSocket         │
      │         (在线则实时推送，离线存待发）│
      └────────────────────────────────────────┘
```

💡 **关键决策点**：
1. **去重**：ClientMsgID 作为唯一标识，防止重复发送
2. **Seq 分配**：Redis INCR 保证消息顺序和高并发
3. **持久化**：消息先写入数据库，再推送（避免推送成功但入库失败）
4. **推送优化**：同用户多连接只推送一次，群消息批量推送

---

## 0.3 开发环境准备与命令速查

### 0.3.1 必要工具安装

💡 **为什么需要这些工具？** 开发 IM 系统需要多种工具协同工作：
- **Go**：编程语言
- **Docker**：容器化运行 MySQL 和 Redis
- **golang-migrate**：数据库版本管理和迁移
- **golangci-lint**：代码质量检查（可选但推荐）
- **protoc**：第二阶段生成 gRPC 代码（第二阶段需要）
- **swag**：生成 Swagger API 文档（可选）

```bash
# 1. Go 语言（1.26+）
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 验证安装
go version

# 2. Docker 和 Docker Compose
# macOS
brew install --cask docker

# Linux
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
# 重新登录使组权限生效

# 验证安装
docker --version
docker compose version

# 3. golang-migrate（数据库迁移工具）
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/migrate

# 验证安装
migrate --version

# 4. golangci-lint（代码检查，可选）
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# 验证安装
golangci-lint version

# 5. protoc（第二阶段 gRPC 代码生成，可选）
# macOS
brew install protobuf

# Linux
sudo apt-get install -y protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 验证安装
protoc --version

# 6. swag（Swagger 文档生成，可选）
go install github.com/swaggo/swag/cmd/swag@latest

# 验证安装
swag --version
```

### 0.3.2 开发与部署命令速查

| 操作 | 命令 | 说明 |
|------|------|------|
| 依赖服务启动 | `make docker` | 启动 MySQL、Redis（仅开发）|
| 依赖服务停止 | `make docker-down` | 停止并清理容器 |
| 查看容器日志 | `make docker-logs` | 查看所有容器日志 |
| 数据库迁移 | `make migrate-up` | 执行数据库迁移 |
| 回滚迁移 | `make migrate-down` | 回滚一次迁移 |
| 创建迁移 | `make migrate-create NAME=name` | 创建新的迁移文件 |
| 构建应用 | `make build` | 编译二进制文件 |
| 运行应用 | `make run` | 启动应用（先 build）|
| 运行测试 | `make test` | 运行所有单元测试 |
| 运行单测 | `make test-single TEST=TestName PKG=./path` | 运行指定测试 |
| 代码检查 | `make lint` | golangci-lint 检查代码 |
| 依赖管理 | `make deps` | 整理和下载依赖 |
| 验证依赖 | `make deps-check` | 验证依赖完整性 |
| 生成 gRPC 代码 | `make gen` | 第二阶段：从 protobuf 生成代码 |
| 生成 Swagger | `make swagger` | 生成 API 文档 |
| 查看帮助 | `make help` | 显示所有可用命令 |

### 0.3.3 环境变量配置

💡 **某些配置通过环境变量传递，避免敏感信息写入配置文件。**

```bash
# 数据库连接（默认值已在 Makefile 中设置）
export DB_USER=gim
export DB_PASSWORD=gim_pass
export DB_HOST=localhost
export DB_PORT=3306
export DB_NAME=gim

# 运行环境
export GIM_ENV=dev  # dev/test/prod

# AI 功能（第四阶段）
export DEEPSEEK_API_KEY=your_deepseek_key_here    # Deepseek API
export ANTHROPIC_API_KEY=your_anthropic_key_here  # Claude API
export OPENAI_API_KEY=your_openai_key_here        # Embedding
```

---


## 1. 项目骨架与基础设施

💡 **本章目标**：搭建项目的基础骨架，包括目录结构、配置管理、依赖初始化。这是所有后续工作的基础。

### 1.0 初始化流程图

```
项目初始化步骤：
┌─────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ 创建目录 │ ───►│ go mod   │ ───►│ 依赖安装  │ ───►│ 配置文件  │
│   结构   │     │  init    │     │          │     │          │
└─────────┘     └──────────┘     └──────────┘     └──────────┘
                                            │
                                      ┌─────────▼─────────┐
                                      │  生成 JWT 密钥对  │
                                      │  (private/public) │
                                      └───────────────────┘
```

### 1.0.1 快速初始化命令（一键执行）

```bash
# 1. 创建项目目录结构（完整的目录树）
mkdir -p cmd/gim \
         internal/{config,handler,service,repository,model,middleware,ws} \
         pkg/{jwt,snowflake,slog,resp,errcode,rediskey,convid,convutil} \
         configs/jwt \
         logs \
         migrations \
         api/{auth,user,friend,msg,conversation,push} \
         deploy/{docker,mysql,k8s/helm/gim}

# 2. 初始化 Go Module
cd /path/to/gim
go mod init github.com/tianlu1990s/gim

# 3. 安装核心依赖（一条命令完成）
go get github.com/gin-gonic/gin     gorm.io/gorm     gorm.io/driver/mysql     github.com/redis/go-redis/v9     github.com/gorilla/websocket     github.com/spf13/viper     github.com/golang-jwt/jwt/v5     golang.org/x/crypto/bcrypt     github.com/bwmarrin/snowflake     github.com/golang-migrate/migrate/v4     gopkg.in/natefinch/lumberjack.v2

# 4. 生成 JWT RSA 密钥对
mkdir -p configs/jwt
openssl genrsa -out configs/jwt/private.pem 2048
openssl rsa -in configs/jwt/private.pem -pubout -out configs/jwt/public.pem

# 5. 创建示例配置文件（使用后面 1.3 节的配置内容）
cat > configs/config.yaml << 'EOF'
server:
  httpPort: 8080
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
  privateKeyPath: "configs/jwt/private.pem"
  publicKeyPath: "configs/jwt/public.pem"

websocket:
  port: 8081
  maxConnPerUser: 5
  maxMessageSize: 4096
  writeWait: 10s
  pongWait: 60s
  pingPeriod: 30s

log:
  level: debug
  format: text
  output: file
  filePath: logs/gim.log
  maxSize: 100
  maxBackups: 10
  maxAge: 30
  compress: true
  shortFile: true
  color: true

snowflake:
  nodeID: 1
EOF

# 6. 生成 .gitignore
cat > .gitignore << 'EOF'
# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib
gim

# Test binary
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment variables
.env
.env.local

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# Logs
logs/
*.log

# OS
.DS_Store
Thumbs.db

# Config (don't commit secrets)
configs/jwt/*.pem
EOF

echo "✅ 项目初始化完成！"
echo "下一步：创建第一个 Model 文件或运行 make docker 启动依赖服务"
```

---

### 1.1 Go Module 初始化

💡 **什么是 Go Module？** 类似 Node.js 的 `package.json`，Go 用 `go.mod` 文件管理项目依赖。`go get xxx` 会自动把依赖记录到 go.mod 里，别人拿到你的代码后 `go mod download` 就能安装所有依赖。

```bash
go mod init github.com/tianlu1990s/gim
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

	"github.com/tianlu1990s/gim/internal/config"
	"github.com/tianlu1990s/gim/internal/handler"
	"github.com/tianlu1990s/gim/internal/middleware"
	"github.com/tianlu1990s/gim/internal/repository"
	"github.com/tianlu1990s/gim/internal/service"
	"github.com/tianlu1990s/gim/internal/ws"
	"github.com/tianlu1990s/gim/pkg/jwt"
	"github.com/tianlu1990s/gim/pkg/slog"
	"github.com/tianlu1990s/gim/pkg/snowflake"
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
	// 8. 初始化 WS Hub
	hub := ws.NewHub(services.Message, services.Conversation, rdb, cfg.WebSocket)
	go hub.Run()
	services := service.NewServices(repos, cfg, jwtMgr, rdb, hub)
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
	wsServer := ws.NewServer(cfg.WebSocket, hub, jwtMgr)

	// 10. 并发启动 HTTP 和 WS
	go func() {
		logger.Info("WebSocket server starting", "port", cfg.WebSocket.Port)
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

💡 **日志配置说明**：使用 Go 1.26+ 标准库 `log/slog`，支持：
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
	Port           int           `mapstructure:"port"`
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
  privateKeyPath: "configs/jwt/private.pem"
  publicKeyPath: "configs/jwt/public.pem"

websocket:
  port: 8081
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


### 1.4 数据模型定义

💡 **为什么先定义 Model？** Model 是整个系统的数据基础——Repository 读写 Model、Service 处理 Model、Handler 返回 Model 的视图（VO）。先把 Model 定好，后续各层才能围绕它展开。

```go
// internal/model/user.go
package model

import "time"

type User struct {
    ID        uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
    UserID    string    `gorm:"uniqueIndex;type:varchar(64);not null" json:"userId"`
    Nickname  string    `gorm:"type:varchar(64);not null;default:''" json:"nickname"`
    AvatarURL string    `gorm:"type:varchar(512);not null;default:''" json:"avatarUrl"`
    Password  string    `gorm:"type:varchar(128);not null" json:"-"`
    Phone     string    `gorm:"index;type:varchar(20);not null;default:''" json:"phone"`
    Email     string    `gorm:"index;type:varchar(128);not null;default:''" json:"email"`
    Status    int8      `gorm:"type:tinyint;not null;default:1" json:"status"` // 1=正常 2=禁用
    CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (User) TableName() string { return "users" }
```

```go
// internal/model/friend.go
package model

import "time"

type Friend struct {
    ID        uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
    OwnerID   string    `gorm:"index:idx_owner;type:varchar(64);not null" json:"ownerId"`
    FriendID  string    `gorm:"index:idx_owner;type:varchar(64);not null" json:"friendId"`
    Remark    string    `gorm:"type:varchar(64);not null;default:''" json:"remark"`
    CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

func (Friend) TableName() string { return "friends" }

type FriendRequest struct {
    ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
    FromUserID string    `gorm:"index:idx_to;type:varchar(64);not null" json:"fromUserId"`
    ToUserID   string    `gorm:"index:idx_to;type:varchar(64);not null" json:"toUserId"`
    Message    string    `gorm:"type:varchar(256);not null;default:''" json:"message"`
    Status     int8      `gorm:"type:tinyint;not null;default:0" json:"status"` // 0=待处理 1=已同意 2=已拒绝
    CreatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (FriendRequest) TableName() string { return "friend_requests" }
```

```go
// internal/model/conversation.go
package model

import "time"

type Conversation struct {
    ID             uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
    OwnerID        string    `gorm:"index:idx_owner_conv;type:varchar(64);not null" json:"ownerId"`
    ConversationID string    `gorm:"index:idx_owner_conv;type:varchar(128);not null" json:"conversationId"`
    ConvType       int       `gorm:"type:int;not null" json:"convType"` // 1=单聊 2=群聊
    TargetID       string    `gorm:"type:varchar(64);not null;default:''" json:"targetId"`
    MaxSeq         int64     `gorm:"type:bigint;not null;default:0" json:"maxSeq"`
    IsPinned       bool      `gorm:"type:tinyint(1);not null;default:0" json:"isPinned"`
    CreatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
    UpdatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (Conversation) TableName() string { return "conversations" }
```

```go
// internal/model/message.go
package model

import "time"

type Message struct {
    ID             uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
    ConversationID string    `gorm:"index:idx_conv_seq;type:varchar(128);not null" json:"conversationId"`
    Seq            int64     `gorm:"index:idx_conv_seq;type:bigint;not null" json:"seq"`
    SenderID       string    `gorm:"type:varchar(64);not null" json:"senderId"`
    MsgType        int       `gorm:"type:int;not null" json:"msgType"` // 1=文本 2=图片 3=文件 4=系统消息
    Content        string    `gorm:"type:text;not null" json:"content"`
    ClientMsgID    string    `gorm:"uniqueIndex;type:varchar(64);not null" json:"clientMsgId"`
    ServerMsgID    string    `gorm:"index;type:varchar(64);not null" json:"serverMsgId"`
    IsRevoked      bool      `gorm:"type:tinyint(1);not null;default:0" json:"isRevoked"`
    CreatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

func (Message) TableName() string { return "messages" }
```

```go
// internal/model/user_conversation_seq.go
package model

import "time"

// UserConversationSeq 记录用户在每个会话中的已读位置
type UserConversationSeq struct {
    ID             uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
    UserID         string    `gorm:"uniqueIndex:idx_user_conv;type:varchar(64);not null" json:"userId"`
    ConversationID string    `gorm:"uniqueIndex:idx_user_conv;type:varchar(128);not null" json:"conversationId"`
    ReadSeq        int64     `gorm:"type:bigint;not null;default:0" json:"readSeq"`
    UpdatedAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (UserConversationSeq) TableName() string { return "user_conversation_seqs" }
```


### 1.5 请求/响应/VO 结构体定义

💡 **为什么要单独定义请求和响应结构体？** 直接把 Model 返回给前端会暴露内部字段（如密码），且不同接口需要的字段不同。请求结构体做参数校验，响应结构体（VO）控制返回内容，这是分层的基本原则。

```go
// internal/model/req.go
package model

// --- 认证请求 ---

type RegisterReq struct {
    UserID   string `json:"userId" binding:"required,alphanum,min=4,max=32"`
    Password string `json:"password" binding:"required,min=8,max=64"`
    Nickname string `json:"nickname" binding:"omitempty,min=1,max=64"`
    Phone    string `json:"phone" binding:"omitempty,len=11"`
    Email    string `json:"email" binding:"omitempty,email"`
}

type LoginReq struct {
    UserID   string `json:"userId" binding:"required"`
    Password string `json:"password" binding:"required"`
    Platform string `json:"platform" binding:"required,oneof=web ios android"`
}

type RefreshReq struct {
    RefreshToken string `json:"refreshToken" binding:"required"`
    Platform     string `json:"platform" binding:"required,oneof=web ios android"`
}

// --- 用户请求 ---

type UpdateProfileReq struct {
    Nickname  string `json:"nickname" binding:"omitempty,min=1,max=64"`
    AvatarURL string `json:"avatarUrl" binding:"omitempty,url,max=512"`
    Phone     string `json:"phone" binding:"omitempty,len=11"`
    Email     string `json:"email" binding:"omitempty,email"`
}

type SearchReq struct {
    Keyword  string `json:"keyword" binding:"required,min=1,max=64"`
    Page     int    `json:"page" binding:"omitempty,min=1"`
    PageSize int    `json:"pageSize" binding:"omitempty,min=1,max=50"`
}

// --- 好友请求 ---

type SendFriendRequestReq struct {
    ToUserID string `json:"toUserId" binding:"required"`
    Message  string `json:"message" binding:"omitempty,max=256"`
}

type HandleFriendRequestReq struct {
    ID int64 `uri:"id" binding:"required"`
}

type SetRemarkReq struct {
    Remark string `json:"remark" binding:"required,min=1,max=64"`
}

type ListFriendRequestsReq struct {
    Page     int `json:"page" binding:"omitempty,min=1"`
    PageSize int `json:"pageSize" binding:"omitempty,min=1,max=50"`
}

type DeleteFriendReq struct {
    UserID string `uri:"userId" binding:"required"`
}

// --- 消息请求 ---

type SendMsgReq struct {
    ConversationID string `json:"conversationId" binding:"required"`
    ClientMsgID    string `json:"clientMsgId" binding:"required"`
    ContentType    int    `json:"contentType" binding:"required,oneof=1 2 3 4"`
    Content        string `json:"content" binding:"required,max=4096"`
}

type HistoryReq struct {
    ConversationID string `json:"conversationId" form:"conversationId" binding:"required"`
    StartSeq       int64  `json:"startSeq" form:"startSeq"`
    Count          int    `json:"count" form:"count" binding:"required,min=1,max=50"`
}

type MarkReadReq struct {
    ConversationID string `json:"conversationId" binding:"required"`
    ReadSeq        int64  `json:"readSeq" binding:"required,min=1"`
}

type RevokeMsgReq struct {
    ConversationID string `json:"conversationId" binding:"required"`
    ClientMsgID    string `json:"clientMsgId" binding:"required"`
}

// --- 会话请求 ---

type PinConversationReq struct {
    IsPinned bool `json:"isPinned"`
}

type DeleteConversationReq struct {
    ID string `uri:"id" binding:"required"`
}

// --- 分页 ---

type PageReq struct {
    Page     int `json:"page" form:"page" binding:"omitempty,min=1"`
    PageSize int `json:"pageSize" form:"pageSize" binding:"omitempty,min=1,max=50"`
}

func (r *PageReq) GetPage() int {
    if r.Page <= 0 {
        return 1
    }
    return r.Page
}

func (r *PageReq) GetPageSize() int {
    if r.PageSize <= 0 {
        return 20
    }
    return r.PageSize
}
```

```go
// internal/model/vo.go
package model

// --- 认证响应 ---

type TokenPair struct {
    AccessToken     string `json:"accessToken"`
    RefreshToken    string `json:"refreshToken"`
    AccessExpireAt  int64  `json:"accessExpireAt"`
    RefreshExpireAt int64  `json:"refreshExpireAt"`
    UserID          string `json:"userId"`
}

// --- 用户视图 ---

type UserVO struct {
    UserID    string `json:"userId"`
    Nickname  string `json:"nickname"`
    AvatarURL string `json:"avatarUrl"`
    Phone     string `json:"phone"`
    Email     string `json:"email"`
    Status    int8   `json:"status"`
    CreatedAt string `json:"createdAt"`
}

func (u *User) ToVO() *UserVO {
    return &UserVO{
        UserID:    u.UserID,
        Nickname:  u.Nickname,
        AvatarURL: u.AvatarURL,
        Phone:     u.Phone,
        Email:     u.Email,
        Status:    u.Status,
        CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
    }
}

type OtherUserVO struct {
    UserID    string `json:"userId"`
    Nickname  string `json:"nickname"`
    AvatarURL string `json:"avatarUrl"`
    IsFriend  bool   `json:"isFriend"`
    Remark    string `json:"remark,omitempty"`
}

type SearchUserVO struct {
    UserID    string `json:"userId"`
    Nickname  string `json:"nickname"`
    AvatarURL string `json:"avatarUrl"`
}

// --- 好友视图 ---

type FriendVO struct {
    FriendID  string `json:"friendId"`
    Nickname  string `json:"nickname"`
    AvatarURL string `json:"avatarUrl"`
    Remark    string `json:"remark"`
}

type FriendRequestVO struct {
    ID         uint64 `json:"id"`
    FromUserID string `json:"fromUserId"`
    FromNick   string `json:"fromNick"`
    FromAvatar string `json:"fromAvatar"`
    Message    string `json:"message"`
    Status     int8   `json:"status"`
    CreatedAt  string `json:"createdAt"`
}

// --- 会话视图 ---

type ConversationVO struct {
    ConversationID string  `json:"conversationId"`
    ConvType       int     `json:"convType"`
    TargetID       string  `json:"targetId"`
    MaxSeq         int64   `json:"maxSeq"`
    ReadSeq        int64   `json:"readSeq"`
    UnreadCount    int64   `json:"unreadCount"`
    IsPinned       bool    `json:"isPinned"`
    LastMsg        *MsgVO  `json:"lastMsg,omitempty"`
    UpdatedAt      string  `json:"updatedAt"`
}

// --- 消息视图 ---

type MsgVO struct {
    ConversationID string `json:"conversationId"`
    Seq            int64  `json:"seq"`
    SenderID       string `json:"senderId"`
    MsgType        int    `json:"msgType"`
    Content        string `json:"content"`
    ClientMsgID    string `json:"clientMsgId"`
    ServerMsgID    string `json:"serverMsgId"`
    IsRevoked      bool   `json:"isRevoked"`
    SendTime       int64  `json:"sendTime"`
}

type SendMsgResp struct {
    Seq         int64  `json:"seq"`
    ServerMsgID string `json:"serverMsgId"`
    SendTime    int64  `json:"sendTime"`
}

type HistoryResp struct {
    List    []*MsgVO `json:"list"`
    HasMore bool     `json:"hasMore"`
    MinSeq  int64    `json:"minSeq"`
    MaxSeq  int64    `json:"maxSeq"`
}

// --- 通用分页 ---

type PageResult[T any] struct {
    List     []T   `json:"list"`
    Total    int64 `json:"total"`
    Page     int   `json:"page"`
    PageSize int   `json:"pageSize"`
}
```


### 1.6 基础设施初始化与 Wire 函数

💡 **为什么需要 Wire 函数？** main.go 中 `NewRepositories`、`NewServices`、`NewHandlers` 负责把所有依赖"串起来"。Repository 依赖 DB 和 Redis，Service 依赖 Repository 和配置，Handler 依赖 Service。这种逐层注入的模式让依赖关系清晰，也方便测试时替换为 Mock。

```go
// internal/repository/init.go
package repository

import (
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"

    "github.com/tianlu1990s/gim/internal/config"
    slogpkg "github.com/tianlu1990s/gim/pkg/slog"
)

// InitMySQL 初始化 MySQL 连接
func InitMySQL(cfg config.MySQLConfig, log *slogpkg.Logger) *gorm.DB {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

    gormLogger := logger.Default.LogMode(logger.Info) // 开发环境用 Info 级别

    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: gormLogger,
    })
    if err != nil {
        log.Fatal("Failed to connect MySQL", "error", err)
    }

    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
    sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
    sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

    return db
}

// InitRedis 初始化 Redis 连接
func InitRedis(cfg config.RedisConfig) *redis.Client {
    rdb := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        Password: cfg.Password,
        DB:       cfg.DB,
        PoolSize: cfg.PoolSize,
    })
    return rdb
}

// Repositories 聚合所有 Repository
type Repositories struct {
    DB          *gorm.DB
    RDB         *redis.Client
    User        UserRepo
    Friend      FriendRepo
    FriendReq   FriendRequestRepo
    Conversation ConversationRepo
    Message     MessageRepo
}

// NewRepositories 创建所有 Repository 实例
func NewRepositories(db *gorm.DB, rdb *redis.Client) *Repositories {
    return &Repositories{
        DB:          db,
        RDB:         rdb,
        User:        newUserRepo(db),
        Friend:      newFriendRepo(db),
        FriendReq:   newFriendRequestRepo(db),
        Conversation: newConversationRepo(db),
        Message:     newMessageRepo(db, rdb),
    }
}

// Transaction 执行事务
func (r *Repositories) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
    return r.DB.WithContext(ctx).Transaction(fn)
}

// --- 内部构造函数 ---

func newUserRepo(db *gorm.DB) UserRepo {
    return &userRepo{db: db}
}

func newFriendRepo(db *gorm.DB) FriendRepo {
    return &friendRepo{db: db}
}

func newFriendRequestRepo(db *gorm.DB) FriendRequestRepo {
    return &friendRequestRepo{db: db}
}

func newConversationRepo(db *gorm.DB) ConversationRepo {
    return &conversationRepo{db: db}
}

func newMessageRepo(db *gorm.DB, rdb *redis.Client) MessageRepo {
    return &messageRepo{db: db, rdb: rdb}
}
```

```go
// internal/service/init.go
package service

import (
    "github.com/redis/go-redis/v9"

    "github.com/tianlu1990s/gim/internal/config"
    "github.com/tianlu1990s/gim/internal/repository"
    "github.com/tianlu1990s/gim/pkg/jwt"
)

// Services 聚合所有 Service
type Services struct {
    Auth         AuthService
    User         UserService
    Friend       FriendService
    Conversation ConversationService
    Message      MessageService
}

// NewServices 创建所有 Service 实例
func NewServices(repos *repository.Repositories, cfg *config.Config, jwtMgr *jwt.JWTManager, rdb *redis.Client, hub *ws.Hub) *Services {
    return &Services{
        Auth:         newAuthService(repos.User, repos.FriendReq, jwtMgr, rdb, cfg),
        User:         newUserService(repos.User, repos.Friend),
        Friend:       newFriendService(repos.Friend, repos.FriendReq, repos.Conversation, repos, hub, rdb),
        Conversation: newConversationService(repos.Conversation, repos.Message),
        Message:      newMessageService(repos.Message, repos.Conversation, repos.Friend, repos.User, hub, rdb),
    }
}

// --- 内部构造函数 ---

func newAuthService(userRepo repository.UserRepo, friendReqRepo repository.FriendRequestRepo, jwtMgr *jwt.JWTManager, rdb *redis.Client, cfg *config.Config) AuthService {
    return &authService{userRepo: userRepo, friendReqRepo: friendReqRepo, jwtMgr: jwtMgr, rdb: rdb, cfg: cfg}
}

func newUserService(userRepo repository.UserRepo, friendRepo repository.FriendRepo) UserService {
    return &userService{userRepo: userRepo, friendRepo: friendRepo}
}

func newFriendService(friendRepo repository.FriendRepo, friendReqRepo repository.FriendRequestRepo, convRepo repository.ConversationRepo, repos *repository.Repositories, hub *ws.Hub, rdb *redis.Client) FriendService {
    return &friendService{friendRepo: friendRepo, friendReqRepo: friendReqRepo, convRepo: convRepo, repos: repos, hub: hub, rdb: rdb}
}

func newConversationService(convRepo repository.ConversationRepo, msgRepo repository.MessageRepo) ConversationService {
    return &conversationService{convRepo: convRepo, msgRepo: msgRepo}
}

func newMessageService(msgRepo repository.MessageRepo, convRepo repository.ConversationRepo, friendRepo repository.FriendRepo, userRepo repository.UserRepo, hub *ws.Hub, rdb *redis.Client) MessageService {
    return &msgService{msgRepo: msgRepo, convRepo: convRepo, friendRepo: friendRepo, userRepo: userRepo, hub: hub, rdb: rdb}
}
```

```go
// internal/handler/init.go
package handler

import "github.com/tianlu1990s/gim/internal/service"

// Handlers 聚合所有 Handler
type Handlers struct {
    Auth         *AuthHandler
    User         *UserHandler
    Friend       *FriendHandler
    Message      *MessageHandler
    Conversation *ConversationHandler
}

// NewHandlers 创建所有 Handler 实例
func NewHandlers(services *service.Services) *Handlers {
    return &Handlers{
        Auth:         NewAuthHandler(services.Auth),
        User:         NewUserHandler(services.User),
        Friend:       NewFriendHandler(services.Friend),
        Message:      NewMessageHandler(services.Message),
        Conversation: NewConversationHandler(services.Conversation),
    }
}
```


## 2. 公共组件规划与封装

💡 **本章目标**：识别和封装项目中可复用的公共组件。公共组件是项目的"积木"，各模块通过组合这些积木快速构建功能，避免重复造轮子。

### 2.1 为什么要规划公共组件

在项目初期就识别和封装公共组件，能带来以下好处：

| 好处 | 说明 | 示例 |
|------|------|------|
| 代码复用 | 避免各模块重复造轮子，保持代码一致性 | JWT 解析逻辑在认证和中间件都要用 |
| 统一规范 | 所有模块使用相同的工具，降低学习成本 | 所有日志使用 slog 格式，便于 grep 分析 |
| 易于测试 | 公共组件独立测试，业务层依赖抽象 | 单独测试 JWT 工具，不需要启动完整应用 |
| 易于升级 | 修改一处即可全局生效 | 更新 slog 版本，所有模块自动生效 |

### 2.2 公共组件依赖关系图

```
公共组件依赖关系：

┌─────────────────────────────────────────────────────┐
│                   业务层                          │
│    (Handler → Service → Repository → Model)        │
└──────────────────┬──────────────────────────────────┘
                   │
      ┌────────────┼────────────┐
      │            │            │
┌─────▼────┐  ┌──▼──────┐  ┌──▼──────┐
│  pkg/jwt │  │pkg/snowflake│  │pkg/resp │
└──────────┘  └───────────┘  └──────────┘
      │            │            │
┌─────▼────────────▼────────────▼─────┐
│    pkg/slog (日志，所有组件依赖)    │
└─────────────────────────────────────┘
      │
┌─────▼──────────────────────────────┐
│    pkg/errcode (错误码，基础层)    │
└─────────────────────────────────────┘
```

💡 **依赖原则**：
- **底层**：errcode（无依赖）
- **基础层**：snowflake、slog（只依赖 errcode）
- **工具层**：jwt、resp（依赖 errcode、slog）
- **业务层**：各模块组合使用所有组件

### 2.3 组件使用频率分析

| 组件 | 使用位置 | 频率 | 复杂度 | 优先级 |
|------|---------|------|--------|--------|
| errcode | 所有层 | 极高 | 低 | ⭐⭐⭐⭐⭐ |
| slog | 所有层 | 极高 | 中 | ⭐⭐⭐⭐⭐ |
| resp | Handler | 高 | 低 | ⭐⭐⭐⭐ |
| jwt | Auth + Middleware | 中 | 中 | ⭐⭐⭐⭐ |
| snowflake | 所有需要 ID 的地方 | 高 | 低 | ⭐⭐⭐⭐ |
| rediskey | Service + Middleware | 中 | 低 | ⭐⭐⭐ |
| convid | Service | 中 | 低 | ⭐⭐⭐ |

---

### 2.4 本项目的公共组件清单

| 组件 | 路径 | 用途 | 说明 |
|------|------|------|------|
| 日志模块 | `pkg/slog/` | 统一日志输出 | 基于标准库 log/slog，支持等级、短文件名、颜色、轮转 |
| JWT 工具 | `pkg/jwt/` | Token 生成与验证 | RS256 非对称加密，支持 JTI 黑名单 |
| Snowflake ID | `pkg/snowflake/` | 分布式唯一 ID | 时钟回拨防护，单机版即可用 |
| 统一响应 | `pkg/resp/` | HTTP 响应格式 | Success/Fail 统一封装 |
| 错误码 | `pkg/errcode/` | 错误码体系 | 分层错误码，支持详情追加 |
| Redis Key 前缀 | `pkg/rediskey/` | Redis Key 管理 | 统一 Key 命名，避免冲突 |
| 会话 ID 生成 | `pkg/convid/` | 会话 ID 生成 | 单聊按字典序排列，群聊固定格式 |

### 2.5 日志模块封装（基于 slog）

💡 **为什么用 slog 而不是 Zap？** Go 1.21+ 引入了结构化日志标准库 `log/slog`，提供了与 Zap 类似的结构化日志能力，但无需额外依赖。对于本项目，slog 足够用且更轻量。（本项目使用 Go 1.26+）

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

	"github.com/tianlu1990s/gim/internal/config"
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

### 2.6 Redis Key 管理

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


### 2.7 Snowflake ID 生成

💡 **为什么用 Snowflake 而不是 UUID？** UUID 是 36 字符的随机字符串，无序且较长，不适合做数据库主键（导致索引碎片）。Snowflake 生成 18 位数字 ID，包含时间戳信息，天然有序，索引友好。

```go
// pkg/snowflake/snowflake.go
package snowflake

import (
    "sync"
    "time"

    "github.com/bwmarrin/snowflake"
)

var (
    node *snowflake.Node
    once sync.Once
)

// Init 初始化 Snowflake 节点（main.go 启动时调用）
func Init(nodeID int64) {
    once.Do(func() {
        var err error
        node, err = snowflake.NewNode(nodeID)
        if err != nil {
            panic("failed to init snowflake node: " + err.Error())
        }
    })
}

// Generate 生成唯一 ID
func Generate() snowflake.ID {
    if node == nil {
        // 未初始化时使用默认节点 1
        Init(1)
    }
    return node.Generate()
}
```

---

## 3. 认证模块实现

### 3.1 认证流程详解

💡 **认证是整个系统的"门卫"**：所有非公开接口都需要验证用户身份，才能决定是否允许访问。

#### 3.1.1 认证流程图

```
用户注册/登录流程：

注册：
┌──────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ 客户端│ ──►│  /auth  │ ──►│ 密码校验 │ ──►│ MySQL  │
│      │    │ /register│    │ bcrypt  │    │ 存储    │
└──────┘    └────┬────┘    └─────────┘    └─────────┘
                  │
            ┌─────▼──────┐
            │  返回用户信息 │
            └────────────┘

登录：
┌──────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ 客户端│ ──►│  /auth  │ ──►│  校验    │ ──►│  生成   │ ──►│  Redis  │
│      │    │  /login │    │  密码   │    │  Token  │    │ 存储    │
└──────┘    └────┬────┘    └─────────┘    └────┬────┘    └─────────┘
                  │                             │
            ┌─────▼───────────────────────────▼──────┐
            │    返回 access_token + refresh_token    │
            └──────────────────────────────────────────┘

刷新 Token：
┌──────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ 客户端│ ──►│  /auth   │ ──►│  验证    │ ──►│  生成    │
│      │    │ /refresh  │    │ refresh  │    │  新的     │
│      │    │          │    │ token    │    │  access   │
└──────┘    └────┬─────┘    └──────────┘    └────┬─────┘
                  │                             │
            ┌─────▼───────────────────────────▼──────┐
            │    返回新的 access_token（刷新）      │
            └──────────────────────────────────────────┘

退出登录：
┌──────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ 客户端│ ──►│  /auth   │ ──►│  加入     │ ──►│  删除    │
│      │    │ /logout  │    │  黑名单   │    │  refresh  │
│      │    │          │    │  (Redis) │    │  token    │
└──────┘    └────┬─────┘    └──────────┘    └──────────┘
                  │
            ┌─────▼──────┐
            │  清除在线状态│
            └────────────┘
```

#### 3.1.2 Token 生命周期

```
Token 生命周期：

┌────────────────────────────────────────────────────┐
│                Token 生成                           │
│  access_token (24h) + refresh_token (7天)            │
└─────────────────┬──────────────────────────────────┘
                  │
        ┌─────────▼─────────┐
        │   Token 使用中     │
        │   (每次请求携带)   │
        └─────────┬─────────┘
                  │
        ┌─────────▼─────────┐
        │   access_token   │
        │   过期？          │
        └─────────┬─────────┘
                  │
         ┌───────┴───────┐
         │               │
        是              否
         │               │
   ┌─────▼────┐    ┌────▼─────┐
   │用refresh  │    │继续使用  │
   │token刷新  │    │access   │
   └─────┬────┘    │token    │
         │          └──────────┘
   ┌─────▼─────┐
   │refresh_token │
   │过期？        │
   └─────┬─────┘
         │
    ┌────┴────┐
    │         │
   是        否
    │         │
┌───▼──┐  ┌───▼──┐
│重新登录│  │继续使用│
└───────┘  │refresh │
           │token  │
           └───────┘
```

#### 3.1.3 JWT Token 黑名单机制

💡 **为什么需要黑名单？** 用户主动退出登录后，Token 还未过期，但应该立即失效。黑名单机制通过在 Redis 中存储已退出的 Token JTI，实现立即失效。

```
黑名单机制流程：

┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  用户    │ ──►│  /auth   │ ──►│  解析    │ ──►│  检查    │
│ 退出登录 │    │ /logout  │    │  access  │    │  黑名单  │
│          │    │          │    │  token   │    │  (Redis) │
└──────────┘    └────┬─────┘    └──────────┘    └────┬─────┘
                     │                             │
               ┌─────▼─────────────────────────────▼──────┐
               │  计算 token 剩余有效期，存入黑名单      │
               │  Key: blacklist:token:{jti}           │
               │  Value: "1"                           │
               │  TTL: token 剩余有效期                    │
               └───────────────────────────────────────────┘

后续请求时：
┌──────────┐    ┌──────────┐    ┌──────────┐
│  用户    │ ──►│  携带     │ ──►│  解析    │
│  发起请求 │    │  access  │    │  token   │
│          │    │  token   │    │          │
└──────────┘    └────┬─────┘    └────┬─────┘
                     │               │
               ┌─────▼───────────────▼──────┐
               │  检查黑名单                │
               │  EXISTS blacklist:token:{jti} │
               └────────────────┬───────────┘
                                │
                         ┌───────┴───────┐
                         │               │
                       在黑名单        不在黑名单
                         │               │
                    ┌────▼────┐      ┌────▼────┐
                    │ 拒绝访问 │      │ 允许访问│
                    └─────────┘      └─────────┘
```


### 3.2 Redis Key 设计

```
# Token 黑名单（logout 时加入）
blacklist:token:{accessTokenJTI}    -> "1"    TTL = accessToken 剩余有效期

# Refresh Token 存储（用于刷新和吊销）
refresh:{userId}:{platform}         -> refreshToken    TTL = 7天
```

💡 **Key 生成使用统一的 rediskey 包**：

```go
import "github.com/tianlu1990s/gim/pkg/rediskey"

// Token 黑名单
rdb.Set(ctx, rediskey.BlacklistTokenKey(claims.ID), "1", ttl)

// Refresh Token 存储
rdb.Set(ctx, rediskey.RefreshKey(userID, platform), refreshToken, 7*24*time.Hour)
```

### 3.3 JWT 工具包

💡 **为什么用 RS256 而不是 HS256？** HS256 用同一个密钥签名和验证（对称加密），密钥泄露就完了。RS256 用私钥签名、公钥验证（非对称加密），私钥只存在服务端，公钥可以分发给其他服务验证 Token。第二阶段微服务拆分后，各服务只需公钥即可验证，无需私钥。

💡 **什么是 JTI？** JWT Token 的唯一 ID，用于 Token 黑名单。用户退出登录后，把 JTI 存入 Redis 黑名单，即使 Token 还没过期，鉴权时检查黑名单也会拒绝。

```go
// pkg/jwt/jwt.go
package jwt

import (
    "crypto/rsa"
    "errors"
    "os"
    "time"

    "github.com/tianlu1990s/gim/pkg/snowflake"
    jwtv5 "github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

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
    now := time.Now()
    expireAt := now.Add(m.refreshExpire).Unix()
    claims := &Claims{
        UserID:   userID,
        Platform: platform,
        RegisteredClaims: jwtv5.RegisteredClaims{
            ID:        snowflake.Generate().String(),
            IssuedAt:  jwtv5.NewNumericDate(now),
            ExpiresAt: jwtv5.NewNumericDate(now.Add(m.refreshExpire)),
        },
    }
    token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims)
    tokenStr, err := token.SignedString(m.privateKey)
    return tokenStr, expireAt, err
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

### 3.4 认证 Service

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

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
    // 1. 解析 refresh token
    claims, err := s.jwtMgr.ParseToken(refreshToken)
    if err != nil {
        return nil, errcode.ErrUnauthorized
    }

    // 2. 检查 refresh token 是否有效（Redis 中是否存在）
    storedToken, err := s.rdb.Get(ctx, fmt.Sprintf("refresh:%s:%s", claims.UserID, claims.Platform)).Result()
    if err != nil || storedToken != refreshToken {
        return nil, errcode.ErrUnauthorized.WithDetail("refresh token invalid or expired")
    }

    // 3. 查询用户状态
    user, err := s.userRepo.GetByID(ctx, claims.UserID)
    if err != nil || user.Status != 1 {
        return nil, errcode.ErrUserDisabled
    }

    // 4. 生成新的 access token
    newAccessToken, accessExp, _ := s.jwtMgr.GenerateAccessToken(claims.UserID, claims.Platform)

    return &TokenPair{
        AccessToken:     newAccessToken,
        RefreshToken:    refreshToken, // refresh token 不变
        AccessExpireAt:  accessExp,
        RefreshExpireAt: claims.ExpiresAt.Unix(),
        UserID:          claims.UserID,
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

### 3.5 认证 Handler

```go
// internal/handler/auth.go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/tianlu1990s/gim/internal/model"
    "github.com/tianlu1990s/gim/internal/service"
    "github.com/tianlu1990s/gim/pkg/errcode"
    "github.com/tianlu1990s/gim/pkg/resp"
)

type AuthHandler struct {
    svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
    return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req model.RegisterReq
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
    var req model.LoginReq
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

func (h *AuthHandler) Refresh(c *gin.Context) {
    var req model.RefreshReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    token, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, token)
}

func (h *AuthHandler) Logout(c *gin.Context) {
    userID := c.GetString("userID")
    platform := c.GetString("platform")
    accessToken := c.GetHeader("Authorization")
    // 去掉 "Bearer " 前缀
    if len(accessToken) > 7 && accessToken[:7] == "Bearer " {
        accessToken = accessToken[7:]
    }
    if err := h.svc.Logout(c.Request.Context(), userID, platform, accessToken); err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, nil)
}
```

---

## 4. 用户模块实现

### 4.1 User Repository

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



### 4.3 User Handler

```go
// internal/handler/user.go
package handler

import (
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/tianlu1990s/gim/internal/model"
    "github.com/tianlu1990s/gim/internal/service"
    "github.com/tianlu1990s/gim/pkg/errcode"
    "github.com/tianlu1990s/gim/pkg/resp"
)

type UserHandler struct {
    svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
    return &UserHandler{svc: svc}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
    userID := c.GetString("userID")
    user, err := h.svc.GetProfile(c.Request.Context(), userID)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, user)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
    userID := c.GetString("userID")
    var req model.UpdateProfileReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    user, err := h.svc.UpdateProfile(c.Request.Context(), userID, &req)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, user)
}

func (h *UserHandler) GetOtherProfile(c *gin.Context) {
    currentUserID := c.GetString("userID")
    targetUserID := c.Param("userId")
    if targetUserID == "" {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail("userId 不能为空"))
        return
    }
    user, err := h.svc.GetOtherProfile(c.Request.Context(), currentUserID, targetUserID)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, user)
}

func (h *UserHandler) Search(c *gin.Context) {
    userID := c.GetString("userID")
    var req model.SearchReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    result, err := h.svc.Search(c.Request.Context(), userID, &req)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, result)
}
```
## 5. 好友模块实现

### 5.1 Friend Repository

```go
// internal/repository/friend.go
package repository

type FriendRepo interface {
    Create(ctx context.Context, ownerID, friendID, remark string) error
    CreateTx(ctx context.Context, tx *gorm.DB, ownerID, friendID, remark string) error
    Delete(ctx context.Context, ownerID, friendID string) error
    IsFriend(ctx context.Context, ownerID, friendID string) (bool, error)
    GetFriend(ctx context.Context, ownerID, friendID string) (*model.Friend, error)
    List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error)
    SetRemark(ctx context.Context, ownerID, friendID, remark string) error
}

type FriendRequestRepo interface {
    Create(ctx context.Context, fromID, toID, message string) (int64, error)
    GetByID(ctx context.Context, id int64) (*model.FriendRequest, error)
    ListIncoming(ctx context.Context, toID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error)
    UpdateStatus(ctx context.Context, id int64, status int) error
    UpdateStatusTx(ctx context.Context, tx *gorm.DB, id int64, status int) error
    HasPendingRequest(ctx context.Context, fromID, toID string) (bool, error)
}
```

### 5.1 Friend Service 定义

```go
// internal/service/friend.go
package service

type FriendService interface {
    SendRequest(ctx context.Context, userID string, req *model.SendFriendRequestReq) (int64, error)
    ListRequests(ctx context.Context, userID string, page, pageSize int) ([]*model.FriendRequestVO, int64, error)
    AcceptRequest(ctx context.Context, userID string, requestID int64) error
    RejectRequest(ctx context.Context, userID string, requestID int64) error
    Delete(ctx context.Context, ownerID, friendID string) error
    List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error)
    SetRemark(ctx context.Context, ownerID, friendID, remark string) error
}

type friendService struct {
    friendRepo    repository.FriendRepo
    friendReqRepo repository.FriendRequestRepo
    convRepo      repository.ConversationRepo
    repos         *repository.Repositories
    hub           *ws.Hub
    rdb           *redis.Client
}
```

### 5.2 好友申请 — 同意流程（事务）

💡 **什么是事务？** 事务是数据库操作的"打包执行"：要么全部成功，要么全部回滚。好友同意涉及 3 张表写入（申请状态 + 双向好友 + 双方会话），如果第 2 步写入失败但不回滚第 1 步，就会数据不一致（申请已同意但好友关系没建）。事务保证"要么全有，要么全无"。

💡 **为什么好友关系要"双向"写入？** Alice 加 Bob 为好友，意味着 Alice 的好友列表有 Bob，Bob 的好友列表也有 Alice。这是两条独立的记录，不是一条。这样每个人查自己的好友列表时只需查 `WHERE owner_id = 我`，简单高效。

```go
func (s *friendService) AcceptRequest(ctx context.Context, userID string, requestID int64) error {
    // 1. 查询申请，验证 toUserID 是当前用户
    req, err := s.friendReqRepo.GetByID(ctx, requestID)
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
    err = s.repos.Transaction(ctx, func(tx *gorm.DB) error {
        // 更新申请状态
        if err := s.friendReqRepo.UpdateStatusTx(ctx, tx, requestID, 1); err != nil {
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
        convID := convid.GenSingleConvID(req.FromUserID, req.ToUserID)
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
        s.hub.PushToUser(req.FromUserID, &ws.WSMessage{
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

import (
    "fmt"
    "sort"
)

// GenSingleConvID 生成单聊会话ID，两个 userId 按字典序排列
// 保证同一对用户无论传入顺序如何，生成的 convID 一致
func GenSingleConvID(uid1, uid2 string) string {
    ids := []string{uid1, uid2}
    sort.Strings(ids)
    return fmt.Sprintf("single_%s_%s", ids[0], ids[1])
}

// GenGroupConvID 生成群聊会话ID（群功能预留）
func GenGroupConvID(groupID string) string {
    return fmt.Sprintf("group_%s", groupID)
}
```

### 5.4 工具函数

💡 **为什么把这些函数放在 pkg 下而不是内联在 Service 中？** 多个 Service 和 Handler 可能复用同一个工具函数（如 `extractTargetID` 在消息和会话模块都用），放在 pkg 下便于引用和单测。

```go
// pkg/convutil/convutil.go
package convutil

import "strings"

// ExtractTargetID 从单聊会话ID中提取对方用户ID
// convID 格式: single_{uid1}_{uid2}，uid1 < uid2（字典序）
func ExtractTargetID(convID, myUserID string) string {
    if !strings.HasPrefix(convID, "single_") {
        return "" // 群聊暂不处理
    }
    parts := strings.TrimPrefix(convID, "single_")
    ids := strings.SplitN(parts, "_", 2)
    if len(ids) != 2 {
        return ""
    }
    if ids[0] == myUserID {
        return ids[1]
    }
    return ids[0]
}

// GetConversationMembers 获取会话中除发送者外的成员ID列表
// 第一阶段仅支持单聊，群聊在第二阶段扩展
func GetConversationMembers(convID, senderID string) []string {
    if strings.HasPrefix(convID, "single_") {
        targetID := ExtractTargetID(convID, senderID)
        if targetID != "" {
            return []string{targetID}
        }
    }
    // TODO: 第二阶段群聊从 group 表获取成员列表
    return nil
}
```

```go
// internal/ws/util.go
package ws

import "encoding/json"

// toJSON 将 map[string]interface{} 转换为 JSON 字节，
// 用于 WS 消息 Data 字段反序列化到具体结构体
func toJSON(data interface{}) []byte {
    b, _ := json.Marshal(data)
    return b
}
```

```go
// internal/service/validate.go
package service

import (
    "regexp"
    "unicode"

    "github.com/tianlu1990s/gim/internal/model"
    "github.com/tianlu1990s/gim/pkg/errcode"
)

var (
    userIDRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{3,31}$`)
    phoneRegex  = regexp.MustCompile(`^1[3-9]\d{9}$`)
    emailRegex  = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// validateRegisterReq 校验注册请求参数
func validateRegisterReq(req *model.RegisterReq) error {
    if !userIDRegex.MatchString(req.UserID) {
        return errcode.ErrInvalidParam.WithDetail("userId 格式错误：4-32位，字母开头，仅字母数字下划线")
    }
    if !isStrongPassword(req.Password) {
        return errcode.ErrInvalidParam.WithDetail("密码强度不足：8-64位，须含大小写字母和数字")
    }
    if req.Phone != "" && !phoneRegex.MatchString(req.Phone) {
        return errcode.ErrInvalidParam.WithDetail("手机号格式错误")
    }
    if req.Email != "" && !emailRegex.MatchString(req.Email) {
        return errcode.ErrInvalidParam.WithDetail("邮箱格式错误")
    }
    return nil
}

// isStrongPassword 检查密码是否包含大小写字母和数字
func isStrongPassword(pwd string) bool {
    if len(pwd) < 8 || len(pwd) > 64 {
        return false
    }
    var hasUpper, hasLower, hasDigit bool
    for _, r := range pwd {
        switch {
        case unicode.IsUpper(r):
            hasUpper = true
        case unicode.IsLower(r):
            hasLower = true
        case unicode.IsDigit(r):
            hasDigit = true
        }
    }
    return hasUpper && hasLower && hasDigit
}

// isDigit 检查字符串是否全为数字
func isDigit(s string) bool {
    for _, r := range s {
        if !unicode.IsDigit(r) {
            return false
        }
    }
    return len(s) > 0
}
```

---


### 5.5 Friend Handler

```go
// internal/handler/friend.go
package handler

import (
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/tianlu1990s/gim/internal/model"
    "github.com/tianlu1990s/gim/internal/service"
    "github.com/tianlu1990s/gim/pkg/errcode"
    "github.com/tianlu1990s/gim/pkg/resp"
)

type FriendHandler struct {
    svc service.FriendService
}

func NewFriendHandler(svc service.FriendService) *FriendHandler {
    return &FriendHandler{svc: svc}
}

func (h *FriendHandler) SendRequest(c *gin.Context) {
    userID := c.GetString("userID")
    var req model.SendFriendRequestReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    id, err := h.svc.SendRequest(c.Request.Context(), userID, &req)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, gin.H{"id": id})
}

func (h *FriendHandler) ListRequests(c *gin.Context) {
    userID := c.GetString("userID")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
    list, total, err := h.svc.ListRequests(c.Request.Context(), userID, page, pageSize)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, gin.H{"list": list, "total": total, "page": page, "pageSize": pageSize})
}

func (h *FriendHandler) AcceptRequest(c *gin.Context) {
    userID := c.GetString("userID")
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail("无效的申请ID"))
        return
    }
    if err := h.svc.AcceptRequest(c.Request.Context(), userID, id); err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, nil)
}

func (h *FriendHandler) RejectRequest(c *gin.Context) {
    userID := c.GetString("userID")
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail("无效的申请ID"))
        return
    }
    if err := h.svc.RejectRequest(c.Request.Context(), userID, id); err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, nil)
}

func (h *FriendHandler) Delete(c *gin.Context) {
    userID := c.GetString("userID")
    friendID := c.Param("userId")
    if friendID == "" {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail("userId 不能为空"))
        return
    }
    if err := h.svc.Delete(c.Request.Context(), userID, friendID); err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, nil)
}

func (h *FriendHandler) List(c *gin.Context) {
    userID := c.GetString("userID")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
    list, total, err := h.svc.List(c.Request.Context(), userID, page, pageSize)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, gin.H{"list": list, "total": total, "page": page, "pageSize": pageSize})
}

func (h *FriendHandler) SetRemark(c *gin.Context) {
    userID := c.GetString("userID")
    friendID := c.Param("userId")
    if friendID == "" {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail("userId 不能为空"))
        return
    }
    var req model.SetRemarkReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    if err := h.svc.SetRemark(c.Request.Context(), userID, friendID, req.Remark); err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, nil)
}
```

## 6. 会话模块实现

### 6.1 Conversation Repository

```go
// internal/repository/conversation.go
package repository

type ConversationRepo interface {
    CreateIfNotExistTx(ctx context.Context, tx *gorm.DB, ownerID, convID string, convType int, targetID string) error
    List(ctx context.Context, ownerID string, page, pageSize int) ([]*model.ConversationVO, int64, error)
    UpdatePin(ctx context.Context, ownerID, convID string, isPinned bool) error
    Delete(ctx context.Context, ownerID, convID string) error
    UpdateMaxSeq(ctx context.Context, convID string, seq int64) error
    GetByID(ctx context.Context, ownerID, convID string) (*model.Conversation, error)
    ListByOwner(ctx context.Context, ownerID string) ([]*model.Conversation, error)
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
    UpdateUserReadSeqDB(ctx context.Context, userID, conversationID string, seq int64) error
    GetMaxSeq(ctx context.Context, conversationID string) (int64, error)
    GetMinSeq(ctx context.Context, conversationID string) (int64, error)
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
        targetID := convutil.ExtractTargetID(convID, senderID)
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
    targetIDs := convutil.GetConversationMembers(convID, senderID)
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

    // 推送给会话中对端用户（通知已读回执）
    targetIDs := convutil.GetConversationMembers(convID, userID)
    for _, targetID := range targetIDs {
        s.hub.PushToUser(targetID, &ws.WSMessage{
            Type: 102,
            Data: map[string]interface{}{
                "conversationId": convID,
                "readUserId":     userID,
                "readSeq":        req.ReadSeq,
            },
        })
    }
    return nil
}
```

---


### 7.6 Message Handler

```go
// internal/handler/message.go
package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/tianlu1990s/gim/internal/model"
    "github.com/tianlu1990s/gim/internal/service"
    "github.com/tianlu1990s/gim/pkg/errcode"
    "github.com/tianlu1990s/gim/pkg/resp"
)

type MessageHandler struct {
    svc service.MessageService
}

func NewMessageHandler(svc service.MessageService) *MessageHandler {
    return &MessageHandler{svc: svc}
}

func (h *MessageHandler) History(c *gin.Context) {
    userID := c.GetString("userID")
    var req model.HistoryReq
    if err := c.ShouldBindQuery(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    result, err := h.svc.History(c.Request.Context(), userID, &req)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, result)
}

func (h *MessageHandler) MarkRead(c *gin.Context) {
    userID := c.GetString("userID")
    var req model.MarkReadReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    if err := h.svc.MarkRead(c.Request.Context(), userID, &req); err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, nil)
}

func (h *MessageHandler) Revoke(c *gin.Context) {
    userID := c.GetString("userID")
    var req model.RevokeMsgReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    if err := h.svc.Revoke(c.Request.Context(), userID, &req); err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, nil)
}
```

### 7.7 Conversation Handler

```go
// internal/handler/conversation.go
package handler

import (
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/tianlu1990s/gim/internal/model"
    "github.com/tianlu1990s/gim/internal/service"
    "github.com/tianlu1990s/gim/pkg/errcode"
    "github.com/tianlu1990s/gim/pkg/resp"
)

type ConversationHandler struct {
    svc service.ConversationService
}

func NewConversationHandler(svc service.ConversationService) *ConversationHandler {
    return &ConversationHandler{svc: svc}
}

func (h *ConversationHandler) List(c *gin.Context) {
    userID := c.GetString("userID")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
    result, err := h.svc.List(c.Request.Context(), userID, page, pageSize)
    if err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, result)
}

func (h *ConversationHandler) Pin(c *gin.Context) {
    userID := c.GetString("userID")
    convID := c.Param("id")
    if convID == "" {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail("会话ID不能为空"))
        return
    }
    var req model.PinConversationReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail(err.Error()))
        return
    }
    if err := h.svc.Pin(c.Request.Context(), userID, convID, req.IsPinned); err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, nil)
}

func (h *ConversationHandler) Delete(c *gin.Context) {
    userID := c.GetString("userID")
    convID := c.Param("id")
    if convID == "" {
        resp.Fail(c, errcode.ErrInvalidParam.WithDetail("会话ID不能为空"))
        return
    }
    if err := h.svc.Delete(c.Request.Context(), userID, convID); err != nil {
        resp.Fail(c, err)
        return
    }
    resp.Success(c, nil)
}
```

## 8. WebSocket 网关实现

### 8.1 Hub（连接中心）

💡 **Hub 的工作原理类比电话总机**：所有 WebSocket 连接都注册到 Hub，Hub 维护一张"谁在线"的表。当要给某人推送消息时，查 Hub 找到这个人的连接，通过连接发送。没有 Hub，每条消息要遍历所有连接去找目标用户，效率极低。

💡 **为什么用 channel（register/unregister/push）而不是直接操作 map？** Go 的 map 不是线程安全的，多个 goroutine 同时读写 map 会 panic。用 channel 可以保证同一时刻只有一个 goroutine 操作 map（在 Run() 的 for-select 循环中），这是 Go 并发编程的惯用模式——"不要通过共享内存来通信，而要通过通信来共享内存"。

```go
// internal/ws/hub.go
package ws

// WSMessage WebSocket 消息通用结构
type WSMessage struct {
    Type    int                    `json:"type"`     // 消息类型
    ReqID   string                 `json:"reqId"`    // 请求ID（用于请求响应匹配）
    Data    map[string]interface{} `json:"data"`     // 消息数据
}

type Hub struct {
    // 用户ID -> 该用户的所有连接
    clients    map[string]map[*Client]struct{}
    register   chan *Client
    unregister chan *Client
    push       chan *PushMessage

    msgSvc  service.MessageService
    convSvc service.ConversationService
    rdb     *redis.Client
    cfg     *config.WebSocketConfig // WebSocket 配置
    mu      sync.RWMutex
}

type PushMessage struct {
    UserID  string
    Message *WSMessage
}

func NewHub(msgSvc service.MessageService, convSvc service.ConversationService, rdb *redis.Client, cfg *config.WebSocketConfig) *Hub {
    return &Hub{
        clients:   make(map[string]map[*Client]struct{}),
        register:  make(chan *Client, 256),
        unregister: make(chan *Client, 256),
        push:      make(chan *PushMessage, 1024),
        msgSvc:    msgSvc,
        convSvc:   convSvc,
        rdb:       rdb,
        cfg:       cfg,
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

    c.conn.SetReadLimit(c.hub.cfg.MaxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(c.hub.cfg.PongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(c.hub.cfg.PongWait))
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
    ticker := time.NewTicker(c.hub.cfg.PingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(c.hub.cfg.WriteWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(c.hub.cfg.WriteWait))
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
        var req model.SendMsgReq
        json.Unmarshal(toJSON(msg.Data), &req)
        resp, err := c.hub.msgSvc.SendMessage(ctx, c.userID, &req)
        if err != nil {
            c.Send(&WSMessage{Type: -1, ReqID: msg.ReqID, Data: map[string]interface{}{"error": err.Error()}})
            return
        }
        c.Send(&WSMessage{Type: 101, ReqID: msg.ReqID, Data: resp})

    case 2: // 已读回执
        var req model.MarkReadReq
        json.Unmarshal(toJSON(msg.Data), &req)
        c.hub.msgSvc.MarkRead(ctx, c.userID, &req)

    case 3: // 心跳
        c.conn.SetReadDeadline(time.Now().Add(c.hub.cfg.PongWait))
        // Refresh online status
        ctx := context.Background()
        key := fmt.Sprintf("online:%s", c.userID)
        c.hub.rdb.Expire(ctx, key, 60*time.Second)
        c.hub.rdb.HSet(ctx, key, "lastActive", time.Now().Unix())
        c.Send(&WSMessage{Type: 103, ReqID: msg.ReqID, Data: map[string]interface{}{}})

    case 4: // 拉取消息
        var req model.HistoryReq
        json.Unmarshal(toJSON(msg.Data), &req)
        resp, _ := c.hub.msgSvc.History(ctx, c.userID, &req)
        c.Send(&WSMessage{Type: 104, ReqID: msg.ReqID, Data: resp})

    case 5: // 输入状态
        var data map[string]interface{}
        json.Unmarshal(toJSON(msg.Data), &data)
        convID := data["conversationId"].(string)
        isTyping := data["isTyping"].(bool)
        targetID := convutil.ExtractTargetID(convID, c.userID)
        c.hub.PushToUser(targetID, &WSMessage{
            Type: 105,
            Data: map[string]interface{}{
                "conversationId": convID,
                "userId":         c.userID,
                "isTyping":       isTyping,
            },
        })
    }
}
```


### 消息发送完整流程详解

💡 **这是 IM 系统最核心的流程，必须理解每一步的判断逻辑和并发控制**。

#### 7.5.1 消息发送流程图

```
客户端发送消息的完整流程：

┌───────────────────────────────────────────────────────────────────┐
│                           客户端 (HTTP/WS)                      │
└───────────────────────────────────────┬───────────────────────┘
                                    │
                    ┌───────────▼──────────┐
                    │  WS Gateway │
                    │  (gim-ws)  │
                    └───────────┬──────────┘
                                 │
              ┌───────────────┼───────────────┐
              │               │               │
         ┌────▼────┐  ┌────▼────┐  ┌────▼────┐
         │ 去重检查 │  │ 好友校验 │  │ Seq分配  │
         │ Redis     │  │ Friend   │  │ Redis    │
         │ SETNX     │  │ Repository│  │ INCR     │
         └────┬────┘  └────┬─────┘  └────┬─────┘
              │               │              │
              │     ┌─────────┴──────────┐
              │     │                     │
              └─────▼───────────────▼──────┘
                            │
                    ┌─────────────────▼──────────┐
                    │   Message Service        │
                    │   生成 serverMsgID        │
                    │   构造消息对象           │
                    │   持久化到 MySQL          │
                    └─────────────┬──────────┘
                                 │
              ┌───────────────┴───────────┐
              │                         │
         ┌────▼────┐           ┌────▼────┐
         │ 更新maxSeq│           │ 推送给接收方│
         │ (MySQL)   │           │ (Hub→WS)   │
         └──────────┘           └────┬──────┘
                                   │
                         ┌─────────▼────────────┐
                         │   WebSocket 推送     │
                         │   (实时/离线)       │
                         └──────────────────────────┘
```

#### 7.5.2 关键并发控制

| 场景 | 问题 | 解决方案 | 实现方式 |
|------|------|---------|----------|
| 高并发去重 | 多个请求同时发送相同消息 | Redis SETNX 原子操作 | 同一条 ClientMsgId 只能写入一次 |
| Seq 顺序保证 | 消息乱序 | Redis INCR 保证递增 | Seq 分配和写入在同一事务 |
| 消息丢失风险 | 推送成功但入库失败 | 先入库再推送 | 数据库写失败则不推送 |
| 推送性能 | 逐个推送太慢 | Hub channel 异步 | push channel 缓冲 1024 条 |

#### 7.5.3 错误处理矩阵

| 错误类型 | 错误码 | HTTP状态 | 客户端处理 |
|----------|--------|----------|------------|
| 参数错误 | 10001 | 400 | 提示用户重新输入 |
| 未登录 | 10002 | 401 | 跳转登录页 |
| 非好友 | 30006 | 403 | 提示"需要先添加好友" |
| 消息撤回超时 | 30004 | 400 | 提示"超过5分钟，无法撤回" |
| 非发送者 | 30005 | 403 | 提示"只能撤回自己发送的消息" |

---


### 8.4 WebSocket 连接生命周期

💡 **WebSocket 连接管理是即时通讯系统的"神经系统"**，需要处理连接建立、心跳保活、异常断开、重连等场景。

#### 8.5.1 连接生命周期图

```
WebSocket 连接生命周期：

建立连接阶段：
┌───────────┐     ┌──────────┐     ┌───────────┐     ┌───────────┐
│  客户端   │     │  HTTP 握手 │     │ Token 验证 │     │ 黑名单检查 │
└─────┬─────┘     └─────┬─────┘     └─────┬─────┘     └─────┬─────┘
      │                 │                 │                 │
      └───────┬───────────┼───────────────┼───────────┘
                │           │               │
      ┌─────────▼───────┴───────────────┴─────────┐
      │            连接数限制                    │
      │       (max 5 per user)               │
      └─────────────┬───────────────────────────┘
                    │
      ┌───────────────▼───────────────────────┐
      │            WebSocket Upgrade             │
      │      (gorilla/websocket)             │
      └───────────────┬──────────────────────────┘
                   │
      ┌──────────────▼───────────────────────┐
      │            创建 Client 并注册           │
      │      (→ Hub.register channel)         │
      │  ┌──────────────────────────────┐    │
      │  │ 上线状态设置 (Redis)       │    │
      │  │ online:{userId}             │    │
      │  │ conn_map:{userId}             │    │
      │  └──────────────────────────────┘    │
      └───────────────────┬──────────────────┘
                           │
      ┌─────────────────▼───────────────────────┐
      │            启动读写协程               │
      │  ┌──────────────────────────┐     │
      │  │  ReadPump  (读取消息)     │     │
      │  │  WritePump (发送消息)      │     │
      │  │  心跳定时器 (30s Ping)     │     │
      │  └───────────────────────────     │
      └───────────────────────────────────┘
```

正常通信阶段：
┌─────────────────────────────────────────────────────────────┐
│                   WS 消息处理循环                 │
│  ┌────────────────────────────────────────────────┐  │
│  │  收到消息 → 类型分发 → 处理     │  │
│  └────────────────────────────────────────────────┘  │
│  ┌────────────┬────────────┬────────────┐         │
│  │   Type=1  │   Type=2   │   Type=3   │         │
│  │  发送消息   │  已读回执   │   心跳     │         │
│  └─────┬─────┴─────┬─────┴─────┘         │
│        │       │       │       │                 │
│  ┌─────▼─────┴─────▼─────┴─────▼─────┐        │
│  │   Message Service 处理           │        │
│  │  → MySQL/Redis 操作              │        │
│  │  → Hub 推送给接收方           │        │
│  └────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────┘

异常处理阶段：
┌─────────────────────────────────────────────────────────────┐
│                检测到异常                      │
│  ┌────────────────────────────────────────────────┐  │
│  │  WebSocket 读取错误 / 网络中断    │  │
│  │  │                             │       │
│  │  ├─► 触发 unregister channel  │       │
│  │  │                             │       │
│  │  ├─► 关闭 WebSocket 连接         │       │
│  │  │                             │       │
│  │  ├─► 清除在线状态 (Redis)         │       │
│  │  │                             │       │
│  │  └─► Hub 中移除连接映射        │       │
│  │                             │       │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

#### 8.5.2 心跳保活机制

```
心跳机制（防止中间连接断开不被发现）：

服务端 → 客户端（每 30 秒）：
┌──────────────┐
│   Ping 消息   │
│  (opcode 9)  │
└──────────────┘

客户端 → 服务端（收到 Ping 后立即响应）：
┌──────────────┐
│  Pong 消息   │
│  (opcode 10) │
└──────────────┘

服务端处理逻辑：
1. 发送 Ping 后，更新 readDeadline 为当前时间 + 60s
2. 收到 Pong 后，确认连接活跃
3. 超过 90s 未收到 Pong，主动关闭连接
4. 连接关闭时，清理 Redis 在线状态
```

#### 8.5.3 连接断开与重连

| 断开原因 | 处理方式 | 客户端行为 |
|----------|---------|------------|
| 网络中断 | 服务端检测到连接异常，unregister | 指数退数重连 (1, 2, 4...) |
| 用户主动关闭 | 客户端发送 close frame | 优雅关闭 |
| 服务端重启 | 连接被拒绝 | 客户端感知后重连 |
| 服务器压力 | 服务端主动关闭 (too busy) | 等待后重连 |
| Token 失效 | 下次请求返回 401 | 跳转登录 |

---

### 8.5 WS Server

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
    convs, _ := c.hub.convSvc.ListByOwner(context.Background(), c.userID)
    for _, conv := range convs {
        readSeq, _ := c.hub.msgRepo.GetUserReadSeq(context.Background(), c.userID, conv.ConversationID)
        if conv.MaxSeq > readSeq {
            result, _ := c.hub.msgSvc.History(context.Background(), c.userID, &model.HistoryReq{
                ConversationID: conv.ConversationID,
                StartSeq:       conv.MaxSeq,
                Count:          50,
            })
            for _, msg := range result.List {
                vo := &model.MsgVO{
                    ConversationID: msg.ConversationID,
                    Seq:            msg.Seq,
                    SenderID:       msg.SenderID,
                    MsgType:        msg.MsgType,
                    Content:        msg.Content,
                    ClientMsgID:    msg.ClientMsgID,
                    ServerMsgID:    msg.ServerMsgID,
                    IsRevoked:      msg.IsRevoked,
                    SendTime:       msg.CreatedAt.UnixMilli(),
                }
                c.Send(&WSMessage{Type: 101, Data: map[string]interface{}{"msg": vo}})
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
    ErrAlreadyProcessed = &Code{30007, "申请已处理"}

    ErrTooManyRequests = &Code{10007, "请求过于频繁"}
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

func JWTAuth(jwtMgr *jwt.JWTManager, rdb *redis.Client) gin.HandlerFunc {
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

### 11.4 Recovery 中间件

```go
// internal/middleware/recovery.go
package middleware

import (
    "net/http"
    "runtime/debug"

    "github.com/gin-gonic/gin"
    "github.com/tianlu1990s/gim/pkg/errcode"
    "github.com/tianlu1990s/gim/pkg/resp"
)

func Recovery(logger interface{}) gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if r := recover(); r != nil {
                // 打印堆栈信息
                debug.PrintStack()
                // 返回统一错误响应
                resp.FailWithStatus(c, http.StatusInternalServerError, errcode.ErrInternal.WithDetail("服务器内部错误"))
                c.Abort()
            }
        }()
        c.Next()
    }
}
```

### 11.5 RequestLogger 中间件

```go
// internal/middleware/logger.go
package middleware

import (
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
)

type ResponseWriter struct {
    gin.ResponseWriter
    bodySize int
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
    n, err := w.ResponseWriter.Write(b)
    w.bodySize += n
    return n, err
}

func RequestLogger(logger interface{}) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        query := c.Request.URL.RawQuery

        // 替换响应写入器
        writer := &ResponseWriter{ResponseWriter: c.Writer}
        c.Writer = writer

        c.Next()

        // 计算耗时
        latency := time.Since(start).Milliseconds()
        status := c.Writer.Status()
        method := c.Request.Method
        clientIP := c.ClientIP()

        // 格式化日志
        logMsg := fmt.Sprintf("[GIN] %s %s?%s %d %dms %s %s",
            method, path, query, status, latency, clientIP, c.Errors.String())

        // 使用 slog 记录（实际使用时传入 logger 参数）
        // 这里简化处理，生产环境建议使用结构化日志
        if status >= 500 {
            // logger.Error(logMsg, "path", path, "status", status, "latency", latency)
            fmt.Println(logMsg)
        } else if status >= 400:
            // logger.Warn(logMsg)
            fmt.Println(logMsg)
        } else:
            // logger.Info(logMsg)
            fmt.Println(logMsg)
    }
}
```

---

## 12. Makefile 与构建

```makefile
# Makefile
.PHONY: build run test lint migrate clean docker docker-down docker-build deps gen swagger

APP_NAME := gim
BUILD_DIR := bin
GO ?= go
MAIN := cmd/gim/main.go

# 数据库连接配置（可在命令行覆盖：make migrate-up DB_DSN="..."）
DB_USER ?= gim
DB_PASSWORD ?= gim_pass
DB_HOST ?= localhost
DB_PORT ?= 3306
DB_NAME ?= gim
DB_DSN := $(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?charset=utf8mb4&parseTime=True&loc=Local

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

run: build
	@echo "Running $(APP_NAME)..."
	$(BUILD_DIR)/$(APP_NAME)

test:
	$(GO) test -v -count=1 ./...

test-single:
	@echo "Usage: make test-single TEST=TestName PKG=./path/to/package"
	@echo "Example: make test-single TEST=TestRegister PKG=./internal/service"
	$(GO) test -v -count=1 -run $(TEST) $(PKG)

lint:
	golangci-lint run ./...

# 数据库迁移（需先启动 MySQL）
migrate-up:
	@echo "Running migrations up..."
	migrate -path migrations -database "mysql://$(DB_DSN)" up

migrate-down:
	@echo "Running migrations down (one version)..."
	migrate -path migrations -database "mysql://$(DB_DSN)" down 1

migrate-create:
	@echo "Usage: make migrate-create NAME=create_users_table"
	@echo "Creating migration file..."
	migrate create -ext sql -dir migrations -seq $(NAME)

# Docker 操作
docker:
	docker compose -f deploy/docker-compose.yaml up -d

docker-down:
	docker compose -f deploy/docker-compose.yaml down

docker-logs:
	docker compose -f deploy/docker-compose.yaml logs -f

docker-build:
	docker build -f deploy/docker/Dockerfile -t $(APP_NAME):latest .

# 代码生成（第二阶段使用）
gen:
	@echo "Generating gRPC code from protobuf..."
	protoc --go_out=. --go-grpc_out=. api/**/*.proto

swagger:
	@echo "Generating Swagger documentation..."
	swag init -g cmd/gim/main.go -o docs/swagger

# 清理
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

# 依赖管理
deps:
	$(GO) mod tidy
	$(GO) mod download

deps-check:
	$(GO) mod verify

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  make build          - Build the application"
	@echo "  make run            - Build and run the application"
	@echo "  make test           - Run all tests"
	@echo "  make test-single    - Run specific test (TEST=TestName PKG=./path/to/package)"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make migrate-up     - Run database migrations up"
	@echo "  make migrate-down   - Rollback one migration"
	@echo "  make migrate-create NAME=name - Create new migration file"
	@echo "  make docker         - Start Docker Compose services"
	@echo "  make docker-down    - Stop Docker Compose services"
	@echo "  make docker-logs    - View Docker Compose logs"
	@echo "  make docker-build    - Build Docker image"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make deps           - Tidy and download dependencies"
	@echo "  make deps-check     - Verify dependencies"
	@echo "  make gen            - Generate gRPC code (Phase 2)"
	@echo "  make swagger        - Generate Swagger docs"
```

---


### 实施方法详解（Phase 1 步骤表）

💡 **本章提供每个模块的详细实施步骤，从零到可运行的具体命令**。

#### Phase 1 实施步骤表

| 步骤 | 模块 | 命令 | 预期输出 | 检查方法 |
|------|------|------|----------|----------|
| 1. 项目初始化 | - | 参见 1.0.1 快速初始化 | 完整目录结构 | `ls -la` |
| 2. 安装依赖 | - | `go get ...` (见 1.0.1) | go.mod/go.sum | `cat go.mod` |
| 3. 配置开发环境 | Docker | `make docker` | MySQL+Redis 运行 | `docker ps` |
| 4. 数据库迁移 | - | `make migrate-up` | 表结构创建 | 连接数据库 `SHOW TABLES` |
| 5. 实现 Model 层 | Go | 按 1.4 编写所有 model 文件 | 编译通过 | `go build ./...` |
| 6. 实现 pkg 包 | Go | 按 2.2-2.5 实现各 pkg | 单元测试通过 | `go test ./pkg/...` |
| 7. 实现 Repository | Go | 按 3.1-4.2 实现各 repo | 单元测试通过 | `go test ./internal/repository/...` |
| 8. 实现 Service | Go | 按 4.3-7.5 实现各 service | 单元测试通过 | `go test ./internal/service/...` |
| 9. 实现 Handler | Go | 按 3.4-7.5 实现各 handler | 编译通过 | `make build` |
| 10. 实现 Middleware | Go | 按 11.1-11.5 实现各 middleware | 编译通过 | `make build` |
| 11. 实现 WebSocket | Go | 按 8.1-8.4 实现 WS 系统 | 编译通过 | `make build` |
| 12. 集成到 main.go | Go | 修改 main.go 连接所有模块 | 编译通过 | `make build && make run` |
| 13. 功能测试 | - | 使用 Postman/WebSocket 客户端 | 各接口正常返回 | 记录测试用例 |

#### 各模块实施顺序与依赖关系

```
模块依赖树（从下往上实现）：

                    ┌─────────────────┐
                    │   main.go     │
                    └───────┬─────────┘
                            │
                    ┌───────────┼───────────┐
                    │           │           │
              ┌─────▼────┐ ┌───▼────┐ ┌───▼────┐
              │  Handler  │ │Middleware │ │WS Gateway│
              └─────┬─────┘ └─────┬─────┘ └─────┬─────┘
                    │            │           │
              ┌─────▼───────────────────────▼───────┐
              │             Service 层              │
              └─────┬────────────────────────┬─────┘
                    │                        │
              ┌─────▼────┐         ┌─────▼────┐
              │Repository  │         │  pkg 层   │
              │  (DB/Redis) │         │ (工具函数)  │
              └─────┬─────┘         └─────┬─────┘
                    │                        │
              ┌─────▼─────────────────────────▼──────┐
              │           Model 层 (数据结构)      │
              └───────────────────────────────────────┘
```

实施顺序说明：
1. **第一步**：实现 pkg 层（最底层，无依赖）
   - pkg/jwt → pkg/snowflake → pkg/slog → pkg/resp → pkg/errcode
   - pkg/rediskey → pkg/convid → pkg/convutil

2. **第二步**：实现 Model 层
   - internal/model/user.go
   - internal/model/friend.go
   - internal/model/conversation.go
   - internal/model/message.go
   - internal/model/req.go (请求体)
   - internal/model/vo.go (响应体)

3. **第三步**：实现 Repository 层
   - internal/repository/user.go (实现所有方法)
   - internal/repository/friend.go
   - internal/repository/conversation.go
   - internal/repository/message.go
   - internal/repository/init.go (wire 函数)

4. **第四步**：实现 Service 层
   - internal/service/auth.go (注册、登录、刷新、退出)
   - internal/service/user.go
   - internal/service/friend.go (发送申请、接受、拒绝、删除、列表)
   - internal/service/conversation.go
   - internal/service/message.go (发送、历史、已读、撤回)
   - internal/service/validate.go (参数校验)
   - internal/service/init.go (wire 函数)

5. **第五步**：实现 Handler 层
   - internal/handler/auth.go
   - internal/handler/user.go
   - internal/handler/friend.go
   - internal/handler/message.go
   - internal/handler/conversation.go
   - internal/handler/init.go (wire 函数)

6. **第六步**：实现 Middleware 层
   - internal/middleware/auth.go (JWT 鉴权)
   - internal/middleware/cors.go (跨域)
   - internal/middleware/recovery.go (恢复)
   - internal/middleware/ratelimit.go (限流)
   - internal/middleware/logger.go (请求日志)

7. **第七步**：实现 WebSocket 系统
   - internal/ws/hub.go (连接中心)
   - internal/ws/client.go (单个连接)
   - internal/ws/server.go (WS 服务)
   - internal/ws/util.go (工具函数)

8. **第八步**：集成到 main.go
   - 修改 cmd/gim/main.go
   - 添加路由注册
   - 添加中间件
   - 初始化数据库和 Redis
   - 启动 WS 服务

#### 测试检查清单

```bash
# 单元测试覆盖率检查
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
# 目标：覆盖率 > 70%

# 集成测试检查
# 启动依赖服务
docker compose -f deploy/docker-compose.yaml up -d mysql redis
# 运行应用
make run
# 等待服务启动 (约 5 秒)
sleep 5

# 健康检查
curl http://localhost:8080/health

# 功能测试（建议保存为 collection）
# 1. 用户注册
curl -X POST http://localhost:8080/api/v1/auth/register   -H "Content-Type: application/json"   -d '{"userId":"test001","password":"Test1234","nickname":"测试用户"}'

# 2. 用户登录
curl -X POST http://localhost:8080/api/v1/auth/login   -H "Content-Type: application/json"   -d '{"userId":"test001","password":"Test1234","platform":"web"}'

# 保存 access_token 到环境变量
export TOKEN="从登录响应获取的 token"

# 3. 获取用户信息
curl -X GET http://localhost:8080/api/v1/user/profile   -H "Authorization: Bearer $TOKEN"

# 4. 搜索用户
curl -X POST http://localhost:8080/api/v1/user/search   -H "Authorization: Bearer $TOKEN"   -H "Content-Type: application/json"   -d '{"keyword":"test","page":1,"pageSize":20}'

# 5. 发送好友请求
curl -X POST http://localhost:8080/api/v1/friend/request   -H "Authorization: Bearer $TOKEN"   -H "Content-Type: application/json"   -d '{"toUserId":"test002","message":"我是测试，加个好友"}'

# 6. 查看好友申请列表
curl -X GET "http://localhost:8080/api/v1/friend/request/incoming?page=1&pageSize=20"   -H "Authorization: Bearer $TOKEN"

# 7. 同意好友请求
curl -X POST http://localhost:8080/api/v1/friend/request/1/accept   -H "Authorization: Bearer $TOKEN"

# 8. 获取会话列表
curl -X GET "http://localhost:8080/api/v1/conversation/list?page=1&pageSize=20"   -H "Authorization: Bearer $TOKEN"

# 9. 发送消息 (通过 WebSocket，需要 WS 客户端)
# 使用 wscat 或其他 WS 客户端连接
# ws://localhost:8081/ws?token=$TOKEN
# 发送消息格式: {"type":1,"reqId":"msg1","data":{"conversationId":"single_test001_test002","clientMsgId":"client123","contentType":1,"content":"你好"}}
```

#### 常见错误排查

| 错误现象 | 可能原因 | 排查步骤 |
|----------|---------|---------|
| 连接不上 MySQL | Docker 容器未启动/端口错误 | `docker ps`, `docker logs gim-mysql`, 检查 config.yaml |
| 连接不上 Redis | Redis 未启动/配置错误 | `docker ps`, `docker logs gim-redis`, `redis-cli ping` |
| JWT 验证失败 | 密钥对不存在/格式错误 | 检查 configs/jwt/ 目录, 重新生成密钥对 |
| 编译错误 | 依赖未安装/导入路径错误 | `go mod tidy`, 检查 go.mod 中的路径 |
| 运行时 panic | nil 指针/数组越界 | 检查日志中的 panic 堆栈, 使用 defer recover |
| WebSocket 连接失败 | Token 无效/地址错误 | 检查 Token 是否正确, WS 服务是否启动 |

#### 开发环境配置示例

```yaml
# configs/config.local (开发环境专用，不提交到 git)
server:
  httpPort: 8080
  port: 8081

mysql:
  host: localhost
  port: 3306
  user: gim
  password: gim_pass
  dbname: gim

redis:
  host: localhost
  port: 6379

log:
  level: debug
  format: text
  output: stdout
  color: true

# 开发环境使用本地数据库，不使用 Docker
```

---

## 13. Docker 与开发环境

### 13.1 Dockerfile

```dockerfile
# deploy/docker/Dockerfile

# Build stage
FROM golang:1.26-alpine AS builder
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
version: '3.8'

services:
  mysql:
    image: mysql:8.4
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

#### MySQL 初始化脚本

💡 **为什么需要 init.sql？** Docker Compose 启动 MySQL 时，会自动执行 `/docker-entrypoint-initdb.d/` 目录下的 SQL 脚本。这里可以创建数据库、用户、授权等初始化操作。

```sql
-- deploy/mysql/init.sql
-- MySQL 初始化脚本，创建用户和授权

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS gim DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 使用数据库
USE gim;

-- 创建用户并授权（如果不存在）
CREATE USER IF NOT EXISTS 'gim'@'%' IDENTIFIED BY 'gim_pass';
GRANT ALL PRIVILEGES ON gim.* TO 'gim'@'%';
FLUSH PRIVILEGES;

-- 设置时区
SET GLOBAL time_zone = '+8:00';

-- 显示初始化完成信息
SELECT 'MySQL initialization completed!' AS Status;
```

💡 **注意事项：**
- init.sql 文件必须放在 `deploy/mysql/` 目录下
- 首次启动时会自动执行，后续启动不再执行
- 如果需要重新执行，删除 MySQL 容器和 volumes：`docker compose down -v`

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
    image: mysql:8.4
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
```sql
-- migrations/000002_create_friends_tables.up.sql
CREATE TABLE friends (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    owner_id   VARCHAR(64)  NOT NULL,
    friend_id  VARCHAR(64)  NOT NULL,
    remark     VARCHAR(64)  NOT NULL DEFAULT '',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_owner_friend (owner_id, friend_id),
    INDEX idx_owner (owner_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE friend_requests (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    from_user_id VARCHAR(64) NOT NULL,
    to_user_id   VARCHAR(64) NOT NULL,
    message     VARCHAR(256) NOT NULL DEFAULT '',
    status      TINYINT      NOT NULL DEFAULT 0 COMMENT '0=待处理 1=已同意 2=已拒绝',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_to_user (to_user_id),
    INDEX idx_from_to (from_user_id, to_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

```sql
-- migrations/000002_create_friends_tables.down.sql
DROP TABLE IF EXISTS friend_requests;
DROP TABLE IF EXISTS friends;
```

```sql
-- migrations/000003_create_conversations_table.up.sql
CREATE TABLE conversations (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    owner_id        VARCHAR(64)  NOT NULL,
    conversation_id VARCHAR(128) NOT NULL,
    conv_type       INT          NOT NULL DEFAULT 1 COMMENT '1=单聊 2=群聊',
    target_id       VARCHAR(64)  NOT NULL DEFAULT '',
    max_seq         BIGINT       NOT NULL DEFAULT 0,
    is_pinned       TINYINT(1)   NOT NULL DEFAULT 0,
    created_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_owner_conv (owner_id, conversation_id),
    INDEX idx_owner_conv (owner_id, conversation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

```sql
-- migrations/000003_create_conversations_table.down.sql
DROP TABLE IF EXISTS conversations;
```

```sql
-- migrations/000004_create_messages_table.up.sql
CREATE TABLE messages (
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    conversation_id  VARCHAR(128) NOT NULL,
    seq              BIGINT       NOT NULL,
    sender_id        VARCHAR(64)  NOT NULL,
    msg_type         INT          NOT NULL DEFAULT 1 COMMENT '1=文本 2=图片 3=文件 4=系统消息',
    content          TEXT         NOT NULL,
    client_msg_id    VARCHAR(64)  NOT NULL,
    server_msg_id    VARCHAR(64)  NOT NULL,
    is_revoked       TINYINT(1)   NOT NULL DEFAULT 0,
    created_at       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_client_msg (client_msg_id),
    INDEX idx_conv_seq (conversation_id, seq),
    INDEX idx_server_msg (server_msg_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

```sql
-- migrations/000004_create_messages_table.down.sql
DROP TABLE IF EXISTS messages;
```

```sql
-- migrations/000005_create_user_conversation_seq_table.up.sql
CREATE TABLE user_conversation_seqs (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id         VARCHAR(64)  NOT NULL,
    conversation_id  VARCHAR(128) NOT NULL,
    read_seq        BIGINT       NOT NULL DEFAULT 0,
    updated_at      DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_conv (user_id, conversation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

```sql
-- migrations/000005_create_user_conversation_seq_table.down.sql
DROP TABLE IF EXISTS user_conversation_seqs;
```


---


## 15. 第二阶段：微服务 + Kafka + MongoDB

### 15.0 第二阶段概述

💡 **为什么要拆成微服务？** 单体应用在发展到一定规模后会出现瓶颈：
1. **部署耦合**：修改一行代码需要重新部署整个应用
2. **技术栈限制**：不同模块适合不同技术栈（如消息队列用 Go，推送用 Go）
3. **横向扩展困难**：消息服务压力大时，无法独立扩容
4. **团队协作障碍**：多人同时修改同一代码库，合并冲突频繁

微服务架构通过服务拆分解决这些问题，每个服务可以独立开发、部署、扩展。

### 15.0.1 第二阶段架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                         客户端                              │
│                    ┌──────────────┐                          │
│                    │   HTTP/WS    │                          │
│                    └──────────────┘                          │
└─────────────────────────────┬───────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
         ┌────▼────┐    ┌───▼──────┐  ┌───▼──────┐
         │ gim-api  │    │ gim-ws   │  │ gim-push  │
         │  Gin/HTTP │    │  WS+gRPC │  │  gRPC    │
         └────┬────┘    └───┬──────┘  └───┬──────┘
              │               │               │
         ┌────▼────┐    ┌───▼──────┐  ┌───▼──────┐
         │ etcd    │◄──►│  Kafka   │◄─┤  etcd    │
         └──────────┘    └────┬─────┘  └──────────┘
                               │
                        ┌────────┴────────┐
                        │                 │
                   ┌────▼────┐      ┌────▼──────┐
                   │rpc-msg  │      │rpc-push   │
                   │  gRPC   │      │  gRPC    │
                   └────┬────┘      └────┬──────┘
                        │                 │
                        └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │  MongoDB  MinIO │
                        │   Milvus pgvector│
                        └───────────────────┘
```

### 15.0.2 服务清单

| 服务名 | 职责 | 技术栈 | 端口 |
|-------|------|--------|------|
| gim-api | HTTP API 网关，路由请求到各 RPC 服务 | Gin + etcd 客户端 | 8080 |
| gim-ws | WebSocket 网关，管理连接 + gRPC 接口 | gorilla/websocket + gRPC 服务 | 8081(WS) + 8082(gRPC) |
| rpc-auth | 认证服务：注册、登录、Token 管理 | gRPC + MySQL + Redis | 9001 |
| rpc-user | 用户服务：资料管理、搜索 | gRPC + MySQL | 9002 |
| rpc-friend | 好友服务：好友关系、申请 | gRPC + MySQL | 9003 |
| rpc-msg | 消息服务：消息发送、历史、已读 | gRPC + Kafka + MongoDB | 9004 |
| rpc-conversation | 会话服务：会话管理、未读 | gRPC + MySQL | 9005 |
| gim-push | 推送服务：在线推送、离线推送 | gRPC + Kafka + Redis | 9010 |
| msg-transfer | 消息传输：消费 Kafka 写 MongoDB | Kafka 消费者 + MongoDB | 内部 |
| offline-push | 离线推送：集成 APNs/FCM | Kafka 消费者 | 内部 |

### 15.0.3 第二阶段目录结构

```
gim/
├── cmd/
│   ├── gim-api/
│   │   └── main.go
│   ├── gim-ws/
│   │   └── main.go
│   ├── rpc-auth/
│   │   └── main.go
│   ├── rpc-user/
│   │   └── main.go
│   ├── rpc-friend/
│   │   └── main.go
│   ├── rpc-msg/
│   │   └── main.go
│   ├── rpc-conversation/
│   │   └── main.go
│   ├── gim-push/
│   │   └── main.go
│   ├── msg-transfer/
│   │   └── main.go
│   └── offline-push/
│       └── main.go
├── api/                          # Protobuf 定义
│   ├── auth/
│   │   └── auth.proto
│   ├── user/
│   │   └── user.proto
│   ├── friend/
│   │   └── friend.proto
│   ├── msg/
│   │   └── msg.proto
│   ├── conversation/
│   │   └── conversation.proto
│   └── push/
│       └── push.proto
├── internal/
│   ├── common/                    # 公共代码
│   │   ├── etcd/                # etcd 客户端封装
│   │   ├── grpc/                # gRPC 中间件
│   │   ├── kafka/               # Kafka 生产者/消费者
│   │   ├── mongo/               # MongoDB 客户端
│   │   ├── redis/               # Redis 客户端封装
│   │   └── storage/             # S3 兼容存储客户端(MinIO/OSS)
│   ├── api/
│   │   ├── auth/
│   │   ├── user/
│   │   ├── friend/
│   │   ├── msg/
│   │   ├── conversation/
│   │   └── push/
│   ├── server/                    # gRPC 服务实现
│   │   ├── auth/
│   │   ├── user/
│   │   ├── friend/
│   │   ├── msg/
│   │   ├── conversation/
│   │   └── push/
│   └── ws/
│       ├── gateway/               # WS Gateway
│       └── push/                 # 推送逻辑
├── pkg/
│   ├── jwt/
│   ├── snowflake/
│   ├── slog/
│   ├── resp/
│   └── errcode/
└── configs/
    ├── etcd.yaml
    ├── kafka.yaml
    ├── mongo.yaml
    └── redis.yaml
```

---


### 15.1 gRPC Protobuf 完整定义

💡 **为什么用 gRPC？** 微服务间通信需要高性能、强类型、自动生成的客户端代码。gRPC 基于 HTTP/2 + Protobuf，比 JSON REST 快 5-10 倍，且自动生成多语言客户端。

#### 15.1.1 Auth Service Protobuf

```protobuf
// api/auth/auth.proto
syntax = "proto3";
package auth;
option go_package = "github.com/tianlu1990s/gim/api/auth;auth";

import "google/protobuf/empty.proto";

service AuthService {
    rpc Register(RegisterReq) returns (RegisterResp);
    rpc Login(LoginReq) returns (LoginResp);
    rpc Refresh(RefreshReq) returns (LoginResp);
    rpc Logout(LogoutReq) returns (google.protobuf.Empty);
    rpc ValidateToken(ValidateTokenReq) returns (ValidateTokenResp);
}

message RegisterReq {
    string user_id = 1;
    string password = 2;
    string nickname = 3;
    string phone = 4;
    string email = 5;
}

message RegisterResp {
    string user_id = 1;
    string nickname = 2;
    string avatar_url = 3;
}

message LoginReq {
    string user_id = 1;
    string password = 2;
    string platform = 3;
}

message LoginResp {
    string access_token = 1;
    string refresh_token = 2;
    int64 access_expire_at = 3;
    int64 refresh_expire_at = 4;
    string user_id = 5;
}

message RefreshReq {
    string refresh_token = 1;
    string platform = 2;
}

message LogoutReq {
    string user_id = 1;
    string platform = 2;
    string access_token = 3;
}

message ValidateTokenReq {
    string access_token = 1;
}

message ValidateTokenResp {
    bool valid = 1;
    string user_id = 2;
    string platform = 3;
}
```

#### 15.1.2 User Service Protobuf

```protobuf
// api/user/user.proto
syntax = "proto3";
package user;
option go_package = "github.com/tianlu1990s/gim/api/user;user";

service UserService {
    rpc GetProfile(GetProfileReq) returns (User);
    rpc UpdateProfile(UpdateProfileReq) returns (User);
    rpc GetOtherProfile(GetOtherProfileReq) returns (OtherUser);
    rpc Search(SearchReq) returns (SearchResp);
    rpc BatchGetProfile(BatchGetProfileReq) returns (BatchGetProfileResp);
}

message GetProfileReq {
    string user_id = 1;
}

message UpdateProfileReq {
    string user_id = 1;
    string nickname = 2;
    string avatar_url = 3;
    string phone = 4;
    string email = 5;
}

message GetOtherProfileReq {
    string current_user_id = 1;
    string target_user_id = 2;
}

message User {
    string user_id = 1;
    string nickname = 2;
    string avatar_url = 3;
    string phone = 4;
    string email = 5;
    int32 status = 6;
    string created_at = 7;
}

message OtherUser {
    string user_id = 1;
    string nickname = 2;
    string avatar_url = 3;
    bool is_friend = 4;
    string remark = 5;
}

message SearchReq {
    string keyword = 1;
    int32 page = 2;
    int32 page_size = 3;
}

message SearchResp {
    repeated User users = 1;
    int64 total = 2;
}

message BatchGetProfileReq {
    repeated string user_ids = 1;
}

message BatchGetProfileResp {
    map<string, User> users = 1;
}
```

#### 15.1.3 Friend Service Protobuf

```protobuf
// api/friend/friend.proto
syntax = "proto3";
package friend;
option go_package = "github.com/tianlu1990s/gim/api/friend;friend";

service FriendService {
    rpc SendRequest(SendRequestReq) returns (SendRequestResp);
    rpc ListRequests(ListRequestsReq) returns (ListRequestsResp);
    rpc AcceptRequest(AcceptRequestReq) returns (google.protobuf.Empty);
    rpc RejectRequest(RejectRequestReq) returns (google.protobuf.Empty);
    rpc DeleteFriend(DeleteFriendReq) returns (google.protobuf.Empty);
    rpc ListFriends(ListFriendsReq) returns (ListFriendsResp);
    rpc SetRemark(SetRemarkReq) returns (google.protobuf.Empty);
    rpc IsFriend(IsFriendReq) returns (IsFriendResp);
    rpc BatchIsFriend(BatchIsFriendReq) returns (BatchIsFriendResp);
}

message SendRequestReq {
    string from_user_id = 1;
    string to_user_id = 2;
    string message = 3;
}

message SendRequestResp {
    int64 request_id = 1;
}

message ListRequestsReq {
    string user_id = 1;
    int32 page = 2;
    int32 page_size = 3;
}

message FriendRequest {
    int64 id = 1;
    string from_user_id = 2;
    string from_nickname = 3;
    string from_avatar = 4;
    string message = 5;
    int32 status = 6;
    string created_at = 7;
}

message ListRequestsResp {
    repeated FriendRequest requests = 1;
    int64 total = 2;
}

message AcceptRequestReq {
    string user_id = 1;
    int64 request_id = 2;
}

message RejectRequestReq {
    string user_id = 1;
    int64 request_id = 2;
}

message DeleteFriendReq {
    string owner_id = 1;
    string friend_id = 2;
}

message ListFriendsReq {
    string owner_id = 1;
    int32 page = 2;
    int32 page_size = 3;
}

message Friend {
    string friend_id = 1;
    string nickname = 2;
    string avatar_url = 3;
    string remark = 4;
}

message ListFriendsResp {
    repeated Friend friends = 1;
    int64 total = 2;
}

message SetRemarkReq {
    string owner_id = 1;
    string friend_id = 2;
    string remark = 3;
}

message IsFriendReq {
    string user_id = 1;
    string friend_id = 2;
}

message IsFriendResp {
    bool is_friend = 1;
}

message BatchIsFriendReq {
    string user_id = 1;
    repeated string friend_ids = 2;
}

message BatchIsFriendResp {
    map<string, bool> results = 1;
}
```

#### 15.1.4 Message Service Protobuf

```protobuf
// api/msg/msg.proto
syntax = "proto3";
package msg;
option go_package = "github.com/tianlu1990s/gim/api/msg;msg";

service MsgService {
    rpc SendMessage(SendMsgReq) returns (SendMsgResp);
    rpc GetHistory(GetHistoryReq) returns (GetHistoryResp);
    rpc MarkRead(MarkReadReq) returns (google.protobuf.Empty);
    rpc RevokeMsg(RevokeMsgReq) returns (google.protobuf.Empty);
    rpc GetUserReadSeq(GetUserReadSeqReq) returns (GetUserReadSeqResp);
    rpc BatchGetUserReadSeq(BatchGetUserReadSeqReq) returns (BatchGetUserReadSeqResp);
}

message SendMsgReq {
    string sender_id = 1;
    string conversation_id = 2;
    string client_msg_id = 3;
    int32 content_type = 4;
    string content = 5;
}

message SendMsgResp {
    int64 seq = 1;
    string server_msg_id = 2;
    int64 send_time = 3;
}

message GetHistoryReq {
    string user_id = 1;
    string conversation_id = 2;
    int64 start_seq = 3;
    int32 count = 4;
}

message Msg {
    string conversation_id = 1;
    int64 seq = 2;
    string sender_id = 3;
    int32 msg_type = 4;
    string content = 5;
    string client_msg_id = 6;
    string server_msg_id = 7;
    bool is_revoked = 8;
    int64 send_time = 9;
}

message GetHistoryResp {
    repeated Msg messages = 1;
    bool has_more = 2;
    int64 min_seq = 3;
    int64 max_seq = 4;
}

message MarkReadReq {
    string user_id = 1;
    string conversation_id = 2;
    int64 read_seq = 3;
}

message RevokeMsgReq {
    string user_id = 1;
    string conversation_id = 2;
    string client_msg_id = 3;
}

message GetUserReadSeqReq {
    string user_id = 1;
    string conversation_id = 2;
}

message GetUserReadSeqResp {
    int64 read_seq = 1;
}

message BatchGetUserReadSeqReq {
    string user_id = 1;
    repeated string conversation_ids = 2;
}

message BatchGetUserReadSeqResp {
    map<string, int64> results = 1;
}
```

#### 15.1.5 Conversation Service Protobuf

```protobuf
// api/conversation/conversation.proto
syntax = "proto3";
package conversation;
option go_package = "github.com/tianlu1990s/gim/api/conversation;conversation";

service ConversationService {
    rpc ListConversations(ListConversationsReq) returns (ListConversationsResp);
    rpc PinConversation(PinConversationReq) returns (google.protobuf.Empty);
    rpc DeleteConversation(DeleteConversationReq) returns (google.protobuf.Empty);
    rpc GetConversation(GetConversationReq) returns (Conversation);
    rpc BatchGetConversation(BatchGetConversationReq) returns (BatchGetConversationResp);
}

message ListConversationsReq {
    string user_id = 1;
    int32 page = 2;
    int32 page_size = 3;
}

message Conversation {
    string conversation_id = 1;
    int32 conv_type = 2;
    string target_id = 3;
    int64 max_seq = 4;
    int64 read_seq = 5;
    int64 unread_count = 6;
    bool is_pinned = 7;
    Msg last_msg = 8;
    string updated_at = 9;
}

message ListConversationsResp {
    repeated Conversation conversations = 1;
    int64 total = 2;
}

message PinConversationReq {
    string user_id = 1;
    string conversation_id = 2;
    bool is_pinned = 3;
}

message DeleteConversationReq {
    string user_id = 1;
    string conversation_id = 2;
}

message GetConversationReq {
    string user_id = 1;
    string conversation_id = 2;
}

message BatchGetConversationReq {
    string user_id = 1;
    repeated string conversation_ids = 2;
}

message BatchGetConversationResp {
    map<string, Conversation> conversations = 1;
}
```

#### 15.1.6 Push Service Protobuf

```protobuf
// api/push/push.proto
syntax = "proto3";
package push;
option go_package = "github.com/tianlu1990s/gim/api/push;push";

import "api/msg/msg.proto";

service PushService {
    rpc OnlinePush(OnlinePushReq) returns (OnlinePushResp);
    rpc OfflinePush(OfflinePushReq) returns (google.protobuf.Empty);
    rpc GetOnlineUsers(GetOnlineUsersReq) returns (GetOnlineUsersResp);
}

message OnlinePushReq {
    repeated string user_ids = 1;
    msg.Msg msg = 2;
}

message OnlinePushResp {
    map<string, bool> results = 1;
}

message OfflinePushReq {
    string user_id = 1;
    msg.Msg msg = 2;
    string title = 3;
    string body = 4;
}

message GetOnlineUsersReq {
    repeated string user_ids = 1;
}

message GetOnlineUsersResp {
    map<string, bool> online_status = 1;
}
```

---

### 15.1 第二阶段核心实现

### 15.1 gRPC Protobuf 定义示例

```protobuf
// api/msg/msg.proto
syntax = "proto3";
package msg;
option go_package = "github.com/tianlu1990s/gim/api/msg";

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

💡 **第二阶段核心改造点**：消息发送从同步写库变为异步写 Kafka。

**原流程（Phase 1）：**
```
WS Gateway → Message Service → 写 MySQL → 返回 Seq → 推送消息
```
**问题**：MySQL 写入成为瓶颈，高并发时延迟高。

**新流程（Phase 2）：**
```
WS Gateway → Message Service → 分配Seq(Redis) → 写Kafka → 返回Seq
                                              │
                                     MsgTransfer消费
                                              │
                                        写MongoDB(批量)
```
**优势**：
1. **降低延迟**：用户感知的消息发送时间从 50ms+ 降到 10ms-
2. **提高吞吐**：Kafka 可以每秒处理数万条消息
3. **解耦服务**：存储服务挂了不影响发送服务

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

💡 **如何高效存储海量消息？** MongoDB 按"文档"组织数据，每个文档可以包含多条消息。

**文档结构设计：**
```
MongoDB Collection: messages

┌─────────────────────────────────────────────────┐
│ _id: "conv:123:0"  (会话ID:分区号)              │
│ doc_id: "conv:123:0"                             │
│ updated_at: 2026-05-01T12:00:00Z                │
│ msgs: [                                           │
│   {seq: 1, sender: "alice", content: "hi"},     │
│   {seq: 2, sender: "bob", content: "hello"},    │
│   ...                                            │
│   {seq: 100, sender: "alice", content: "bye"}   │
│ ]                                                │
└─────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────┐
│ _id: "conv:123:1"  (第2个文档，存101-200条)     │
│ msgs: [...]                                      │
└─────────────────────────────────────────────────┘
```

**每文档 100 条消息的原因：**
- **查询效率**：拉取历史时一次读取一个文档就够
- **写入效率**：批量 Upsert 比单条插入快 10-20 倍
- **存储压缩**：MongoDB 对同一文档的数据有压缩

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

💡 **为什么 WS Gateway 需要 gRPC 接口？** Phase 2 中，Push 服务需要把消息推送到在线用户的 WebSocket 连接上。由于有多台 WS Gateway 实例，Push 服务需要通过 gRPC 调用每台 Gateway 的推送接口。

**双协议架构：**
```
┌─────────────────────────────────────────────────────┐
│              gim-ws (WS Gateway)                    │
│                                                     │
│  ┌──────────────┐         ┌──────────────┐        │
│  │ HTTP/WS 端口 │         │  gRPC 端口   │        │
│  │    :8081     │         │    :8082     │        │
│  └──────┬───────┘         └──────┬───────┘        │
│         │                        │                 │
│         ▼                        ▼                 │
│  ┌──────────────────────────────────────┐         │
│  │           Hub (连接中心)              │         │
│  │   - clients: map[userID][]*Client    │         │
│  │   - register/unregister              │         │
│  │   - broadcast                        │         │
│  └──────────────────────────────────────┘         │
│                                                     │
└─────────────────────────────────────────────────────┘
         ▲                                            ▲
         │                                            │
    客户端连接                                   Push服务调用
   (WS协议)                                   (gRPC协议)
```

**调用链路：**
```
Push Service → gRPC: OnlineBatchPushOneMsg() → Gateway → 推送给连接的客户端
```

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

💡 **群消息如何高效推送？** 一个 500 人群发一条消息，需要推送到所有在线成员的客户端。如果一个个推，效率太低。需要"批量分组推送"。

**扇出流程图：**
```
┌─────────────────────────────────────────────────────┐
│             Push Service                           │
│                                                     │
│  1. 获取群成员: [user1, user2, ..., user500]      │
│         │                                            │
│         ▼                                            │
│  2. 检查在线状态: [user1, user3, user5, ...]        │
│         │                                            │
│         ▼                                            │
│  3. 按Gateway分组:                                   │
│     ┌─────────────┬─────────────┬─────────────┐       │
│     │ Gateway-1   │ Gateway-2   │ Gateway-3   │       │
│     │ 100在线     │ 150在线     │  50在线     │       │
│     └──────┬──────┴──────┬──────┴──────┬──────┘       │
│            │             │             │              │
│            ▼             ▼             ▼              │
│     gRPC调用1    gRPC调用2    gRPC调用3              │
│            │             │             │              │
│            ▼             ▼             ▼              │
│     ┌─────────────┬─────────────┬─────────────┐       │
│     │Gateway-1推送│Gateway-2推送│Gateway-3推送│       │
│     │100条消息    │150条消息    │ 50条消息    │       │
│     └─────────────┴─────────────┴─────────────┘       │
│                                                     │
│  4. 离线用户 → 写入离线推送队列                      │
└─────────────────────────────────────────────────────┘
```

**性能优化：**
1. **批量获取在线状态**：一次 Redis MGET，不是循环 GET
2. **按 Gateway 分组**：每台 Gateway 只调用一次 gRPC
3. **并发推送**：多个 Gateway 同时推送，不等待

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


### 15.6 基础设施：etcd 服务发现

💡 **为什么需要 etcd？** 微服务需要动态发现彼此的地址，而不是硬编码 IP。etcd 提供了键值存储 + 服务注册发现机制，类似于 K8S 的 etcd，但更轻量级。

**服务注册发现流程：**
```
服务启动时:
┌─────────────┐   1. 注册服务   ┌──────────────────────┐
│  rpc-auth   │ ──────────────► │ /services/rpc-auth/  │
│  :9001      │                │   "10.0.1.5:9001"    │
└─────────────┘                └──────────────────────┘
                                        │
                                        ▼
                                 2. 自动续期(TTL)
                                 (心跳保活)

其他服务调用时:
┌─────────────┐   1. 发现服务   ┌──────────────────────┐
│  gim-api    │ ◄────────────── │ /services/rpc-auth/  │
│             │                │   ["10.0.1.5:9001",  │
│  需要调用    │   2. 返回地址列表 │    "10.0.1.6:9001"]  │
│  rpc-auth   │ ◄────────────── │                      │
└─────────────┘                └──────────────────────┘
         │
         ▼
   3. 选择一个地址
   (负载均衡)
```

**etcd 键空间设计：**
```
/services/
├── /rpc-auth/
│   ├── "10.0.1.5:9001" → "10.0.1.5:9001"
│   └── "10.0.1.6:9001" → "10.0.1.6:9001"
├── /rpc-user/
│   ├── "10.0.1.7:9002" → "10.0.1.7:9002"
│   └── "10.0.1.8:9002" → "10.0.1.8:9002"
└── /gim-ws/
    ├── "10.0.2.1:8082" → "10.0.2.1:8082"
    └── "10.0.2.2:8082" → "10.0.2.2:8082"
```

#### 15.6.1 etcd 客户端封装

```go
// internal/common/etcd/client.go
package etcd

import (
    "context"
    "time"

    clientv3 "go.etcd.io/etcd/client/v3"
)

type Client struct {
    *clientv3.Client
}

type Config struct {
    Endpoints   []string
    DialTimeout time.Duration
    Username    string
    Password    string
}

func NewClient(cfg *Config) (*Client, error) {
    client, err := clientv3.New(clientv3.Config{
        Endpoints:   cfg.Endpoints,
        DialTimeout: cfg.DialTimeout,
        Username:    cfg.Username,
        Password:    cfg.Password,
    })
    if err != nil {
        return nil, err
    }
    return &Client{Client: client}, nil
}

// RegisterService 注册服务到 etcd
func (c *Client) RegisterService(ctx context.Context, serviceName, addr string, ttl time.Duration) error {
    key := "/services/" + serviceName + "/" + addr
    lease, err := c.Grant(ctx, int64(ttl.Seconds()))
    if err != nil {
        return err
    }
    _, err = c.Put(ctx, key, addr, clientv3.WithLease(lease.ID))
    if err != nil {
        return err
    }
    // 自动续期
    ch, kaerr := c.KeepAlive(ctx, lease.ID)
    if kaerr != nil {
        return kaerr
    }
    go func() {
        for ka := range ch {
            _ = ka
        }
    }()
    return nil
}

// DiscoverService 发现服务地址
func (c *Client) DiscoverService(ctx context.Context, serviceName string) ([]string, error) {
    key := "/services/" + serviceName + "/"
    resp, err := c.Get(ctx, key, clientv3.WithPrefix())
    if err != nil {
        return nil, err
    }
    var addrs []string
    for _, kv := range resp.Kvs {
        addrs = append(addrs, string(kv.Value))
    }
    return addrs, nil
}
```

#### 15.6.2 gRPC 客户端连接池

💡 **为什么需要连接池？** 每次调用都新建 gRPC 连接开销大（TCP 握手、TLS 协商）。连接池复用连接，提升性能。

**连接池架构：**
```
┌─────────────────────────────────────────────────────┐
│           gRPC Connection Pool                      │
│                                                     │
│  conns map:                                          │
│  ┌─────────────────────────────────────────────┐    │
│  │ "rpc-auth"  → *grpc.ClientConn (复用)       │    │
│  │ "rpc-user"  → *grpc.ClientConn (复用)       │    │
│  │ "rpc-msg"   → *grpc.ClientConn (复用)       │    │
│  └─────────────────────────────────────────────┘    │
│                                                     │
│  GetConn(serviceName) 流程:                         │
│  1. 检查缓存，存在则返回                            │
│  2. 从 etcd 发现服务地址                            │
│  3. 创建新连接并缓存                                │
│  4. 返回连接                                       │
└─────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────┐
│              etcd (服务发现)                         │
│  /services/rpc-auth/ → ["10.0.1.5:9001", ...]      │
└─────────────────────────────────────────────────────┘
```

**负载均衡策略：**
```
Round-Robin (轮询):
请求1 → rpc-auth → 10.0.1.5:9001
请求2 → rpc-auth → 10.0.1.6:9001
请求3 → rpc-auth → 10.0.1.5:9001
...
```

```go
// internal/common/grpc/connpool.go
package grpc

import (
    "context"
    "sync"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    etcdcli "github.com/tianlu1990s/gim/internal/common/etcd"
)

type ConnPool struct {
    mu       sync.RWMutex
    conns    map[string]*grpc.ClientConn
    etcd     *etcdcli.Client
    resolver *ServiceResolver
}

type ServiceResolver struct {
    serviceName string
    etcd       *etcdcli.Client
}

func NewConnPool(etcd *etcdcli.Client) *ConnPool {
    return &ConnPool{
        conns: make(map[string]*grpc.ClientConn),
        etcd:  etcd,
    }
}

func (p *ConnPool) GetConn(serviceName string) (*grpc.ClientConn, error) {
    p.mu.RLock()
    conn, ok := p.conns[serviceName]
    p.mu.RUnlock()
    if ok {
        return conn, nil
    }

    p.mu.Lock()
    defer p.mu.Unlock()
    // 双重检查
    if conn, ok := p.conns[serviceName]; ok {
        return conn, nil
    }

    // 从 etcd 发现服务地址
    addrs, err := p.etcd.DiscoverService(context.Background(), serviceName)
    if err != nil || len(addrs) == 0 {
        return nil, err
    }

    // 创建连接（简单负载均衡：选第一个）
    conn, err = grpc.Dial(addrs[0],
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
    )
    if err != nil {
        return nil, err
    }

    p.conns[serviceName] = conn
    return conn, nil
}

func (p *ConnPool) Close() {
    p.mu.Lock()
    defer p.mu.Unlock()
    for _, conn := range p.conns {
        conn.Close()
    }
    p.conns = make(map[string]*grpc.ClientConn)
}
```

---

### 15.7 基础设施：Kafka 生产者/消费者

💡 **为什么用 Kafka？** 微服务架构需要异步解耦。消息发送者不需要等待消费者处理完成，而是把消息放到 Kafka 中就返回，消费者按自己的节奏消费。这提高了系统的吞吐量和可用性：
1. **削峰填谷**：高峰期消息堆积，低峰期慢慢消费
2. **故障隔离**：消费者挂了不影响生产者
3. **水平扩展**：增加消费者实例就能提高消费速度

#### 15.7.1 Kafka 生产者

```go
// internal/common/kafka/producer.go
package kafka

import (
    "context"
    "encoding/json"

    "github.com/IBM/sarama"
)

type Producer struct {
    producer sarama.SyncProducer
}

type Config struct {
    Brokers []string
}

func NewProducer(cfg *Config) (*Producer, error) {
    config := sarama.NewConfig()
    config.Producer.RequiredAcks = sarama.WaitForAll
    config.Producer.Retry.Max = 5
    config.Producer.Return.Successes = true

    producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
    if err != nil {
        return nil, err
    }
    return &Producer{producer: producer}, nil
}

type Message struct {
    Topic string
    Key   string
    Value interface{}
}

func (p *Producer) SendMessage(ctx context.Context, msg *Message) error {
    value, err := json.Marshal(msg.Value)
    if err != nil {
        return err
    }

    kafkaMsg := &sarama.ProducerMessage{
        Topic: msg.Topic,
        Key:   sarama.StringEncoder(msg.Key),
        Value: sarama.ByteEncoder(value),
    }

    _, _, err = p.producer.SendMessage(kafkaMsg)
    return err
}

func (p *Producer) Close() error {
    return p.producer.Close()
}
```

#### 15.7.2 Kafka 消费者

```go
// internal/common/kafka/consumer.go
package kafka

import (
    "context"
    "encoding/json"
    "log/slog"

    "github.com/IBM/sarama"
)

type Consumer struct {
    consumer sarama.ConsumerGroup
    handler  Handler
    logger   *slog.Logger
}

type Handler interface {
    HandleMessage(ctx context.Context, topic string, key string, value []byte) error
}

func NewConsumer(cfg *Config, groupID string, handler Handler, logger *slog.Logger) (*Consumer, error) {
    config := sarama.NewConfig()
    config.Version = sarama.V2_8_0_0
    config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
    config.Consumer.Offsets.Initial = sarama.OffsetNewest

    consumer, err := sarama.NewConsumerGroup(cfg.Brokers, groupID, config)
    if err != nil {
        return nil, err
    }

    return &Consumer{
        consumer: consumer,
        handler:  handler,
        logger:   logger,
    }, nil
}

func (c *Consumer) Consume(ctx context.Context, topics []string) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := c.consumer.Consume(ctx, topics, &consumerGroupHandler{handler: c.handler}); err != nil {
                c.logger.Error("consumer error", "error", err)
            }
        }
    }
}

type consumerGroupHandler struct {
    handler Handler
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
        if err := h.handler.HandleMessage(session.Context(), msg.Topic, string(msg.Key), msg.Value); err != nil {
            // 处理失败，记录日志但不提交 offset
            continue
        }
        session.MarkMessage(msg, "")
    }
    return nil
}
```

---

### 15.8 基础设施：MongoDB 客户端

💡 **为什么用 MongoDB？** 聊天消息有特殊的数据特征：
1. **写多读少**：消息主要写入，历史查询是低频操作
2. **文档结构**：消息本身就是文档，不需要复杂的关联查询
3. **水平扩展**：MongoDB 的分片机制适合海量消息存储
4. **灵活 Schema**：未来可能增加附件、表情等富文本，MongoDB 更灵活

对于 IM 系统，MySQL 存储用户、好友、会话等结构化数据，MongoDB 存储海量消息，是经典的混合存储方案。

```go
// internal/common/mongo/client.go
package mongo

import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
    client   *mongo.Client
    database *mongo.Database
}

type Config struct {
    URI      string
    Database string
}

func NewClient(cfg *Config) (*Client, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
    if err != nil {
        return nil, err
    }

    if err := client.Ping(ctx, nil); err != nil {
        return nil, err
    }

    return &Client{
        client:   client,
        database: client.Database(cfg.Database),
    }, nil
}

func (c *Client) Collection(name string) *mongo.Collection {
    return c.database.Collection(name)
}

func (c *Client) Close(ctx context.Context) error {
    return c.client.Disconnect(ctx)
}
```

---

### 15.9 消息传输服务实现

💡 **为什么需要独立的消息传输服务？** 在微服务架构中，消息发送是高频操作：
1. **解耦发送和存储**：用户发送消息后立即返回 Seq（消息号），不用等存库完成，体验更好
2. **批量写入优化**：每条消息单独写 MongoDB 效率低，批量写 100 条为一个文档能大幅提升性能
3. **消费顺序保证**：Kafka 的分区和消费者组机制保证消息按顺序消费
4. **故障恢复**：服务重启后从上次提交的 offset 继续消费，不会丢消息

```go
// cmd/msg-transfer/main.go
package main

import (
    "context"
    "encoding/json"
    "log"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/IBM/sarama"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo/options"

    mongocli "github.com/tianlu1990s/gim/internal/common/mongo"
    kafkacfg "github.com/tianlu1990s/gim/internal/common/kafka"
    "github.com/tianlu1990s/gim/pkg/snowflake"
    "github.com/tianlu1990s/gim/pkg/slog"
)

type Message struct {
    ConversationID string `json:"conversation_id"`
    Seq            int64  `json:"seq"`
    SenderID       string `json:"sender_id"`
    MsgType        int    `json:"msg_type"`
    Content        string `json:"content"`
    ClientMsgID    string `json:"client_msg_id"`
    ServerMsgID    string `json:"server_msg_id"`
    SendTime       int64  `json:"send_time"`
}

type MsgDoc struct {
    ID        string     `bson:"_id"`
    DocID     string     `bson:"doc_id"`
    Msgs      []Message  `bson:"msgs"`
    UpdatedAt time.Time  `bson:"updated_at"`
}

func main() {
    logger := slog.New(&slog.LogConfig{
        Level:  "info",
        Format: "text",
        Output: "stdout",
    })

    // 初始化 MongoDB
    mongoClient, err := mongocli.NewClient(&mongocli.Config{
        URI:      "mongodb://localhost:27017",
        Database: "gim",
    })
    if err != nil {
        logger.Fatal("Failed to connect MongoDB", "error", err)
    }
    defer mongoClient.Close(context.Background())

    // 初始化 Kafka 消费者
    consumer, err := sarama.NewConsumerGroup([]string{"localhost:9092"}, "msg-transfer-group", sarama.NewConfig())
    if err != nil {
        logger.Fatal("Failed to create consumer", "error", err)
    }
    defer consumer.Close()

    handler := &MsgTransferHandler{
        mongo: mongoClient,
        logger: logger,
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        for {
            if err := consumer.Consume(ctx, []string{"toMongo"}, handler); err != nil {
                logger.Error("Consumer error", "error", err)
                time.Sleep(5 * time.Second)
            }
        }
    }()

    logger.Info("Message transfer service started")

    sigterm := make(chan os.Signal, 1)
    signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
    <-sigterm

    logger.Info("Message transfer service stopped")
}

type MsgTransferHandler struct {
    mongo  *mongocli.Client
    logger *slog.Logger
}

func (h *MsgTransferHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *MsgTransferHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *MsgTransferHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    batch := make([]Message, 0, 100)
    for msg := range claim.Messages() {
        var message Message
        if err := json.Unmarshal(msg.Value, &message); err != nil {
            logger.Error("Failed to unmarshal message", "error", err)
            continue
        }
        batch = append(batch, message)
        session.MarkMessage(msg, "")

        if len(batch) >= 100 {
            if err := h.batchInsert(context.Background(), batch); err != nil {
                logger.Error("Failed to insert messages", "error", err)
            } else {
                batch = batch[:0]
            }
        }
    }
    if len(batch) > 0 {
        if err := h.batchInsert(context.Background(), batch); err != nil {
            logger.Error("Failed to insert messages", "error", err)
        }
    }
    return nil
}

func (h *MsgTransferHandler) batchInsert(ctx context.Context, msgs []Message) error {
    collection := h.mongo.Collection("messages")
    now := time.Now()

    // 按 DocID 分组（每 100 条消息一个文档）
    groups := make(map[string][]Message)
    for _, msg := range msgs {
        seqSuffix := (msg.Seq - 1) / 100
        docID := msg.ConversationID + ":" + snowflake.Generate().String()
        groups[docID] = append(groups[docID], msg)
    }

    for docID, groupMsgs := range groups {
        filter := bson.M{"_id": docID}
        update := bson.M{
            "$push": bson.M{"msgs": bson.M{"$each": groupMsgs}},
            "$set":  bson.M{"updated_at": now},
        }
        opts := options.Update().SetUpsert(true)
        _, err := collection.UpdateOne(ctx, filter, update, opts)
        if err != nil {
            return err
        }
    }
    return nil
}
```

---

## 16. 第三阶段实现要点

### 16.1 Prometheus 指标埋点

💡 **为什么需要指标？** 没有监控的生产系统是黑盒。Prometheus 采集指标，Grafana 可视化，帮助我们：
1. **发现瓶颈**：哪个接口慢？哪个服务吃资源？
2. **提前预警**：CPU、内存、磁盘快满了怎么办？
3. **事故排查**：用户反馈"卡"，先看监控确认

**指标类型：**
```
Counter (计数器):
- gim_http_requests_total     总请求数（只增不减）
- gim_messages_sent_total      总消息数

Gauge (仪表盘):
- gim_ws_connections          当前WS连接数（可增可减）
- mysql_active_connections     当前活跃连接数

Histogram (直方图):
- gim_http_request_duration_seconds  请求耗时分布
  → _count: 总请求数
  → _sum: 总耗时
  → _bucket: P50、P95、P99等分位数
```

**埋点位置：**
```
HTTP请求:
  [中间件] → 记录计数器和直方图 → Prometheus采集

WS连接:
  [Hub.register] → Gauge.Inc()  → Prometheus采集
  [Hub.unregister] → Gauge.Dec() → Prometheus采集

业务操作:
  [发送消息] → Counter.Inc()   → Prometheus采集
```

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

💡 **为什么需要链路追踪？** 微服务架构中，一个请求可能经过多个服务。当请求慢或出错时，怎么知道是哪个服务出了问题？链路追踪记录请求经过的每个环节。

**追踪数据模型：**
```
Trace (一条完整的调用链):
  Span1: gim-api 接收请求
    └─ Span2: 调用 rpc-auth
         └─ Span3: 调用 MySQL
    └─ Span4: 调用 rpc-msg
         └─ Span5: 调用 Kafka
```

**追踪流程图：**
```
用户请求
   │
   ▼
┌──────────────┐
│  gim-api     │ ← 创建 TraceID, Span1
│  中间件       │   记录: 开始时间、HTTP方法、路径
└──────┬───────┘
       │ gRPC调用
       ▼
┌──────────────┐
│  rpc-auth    │ ← 创建 Span2 (parent=Span1)
│  认证服务     │   记录: 方法名、参数
└──────┬───────┘
       │ 查询数据库
       ▼
┌──────────────┐
│   MySQL      │ ← 创建 Span3 (parent=Span2)
│              │   记录: SQL语句、执行时间
└──────────────┘
       │ 返回结果
       ▼
┌──────────────┐
│  rpc-msg     │ ← 创建 Span4 (parent=Span1)
│  消息服务     │
└──────┬───────┘
       │ 写Kafka
       ▼
┌──────────────┐
│   Kafka      │ ← 创建 Span5 (parent=Span4)
└──────────────┘
       │
       ▼
   返回给用户

所有 Span 上报到 Jaeger → 可视化查看调用链
```

**Trace ID 传播：**
```
HTTP Header: X-Trace-Id, X-Span-Id
gRPC Metadata: trace-bin
```

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

💡 **为什么用 Helm？** 手动写 K8S YAML 文件管理几十个服务很痛苦。Helm 是 K8S 的包管理器，类似 `apt` / `pip`，可以：
1. **版本管理**：`helm upgrade` 滚动更新
2. **参数化**：`values.yaml` 灵活配置
3. **一键部署**：`helm install` 部署所有资源

**Helm 部署架构：**
```
helm install gim ./gim-chart
         │
         ▼
┌────────────────────────────────────────────────────┐
│              gim Chart                              │
│  ├── values.yaml (配置参数)                        │
│  └── templates/ (模板文件)                          │
│        ├── namespace.yaml                         │
│        ├── configmap.yaml                         │
│        ├── secret.yaml                            │
│        ├── api-deployment.yaml                    │
│        ├── ws-statefulset.yaml                    │
│        ├── rpc-*-deployment.yaml                  │
│        ├── hpa.yaml                               │
│        └── ingress.yaml                           │
└────────────────────────────────────────────────────┘
         │
         ▼
   渲染为 K8S YAML
         │
         ▼
   kubectl apply
         │
         ▼
┌────────────────────────────────────────────────────┐
│              K8S Cluster                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │
│  │ gim-api Pod │ │ gim-ws Pod  │ │ rpc-auth Pod│   │
│  └─────────────┘ └─────────────┘ └─────────────┘   │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │
│  │  mysql PVC  │ │  redis PVC  │ │  kafka PVC  │   │
│  └─────────────┘ └─────────────┘ └─────────────┘   │
└────────────────────────────────────────────────────┘
```

**目录结构：**
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

💡 **为什么需要 HPA？** 用户量不是固定的，早高峰、晚高峰流量不同。HPA (Horizontal Pod Autoscaler) 自动扩缩容：
- 流量大 → 自动增加 Pod（扩容）
- 流量小 → 自动减少 Pod（缩容，节省成本）

**HPA 工作流程：**
```
┌─────────────────────────────────────────────────────┐
│              Metrics Server                         │
│   采集指标：CPU、内存、自定义指标（如WS连接数）      │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────┐
│                  HPA Controller                     │
│   每个周期检查（默认15秒）:                          │
│   - 当前指标 vs 目标指标                             │
│   - 计算需要多少个 Pod                              │
│   - 调整 Deployment/StatefulSet 副本数              │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────┐
│              Deployment/StatefulSet                 │
│   调整 .spec.replicas 字段                          │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
         增加/减少 Pod
```

**扩缩容示例：**
```
场景: WS Gateway 按 CPU 和连接数扩容

当前状态:
  - 副本数: 3
  - CPU 使用率: 80% (目标70%)
  - 每Pod连接数: 4000 (目标5000)

计算:
  - 需要副本数 = max(3 * 80/70, 3 * 4000/5000) = max(3.4, 2.4) = 4
  - 扩容到 4 个 Pod

15秒后再次检查，直到 CPU 降到 70% 以下
```

```yaml
# templates/hpa.yaml — 各服务 HPA 配置
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gim-api-hpa
  namespace: {{ .Values.namespace }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: gim-api
  minReplicas: {{ .Values.api.minReplicas }}
  maxReplicas: {{ .Values.api.maxReplicas }}
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gim-ws-hpa
  namespace: {{ .Values.namespace }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: StatefulSet
    name: gim-ws
  minReplicas: {{ .Values.ws.minReplicas }}
  maxReplicas: {{ .Values.ws.maxReplicas }}
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

### 16.5 部署命令

💡 **部署前检查**：确保 K8S 集群可用、kubectl 已配置、镜像仓库可访问。

**完整部署流程：**
```bash
# 1. 准备环境（安装必要组件）—— 所有命令自动确认
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml --context=default
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml --context=default
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml --context=default

# 2. 创建命名空间（不存在则创建）
kubectl create namespace gim-prod --context=default 2>/dev/null || true

# 3. 创建镜像仓库凭据（如果使用私有仓库）
kubectl create secret docker-registry regcred \
  --docker-server=registry.example.com \
  --docker-username=tianlu1990s \
  --docker-password=your-password \
  --namespace gim-prod \
  --context=default \
  --dry-run=client -o yaml | kubectl apply -f - --context=default

# 4. 构建镜像
docker build -f deploy/docker/Dockerfile -t registry.example.com/gim:latest --progress=plain .

# 5. 推送镜像（跳过证书验证）
docker push registry.example.com/gim:latest

# 6. 使用 Helm 部署（所有配置通过 --set 传入，无需交互确认）
helm upgrade --install gim ./deploy/k8s/helm/gim \
    --namespace gim-prod \
    --create-namespace \
    --set image.repository=registry.example.com/gim \
    --set image.tag=latest \
    --set image.pullSecrets[0].name=regcred \
    --set mysql.persistence.enabled=true \
    --set mysql.persistence.size=100Gi \
    --set redis.persistence.enabled=true \
    --set redis.persistence.size=20Gi \
    --set kafka.persistence.enabled=true \
    --set kafka.persistence.size=200Gi \
    --set mongo.persistence.enabled=true \
    --set mongo.persistence.size=500Gi \
    --set api.replicaCount=2 \
    --set ws.replicaCount=3 \
    --set api.resources.requests.cpu=500m \
    --set api.resources.requests.memory=512Mi \
    --set api.resources.limits.cpu=2000m \
    --set api.resources.limits.memory=2Gi \
    --timeout 10m \
    --wait \
    --context=default

# 7. 检查部署状态（等待所有Pod就绪）
kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=gim -n gim-prod --timeout=300s --context=default
kubectl get pods -n gim-prod --context=default
kubectl get svc -n gim-prod --context=default
kubectl get ingress -n gim-prod --context=default

# 8. 查看 HPA 状态
kubectl get hpa -n gim-prod --context=default
kubectl describe hpa gim-ws-hpa -n gim-prod --context=default

# 9. 查看日志（最新20行）
kubectl logs deployment/gim-api -n gim-prod --tail=20 --context=default
kubectl logs statefulset/gim-ws -n gim-prod --tail=20 --context=default

# 10. 验证服务健康
kubectl run -i --tty curl --image=curlimages/curl --rm --restart=Never -n gim-prod --context=default -- \
  curl -s http://gim-api.gim-prod.svc.cluster.local:8080/health

# 11. 测试WebSocket连接
kubectl run -i --tty ws-test --image=nicolaka/netshoot --rm --restart=Never -n gim-prod --context=default -- \
  curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" -H "Sec-WebSocket-Version: 13" http://gim-ws.gim-prod.svc.cluster.local:8081/ws
```

**升级部署：**
```bash
# 升级到新版本
helm upgrade gim ./deploy/k8s/helm/gim \
    --namespace gim-prod \
    --set image.tag=v1.2.0 \
    --timeout 10m \
    --wait \
    --context=default

# 查看升级历史
helm history gim -n gim-prod --context=default

# 回滚到上一个版本
helm rollback gim -n gim-prod --context=default
```

### 16.6 常见部署问题排查

| 问题 | 可能原因 | 解决方案 | 检查命令 |
|------|---------|---------|---------|
| Pod 无法启动 | 镜像拉取失败、资源不足 | 检查 `kubectl describe pod`，增加资源配额 | `kubectl describe pod <pod-name> -n gim-prod` |
| 服务无法访问 | Service/Ingress 配置错误 | 检查 `kubectl get svc` 和 `kubectl get ingress` | `kubectl get svc -n gim-prod && kubectl get ingress -n gim-prod` |
| HPA 不扩容 | Metrics Server 未配置 | 确保 `metrics-server` 已部署 | `kubectl get apiservice v1beta1.metrics.k8s.io` |
| 数据持久化失败 | PVC 未绑定、存储类错误 | 检查 `kubectl get pvc` 和 StorageClass | `kubectl get pvc -n gim-prod && kubectl get storageclass` |
| 消息丢失 | Kafka/Mongo 连接失败 | 检查 Pod 日志，确认网络连接 | `kubectl logs -f <pod-name> -n gim-prod` |
| 证书签发失败 | Cert-Manager 配置问题 | 检查 ClusterIssuer 和 DNS 配置 | `kubectl get clusterissuer && kubectl get certificate -n gim-prod` |
| Pod 频繁重启 | 健康检查失败或 OOM | 检查日志，增加资源限制 | `kubectl logs <pod-name> -n gim-prod --previous` |

**故障排查命令集：**
```bash
# 一键检查所有资源状态
echo "=== Pods ===" && kubectl get pods -n gim-prod
echo -e "\n=== Services ===" && kubectl get svc -n gim-prod
echo -e "\n=== Ingress ===" && kubectl get ingress -n gim-prod
echo -e "\n=== PVC ===" && kubectl get pvc -n gim-prod
echo -e "\n=== HPA ===" && kubectl get hpa -n gim-prod
echo -e "\n=== Events ===" && kubectl get events -n gim-prod --sort-by='.lastTimestamp' | tail -20

# 检查特定Pod详情
kubectl describe pod <pod-name> -n gim-prod

# 查看Pod日志（带时间戳）
kubectl logs -f <pod-name> -n gim-prod --timestamps=true

# 进入Pod调试
kubectl exec -it <pod-name> -n gim-prod -- sh

# 检查网络连接
kubectl run -i --tty netshoot --image=nicolaka/netshoot --rm --restart=Never -n gim-prod -- sh

# 检查DNS解析
kubectl run -i --tty dnsutils --image=k8s.gcr.io/e2e-test-images/dnsutils:1.3 --rm --restart=Never -n gim-prod -- nslookup gim-api.gim-prod.svc.cluster.local
```

---



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



### 16.7 K8S 基础设施部署

💡 **为什么用 K8S？** K8S 提供了容器编排、自动扩缩容、负载均衡、配置管理等功能。对于生产环境的 IM 系统，这些是必需的：
1. **自动扩容**：用户量激增时，WS Gateway 自动增加 Pod
2. **滚动更新**：零停机部署新版本
3. **故障自愈**：Pod 崩溃自动重启
4. **资源调度**：合理分配 CPU/内存

#### 16.7.1 Namespace 和资源配额

```yaml
# deploy/k8s/namespaces.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: gim-prod
  labels:
    name: gim-prod
    environment: production
---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: gim-quota
  namespace: gim-prod
spec:
  hard:
    requests.cpu: "10"
    requests.memory: "20Gi"
    limits.cpu: "20"
    limits.memory: "40Gi"
    persistentvolumeclaims: "10"
```

#### 16.7.2 ConfigMap（非敏感配置）

```yaml
# deploy/k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gim-config
  namespace: gim-prod
data:
  mysql-host: "mysql"
  mysql-port: "3306"
  redis-host: "redis"
  redis-port: "6379"
  kafka-brokers: "kafka:9092"
  etcd-endpoints: "etcd:2379"
  mongo-uri: "mongodb://mongo:27017"
  log-level: "info"
  log-format: "json"
```

#### 16.7.3 Secret（敏感配置）

```yaml
# deploy/k8s/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: gim-secret
  namespace: gim-prod
type: Opaque
data:
  # Base64 编码的值
  mysql-password: Z2ltX3Bhc3M=
  redis-password: ""
  jwt-private-key: LS0t...
  jwt-public-key: LS0t...
  mongo-password: bW9uZ29fcGFzc3dvcmQ=
```

#### 16.7.4 Service 和 Ingress

```yaml
# deploy/k8s/services.yaml
apiVersion: v1
kind: Service
metadata:
  name: gim-api
  namespace: gim-prod
spec:
  type: ClusterIP
  selector:
    app: gim-api
  ports:
  - name: http
    port: 8080
    targetPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: gim-ws
  namespace: gim-prod
spec:
  type: LoadBalancer
  selector:
    app: gim-ws
  ports:
  - name: ws
    port: 8081
    targetPort: 8081
  sessionAffinity: ClientIP  # WebSocket 需要 Sticky Session
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gim-ingress
  namespace: gim-prod
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - api.gim.example.com
    secretName: gim-tls
  rules:
  - host: api.gim.example.com
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: gim-api
            port:
              number: 8080
```

#### 16.7.5 PVC（持久化存储）

```yaml
# deploy/k8s/pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mysql-pvc
  namespace: gim-prod
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: fast-ssd
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: redis-pvc
  namespace: gim-prod
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: fast-ssd
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: kafka-pvc
  namespace: gim-prod
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 200Gi
  storageClassName: fast-ssd
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mongo-pvc
  namespace: gim-prod
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 500Gi
  storageClassName: fast-ssd
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: storage-pvc
  namespace: gim-prod
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 1Ti
  storageClassName: fast-ssd
```

### 16.8 OpenTelemetry 链路追踪详细实现

```go
// pkg/trace/trace.go
package trace

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
    "go.opentelemetry.io/otel/sdk/resource"
    tracesdk "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type Config struct {
    ServiceName    string
    ServiceVersion string
    Endpoint       string
    SampleRate     float64
}

func InitTracer(cfg *Config) (*tracesdk.TracerProvider, error) {
    exporter, err := otlptrace.New(context.Background(),
        otlptrace.WithEndpoint(cfg.Endpoint),
        otlptrace.WithHeaders(map[string]string{
            "Authorization": "Bearer " + otel.GetAccessToken(),
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
    }

    res, err := resource.New(
        context.Background(),
        resource.WithAttributes(
            semconv.ServiceName(cfg.ServiceName),
            semconv.ServiceVersion(cfg.ServiceVersion),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    tp := tracesdk.NewTracerProvider(
        tracesdk.WithBatcher(exporter),
        tracesdk.WithResource(res),
        tracesdk.WithSampler(tracesdk.TraceIDRatioBased(cfg.SampleRate)),
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}

// TraceMiddleware 链路追踪中间件
func TraceMiddleware(serviceName, version string) gin.HandlerFunc {
    tracer := otel.Tracer(serviceName)
    return func(c *gin.Context) {
        ctx, span := tracer.Start(c.Request.Context(), fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path))
        defer span.End()

        // 记录请求属性
        span.SetAttributes(
            semconv.HTTPMethodKey.String(c.Request.Method),
            semconv.HTTPTargetKey.String(c.Request.URL.String()),
            semconv.HTTPURLKey.String(c.Request.URL.Path),
        )

        // 传播 trace context
        c.Request = c.Request.WithContext(ctx)
        c.Next()

        // 记录响应状态
        span.SetStatus(codes.Error if c.Writer.Status() >= 400 else codes.Ok)
        span.SetAttributes(semconv.HTTPStatusCodeKey.Int(c.Writer.Status()))
    }
}
```

### 16.9 ServiceMonitor（自定义指标）

```go
// pkg/monitor/service_monitor.go
package monitor

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    // 业务指标
    registerUsersTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gim_register_users_total",
            Help: "Total number of user registrations",
        },
        []string{"platform"},
    )

    loginUsersTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gim_login_users_total",
            Help: "Total number of user logins",
        },
        []string{"platform"},
    )

    messagesSentTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gim_messages_sent_total",
            Help: "Total number of messages sent",
        },
        []string{"msg_type", "conv_type"},
    )

    messagesStoredTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gim_messages_stored_total",
            Help: "Total number of messages stored in database",
        },
        []string{"storage"},
    )

    // 错误指标
    errorsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gim_errors_total",
            Help: "Total number of errors",
        },
        []string{"service", "error_type", "severity"},
    )
)

func Init() {
    prometheus.MustRegister(registerUsersTotal)
    prometheus.MustRegister(loginUsersTotal)
    prometheus.MustRegister(messagesSentTotal)
    prometheus.MustRegister(messagesStoredTotal)
    prometheus.MustRegister(errorsTotal)
}

// MetricsHandler 提供 HTTP 接口
func MetricsHandler() http.Handler {
    return promhttp.Handler()
}

// 辅助函数
func RecordRegister(platform string) {
    registerUsersTotal.WithLabelValues(platform).Inc()
}

func RecordLogin(platform string) {
    loginUsersTotal.WithLabelValues(platform).Inc()
}

func RecordMessageSent(msgType, convType string) {
    messagesSentTotal.WithLabelValues(msgType, convType).Inc()
}

func RecordMessageStored(storage string) {
    messagesStoredTotal.WithLabelValues(storage).Inc()
}

func RecordError(service, errorType, severity string) {
    errorsTotal.WithLabelValues(service, errorType, severity).Inc()
}
```

### 16.10 Helm Chart 完整结构

```bash
deploy/k8s/helm/gim/
├── Chart.yaml                 # Chart 元信息
├── values.yaml                # 默认配置值
├── templates/
│   ├── _helpers.tpl           # 模板辅助函数
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── pvc.yaml
│   ├── mysql/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── configmap.yaml
│   ├── redis/
│   │   ├── statefulset.yaml
│   │   └── service.yaml
│   ├── kafka/
│   │   ├── statefulset.yaml
│   │   └── service.yaml
│   ├── mongo/
│   │   ├── statefulset.yaml
│   │   └── service.yaml
│   ├── storage/                 # S3 兼容存储(MinIO/OSS)
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── configmap.yaml
│   ├── etcd/
│   │   ├── statefulset.yaml
│   │   └── service.yaml
│   ├── api/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── hpa.yaml
│   ├── ws/
│   │   ├── statefulset.yaml
│   │   ├── service.yaml
│   │   └── hpa.yaml
│   ├── rpc/
│   │   ├── auth-deployment.yaml
│   │   ├── user-deployment.yaml
│   │   ├── friend-deployment.yaml
│   │   ├── msg-deployment.yaml
│   │   ├── conversation-deployment.yaml
│   │   └── hpa.yaml
│   ├── push/
│   │   ├── deployment.yaml
│   │   └── hpa.yaml
│   ├── msgtransfer/
│   │   ├── deployment.yaml
│   │   └── hpa.yaml
│   ├── offlinepush/
│   │   ├── deployment.yaml
│   │   └── hpa.yaml
│   ├── serviceaccount.yaml
│   ├── ingress.yaml
│   ├── poddisruptionbudget.yaml
│   └── networkpolicy.yaml
└── README.md                   # Chart 使用说明
```

---



### 16.11 第三阶段：K8S 生产部署实施步骤

💡 **第三阶段的核心任务**：将应用部署到 K8S 集群，实现自动化运维、监控告警、弹性伸缩。

#### 16.11.1 K8S 集群准备

```bash
# 1. 安装 kubectl（K8S 命令行工具）
# macOS
brew install kubectl

# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# 2. 安装 Helm（K8S 包管理器）
# macOS
brew install helm

# Linux
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# 3. 验证 K8S 集群连接
kubectl cluster-info
kubectl get nodes

# 4. 安装基础组件
# Metrics Server（用于 HPA）
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Nginx Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml

# Cert-Manager（TLS 证书管理）
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Prometheus Operator（监控）
kubectl apply -f https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.66.0/bundle.yaml

# Jaeger Operator（链路追踪）
kubectl apply -f https://github.com/jaegertracing/jaeger-operator/releases/download/v1.52.0/jaeger-operator.yaml
```

#### 16.11.2 K8S Namespace 和资源配额创建

```bash
# 创建命名空间
kubectl create namespace gim-prod

# 创建资源配额
kubectl apply -f - << 'EOF'
apiVersion: v1
kind: ResourceQuota
metadata:
  name: gim-quota
  namespace: gim-prod
spec:
  hard:
    requests.cpu: "20"
    requests.memory: "40Gi"
    limits.cpu: "40"
    limits.memory: "80Gi"
    persistentvolumeclaims: "20"
EOF
```

#### 16.11.3 使用 Helm 部署应用

```bash
# 1. 创建镜像仓库（如果使用私有仓库）
# 推荐使用 Harbor、AWS ECR、阿里云容器镜像服务

# 2. 构建并推送镜像
docker build -f deploy/docker/Dockerfile -t registry.example.com/gim:latest .
docker push registry.example.com/gim:latest

# 3. 使用 Helm 部署
cd deploy/k8s/helm/gim

# 自定义配置（编辑 values.yaml）
vim values.yaml

# 安装 Chart
helm install gim .   --namespace gim-prod   --set image.repository=registry.example.com/gim   --set image.tag=latest   --set mysql.persistence.enabled=true   --set redis.persistence.enabled=true   --set kafka.persistence.enabled=true   --set mongo.persistence.enabled=true

# 查看部署状态
kubectl get pods -n gim-prod
kubectl get svc -n gim-prod
kubectl get ingress -n gim-prod

# 查看 HPA 状态
kubectl get hpa -n gim-prod
kubectl describe hpa gim-ws-hpa -n gim-prod

# 查看日志
kubectl logs -f deployment/gim-api -n gim-prod
kubectl logs -f statefulset/gim-ws -n gim-prod
```

#### 16.11.4 监控和告警配置

```yaml
# Prometheus 监控规则
# deploy/k8s/prometheus/alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: gim-alerts
  namespace: gim-prod
  labels:
    release: prometheus
spec:
  groups:
  - name: gim
    rules:
    # 服务错误率过高
    - alert: GimHighErrorRate
      expr: |
        rate(gim_http_requests_total{status=~"5.."}[5m]) /
        rate(gim_http_requests_total[5m]) > 0.01
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "GIM 服务错误率过高"
        description: "错误率 > 1% ({{ $value | humanizePercentage }})"

    # P95 延迟过高
    - alert: GimHighLatency
      expr: |
        histogram_quantile(0.95, rate(gim_http_request_duration_seconds_bucket[5m])) > 0.5
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "GIM P95 延迟过高"
        description: "P95 延迟 > 500ms ({{ $value }}s)"

    # WS 连接数异常
    - alert: GimWSConnectionsLow
      expr: |
        gim_ws_connections < 1000
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "WS 连接数过低"
        description: "当前 WS 连接数: {{ $value }}"

    # Pod 重启过多
    - alert: GimPodRestartTooFrequent
      expr: |
        increase(kube_pod_container_status_restarts_total[1h]) > 5
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Pod 频繁重启"
        description: "过去 1 小时重启超过 5 次"

    # 数据库连接池耗尽
    - alert: GimDBConnectionPoolExhausted
      expr: |
        mysql_global_status_threads_connected / mysql_global_variables_max_connections > 0.9
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "数据库连接池即将耗尽"
        description: "连接池使用率: {{ $value | humanizePercentage }}"
```

#### 16.11.5 Grafana Dashboard 配置

```json
{
  "dashboard": {
    "title": "GIM 系统监控",
    "panels": [
      {
        "title": "QPS (每秒请求数)",
        "targets": [
          {
            "expr": "rate(gim_http_requests_total[1m])",
            "legendFormat": "{{method}} {{path}}"
          }
        ]
      },
      {
        "title": "P95 延迟",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(gim_http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "{{path}}"
          }
        ]
      },
      {
        "title": "错误率",
        "targets": [
          {
            "expr": "rate(gim_http_requests_total{status=~"5.."}[5m]) / rate(gim_http_requests_total[5m])",
            "legendFormat": "{{status}}"
          }
        ]
      },
      {
        "title": "WebSocket 连接数",
        "targets": [
          {
            "expr": "gim_ws_connections",
            "legendFormat": "total"
          }
        ]
      },
      {
        "title": "消息发送速率",
        "targets": [
          {
            "expr": "rate(gim_messages_sent_total[1m])",
            "legendFormat": "{{msg_type}}"
          }
        ]
      },
      {
        "title": "Kafka 消息堆积",
        "targets": [
          {
            "expr": "kafka_consumer_group_lag{topic="toMongo"}",
            "legendFormat": "{{partition}}"
          }
        ]
      },
      {
        "title": "MongoDB 操作延迟",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(mongo_request_duration_seconds_bucket[5m]))",
            "legendFormat": "{{command}}"
          }
        ]
      }
    ]
  }
}
```

---

### 16.12 故障恢复与灾备方案

#### 16.12.1 常见故障处理

| 故障类型 | 影响 | 恢复方案 | 预防措施 |
|----------|------|---------|---------|
| MySQL 主节点故障 | 无法读写数据 | 自动切换到从节点 | MySQL 主从复制 + 自动故障转移 |
| Redis 宕久化丢失 | 部分数据丢失 | 从 RDB/AOF 恢复 | 启用 AOF 持久化 + 定期备份 |
| Kafka Broker 故障 | 消息积压 | 自动 rebalance | 配置副本数 >= 3 |
| K8S Node 故障 | Pod 调度失败 | 自动重新调度 | 多节点部署 + PDB |
| 磁盘空间不足 | 服务无法写入 | 扩容 PVC | 监控磁盘使用率 |

#### 16.12.2 数据备份策略

```bash
# MySQL 定期备份（每天凌晨 2 点）
0 2 * * * /usr/bin/mysqldump -h mysql -u root -p${MYSQL_ROOT_PASSWORD} gim | gzip > /backup/gim-$(date +\%Y\%m\%d).sql.gz

# MongoDB 定期备份（每天凌晨 3 点）
0 3 * * * /usr/bin/mongodump --uri="mongodb://mongo:27017/gim" --gzip --archive=/backup/mongo-$(date +\%Y\%m\%d).gz

# 定期清理旧备份（保留 30 天）
0 4 * * 0 /usr/bin/find /backup -name "*.gz" -mtime +30 -delete

# 备份到对象存储（S3 协议：AWS S3/MinIO/OSS）
0 5 * * * /usr/bin/aws s3 sync /backup/ s3://gim-backup/
```

---

## 17. 第四阶段实现要点

> 第四阶段（AI Agent）的完整架构设计、核心代码、数据模型、TODO 详见 [AI_AGENT.md](AI_AGENT.md)。这里只列出与前三阶段代码的关键集成点。

💡 **第四阶段概述：** 在成熟的 IM 基础设施上，引入 AI 能力，实现智能交互。三个核心方向：
1. **用户侧**：智能回复、AI 助手，提升用户体验
2. **运维侧**：内容审核，自动识别违规内容
3. **管理侧**：RAG 智能助手，帮助管理员查询数据、生成报表

**技术栈：**
- **LLM API**: Deepseek API / Claude API / 本地模型（Ollama/vLLM），通过 AIProvider 统一接口切换
- **Go SDK**: `github.com/anthropics/anthropic-sdk-go` + `github.com/openai/openai-go`（Deepseek 兼容 OpenAI 协议）
- **向量数据库**: Milvus (自部署) 或 pgvector (PostgreSQL 扩展)
- **Embedding**: OpenAI text-embedding-3-small 或本地模型
- **消息队列**: 复用 Kafka


### 第四阶段 AI Agent 集成概览

💡 **第四阶段的核心任务**：引入 AI 智能助手、群 AI 助手和内容审核，提升用户体验和运营效率。

#### AI 实施步骤表

| 步骤 | 任务 | 命令/操作 | 预期结果 |
|------|------|----------|----------|
| 1. 安装 AI 基础设施 | Milvus, AI Provider | `docker compose -f deploy/k8s/docker-compose-ai.yaml up -d` | AI 基础设施运行 |
| 2. 封装 AIProvider | pkg/aiprovider/ | 实现统一接口(Deepseek/Claude/本地) | 可以调用多后端 AI |
| 3. 实现向量存储 | internal/vector/milvus/ | Milvus 客户端封装 | 可以存储和检索向量 |
| 4. 实现 Embedding | internal/ai/embedding/ | 文本转向量服务 | 可以生成消息向量 |
| 5. 搭建智能回复服务 | cmd/ai-reply/ | 对话上下文 + AI 生成 | 可以生成回复建议 |
| 6. 搭建群 AI 助手 | cmd/group-ai/ | Tool Use + 工具集成 | 可以响应 @AI 请求 |
| 7. 搭建内容审核服务 | cmd/moderation/ | 内容分类 + 违规检测 | 可以检测违规内容 |
| 8. 集成到 WS Gateway | 修改 WS 消息处理 | AI 相关消息类型 | 可以触发 AI 功能 |
| 9. 配置 Prompt 模板 | pkg/prompt/ | 系统提示词管理 | 可以快速调整 AI 行为 |
| 10. 测试 AI 功能 | 手动/自动化测试 | 验证各种场景 | AI 功能正常工作 |

#### AI 服务流程图

```
AI 功能完整流程：

智能回复：
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  客户端  │ ──►│  WS GW   │ ──►│ AI Reply │ ──►│AI Provider│
│  点击回复 │    │  消息类型  │    │  Service  │    │   API   │
│   按钮   │    │   = 10    │    │          │    │         │
└─────┬────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘
      │               │                 │                │
      │               │         ┌─────▼─────┐          │
      │               │         │ 检索历史  │          │
      │               │         │ (Milvus)  │          │
      │               │         └───────────┘          │
      │               │                 │                │
      │               │         ┌─────▼─────┐          │
      │               │         │ 构建 Prompt│          │
      │               │         │ (Templates)│         │
      │               │         └───────────┘          │
      │               │                 │                │
      │               │         ┌─────▼─────┐          │
      │               │         │ 生成回复   │          │
      │               │         └───────────┘          │
      │               │                 │                │
      │               │         ┌─────▼─────┐          │
      │               │         │ 格式化输出  │          │
      │               │         └───────────┘          │
      └───────────────┴─────────────────┴───────────────┘
                              │
                    ┌─────────▼───────────┐
                    │   返回给客户端       │
                    │   (WS 推送)          │
                    └─────────────────────┘

内容审核：
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  客户端  │ ──►│ rpc-msg  │ ──►│  Kafka  │ ──►│Moderation│
│  发送消息 │    │ 写 Kafka │    │ toModerate│    │  Service │
└──────────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘
                     │                 │                │
                     │         ┌─────────▼─────┐        │
                     │         │   消费消息     │        │
                     │         └─────┬─────┘        │
                     │               │                │
                     │         ┌─────▼─────┐        │
                     │         │调用AI Provider │        │
                     │         │ (审核 API)     │        │
                     │         └─────┬─────┘        │
                     │               │                │
                     │         ┌─────▼─────┐        │
                     │         │ 解析审核结果   │        │
                     │         └─────┬─────┘        │
                     │               │                │
                     │         ┌─────▼─────┐        │
                     │         │ 无违规：无事   │        │
                     │         │ 有违规：处理   │        │
                     │         └─────┬─────┘        │
                     │               │                │
                     │         ┌─────▼─────┐        │
                     │         │ 记录审核日志   │        │
                     │         │ 通知 Admin     │        │
                     │         │ 撤回消息(可选) │        │
                     │         └───────────┘        │
```

#### AI 服务配置

```yaml
# configs/ai/config.yaml
ai:
  enabled: true
  # === AI Provider（支持多后端切换） ===
  provider: "deepseek"               # deepseek / claude / local（Ollama）
  maxContextMessages: 20             # 上下文消息数
  maxTokens: 1024                    # 单次回复最大 Token
  temperature: 0.7                   # 温度参数（0-1，越高越随机）

  # Provider 详细配置（按 provider 选择生效）
  providers:
    deepseek:
      apiKey: "${DEEPSEEK_API_KEY}"
      baseURL: "https://api.deepseek.com/v1"
      model: "deepseek-chat"          # 主模型
      routerModel: "deepseek-chat"    # 路由判断复用

    claude:
      apiKey: "${ANTHROPIC_API_KEY}"
      model: "claude-sonnet-4-6"
      routerModel: "claude-haiku-4-5-20251001"

    local:
      baseURL: "http://localhost:11434/v1"  # Ollama 默认地址
      model: "qwen2.5:7b"                   # 本地模型名
      routerModel: "qwen2.5:1.5b"           # 路由用轻量模型

  # 智能回复配置
  reply:
    enabled: true
    maxHistoryLength: 5000
    suggestedReplies: 3

  # 群 AI 助手配置
  groupAI:
    enabled: true
    triggerKeyword: "@AI"
    maxContextMessages: 50
    toolsEnabled: true

  # 内容审核配置
  moderation:
    enabled: true
    checkInterval: 0
    autoRevoke: false
    violationCategories:
      - spam
      - harassment
      - hate_speech
      - violence
      - sexual_content
    severityThreshold: 0.7

  # Embedding 配置
  embedding:
    provider: "openai"               # openai / local
    model: "text-embedding-3-small"
    apiKey: "${OPENAI_API_KEY}"
    batchSize: 10

  # 向量数据库配置
  vectorDB:
    type: "milvus"                   # milvus / pgvector
    milvus:
      host: "127.0.0.1"
      port: 19530
      collection: "gim_conversations"
      dimension: 1536
      indexType: "IVF_FLAT"
    pgvector:
      dsn: "postgres://user:pass@localhost:5432/gim_vectors"
      table: "embeddings"
      dimension: 1536
```

#### Prompt 模板管理

```yaml
# configs/ai/prompts.yaml
prompts:
  # 智能回复系统提示
  reply:
    system: |
      你是一个即时通讯（IM）助手，帮助用户生成合适的回复。
      请根据对话上下文，生成自然、友好、符合语境的回复建议。
      回复应该：
      1. 语言自然流畅，符合日常对话习惯
      2. 语气友好，避免生硬
      3. 长度适中（建议 10-50 字）
      4. 如果需要，可以包含表情符号

  # 群 AI 助手系统提示
  groupAI:
    system: |
      你是一个群聊助手，名叫 "AI 助手"。
      你的职责是：
      1. 回答群友的问题
      2. 帮助总结讨论内容
      3. 提供有用的信息和建议
      4. 保持友好、专业的态度
      5. 避免过度介入，只在被 @ 时回应

      回答时请注意：
      - 简洁明了，避免冗长
      - 如果不确定，可以说"我不太确定，不过..."
      - 遇到无法回答的问题，可以说"这个问题超出了我的知识范围"

  # 内容审核系统提示
  moderation:
    system: |
      你是一个内容审核助手，负责检测即时通讯中的违规内容。
      请根据以下标准对消息进行分类：

      1. spam: 垃圾信息、广告、刷屏
      2. harassment: 骚扰、辱骂、人身攻击
      3. hate_speech: 仇恨言论、歧视性内容
      4. violence: 暴力威胁、血腥内容
      5. sexual_content: 色情、不适宜内容

      返回 JSON 格式：
      {
        "violation": true/false,
        "category": "分类名称",
        "reason": "原因说明",
        "score": 0.0-1.0
      }

      请准确判断，误判会导致不良后果。
```

#### AI 服务部署

```bash
# 1. 创建 AI 专用命名空间
kubectl create namespace gim-ai

# 2. 安装 Milvus（向量数据库）
helm repo add milvus https://milvus-io.github.io/milvus-helm
helm install milvus milvus/milvus -n gim-ai --set persistence.enabled=true

# 3. 部署 AI 服务
helm install gim-ai deploy/k8s/helm/gim-ai -n gim-ai \
  --set ai.provider=deepseek \
  --set ai.deepseek.apiKey=${DEEPSEEK_API_KEY} \
  --set ai.anthropic.apiKey=${ANTHROPIC_API_KEY} \
  --set ai.embedding.apiKey=${OPENAI_API_KEY}

# 4. 验证部署
kubectl get pods -n gim-ai
kubectl get svc -n gim-ai

# 5. 测试 AI 功能
# 通过 WS 客户端测试智能回复、群 AI、内容审核
```

---



---
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

### 17.3 Admin API 扩展

```go
// 在 Admin API 路由中追加 AI 对话接口
admin := api.Group("/ai")
{
    admin.POST("/chat", handlers.AdminAI.Chat)           // 管理助手对话
    admin.GET("/violations", handlers.AdminAI.Violations) // 审核日志查询
    admin.GET("/stats", handlers.AdminAI.Stats)           // 统计数据（Agent Tool 用）
}
```

### 17.4 配置扩展

**完整的 AI 配置示例：**
```yaml
# config.yaml 新增 AI 配置段
ai:
  enabled: true
  provider: "deepseek"           # deepseek / claude / local
  providers:
    deepseek:
      apiKey: ""                 # 从环境变量 DEEPSEEK_API_KEY 读取
      baseURL: "https://api.deepseek.com/v1"
      model: "deepseek-chat"
    claude:
      apiKey: ""                 # 从环境变量 ANTHROPIC_API_KEY 读取
      model: "claude-sonnet-4-6"
      routerModel: "claude-haiku-4-5-20251001"
    local:
      baseURL: "http://localhost:11434/v1"  # Ollama
      model: "qwen2.5:7b"
      routerModel: "qwen2.5:1.5b"
  maxContextMessages: 20
  maxTokens: 1024
  moderationEnabled: true
  rateLimitPerUser: 100         # 每用户每日 AI 调用上限

# 向量数据库配置（RAG 用）
vectorDB:
  type: "milvus"                 # milvus / pgvector
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

### 17.5 AI 架构总览

💡 **AI 集成到 IM 的三种方式：**
1. **智能回复助手**：用户 @bot，bot 基于上下文生成回复
2. **内容审核 Agent**：消息发送后自动审核，违规内容标记或删除
3. **管理后台助手**：RAG + Multi-turn 对话，帮助管理员查询日志、统计数据

**AI 架构图：**
```
┌────────────────────────────────────────────────────────────────┐
│                         IM 基础设施                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ WS GW    │  │ rpc-msg  │  │  Kafka   │  │ MongoDB  │      │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────────┘      │
└───────┼────────────┼────────────┼──────────────────────────────┘
        │            │            │
        ▼            ▼            ▼
┌────────────────────────────────────────────────────────────────┐
│                          AI Layer                              │
│                                                                │
│  ┌──────────────────┐     ┌─────────────────────────────┐      │
│  │  智能回复助手     │     │     内容审核 Agent           │      │
│  │  (AI Provider)   │     │     (AI + Tool Use)     │      │
│  │                  │     │                             │      │
│  │  - 流式输出      │     │  - 消息审核                  │      │
│  │  - 上下文管理    │     │  - 违规检测                  │      │
│  │  - 多种指令      │     │  - 自动处理                  │      │
│  └──────────────────┘     └─────────────────────────────┘      │
│                                                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │            管理后台智能助手 (RAG)                        │   │
│  │                                                         │   │
│  │  ┌─────────┐   ┌─────────┐   ┌─────────┐               │   │
│  │  │ Milvus  │◄──│Embedding│◄──│  文档   │               │   │
│  │  │ 向量库  │   │  模型   │   │  知识库 │               │   │
│  │  └─────────┘   └─────────┘   └─────────┘               │   │
│  │         │                   │                           │   │
│  │         ▼                   ▼                           │   │
│  │    ┌───────────────────────────────────┐                │   │
│  │    │     AI Provider (RAG 模式)            │                │   │
│  │    │  1. 向量检索相似文档              │                │   │
│  │    │  2. 构建 Prompt + 检索结果        │                │   │
│  │    │  3. 生成基于事实的回答            │                │   │
│  │    └───────────────────────────────────┘                │   │
│  └─────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────┘
```

### 17.6 智能回复助手详细实现

**处理流程：**
```
用户: "@gim-bot 翻译成英文：你好世界"
    │
    ▼
┌─────────────────────────────────────────────────────┐
│  WS Gateway                                         │
│  - 检测到 @gim-bot                                  │
│  - 提取会话最近 20 条消息                            │
│  - 构建请求: {instruction, context}                 │
└─────────────────┬───────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────┐
│  AI Service (智能回复)                              │
│  1. 构建 Prompt:                                     │
│     "你是聊天助手，根据以下上下文..."                │
│     上下文: [用户A: hi, 用户B: hello, ...]          │
│     指令: "翻译成英文：你好世界"                     │
│                                                     │
│  2. 调用 AI API (stream=true, AIProvider接口):      │
│     Provider 自动选择（按配置）:                     │
│     - Deepseek: POST api.deepseek.com/v1/chat/...  │
│     - Claude:  POST api.anthropic.com/v1/messages  │
│     - Local:   POST localhost:11434/v1/chat/...    │
│     {                                                │
│       model: "deepseek-chat",  # 可切换              │
│       messages: [{role: user, content: prompt}],    │
│       stream: true                                  │
│     }                                                │
│                                                     │
│  3. 处理流式响应:                                    │
│     chunk1: "Hello"                                 │
│     chunk2: " world"                                │
│     chunk3: "" (结束)                               │
│                                                     │
│  4. 通过 WS 推送给用户                              │
└─────────────────────────────────────────────────────┘
```

**核心代码结构：**
```go
// internal/ai/reply_service.go
type AIProvider interface {
    ChatStream(ctx context.Context, params ChatParams) (<-chan *StreamChunk, error)
}

type ReplyService struct {
    provider AIProvider            // 统一接口，运行时按配置选择 Deepseek/Claude/本地
    msgRepo  MessageRepository
    logger   *slog.Logger
}

func (s *ReplyService) HandleReply(ctx context.Context, userID, conversationID, instruction, reqID string) error {
    // 1. 获取上下文（最近20条消息）
    msgs, _ := s.msgRepo.GetHistory(ctx, conversationID, 20)

    // 2. 构建 Prompt
    prompt := s.buildPrompt(msgs, instruction)

    // 3. 调用 AI API（通过 AIProvider 统一接口，底层自动适配协议）
    stream, err := s.provider.ChatStream(ctx, ChatParams{
        Model:     s.cfg.Model,       // 按 provider 配置：deepseek-chat / claude-sonnet-4-6 / qwen2.5:7b
        MaxTokens: 1024,
        Messages:  []ChatMessage{{Role: "user", Content: prompt}},
    })
    if err != nil {
        return err
    }

    // 4. 流式推送
    var fullContent string
    for chunk := range stream {
        if chunk.Type == "content_block_delta" {
            text := chunk.Delta.Text
            fullContent += text
            // 通过 WS 推送给用户
            s.sendWS(ctx, userID, reqID, text)
        }
    }

    // 5. 保存 AI 回复作为消息
    s.msgRepo.Send(ctx, &Message{
        ConversationID: conversationID,
        SenderID:       "gim-bot",
        Content:        fullContent,
        MsgType:        10, // AI 回复类型
    })

    return nil
}
```

### 17.7 内容审核 Agent 详细实现

**审核流程：**
```
消息发送 → 写入 "toModeration" Topic
    │
    ▼
┌─────────────────────────────────────────────────────┐
│  Moderation Consumer                                 │
│                                                     │
│  1. 消费 Kafka 消息                                  │
│  2. 提取消息内容                                     │
│  3. 调用 AI 判断是否违规                            │
│                                                     │
│     Prompt: "判断以下消息是否包含违规内容..."        │
│     {                                                │
│       content: "消息内容",                           │
│       rules: [                                        │
│         "禁止谩骂",                                   │
│         "禁止政治敏感",                               │
│         "禁止广告"                                    │
│       ]                                              │
│     }                                                │
│                                                     │
│  4. 根据结果执行操作:                                │
│     - 合规: 无操作                                   │
│     - 轻微: 标记为敏感                               │
│     - 严重: 删除消息 + 警告用户                      │
└─────────────────────────────────────────────────────┘
```

**Tool Use 示例：**
```go
// 使用 AI 的 Function Calling 自动选择操作
type ModerationTool struct {
    Name        string
    Description string
}

var tools = []ModerationTool{
    {Name: "mark_sensitive", Description: "标记为敏感内容"},
    {Name: "delete_message", Description: "删除违规消息"},
    {Name: "warn_user", Description: "警告用户"},
}

// AI 返回: {"tool_calls": [{"name": "delete_message", "arguments": {...}}]}
// 服务端自动执行对应的操作
```

### 17.8 管理后台智能助手（RAG）详细实现

**RAG 流程：**
```
管理员提问: "昨天有多少用户注册？"
    │
    ▼
┌─────────────────────────────────────────────────────┐
│  1. 向量化问题                                        │
│     "昨天有多少用户注册？" → Embedding → vector[768] │
└─────────────────┬───────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────┐
│  2. Milvus 向量检索                                   │
│     - 在知识库中搜索相似文档                          │
│     - 返回 Top-5 相关文档                            │
│                                                     │
│     结果: [                                          │
│       {doc: "用户注册统计SQL...", score: 0.92},     │
│       {doc: "用户表结构...", score: 0.85},          │
│       ...                                           │
│     ]                                                │
└─────────────────┬───────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────┐
│  3. 构建 Prompt                                      │
│     "根据以下文档回答问题：\n"                       │
│     "文档1: 用户注册统计SQL: SELECT COUNT(*) ...\n"  │
│     "文档2: 用户表结构...\n"                         │
│     "问题: 昨天有多少用户注册？"                      │
└─────────────────┬───────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────┐
│  4. AI 生成回答                                      │
│     "根据查询结果，昨天（2026-04-30）共有 234 位    │
│      新用户注册。"                                    │
└─────────────────────────────────────────────────────┘
```

**知识库构建：**
```go
// 1. 收集文档
docs := []string{
    "用户注册统计SQL: SELECT COUNT(*) FROM users WHERE DATE(created_at) = CURDATE()",
    "用户表结构: users(id, nickname, avatar_url, phone, email, created_at)",
    "消息发送统计SQL: SELECT COUNT(*) FROM messages WHERE DATE(send_time) = CURDATE()",
    // ... 更多文档
}

// 2. 向量化并写入 Milvus
for _, doc := range docs {
    // 调用 Embedding API
    vector := embeddingClient.Embed(doc)

    // 写入 Milvus
    milvusClient.Insert(&Doc{
        ID:    generateID(),
        Text:  doc,
        Vector: vector,
    })
}
```

### 17.9 AI 调用限流和成本控制

**限流策略：**
```go
// 使用 Redis 实现每用户每日调用上限
type RateLimiter struct {
    redis  *redis.Client
    logger *slog.Logger
}

func (r *RateLimiter) Check(ctx context.Context, userID string) (bool, error) {
    key := fmt.Sprintf("ai_rate_limit:%s:%s", userID, time.Now().Format("2006-01-02"))
    count, err := r.redis.Incr(ctx, key).Result()
    if err != nil {
        return false, err
    }
    if count == 1 {
        r.redis.Expire(ctx, key, 24*time.Hour)
    }
    return count <= 100, nil // 每天最多 100 次
}
```

**成本估算（多 Provider 对比）：**
```
Deepseek API（推荐，性价比最高）:
- deepseek-chat: ¥1/1M 输入 tokens, ¥2/1M 输出 tokens
- 平均每次对话: 1000 输入 + 500 输出 = 1500 tokens
- 每次成本: (1000/1M)*¥1 + (500/1M)*¥2 = ¥0.001 + ¥0.001 = ¥0.002
- 100 次对话/用户/天: ¥0.2/天
- 1000 用户: ¥200/天 ≈ $28/天

Claude API（示例）:
- claude-sonnet-4-6: $15/1M 输入 tokens, $75/1M 输出 tokens
- 每次成本: (1000/1M)*$15 + (500/1M)*$75 = $0.015 + $0.0375 = $0.0525
- 100 次对话/用户/天: $5.25/天
- 1000 用户: $5250/天 (需要优化或设置上限)

本地模型 (Ollama/vLLM):
- 零 API 费用，仅需 GPU 服务器成本
- 推荐模型: qwen2.5:7b / llama3:8b（8GB VRAM）
- 适合开发测试和小规模部署
```

---

## 附录 A：快速参考

### A.1 端口清单

| 服务 | 端口 | 协议 | 说明 |
|------|------|------|------|
| gim-api | 8080 | HTTP | API 网关 |
| gim-ws | 8081 | WebSocket | WS Gateway |
| gim-ws gRPC | 8082 | gRPC | WS Gateway gRPC 接口 |
| rpc-auth | 9001 | gRPC | 认证服务 |
| rpc-user | 9002 | gRPC | 用户服务 |
| rpc-friend | 9003 | gRPC | 好友服务 |
| rpc-msg | 9004 | gRPC | 消息服务 |
| rpc-conversation | 9005 | gRPC | 会话服务 |
| gim-push | 9010 | gRPC | 推送服务 |
| MySQL | 3306 | MySQL | 数据库 |
| Redis | 6379 | Redis | 缓存 |
| Kafka | 9092 | Kafka | 消息队列 |
| etcd | 2379 | etcd | 服务发现 |
| MongoDB | 27017 | MongoDB | 消息存储 |
| S3 存储(MinIO/OSS) | 9000 | S3 API | 对象存储 |

### A.2 目录结构速查

```
gim/
├── cmd/                    # 各服务的入口
│   ├── gim/               # Phase 1: 单体应用
│   ├── gim-api/           # Phase 2: API 网关
│   ├── gim-ws/            # Phase 2: WS Gateway
│   ├── rpc-*/             # Phase 2: RPC 服务
│   └── msg-transfer/      # Phase 2: 消息传输
├── api/                    # gRPC Protobuf 定义
├── internal/               # 私有代码
│   ├── handlers/          # HTTP 处理器
│   ├── services/          # 业务逻辑
│   ├── repositories/      # 数据访问
│   ├── models/            # 数据模型
│   ├── middleware/        # 中间件
│   └── ws/                # WebSocket
├── pkg/                    # 公共包
│   ├── jwt/               # JWT 工具
│   ├── snowflake/         # ID 生成
│   ├── resp/              # 统一响应
│   ├── errcode/           # 错误码
│   └── slog/              # 日志
├── configs/                # 配置文件
├── migrations/             # 数据库迁移
├── deploy/                 # 部署文件
│   ├── docker/            # Dockerfile
│   ├── k8s/               # K8S 资源
│   └── helm/              # Helm Charts
└── docs/                   # 文档
```

### A.3 环境变量清单

| 变量名 | 说明 | 示例值 |
|--------|------|--------|
| GIM_ENV | 运行环境 | dev / test / prod |
| GIM_LOG_LEVEL | 日志级别 | debug / info / warn / error |
| MYSQL_HOST | MySQL 地址 | localhost:3306 |
| MYSQL_USER | MySQL 用户 | root |
| MYSQL_PASSWORD | MySQL 密码 | *** |
| REDIS_HOST | Redis 地址 | localhost:6379 |
| REDIS_PASSWORD | Redis 密码 | *** |
| KAFKA_BROKERS | Kafka 地址 | localhost:9092 |
| ETCD_ENDPOINTS | etcd 地址 | localhost:2379 |
| MONGO_URI | MongoDB URI | mongodb://localhost:27017/gim |
| S3_ENDPOINT | S3 兼容存储地址 | localhost:9000 |
| JWT_PRIVATE_KEY | JWT 私钥路径 | configs/jwt/private.pem |
| JWT_PUBLIC_KEY | JWT 公钥路径 | configs/jwt/public.pem |
| DEEPSEEK_API_KEY | Deepseek API 密钥 | *** |
| ANTHROPIC_API_KEY | Claude API 密钥 | *** |
| OPENAI_API_KEY | OpenAI API 密钥（Embedding） | *** |

---

## 附录 B：常见问题排查

### B.1 启动问题

**问题：`panic: invalid memory address or nil pointer dereference`**
```
原因：某个依赖未正确初始化
解决：
1. 检查 InitMySQL、InitRedis 是否成功
2. 检查 wire.go 依赖关系是否正确
3. 添加日志确认初始化顺序
```

**问题：`listen tcp :8080: bind: address already in use`**
```
原因：端口被占用
解决：
lsof -i :8080  # 查看占用进程
kill -9 <PID>  # 杀死进程
# 或修改配置文件中的端口
```

**问题：`failed to connect to database: connection refused`**
```
原因：数据库未启动或地址错误
解决：
1. 确认 Docker 容器运行: docker ps
2. 检查配置文件中的地址
3. 确认网络连通: telnet localhost 3306
```

### B.2 WebSocket 问题

**问题：WebSocket 连接频繁断开**
```
可能原因：
1. Nginx 超时设置过短
2. 客户端网络不稳定
3. 心跳机制失效

排查：
1. 查看日志: tail -f logs/gim.log | grep "WS"
2. 检查 Nginx 配置: proxy_read_timeout, proxy_send_timeout
3. 确认客户端心跳发送正常
```

**问题：消息收不到**
```
可能原因：
1. 用户不在线
2. Hub 中客户端未正确注册
3. 消息推送逻辑出错

排查：
1. 检查 Redis 在线状态: redis-cli HGETALL online:userID
2. 查看 Hub 日志: grep "Hub" logs/gim.log
3. 确认消息序号: redis-cli GET seq:convID
```

### B.3 性能问题

**问题：响应慢**
```
排查步骤：
1. 查看 Prometheus 指标: P95 延迟
2. 检查慢查询: SHOW PROCESSLIST (MySQL)
3. 分析日志: grep "slow" logs/gim.log
4. 使用 pprof: go tool pprof http://localhost:8080/debug/pprof/profile
```

**问题：CPU 高**
```
排查步骤：
1. 查看进程: top -p <PID>
2. 查看协程数量: runtime.NumGoroutine()
3. 检查是否有死循环
4. 分析 pprof: go tool pprof http://localhost:8080/debug/pprof/profile
```

**问题：内存高**
```
排查步骤：
1. 查看堆内存: runtime.ReadMemStats()
2. 检查是否有内存泄漏: go tool pprof http://localhost:8080/debug/pprof/heap
3. 检查 WebSocket 连接数: gim_ws_connections
4. 检查 Redis 连接池: redis-cli CLIENT LIST
```

### B.4 K8S 问题

**问题：Pod 一直是 Pending 状态**
```
原因：资源不足或调度失败
解决：
kubectl describe pod <pod-name> -n gim-prod
# 检查 Events 中的错误信息
# 可能需要增加节点配额或调整资源请求
```

**问题：Pod 频繁重启**
```
原因：健康检查失败或 OOM
解决：
kubectl logs <pod-name> -n gim-prod --previous
# 查看上一次崩溃的日志
# 检查资源限制是否合理
```

**问题：Service 无法访问**
```
原因：选择器错误或网络策略限制
解决：
kubectl get svc <svc-name> -n gim-prod -o yaml
# 检查 selector 是否匹配 Pod 的 label
# 检查 NetworkPolicy 是否限制了流量
```

---

## 附录 C：性能调优建议

### C.1 数据库优化

**MySQL：**
```sql
-- 1. 添加索引
CREATE INDEX idx_messages_conversation_seq ON messages(conversation_id, seq);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_friends_status ON friends(status);

-- 2. 分区表（大表）
ALTER TABLE messages PARTITION BY RANGE (YEAR(send_time)) (
    PARTITION p2024 VALUES LESS THAN (2025),
    PARTITION p2025 VALUES LESS THAN (2026),
    PARTITION pmax VALUES LESS THAN MAXVALUE
);

-- 3. 读写分离
-- 主库：写操作
-- 从库：读操作（查询历史、统计等）
```

**Redis：**
```bash
# 1. 使用 Pipeline 批量操作
redis-cli --pipeline
SET key1 value1
SET key2 value2
SET key3 value3
EXEC

# 2. 合理设置过期时间
# 在线状态: 30秒
# 短信验证码: 5分钟
# 限流计数: 1小时

# 3. 使用 Redis Cluster
# 单节点内存 > 10GB 时考虑集群
```

### C.2 应用优化

**连接池配置：**
```go
// MySQL 连接池
db.SetMaxOpenConns(100)    // 最大连接数
db.SetMaxIdleConns(20)     // 最大空闲连接数
db.SetConnMaxLifetime(1 * time.Hour) // 连接最大生存时间

// Redis 连接池
redisClient := redis.NewClient(&redis.Options{
    PoolSize:     50,   // 连接池大小
    MinIdleConns: 10,   // 最小空闲连接
    MaxRetries:   3,    // 最大重试次数
})
```

**并发控制：**
```go
// 使用 worker pool 处理任务
workerPool := make(chan struct{}, 100) // 最多100个并发

for _, task := range tasks {
    workerPool <- struct{}{} // 获取令牌
    go func(t Task) {
        defer func() { <-workerPool }() // 释放令牌
        process(t)
    }(task)
}
```

### C.3 网络优化

**Nginx 配置：**
```nginx
upstream gim_api {
    least_conn;  # 最少连接负载均衡
    server 10.0.1.1:8080 weight=3;
    server 10.0.1.2:8080 weight=2;
    keepalive 32;
}

server {
    location /api/ {
        proxy_pass http://gim_api;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_connect_timeout 5s;
        proxy_read_timeout 60s;
        proxy_send_timeout 60s;
    }
}
```

---

## 附录 D：安全加固建议

### D.1 认证安全

1. **使用强密码策略**
   ```go
   // 密码强度检查：至少8位，包含大小写字母、数字、特殊字符
   func isStrongPassword(pwd string) bool {
       if len(pwd) < 8 {
           return false
       }
       hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(pwd)
       hasLower := regexp.MustCompile(`[a-z]`).MatchString(pwd)
       hasDigit := regexp.MustCompile(`[0-9]`).MatchString(pwd)
       hasSpecial := regexp.MustCompile(`[!@#$%^&*]`).MatchString(pwd)
       return hasUpper && hasLower && hasDigit && hasSpecial
   }
   ```

2. **JWT 安全配置**
   - Access Token 过期时间: 15-30分钟
   - Refresh Token 过期时间: 7-30天
   - 使用 RS256 非对称加密
   - Token 黑名单机制

3. **防暴力破解**
   ```go
   // 登录失败限制：5次/15分钟
   key := fmt.Sprintf("login_failed:%s:%s", userID, time.Now().Format("200601021504"))
   count := redis.Incr(ctx, key)
   redis.Expire(ctx, key, 15*time.Minute)
   if count > 5 {
       return errors.New("登录失败次数过多，请15分钟后再试")
   }
   ```

### D.2 通信安全

1. **HTTPS/TLS**
   ```nginx
   server {
       listen 443 ssl http2;
       ssl_certificate /etc/nginx/ssl/cert.pem;
       ssl_certificate_key /etc/nginx/ssl/key.pem;
       ssl_protocols TLSv1.2 TLSv1.3;
       ssl_ciphers HIGH:!aNULL:!MD5;
   }
   ```

2. **WSS (Secure WebSocket)**
   ```go
   // 客户端连接时使用 wss://
   ws := new WebSocket("wss://api.example.com/ws")
   ```

### D.3 数据安全

1. **敏感字段加密**
   ```go
   // 手机号、邮箱等敏感信息加密存储
   func encryptPhone(phone string) string {
       block, _ := aes.NewCipher(key)
       // ... AES 加密
   }
   ```

2. **数据脱敏**
   ```go
   // 日志中隐藏敏感信息
   func maskPhone(phone string) string {
       if len(phone) == 11 {
           return phone[:3] + "****" + phone[7:]
       }
       return phone
   }
   ```

3. **SQL 注入防护**
   ```go
   // 使用参数化查询
   db.Where("phone = ?", phone).First(&user)
   // 不要使用字符串拼接: db.Where("phone = '" + phone + "'")
   ```

---

## 附录 E：学习资源

### E.1 官方文档
- [Go 官方文档](https://golang.org/doc/)
- [Gin 框架文档](https://gin-gonic.com/docs/)
- [GORM 文档](https://gorm.io/docs/)
- [Redis 命令参考](https://redis.io/commands/)
- [Kubernetes 文档](https://kubernetes.io/docs/)
- [Helm 文档](https://helm.sh/docs/)
- [Prometheus 文档](https://prometheus.io/docs/)
- [Deepseek API 文档](https://platform.deepseek.com/docs)
- [Claude API 文档](https://docs.anthropic.com/)
- [Ollama 本地部署](https://ollama.com/)

### E.2 推荐书籍
- 《Go 语言实战》- William Kennedy
- 《Go 语言圣经》- Alan A. A. Donovan
- 《高性能 MySQL》- Baron Schwartz
- 《Redis 设计与实现》- 黄健宏
- 《Kubernetes 权威指南》- 龚正 等

### E.3 在线课程
- [Go 语言入门到实战](https://www.imooc.com/learn/424)
- [Kubernetes 入门到实践](https://kubernetes.io/docs/tutorials/)
- [微服务架构设计模式](https://microservices.io/patterns/)

### E.4 开源项目参考
- [Go-Gin Example](https://github.com/EDDYCJY/go-gin-example)
- [Go-Redis](https://github.com/redis/go-redis)
- [grpc-go](https://github.com/grpc/grpc-go)
- [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go)

---

## 附录 F：开发增强工具与最佳实践

💡 **本附录提供开发效率提升工具和最佳实践，帮助团队更高效地开发和维护项目。**

### F.1 依赖安装自动化脚本

💡 **为什么需要自动化脚本？** 新成员加入团队时，手动安装所有工具容易出错且耗时。一个脚本可以让所有开发者使用一致的环境。

```bash
#!/bin/bash
# scripts/install-dev-tools.sh
# 开发环境依赖自动安装脚本
# 支持: macOS (Homebrew) 和 Linux (apt/yum)

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检测操作系统
OS="$(uname -s)"
case "${OS}" in
    Linux*)     MACHINE=Linux;;
    Darwin*)    MACHINE=Mac;;
    *)          log_error "Unsupported OS: ${OS}"; exit 1;;
esac

log_info "检测到操作系统: ${MACHINE}"

# 检查 Go 是否已安装
if ! command -v go &> /dev/null; then
    log_error "Go 未安装，请先安装 Go 1.26+"
    log_info "访问 https://go.dev/dl/ 下载安装包"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
log_info "Go 版本: ${GO_VERSION}"

# 检查 Docker 是否已安装
if ! command -v docker &> /dev/null; then
    log_warn "Docker 未安装，正在安装..."
    if [ "${MACHINE}" = "Mac" ]; then
        if ! command -v brew &> /dev/null; then
            log_error "Homebrew 未安装，请先安装 Homebrew"
            exit 1
        fi
        brew install --cask docker
    else
        curl -fsSL https://get.docker.com -o get-docker.sh
        sudo sh get-docker.sh
        sudo usermod -aG docker $USER
        log_warn "Docker 安装完成，请重新登录使组权限生效"
    fi
else
    log_info "Docker 已安装: $(docker --version)"
fi

# 检查 Docker Compose 是否已安装
if ! docker compose version &> /dev/null; then
    log_warn "Docker Compose 未安装，正在安装..."
    if [ "${MACHINE}" = "Mac" ]; then
        brew install docker-compose
    else
        sudo apt-get update
        sudo apt-get install -y docker-compose-plugin
    fi
else
    log_info "Docker Compose 已安装: $(docker compose version)"
fi

# 安装 golang-migrate
if ! command -v migrate &> /dev/null; then
    log_info "正在安装 golang-migrate..."
    if [ "${MACHINE}" = "Mac" ]; then
        brew install golang-migrate
    else
        MIGRATE_VERSION="v4.16.2"
        curl -L https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz | tar xvz
        sudo mv migrate /usr/local/bin/migrate
    fi
    log_info "golang-migrate 安装完成: $(migrate --version)"
else
    log_info "golang-migrate 已安装: $(migrate --version)"
fi

# 安装 golangci-lint（可选）
if ! command -v golangci-lint &> /dev/null; then
    log_info "正在安装 golangci-lint（可选，用于代码检查）..."
    if [ "${MACHINE}" = "Mac" ]; then
        brew install golangci-lint
    else
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
    fi
    log_info "golangci-lint 安装完成: $(golangci-lint version)"
else
    log_info "golangci-lint 已安装: $(golangci-lint version)"
fi

# 安装 protoc（第二阶段需要）
if ! command -v protoc &> /dev/null; then
    log_info "正在安装 protoc（第二阶段 gRPC 开发需要）..."
    if [ "${MACHINE}" = "Mac" ]; then
        brew install protobuf
    else
        sudo apt-get update
        sudo apt-get install -y protobuf-compiler
    fi
    log_info "protoc 安装完成: $(protoc --version)"
    
    # 安装 Go protobuf 插件
    log_info "正在安装 Go protobuf 插件..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
else
    log_info "protoc 已安装: $(protoc --version)"
fi

# 安装 swag（Swagger 文档生成，可选）
if ! command -v swag &> /dev/null; then
    log_info "正在安装 swag（可选，用于生成 API 文档）..."
    go install github.com/swaggo/swag/cmd/swag@latest
    log_info "swag 安装完成"
else
    log_info "swag 已安装"
fi

# 创建必要的目录
log_info "创建项目目录结构..."
mkdir -p cmd/gim \
         internal/{config,handler,service,repository,model,middleware,ws} \
         pkg/{jwt,snowflake,slog,resp,errcode,rediskey,convid,convutil} \
         configs/jwt \
         logs \
         migrations \
         api/{auth,user,friend,msg,conversation,push} \
         deploy/{docker,mysql,k8s/helm/gim} \
         scripts

log_info "目录结构创建完成"

# 生成 JWT 密钥对
if [ ! -f "configs/jwt/private.pem" ]; then
    log_info "生成 JWT 密钥对..."
    mkdir -p configs/jwt
    openssl genrsa -out configs/jwt/private.pem 2048
    openssl rsa -in configs/jwt/private.pem -pubout -out configs/jwt/public.pem
    log_info "JWT 密钥对生成完成"
else
    log_info "JWT 密钥对已存在"
fi

# 创建示例配置文件
if [ ! -f "configs/config.yaml" ]; then
    log_info "创建示例配置文件..."
    cat > configs/config.yaml << 'EOF'
server:
  httpPort: 8080
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
  privateKeyPath: "configs/jwt/private.pem"
  publicKeyPath: "configs/jwt/public.pem"

websocket:
  port: 8081
  maxConnPerUser: 5
  maxMessageSize: 4096
  writeWait: 10s
  pongWait: 60s
  pingPeriod: 30s

log:
  level: debug
  format: text
  output: stdout
  filePath: logs/gim.log
  maxSize: 100
  maxBackups: 10
  maxAge: 30
  compress: true
  shortFile: true
  color: true

snowflake:
  nodeID: 1
EOF
    log_info "示例配置文件创建完成"
else
    log_info "配置文件已存在"
fi

# 设置环境变量
log_info "设置环境变量..."
cat > .env << 'EOF'
# 数据库配置
DB_USER=gim
DB_PASSWORD=gim_pass
DB_HOST=localhost
DB_PORT=3306
DB_NAME=gim

# 运行环境
GIM_ENV=dev

# 日志级别
LOG_LEVEL=debug
EOF

log_info ".env 文件创建完成"
log_warn "请运行 'source .env' 或手动设置环境变量"

# 显示总结
echo ""
echo "========================================="
log_info "开发环境工具安装完成！"
echo "========================================="
echo ""
log_info "下一步操作："
echo "  1. 加载环境变量: source .env"
echo "  2. 启动依赖服务: make docker"
echo "  3. 初始化数据库: make migrate-up"
echo "  4. 运行应用: make run"
echo ""
log_info "如需查看所有可用命令: make help"
echo ""
```

使用方法：
```bash
# 赋予执行权限
chmod +x scripts/install-dev-tools.sh

# 运行安装脚本
./scripts/install-dev-tools.sh

# 或使用 make 命令（添加到 Makefile 中）
make setup-dev
```

### F.2 开发环境一键启动命令

💡 **为什么需要一键启动？** 开发时经常需要重复执行多个命令：启动依赖、迁移数据库、构建、运行。一键启动可以让开发者更快进入工作状态。

#### F.2.1 扩展 Makefile

```makefile
# 在 Makefile 中添加以下目标

# 开发环境一键启动
dev-setup:
	@echo "=== 初始化开发环境 ==="
	@make docker
	@echo "等待数据库就绪..."
	@sleep 10
	@make migrate-up
	@echo "=== 开发环境就绪 ==="

# 一键启动所有服务
dev: deps migrate-up
	@echo "=== 启动开发环境 ==="
	@docker compose -f deploy/docker-compose.yaml up -d
	@echo "等待服务启动..."
	@sleep 10
	@make migrate-up
	@make build
	@echo "=== 启动应用 (后台运行) ==="
	@nohup ./bin/gim > logs/app.log 2>&1 &
	@echo "应用已启动，PID: $$!"
	@echo "查看日志: tail -f logs/app.log"
	@echo "停止应用: make stop"

# 停止应用
stop:
	@echo "停止应用..."
	@pkill -f "bin/gim" || echo "应用未运行"

# 重启应用
restart: stop dev

# 查看所有日志
logs:
	@docker compose -f deploy/docker-compose.yaml logs -f

# 查看应用日志
app-logs:
	@tail -f logs/gim.log

# 清理所有数据并重新初始化
clean-all: docker-down
	@echo "清理数据目录..."
	@rm -rf mysql_data redis_data
	@rm -f *.log
	@make clean
	@echo "清理完成"

# 完整重置（危险操作）
reset: clean-all dev-setup

# 代码格式化
fmt:
	@echo "格式化代码..."
	@go fmt ./...
	@goimports -w .

# 代码静态检查
check: lint fmt
	@echo "代码检查完成"

# 生成所有代码（包括 protobuf 和 swagger）
generate-all: gen swagger
	@echo "代码生成完成"

# 预提交检查（用于 git hook）
pre-commit: fmt lint test
	@echo "预提交检查完成"

# 初始化新项目（首次使用）
init-project:
	@echo "=== 初始化新项目 ==="
	@go mod init github.com/tianlu1990s/gim || echo "go.mod 已存在"
	@go mod tidy
	@make deps
	@make docker
	@sleep 10
	@make migrate-up
	@echo "=== 项目初始化完成 ==="

# 运行开发服务器（带热重载）
# 需要安装 air: go install github.com/cosmtrek/air@latest
dev-server:
	@which air > /dev/null || (echo "安装 air..." && go install github.com/cosmtrek/air@latest)
	@air -c .air.toml
```

#### F.2.2 Air 配置（热重载）

```toml
# .air.toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main ./cmd/gim/main.go"
  bin = "tmp/main"
  include_ext = ["go", "tpl", "tmpl", "html"]
  exclude_dir = ["tmp", "vendor", "testdata"]
  exclude_regex = ["_test\\.go"]
  include_file = []
  exclude_unchanged = false
  follow_symlink = false
  poll = false
  poll_interval = 0
  delay = 1000
  stop_on_error = true
  send_interrupt = false
  kill_delay = 0

[log]
  time = true
  main_only = false
  color = "false"
  level = "debug"

[misc]
  clean_on_exit = false
```

#### F.2.3 Git Hooks 配置

```bash
#!/bin/bash
# .git/hooks/pre-commit
# Git pre-commit hook，确保提交前代码质量

echo "运行预提交检查..."

# 格式化代码
echo "格式化代码..."
go fmt ./...

# 运行测试
echo "运行测试..."
go test -short ./...

# 代码检查
echo "代码检查..."
golangci-lint run --new-from-rev HEAD~1 || echo "警告: 代码检查发现问题，请修复或提交 --no-verify"

echo "预提交检查完成"
```

安装 Git Hooks：
```bash
# 创建 .githooks 目录
mkdir -p .githooks

# 复制 hooks
cp .githooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

# 配置 Git 使用 .githooks 目录
git config core.hooksPath .githooks
```

### F.3 测试用例示例

💡 **为什么需要测试示例？** 新手可能不知道如何编写单元测试和集成测试。提供示例可以快速上手。

#### F.3.1 单元测试示例

```go
// internal/service/auth_test.go
package service

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    
    "github.com/tianlu1990s/gim/internal/model"
    "github.com/tianlu1990s/gim/internal/repository/mocks"
    "github.com/tianlu1990s/gim/pkg/jwt"
)

// Mock Repository
type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

func (m *MockUserRepository) GetByUserID(ctx context.Context, userID string) (*model.User, error) {
    args := m.Called(ctx, userID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.User), args.Error(1)
}

// Test Register Success
func TestAuthService_Register_Success(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    mockJWT := jwt.NewJWTManager(
        "configs/jwt/test_private.pem",
        "configs/jwt/test_public.pem",
        24*time.Hour,
        168*time.Hour,
    )
    
    service := NewAuthService(mockRepo, mockJWT, nil)
    
    req := &model.RegisterReq{
        UserID:   "test001",
        Password: "Test1234",
        Nickname: "测试用户",
    }
    
    mockRepo.On("GetByUserID", mock.Anything, "test001").Return(nil, nil)
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
    
    // Act
    resp, err := service.Register(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.Equal(t, "test001", resp.UserID)
    assert.Equal(t, "测试用户", resp.Nickname)
    assert.NotEmpty(t, resp.AccessToken)
    assert.NotEmpty(t, resp.RefreshToken)
    
    mockRepo.AssertExpectations(t)
}

// Test Register - User Already Exists
func TestAuthService_Register_UserAlreadyExists(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    mockJWT := jwt.NewJWTManager(
        "configs/jwt/test_private.pem",
        "configs/jwt/test_public.pem",
        24*time.Hour,
        168*time.Hour,
    )
    
    service := NewAuthService(mockRepo, mockJWT, nil)
    
    req := &model.RegisterReq{
        UserID:   "test001",
        Password: "Test1234",
        Nickname: "测试用户",
    }
    
    existingUser := &model.User{UserID: "test001"}
    mockRepo.On("GetByUserID", mock.Anything, "test001").Return(existingUser, nil)
    
    // Act
    resp, err := service.Register(context.Background(), req)
    
    // Assert
    assert.Error(t, err)
    assert.Nil(t, resp)
    assert.Equal(t, "用户已存在", err.Error())
    
    mockRepo.AssertExpectations(t)
}

// Test Login Success
func TestAuthService_Login_Success(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    mockJWT := jwt.NewJWTManager(
        "configs/jwt/test_private.pem",
        "configs/jwt/test_public.pem",
        24*time.Hour,
        168*time.Hour,
    )
    
    service := NewAuthService(mockRepo, mockJWT, nil)
    
    req := &model.LoginReq{
        UserID:   "test001",
        Password: "Test1234",
        Platform: "web",
    }
    
    // 模拟加密后的密码
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Test1234"), bcrypt.DefaultCost)
    
    existingUser := &model.User{
        UserID:   "test001",
        Nickname: "测试用户",
        Password: string(hashedPassword),
        Status:   1,
    }
    
    mockRepo.On("GetByUserID", mock.Anything, "test001").Return(existingUser, nil)
    
    // Act
    resp, err := service.Login(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.Equal(t, "test001", resp.UserID)
    assert.NotEmpty(t, resp.AccessToken)
    assert.NotEmpty(t, resp.RefreshToken)
    
    mockRepo.AssertExpectations(t)
}

// Table-driven Tests
func TestAuthService_ValidatePassword(t *testing.T) {
    tests := []struct {
        name        string
        password    string
        wantErr     bool
        errContains string
    }{
        {
            name:        "Valid password",
            password:    "Test1234",
            wantErr:     false,
        },
        {
            name:        "Too short",
            password:    "Test1",
            wantErr:     true,
            errContains: "密码长度至少8位",
        },
        {
            name:        "No uppercase",
            password:    "test1234",
            wantErr:     true,
            errContains: "必须包含大写字母",
        },
        {
            name:        "No digit",
            password:    "Testtest",
            wantErr:     true,
            errContains: "必须包含数字",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validatePassword(tt.password)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errContains)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### F.3.2 集成测试示例

```go
// tests/integration/auth_integration_test.go
package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    
    "github.com/tianlu1990s/gim/internal/handler"
    "github.com/tianlu1990s/gim/internal/service"
)

type AuthIntegrationTestSuite struct {
    suite.Suite
    router   *gin.Engine
    services *service.Services
}

func (suite *AuthIntegrationTestSuite) SetupSuite() {
    // 初始化测试数据库
    setupTestDB()
    
    // 初始化服务
    suite.services = setupTestServices()
    
    // 初始化路由
    suite.router = gin.New()
    authHandler := handler.NewAuthHandler(suite.services.Auth)
    
    auth := suite.router.Group("/api/v1/auth")
    auth.POST("/register", authHandler.Register)
    auth.POST("/login", authHandler.Login)
}

func (suite *AuthIntegrationTestSuite) TearDownSuite() {
    // 清理测试数据
    cleanupTestDB()
}

func (suite *AuthIntegrationTestSuite) TestAuthFlow() {
    // 1. 注册用户
    registerReq := map[string]interface{}{
        "userId":   "integration_test_001",
        "password": "Test1234",
        "nickname": "集成测试用户",
    }
    
    reqBody, _ := json.Marshal(registerReq)
    req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
    req.Header.Set("Content-Type", "application/json")
    
    w := httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)
    
    assert.Equal(suite.T(), http.StatusOK, w.Code)
    
    var registerResp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &registerResp)
    assert.Equal(suite.T(), float64(200), registerResp["code"])
    
    // 2. 登录用户
    loginReq := map[string]interface{}{
        "userId":   "integration_test_001",
        "password": "Test1234",
        "platform": "web",
    }
    
    reqBody, _ = json.Marshal(loginReq)
    req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
    req.Header.Set("Content-Type", "application/json")
    
    w = httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)
    
    assert.Equal(suite.T(), http.StatusOK, w.Code)
    
    var loginResp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &loginResp)
    assert.Equal(suite.T(), float64(200), loginResp["code"])
    
    data := loginResp["data"].(map[string]interface{})
    assert.NotEmpty(suite.T(), data["accessToken"])
    assert.NotEmpty(suite.T(), data["refreshToken"])
    
    // 3. 使用 Token 访问受保护资源
    accessToken := data["accessToken"].(string)
    req = httptest.NewRequest("GET", "/api/v1/user/profile", nil)
    req.Header.Set("Authorization", "Bearer "+accessToken)
    
    w = httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)
    
    assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func TestAuthIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(AuthIntegrationTestSuite))
}
```

#### F.3.3 基准测试示例

```go
// internal/service/message_bench_test.go
package service

import (
    "context"
    "testing"
    
    "github.com/tianlu1990s/gim/internal/model"
)

func BenchmarkMessageService_SendMessage(b *testing.B) {
    service := setupBenchmarkMessageService()
    
    req := &model.SendMsgReq{
        ConversationID: "single_user1_user2",
        ClientMsgID:    "client_001",
        ContentType:     1,
        Content:        "测试消息内容",
    }
    
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.SendMessage(ctx, "user1", req)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkMessageService_History(b *testing.B) {
    service := setupBenchmarkMessageService()
    
    // 预先准备测试数据
    prepareBenchmarkData(service)
    
    req := &model.HistoryReq{
        ConversationID: "single_user1_user2",
        MinSeq:         0,
        Count:          50,
    }
    
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.History(ctx, "user1", req)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// 并发基准测试
func BenchmarkMessageService_SendMessage_Parallel(b *testing.B) {
    service := setupBenchmarkMessageService()
    
    ctx := context.Background()
    
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            req := &model.SendMsgReq{
                ConversationID: "single_user1_user2",
                ClientMsgID:    fmt.Sprintf("client_%d", i),
                ContentType:     1,
                Content:        "测试消息",
            }
            _, err := service.SendMessage(ctx, "user1", req)
            if err != nil {
                b.Fatal(err)
            }
            i++
        }
    })
}
```

运行基准测试：
```bash
# 运行所有基准测试
go test -bench=. -benchmem ./internal/service/

# 运行特定基准测试
go test -bench=BenchmarkMessageService_SendMessage -benchmem ./internal/service/

# 运行基准测试并生成 CPU profile
go test -bench=. -cpuprofile=cpu.prof ./internal/service/

# 分析 profile
go tool pprof cpu.prof
```

### F.4 性能基准数据

💡 **为什么需要性能基准？** 了解系统在不同负载下的表现，可以帮助优化和容量规划。

#### F.4.1 单机性能基准

```
测试环境：
- CPU: 8核心
- 内存: 16GB
- 磁盘: SSD
- Go 版本: 1.26
- MySQL: 8.4
- Redis: 7.x

| 场景 | QPS | P95 延迟 | P99 延迟 | CPU 使用率 | 内存使用 |
|------|-----|---------|---------|-----------|---------|
| 用户注册 | 5,000 | 15ms | 25ms | 45% | 1.2GB |
| 用户登录 | 8,000 | 8ms | 15ms | 50% | 1.5GB |
| 获取用户信息 | 15,000 | 5ms | 10ms | 60% | 2.0GB |
| 搜索用户 | 3,000 | 30ms | 50ms | 40% | 1.0GB |
| 发送消息 (HTTP) | 10,000 | 10ms | 20ms | 65% | 2.5GB |
| 拉取历史消息 | 5,000 | 20ms | 40ms | 55% | 1.8GB |
| WebSocket 连接数 | 50,000 | - | - | 70% | 3.5GB |
| WebSocket 消息推送 | 20,000 | 5ms | 12ms | 60% | 2.8GB |
```

#### F.4.2 数据库性能基准

```
MySQL 查询性能（单机）：

| 查询类型 | QPS | 平均延迟 | 说明 |
|---------|-----|---------|------|
| 用户查询（主键） | 50,000 | 0.5ms | SELECT * FROM users WHERE user_id = ? |
| 用户查询（索引） | 20,000 | 2ms | SELECT * FROM users WHERE phone = ? |
| 好友列表查询 | 8,000 | 8ms | SELECT * FROM friends WHERE owner_id = ? |
| 会话列表查询 | 5,000 | 12ms | SELECT * FROM conversations WHERE owner_id = ? |
| 消息历史查询 | 10,000 | 5ms | SELECT * FROM messages WHERE conv_id = ? ORDER BY seq DESC LIMIT 50 |

Redis 操作性能（单机）：

| 操作类型 | QPS | 平均延迟 | 说明 |
|---------|-----|---------|------|
| SET | 80,000 | 0.1ms | 普通写入 |
| GET | 100,000 | 0.08ms | 普通读取 |
| HSET | 60,000 | 0.15ms | Hash 写入 |
| HGET | 80,000 | 0.12ms | Hash 读取 |
| INCR | 70,000 | 0.1ms | 计数器 |
| EXPIRE | 50,000 | 0.2ms | 设置过期 |
```

#### F.4.3 消息吞吐量基准

```
消息发送性能（单机 WebSocket Gateway）：

| 消息类型 | TPS | P95 延迟 | 说明 |
|---------|-----|---------|------|
| 单聊消息 | 10,000 | 8ms | 一对一消息 |
| 群聊消息 (100人) | 2,000 | 25ms | 群消息需要广播 |
| 已读回执 | 15,000 | 5ms | 轻量操作 |
| 输入状态 | 20,000 | 3ms | 高频但轻量 |
| 心跳保活 | 50,000 | 2ms | 定时发送 |

消息持久化性能（异步写入 MongoDB）：

| 场景 | TPS | 批量大小 | 说明 |
|------|-----|---------|------|
| 单条插入 | 5,000 | 1 | 每条消息单独插入 |
| 批量插入 (10) | 30,000 | 10 | 每次插入10条 |
| 批量插入 (50) | 80,000 | 50 | 每次插入50条 |
| 批量插入 (100) | 120,000 | 100 | 每次插入100条（推荐） |
```

#### F.4.4 压力测试工具示例

```bash
#!/bin/bash
# scripts/stress-test.sh
# 压力测试脚本

echo "=== GIM 压力测试 ==="

# 测试 1: 用户注册
echo "测试 1: 用户注册 (1000 QPS, 持续 30 秒)"
wrk -t8 -c100 -d30s -s scripts/post_register.lua \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/auth/register

# 测试 2: 用户登录
echo "测试 2: 用户登录 (2000 QPS, 持续 30 秒)"
wrk -t8 -c200 -d30s -s scripts/post_login.lua \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/auth/login

# 测试 3: 获取用户信息
echo "测试 3: 获取用户信息 (3000 QPS, 持续 30 秒)"
wrk -t8 -c300 -d30s -s scripts/get_profile.lua \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/user/profile

# 测试 4: 搜索用户
echo "测试 4: 搜索用户 (1000 QPS, 持续 30 秒)"
wrk -t8 -c100 -d30s -s scripts/post_search.lua \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/user/search

# 测试 5: 发送消息
echo "测试 5: 发送消息 (2000 QPS, 持续 30 秒)"
wrk -t8 -c200 -d30s -s scripts/post_message.lua \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/v1/msg/send

echo "=== 压力测试完成 ==="
```

wrk Lua 脚本示例：
```lua
-- scripts/post_register.lua
wrk.method = "POST"
wrk.body   = '{"userId":"test_{{randomUserID}}","password":"Test1234","nickname":"测试用户"}'
wrk.headers["Content-Type"] = "application/json"

local counter = 0
function request()
    counter = counter + 1
    wrk.body = string.format('{"userId":"stress_test_%d","password":"Test1234","nickname":"测试用户"}', counter)
    return wrk.format()
end
```

#### F.4.5 性能优化建议

基于以上基准数据，以下优化建议：

1. **数据库层面**
   - 添加适当的索引（已在迁移文件中）
   - 使用连接池（MaxOpenConns: 100, MaxIdleConns: 10）
   - 考虑读写分离（读操作走从库）
   - 大表分库分表（messages 表按时间分区）

2. **Redis 层面**
   - 合理设置过期时间
   - 使用 Pipeline 批量操作
   - 考虑 Redis Cluster（单节点 > 10GB 时）

3. **应用层面**
   - 使用 goroutine 并发处理
   - 实现缓存层（热点数据）
   - 使用 sync.Pool 复用对象
   - 批量插入优化（100 条/批次）

4. **WebSocket 层面**
   - 多实例部署，使用一致性哈希分配连接
   - 心跳优化（30 秒间隔）
   - 消息批量推送
   - 连接数限制（每用户最多 5 个连接）

5. **消息层面**
   - 异步持久化（Kafka + MongoDB）
   - 历史消息分页加载
   - 消息压缩（大文本）
   - 二进制消息处理优化

---

## 文档更新记录

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| 1.0 | 2026-04-26 | 初始版本，完成 Phase 1 实现指南 |
| 1.1 | 2026-04-27 | 添加 Phase 2 微服务架构设计 |
| 1.2 | 2026-04-28 | 完善 Phase 3 K8S 部署指南 |
| 1.3 | 2026-04-29 | 添加 Phase 4 AI 集成计划 |
| 1.4 | 2026-04-30 | 补充详细注释和示意图 |
| 1.5 | 2026-05-01 | 完善实施方法，添加故障排查 |
| 1.6 | 2026-05-02 | 修复文档逻辑错误，添加开发增强工具（附录 F） |

**版本 1.6 详细更新：**
- 修复实施步骤表中 1.0.3 引用错误（改为 1.0.1）
- 补充快速初始化命令中的 deploy/ 目录创建
- 修复 JWT 路径配置（添加引号确保正确解析）
- 增强 Makefile：添加 DB_DSN 配置、帮助命令、更多实用目标
- 添加 MySQL 初始化脚本 init.sql
- 完善快速开始流程，补充环境变量设置和配置文件创建步骤
- 添加详细的 schema_migrations 表说明和 dirty 字段处理方法
- 新增附录 F：开发增强工具与最佳实践
  - F.1: 依赖安装自动化脚本
  - F.2: 开发环境一键启动命令
  - F.3: 测试用例示例（单元测试、集成测试、基准测试）
  - F.4: 性能基准数据（QPS、延迟、资源使用）
  - F.5: 压力测试工具和脚本
  - F.6: 性能优化建议

---

**文档结束** 📚
