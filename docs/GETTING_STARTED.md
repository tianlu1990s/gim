# GIM 零基础入门指南

> 本文档假设你是一个后端新手，手把手带你从零搭建开发环境、理解核心概念、跑通第一个模块。

---

## 一、前置概念速览

在动手之前，先理解这些你会反复遇到的术语。不需要完全记住，遇到不懂的回来看。

### 1.1 什么是 HTTP API？

HTTP 就是浏览器和服务器通信的协议。你在浏览器地址栏输入网址，就是发了一个 HTTP **GET** 请求。

IM 系统中，客户端（手机 App / 网页）和服务器之间主要通过两种方式通信：

| 方式 | 类比 | 特点 | 本项目用途 |
|------|------|------|-----------|
| HTTP API | 发短信 | 一问一答，发完就断 | 注册、登录、拉好友列表、查历史消息 |
| WebSocket | 打电话 | 建立连接后持续通话，双方随时说话 | 实时收发消息、在线状态、输入提示 |

**为什么收消息不能只用 HTTP？** 因为 HTTP 是"客户端主动问，服务端才答"。如果有人给你发消息，服务器没法主动告诉你。WebSocket 建立"通话"后，服务器可以随时推送消息给你。

### 1.2 什么是 JSON？

JSON 是一种数据格式，长得像这样：

```json
{
  "userId": "alice",
  "nickname": "爱丽丝",
  "age": 25
}
```

就是"键-值对"，和编程里的字典/Map/对象一个意思。客户端和服务器之间传递数据就用 JSON。

### 1.3 什么是 Redis？为什么需要它？

Redis 是一个**内存数据库**，数据存在内存中（不是硬盘），读写极快（微秒级）。

**MySQL vs Redis 的关系**（好比银行金库 vs 你口袋里的钱包）：

| | MySQL | Redis |
|--|-------|-------|
| 存储 | 硬盘（慢但可靠） | 内存（快但断电丢失） |
| 用途 | 存所有重要数据 | 存临时/高频访问的数据 |
| 类比 | 银行金库 | 随身钱包 |
| 本项目存什么 | 用户信息、好友关系、消息历史 | 在线状态、Token 黑名单、消息序号(Seq) |

**为什么在线状态放 Redis？** 因为在线状态变化极其频繁（每秒可能有成千上万用户上下线），写 MySQL 会拖垮数据库。而且在线状态丢了也无所谓——用户重新上线就好。

### 1.4 什么是 GORM？

GORM 是 Go 语言的 ORM（Object-Relational Mapping，对象关系映射）。

**不用 ORM（手写 SQL）：**
```go
db.Exec("INSERT INTO users (user_id, nickname) VALUES (?, ?)", "alice", "爱丽丝")
```

**用 GORM：**
```go
db.Create(&User{UserID: "alice", Nickname: "爱丽丝"})
```

GORM 把 Go 的结构体和数据库的表对应起来，你写 Go 代码，GORM 帮你生成 SQL。好处是不容易写错 SQL，代码可读性高。

### 1.5 什么是 JWT？

JWT（JSON Web Token）是一种无状态认证方案。

**传统方式（Session）：** 服务器记住"用户 A 已经登录了"（存内存/Redis），客户端每次请求带 cookie，服务器查 session 表。
**JWT 方式：** 服务器不记状态，登录时发一个"令牌"（Token），客户端每次请求带上这个令牌，服务器验证令牌的真伪即可。

JWT 的好处：服务器不需要存储 session，天然适合分布式（多个服务器都能验证同一个令牌）。

**类比：** Session 像是酒店房卡——前台记住你住哪间房；JWT 像是加密的身份证——谁看到都能验证你身份，不用去前台查。

### 1.6 什么是 WebSocket？和 HTTP 有什么区别？

```
HTTP（一问一答）：
客户端：给我好友列表
服务端：[alice, bob, carol]
（连接断开）

客户端：给我消息
服务端：[msg1, msg2]
（连接断开）

WebSocket（持续通话）：
客户端 <-> 服务端（建立连接，保持打开）
客户端：发消息给bob
服务端：收到，seq=42
服务端：bob给你发了消息（服务端主动推送！）
服务端：carol也给你发了消息
客户端：标记已读
...
```

WebSocket 的关键优势：**服务器可以主动推送消息**，不需要客户端不停轮询。

### 1.7 什么是 Seq？为什么消息需要序号？

Seq（Sequence，序列号）是消息的"编号"，每条消息在同一个会话内递增。

```
alice 和 bob 的会话：
  seq=1: alice: "你好"
  seq=2: bob: "在吗"
  seq=3: alice: "今天有空吗"
  seq=4: bob: "有啊"
```

**Seq 解决三大问题：**

1. **消息不丢**：客户端记录自己已收到的最大 seq（比如 2），上线后请求"给我 seq > 2 的消息"，就能拿到 3、4，不会漏。
2. **消息有序**：即使网络导致消息乱序到达，客户端按 seq 排序即可恢复正确顺序。
3. **已读回执**：bob 的 readSeq=3，表示 bob 读到了 seq=3 的消息。alice 看到后就知道自己 seq=1,2,3 的消息 bob 都看了。

---

## 二、开发环境搭建

### 2.1 安装 Go

```bash
# 下载 Go 1.21+（推荐用最新稳定版）
# 访问 https://go.dev/dl/ 下载对应平台的安装包

# Debian/Ubuntu 也可以用：
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz

# 配置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# 验证
go version
# 输出：go version go1.26.0 linux/amd64
```

**Go 基础速记**（如果你不熟悉 Go）：

```go
// 变量声明
var name string = "alice"    // 完整写法
name := "alice"              // 短写法（函数内，Go 自动推断类型）

// 函数
func add(a int, b int) int {
    return a + b
}

// 结构体（类似 C 的 struct 或 Java 的 class）
type User struct {
    UserID   string
    Nickname string
    Age      int
}

// 创建实例
u := User{UserID: "alice", Nickname: "爱丽丝"}

// 方法（给结构体绑定函数）
func (u *User) Greet() string {
    return "Hello, " + u.Nickname
}

// 接口（不需要显式 implement，只要方法签名匹配）
type Greeter interface {
    Greet() string
}
var g Greeter = &u  // User 有 Greet() 方法，所以实现了 Greeter

// 错误处理（Go 没有 try-catch，用返回值）
result, err := someFunction()
if err != nil {
    // 处理错误
}

// goroutine（轻量级线程，Go 的核心并发机制）
go doSomething()  // 在后台并发执行

// channel（goroutine 之间通信的管道）
ch := make(chan string)
go func() { ch <- "hello" }()  // 发送
msg := <-ch                      // 接收
```

### 2.2 安装 Docker 和 Docker Compose

Docker 是容器化工具——把你的应用和它的依赖（MySQL、Redis 等）打包在一起运行，不用在本机装一堆软件。

```bash
# Debian/Ubuntu 安装 Docker
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg
curl -fsSL https://download.docker.com/linux/debian/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo $VERSION_CODENAME) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# 把当前用户加入 docker 组（免 sudo）
sudo usermod -aG docker $USER
newgrp docker

# 验证
docker --version
docker compose version
```

### 2.3 启动开发依赖（MySQL + Redis）

不用在本机装 MySQL 和 Redis，用 Docker Compose 一键启动：

```bash
# 在项目根目录创建开发环境配置
mkdir -p deploy

# 启动
docker compose -f deploy/docker-compose.yaml up -d

# 查看是否启动成功
docker compose -f deploy/docker-compose.yaml ps

# 连接 MySQL 测试
docker exec -it gim-mysql mysql -ugim -pgim_pass gim

# 连接 Redis 测试
docker exec -it gim-redis redis-cli ping
# 输出 PONG 就表示成功
```

**MySQL 初学者要点：**
- 数据库是一个"文件夹"，表是"文件"，每行数据是"一条记录"
- 我们的项目创建一个叫 `gim` 的数据库，里面有多张表（users、friends、messages 等）
- SQL 命令举例：
  ```sql
  -- 查看所有数据库
  SHOW DATABASES;
  -- 使用 gim 数据库
  USE gim;
  -- 查看所有表
  SHOW TABLES;
  -- 查看 users 表的所有数据
  SELECT * FROM users;
  ```

**Redis 初学者要点：**
- Redis 是"键值存储"，就像一个大字典：`SET key value` 存，`GET key` 取
- 常用命令：
  ```bash
  SET name "alice"       # 存
  GET name               # 取，返回 "alice"
  DEL name               # 删
  INCR counter           # 计数器 +1（原子操作，多线程安全）
  SETNX lock "1"         # 只在 key 不存在时设置（分布式锁/去重的关键）
  EXPIRE name 60         # 设置 60 秒后过期
  SADD myset "a" "b"     # 集合操作
  SMEMBERS myset         # 查看集合所有成员
  ```

### 2.4 安装开发工具

```bash
# 安装 golangci-lint（代码检查工具）
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 安装 golang-migrate（数据库迁移工具）
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 安装 swag（Swagger 文档生成，可选）
go install github.com/swaggo/swag/cmd/swag@latest

# 验证
golangci-lint --version
migrate --version
```

**推荐 IDE：** VS Code + Go 扩展，或 JetBrains GoLand。

---

## 三、项目结构解读

```
gim/
├── cmd/gim/main.go          ← 程序入口，从这里开始运行
├── internal/                 ← 私有代码（其他项目不能 import）
│   ├── config/              ← 读取 config.yaml 配置文件
│   ├── handler/             ← HTTP 请求处理器（收请求、调 service、返响应）
│   ├── service/             ← 业务逻辑（核心计算、校验、流程编排）
│   ├── repository/          ← 数据库操作（SQL/Redis 读写）
│   ├── model/               ← 数据结构定义（对应数据库表）
│   ├── middleware/           ← 中间件（鉴权、限流、日志等"拦截器"）
│   └── ws/                  ← WebSocket 网关（长连接管理）
├── pkg/                      ← 公共工具包（可以被其他项目复用）
│   ├── jwt/                 ← JWT 令牌生成和验证
│   ├── snowflake/           ← ID 生成器
│   ├── resp/                ← 统一响应格式
│   └── errcode/             ← 错误码定义
├── migrations/               ← 数据库建表脚本（按顺序执行）
├── configs/config.yaml       ← 配置文件
├── deploy/                   ← 部署相关（Docker、K8S）
├── docs/                     ← 文档
├── go.mod                    ← 依赖管理（类似 package.json）
└── Makefile                  ← 常用命令快捷方式
```

### 三层架构：Handler → Service → Repository

这是后端最经典的分层模式，每一层只做自己的事：

```
客户端请求
    │
    ▼
┌─────────────────────────────────────┐
│  Handler（处理器）                    │
│  只做：接收请求、参数校验、调用       │
│  Service、返回响应                    │
│  不做：不写业务逻辑，不直接操作数据库  │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  Service（业务层）                    │
│  只做：业务逻辑、流程编排、校验       │
│  不做：不直接写 SQL，不关心 HTTP 格式  │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  Repository（数据层）                 │
│  只做：数据库读写（SQL/Redis 操作）   │
│  不做：不包含业务逻辑                 │
└─────────────────────────────────────┘
```

**举个具体例子——用户注册：**

```
Handler:
  1. 解析请求 JSON -> RegisterReq{userId, password, nickname}
  2. 调用 service.Register(req)
  3. 把返回的 User 转成 JSON 响应返回

Service:
  1. 检查 userId 是否已存在 -> 调用 repo.ExistsByID()
  2. 密码哈希 -> bcrypt.GenerateFromPassword()
  3. 保存用户 -> 调用 repo.Create()
  4. 返回 User

Repository:
  1. 执行 SQL: INSERT INTO users ...
```

### 数据流向图（一条消息的完整旅程）

```
alice 点击"发送"
    │
    ▼
┌──────────────┐
│  客户端 WS    │  发送 JSON: {type:1, data:{conversationId, content, ...}}
└──────┬───────┘
       │ WebSocket 连接
       ▼
┌──────────────┐
│  WS Gateway   │  Hub 找到 alice 的连接，调用 handleMessage()
│  (ws/client)  │  解析消息类型 type=1 -> 调用 msgService.SendMessage()
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Message      │  1. Redis SETNX 去重(clientMsgId)
│  Service      │  2. 检查好友关系
│               │  3. Redis INCR 分配 Seq
│               │  4. 生成 serverMsgId (Snowflake)
│               │  5. 写入 MySQL (messages 表)
│               │  6. 更新会话 maxSeq
└──────┬───────┘
       │
       ├──→ 返回 Seq 给 alice（通过 WS 连接，type=101 发送确认）
       │
       └──→ 推送消息给 bob（通过 Hub.PushToUser，type=101 新消息通知）
                │
                ▼
           ┌──────────────┐
           │  bob 的 WS    │  bob 收到推送，UI 显示新消息
           │  连接         │
           └──────────────┘
```

---

## 四、第一个模块实战：跑通认证模块

跟着以下步骤，从零跑通"注册 → 登录 → 访问受保护接口"的完整链路。

### Step 1：初始化项目

```bash
cd ~/github.com/gim

# 初始化 Go Module
go mod init github.com/yourname/gim

# 安装核心依赖
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/mysql
go get github.com/redis/go-redis/v9
go get github.com/spf13/viper
go get go.uber.org/zap
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
go get github.com/bwmarrin/snowflake
go get github.com/golang-migrate/migrate/v4
```

**go.mod 是什么？** 类似 Node.js 的 package.json，记录项目依赖。`go get xxx` 会自动更新 go.mod 和 go.sum。

### Step 2：创建配置文件

```bash
mkdir -p configs
```

创建 `configs/config.yaml`（内容见 API.md §9 配置文件参考）。

### Step 3：创建目录结构

```bash
mkdir -p cmd/gim
mkdir -p internal/{config,handler,service,repository,model,middleware,ws}
mkdir -p pkg/{jwt,snowflake,resp,errcode}
mkdir -p migrations
mkdir -p deploy/docker
```

### Step 4：编写最简 main.go

先写一个能启动的骨架，验证 Gin 框架工作正常：

```go
// cmd/gim/main.go
package main

import (
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()

    r.GET("/ping", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "pong"})
    })

    fmt.Println("Server starting on :8080")
    r.Run(":8080")
}
```

运行：

```bash
go run cmd/gim/main.go

# 另一个终端测试
curl http://localhost:8080/ping
# 输出：{"message":"pong"}
```

**恭喜，你的第一个 HTTP 接口已经跑通了！**

### Step 5：连接 MySQL 和 Redis

按照 IMPLEMENTATION.md §1.2 的 Config 结构体和 §1 的 main.go 完整版本，逐步替换简单骨架。每加一个组件就运行验证：

```bash
# 加了 MySQL 连接后运行，看日志有没有报错
go run cmd/gim/main.go

# 加了 Redis 连接后运行
go run cmd/gim/main.go
```

**排错技巧：**
- `dial tcp 127.0.0.1:3306: connect: connection refused` → MySQL 没启动，运行 `docker compose up -d`
- `dial tcp 127.0.0.1:6379: connect: connection refused` → Redis 没启动
- `Access denied for user` → 检查 config.yaml 中的用户名密码

### Step 6：执行数据库迁移

```bash
# 创建第一张表的迁移文件
migrate create -ext sql -dir migrations -seq create_users_table

# 编辑 migrations/000001_create_users_table.up.sql（内容见 PLAN.md §五）
# 编辑 migrations/000001_create_users_table.down.sql（写 DROP TABLE IF EXISTS users;）

# 执行迁移
migrate -path migrations -database "mysql://gim:gim_pass@tcp(localhost:3306)/gim" up

# 验证
docker exec -it gim-mysql mysql -ugim -pgim_pass gim -e "SHOW TABLES;"
```

### Step 7：实现注册接口

按 IMPLEMENTATION.md §2 的代码，依次实现：

1. `pkg/errcode/errcode.go` — 错误码定义
2. `pkg/resp/resp.go` — 统一响应
3. `internal/model/user.go` — User 结构体
4. `internal/repository/user.go` — 用户数据库操作
5. `internal/service/auth.go` — 注册逻辑
6. `internal/handler/auth.go` — HTTP 处理

**验证：**

```bash
# 注册
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"userId":"alice","password":"P@ssw0rd123","nickname":"爱丽丝"}'

# 成功响应：
# {"code":0,"msg":"success","data":{"userId":"alice","nickname":"爱丽丝",...}}
```

### Step 8：实现登录接口

继续按 IMPLEMENTATION.md §2.3 实现 Login 方法。

**验证：**

```bash
# 登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"userId":"alice","password":"P@ssw0rd123"}'

# 成功响应：
# {"code":0,"msg":"success","data":{"accessToken":"eyJ...","refreshToken":"eyJ...",...}}
```

### Step 9：实现鉴权中间件

按 IMPLEMENTATION.md §10.1 实现 JWT 鉴权中间件。

**验证鉴权生效：**

```bash
# 不带 Token 访问受保护接口
curl http://localhost:8080/api/v1/user/profile
# 返回 401 Unauthorized

# 带上 Token
curl http://localhost:8080/api/v1/user/profile \
  -H "Authorization: Bearer eyJ..."
# 返回用户信息
```

**到这一步，认证模块的核心链路已跑通。** 接下来的模块（用户、好友、消息）模式相同：Model → Repository → Service → Handler，逐步实现即可。

---

## 五、常见问题与排错

### Q: `go run` 报 `package xxx is not in GOROOT`

Go 找不到依赖包。运行：
```bash
go mod tidy    # 整理依赖
go mod download # 下载依赖
```

### Q: MySQL 连接报 `too many connections`

连接池配置太大或忘记关闭连接。检查 config.yaml 的 `maxOpenConns`，确保 GORM 初始化后不要重复创建连接。

### Q: Redis 报 `connection refused`

1. 确认 Docker 容器在运行：`docker ps`
2. 确认端口没被占用：`ss -tlnp | grep 6379`
3. 检查 config.yaml 的 host 和 port

### Q: WebSocket 连接失败

1. 确认 WS 端口（默认 8081）已启动
2. 用浏览器工具测试：`new WebSocket("ws://localhost:8081/ws?token=xxx")`
3. 检查 Token 是否有效（先用 HTTP 登录获取 Token）

### Q: 编码时不知道方法该写在哪个层？

| 你要做的事 | 写在哪一层 | 举例 |
|-----------|-----------|------|
| 解析请求参数、返回 JSON | Handler | `c.ShouldBindJSON(&req)` |
| 判断业务条件（是否好友、是否已存在） | Service | `if !isFriend { return err }` |
| 执行 SQL 查询 | Repository | `db.Where(...).First(&user)` |
| 生成 Token、哈希密码 | Service（调用 pkg 工具） | `bcrypt.GenerateFromPassword()` |
| 校验 Token、检查权限 | Middleware | `jwtMgr.ParseToken(tokenStr)` |

---

## 六、学习路径建议

按以下顺序学习，每个阶段都有明确的验收标准：

| 阶段 | 学什么 | 验收标准 |
|------|--------|----------|
| 第1周 | Go 基础语法、项目骨架搭建 | `go run` 启动服务，`/ping` 返回 pong |
| 第2周 | 认证模块（HTTP + MySQL + Redis + JWT） | 注册→登录→带 Token 访问接口 |
| 第3周 | 用户/好友模块（CRUD、事务） | 能添加好友、查看好友列表 |
| 第4周 | 会话模块、消息模块 HTTP 部分 | 能通过 HTTP 拉取消息历史 |
| 第5-7周 | WebSocket 网关、消息实时收发 | 两个用户能实时聊天 |
| 第8周 | 收尾、Docker、压测 | 完整 docker-compose up 跑通 |

### 推荐学习顺序（每学一个技术点就用在项目里）

1. **Go 基础** → 写 config/model/resp 这类简单结构体
2. **Gin 框架** → 写 Handler，理解路由和中间件
3. **GORM** → 写 Repository，理解数据库 CRUD
4. **Redis** → 在线状态、Token 黑名单、Seq 生成
5. **JWT** → 认证中间件
6. **WebSocket** → 消息实时收发（最难的部分，留到后面）
7. **Docker** → 容器化部署（理解了代码再打包）
8. **K8S** → 第三阶段再学（有 Docker 基础后才好理解）
