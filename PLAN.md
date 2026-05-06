# GIM — Go IM 项目完整计划

## 一、项目定位与目标

基于云原生架构的即时通讯系统，以实战驱动深度学习 Go 语言、Kubernetes、高并发后端技术栈。

### 核心学习目标

| 领域 | 具体技能 |
|------|----------|
| Go 深入 | 并发模型(goroutine/channel/context)、接口设计、错误处理、性能调优(pprof)、代码组织 |
| 云原生 | Docker 多阶段构建、K8S 编排(Deployment/Service/ConfigMap/HPA)、Helm Chart |
| 高并发 | WebSocket 长连接管理、消息队列解耦、Redis 缓存策略、连接池、限流/熔断 |
| 数据库 | MySQL 事务与索引设计、MongoDB 文档模型、Redis 数据结构应用 |
| 消息可靠性 | Seq 序号机制、消息去重、离线消息拉取、已读回执 |

### 对初始方案的修正

1. **不建议第一阶段就使用 MongoDB**：第一阶段以 MySQL 为主存储，降低复杂度。MongoDB 在第二阶段引入，专门用于消息存储（参考 OpenIM 的文档分片设计）。
2. **不建议一开始就引入 Kafka**：第一阶段用 Redis Stream 或直接写库，Kafka 在第二阶段引入。Kafka 运维成本高，过早引入会拖慢第一阶段交付。
3. **单聊/群聊不宜同时实现**：第一阶段先做单聊，群聊涉及群成员管理、权限、消息扇出等复杂逻辑，放在第二阶段。
4. **后台管理系统放在第三阶段合理**：但账号管理（注册/登录/Token）属于基础设施，必须在第一阶段完成。

---

## 二、四阶段规划总览

### 第一阶段：最小可运行单机版（预计 6-8 周）

**目标**：一个能跑起来的单机 IM，支持注册登录、单聊、消息可靠投递。

### 第二阶段：分布式架构与核心功能（预计 8-10 周）

**目标**：微服务拆分、消息队列解耦、群聊、MongoDB 消息存储、水平扩展能力。

### 第三阶段：运营支撑与生产化（预计 6-8 周）

**目标**：后台管理系统、监控告警、压力测试、K8S 生产部署方案。

### 第四阶段：AI Agent 集成（预计 8-10 周）

**目标**：在 IM 系统中嵌入 AI Agent，实战学习 LLM API 调用、Tool Use、RAG、Multi-Agent 协作。
**详细方案**：见 [docs/AI_AGENT.md](docs/AI_AGENT.md)

---

## 三、系统架构

### 第一阶段架构（单体）

```
┌─────────────────────────────────────────────────────┐
│                    gim (单体服务)                      │
│                                                       │
│  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌────────┐ │
│  │ HTTP API│  │   WS    │  │  消息处理  │  │ 认证   │ │
│  │ (Gin)   │  │ Gateway │  │  发送/拉取 │  │ 模块   │ │
│  └────┬────┘  └────┬────┘  └─────┬─────┘  └───┬────┘ │
│       │            │              │             │      │
│  ┌────┴────────────┴──────────────┴─────────────┴────┐ │
│  │                  业务逻辑层                         │ │
│  │  用户管理 │ 会话管理 │ 消息管理 │ 好友管理          │ │
│  └────────────────────┬──────────────────────────────┘ │
│                       │                                │
│  ┌────────────────────┴──────────────────────────────┐ │
│  │                  数据访问层                         │ │
│  └───────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────┘
          │              │               │
     ┌────┴────┐   ┌────┴────┐    ┌─────┴─────┐
     │  MySQL  │   │  Redis  │    │  本地文件  │
     │用户/会话 │   │在线状态  │    │  日志等    │
     │消息/好友 │   │Token    │    │           │
     └─────────┘   │Seq缓存  │    └───────────┘
                   └─────────┘
```

### 第二阶段架构（微服务）

```
                    ┌──────────────┐
                    │   客户端      │
                    └──┬───────┬───┘
                  HTTP│       │WS
            ┌─────────┴─┐  ┌──┴──────────┐
            │  API 网关  │  │  WS Gateway  │
            │  (Gin)     │  │ (WS+gRPC)   │
            └─────┬──────┘  └──────┬──────┘
                  │ gRPC           │ gRPC
        ┌─────────┼────────────────┼──────────┐
        │         │                │           │
  ┌─────┴───┐ ┌──┴────┐ ┌────────┴──┐ ┌─────┴──────┐
  │  Auth   │ │  User │ │   Msg     │ │   Push     │
  │  RPC    │ │  RPC  │ │   RPC     │ │   Service  │
  └─────────┘ │  +    │ └─────┬─────┘ └─────┬──────┘
              │ Friend│       │             │
              │  RPC  │       │ Kafka       │ Kafka
              └───────┘       │             │
                          ┌───┴───┐    ┌────┴──────┐
                          │  Msg  │    │  离线推送  │
                          │Transfer│   │  Service   │
                          └───┬───┘    └───────────┘
                              │
                    ┌─────────┼──────────┐
                    │         │          │
               ┌────┴───┐ ┌──┴───┐ ┌───┴────┐
               │ MySQL  │ │MongoDB│ │ Redis  │
               │用户/群组│ │消息   │ │缓存/Seq│
               │好友/会话│ │       │ │在线状态 │
               └────────┘ └──────┘ └────────┘
```

### 第三阶段完整架构

在第二阶段基础上增加：

```
┌──────────────────────────────────────────────────────────┐
│                     运维与运营层                           │
│                                                            │
│  ┌────────────┐  ┌────────────┐  ┌─────────────────────┐ │
│  │ Admin API  │  │ Prometheus │  │   日志聚合           │ │
│  │ + Web UI   │  │ + Grafana  │  │  (Loki/ELK)         │ │
│  └─────┬──────┘  └─────┬──────┘  └─────────────────────┘ │
│        │               │                                    │
│  ┌─────┴───────────────┴──────────────────────────────────┤│
│  │              K8S 集群                                    ││
│  │  ┌──────────────────────────────────────────────────┐  ││
│  │  │  Ingress / LoadBalancer                          │  ││
│  │  ├──────────────────────────────────────────────────┤  ││
│  │  │  API Gateway (Deployment + HPA)                  │  ││
│  │  │  WS Gateway  (StatefulSet + HPA)                 │  ││
│  │  │  Auth/User/Friend/Msg RPC (Deployment)           │  ││
│  │  │  Push/MsgTransfer (Deployment + HPA)             │  ││
│  │  │  Admin API (Deployment)                          │  ││
│  │  ├──────────────────────────────────────────────────┤  ││
│  │  │  Infrastructure: MySQL/MongoDB/Redis/Kafka       │  ││
│  │  │  (StatefulSet 或外部托管)                         │  ││
│  │  └──────────────────────────────────────────────────┘  ││
│  └────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

### 第四阶段架构（AI Agent 层）

在第三阶段基础上，AI Agent 作为消息管道上的"智能消费者"接入：

```
┌──────────────────────────────────────────────────────────┐
│                    AI Agent 层                            │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │  Agent Router（路由器）                               │ │
│  │  解析用户意图 → 分发到对应 Agent                      │ │
│  └──────┬──────────┬──────────┬──────────┬─────────────┘ │
│         │          │          │          │                │
│  ┌──────┴───┐ ┌────┴────┐ ┌──┴───────┐ ┌┴───────────┐  │
│  │ 智能回复  │ │内容审核  │ │管理助手   │ │群聊多Agent  │  │
│  │ Agent    │ │Agent    │ │Agent     │ │协作        │  │
│  │          │ │         │ │          │ │            │  │
│  │ Claude   │ │规则引擎 │ │Tool Use  │ │Summary     │  │
│  │ 流式输出  │ │+Claude  │ │+RAG      │ │Todo        │  │
│  │          │ │Tool Use │ │          │ │Remind      │  │
│  └──────────┘ └─────────┘ └──────────┘ │QA          │  │
│                                          └────────────┘  │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │  基础设施                                            │ │
│  │  AI Provider(Deepseek/Claude/Local) │ Milvus(向量库) │ Kafka(toModeration)  │ │
│  └─────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘

接入方式：
  WS Gateway ──(检测@gim-bot)──→ AI Service ──(流式推送)──→ 客户端
  AI Service ──(AIProvider接口)──→ Deepseek/Claude/本地模型（统一调用）
  Kafka toModeration ──(消费)──→ Moderation Agent ──(审核结果)──→ 业务处理
  Admin API ──(对话接口)──→ Admin Assistant Agent ──(Tool Use)──→ 管理操作
```

---

## 四、技术栈

### 第一阶段

| 类别 | 技术 | 用途 |
|------|------|------|
| 语言 | Go 1.26+ | 全部业务代码 |
| HTTP 框架 | Gin | REST API |
| WebSocket | gorilla/websocket | 长连接 |
| 数据库 | MySQL 8.4 LTS | 全部持久化 |
| 缓存 | Redis 7.x | 在线状态、Token、Seq 缓存 |
| 认证 | JWT (RS256) | 无状态 Token 认证 |
| 配置 | Viper | YAML 配置管理 |
| 日志 | Zap | 结构化日志 |
| ORM | GORM | 数据库操作（第一阶段简化开发） |
| 迁移 | golang-migrate | 数据库 Schema 版本管理 |
| 容器 | Docker | 开发环境一致性 |

### 第二阶段新增

| 类别 | 技术 | 用途 |
|------|------|------|
| RPC | gRPC + Protobuf | 服务间通信 |
| 服务发现 | etcd | RPC 服务注册与发现 |
| 消息队列 | Kafka | 消息异步写入解耦 |
| 消息存储 | MongoDB | 消息文档分片存储 |
| 对象存储 | S3/MinIO/OSS | 文件/图片/音视频（S3 协议兼容） |
| 限流熔断 | sentinel/ratelimit | 服务保护 |

### 第三阶段新增

| 类别 | 技术 | 用途 |
|------|------|------|
| 监控 | Prometheus + Grafana | 指标采集与可视化 |
| 日志聚合 | Loki + Promtail | 日志采集与查询 |
| 链路追踪 | OpenTelemetry + Jaeger | 分布式追踪 |
| K8S 部署 | Helm | 包管理与服务编排 |
| 前端 | React + Ant Design Pro | 后台管理界面 |

### 第四阶段新增

| 类别 | 技术 | 用途 |
|------|------|------|
| LLM API | Deepseek API / Claude API / 本地部署 | 智能回复、内容审核、管理助手（多 Provider 可切换） |
| 本地模型 | Ollama / vLLM | 开发环境本地运行，降低 API 成本 |
| Go SDK | anthropic-sdk-go / openai-go | API 调用（流式+Tool Use），OpenAI 兼容协议 |
| 向量数据库 | Milvus / pgvector | RAG 知识库存储与检索 |
| Embedding | OpenAI text-embedding-3-small / 本地 Embedding | 文档向量化 |

> **AI Provider 设计**：系统通过统一的 `AIProvider` 接口支持多后端切换。开发阶段默认使用本地部署模型（Ollama），生产环境可按需切换 Deepseek API 或 Claude API。Deepseek API 兼容 OpenAI 协议，迁移成本低。

**详细方案**：见 [docs/AI_AGENT.md](docs/AI_AGENT.md)

---

## 五、数据模型

### 第一阶段（MySQL）

#### users — 用户表

```sql
CREATE TABLE users (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id    VARCHAR(64)  NOT NULL UNIQUE COMMENT '用户唯一ID',
    nickname   VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '昵称',
    avatar_url VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像URL',
    password   VARCHAR(128) NOT NULL COMMENT 'bcrypt哈希密码',
    phone      VARCHAR(20)  NOT NULL DEFAULT '' COMMENT '手机号',
    email      VARCHAR(128) NOT NULL DEFAULT '' COMMENT '邮箱',
    status     TINYINT      NOT NULL DEFAULT 1 COMMENT '1-正常 2-禁用',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_phone (phone),
    INDEX idx_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### friends — 好友关系表

```sql
CREATE TABLE friends (
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    owner_id     VARCHAR(64)  NOT NULL COMMENT '好友关系拥有者',
    friend_id    VARCHAR(64)  NOT NULL COMMENT '好友UserID',
    remark       VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '好友备注',
    is_pinned    TINYINT      NOT NULL DEFAULT 0 COMMENT '是否置顶',
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_owner_friend (owner_id, friend_id),
    INDEX idx_friend (friend_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### friend_requests — 好友申请表

```sql
CREATE TABLE friend_requests (
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    from_id      VARCHAR(64)  NOT NULL COMMENT '申请人',
    to_id        VARCHAR(64)  NOT NULL COMMENT '被申请人',
    message      VARCHAR(256) NOT NULL DEFAULT '' COMMENT '申请消息',
    status       TINYINT      NOT NULL DEFAULT 0 COMMENT '0-待处理 1-已同意 2-已拒绝',
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_to_status (to_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### conversations — 会话表

```sql
CREATE TABLE conversations (
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    conversation_id  VARCHAR(128) NOT NULL UNIQUE COMMENT '会话ID: single_userA_userB / group_groupID',
    owner_id         VARCHAR(64)  NOT NULL COMMENT '会话所属用户',
    type             TINYINT      NOT NULL COMMENT '1-单聊 2-群聊',
    target_id        VARCHAR(64)  NOT NULL COMMENT '对方UserID或GroupID',
    is_pinned        TINYINT      NOT NULL DEFAULT 0,
    max_seq          BIGINT       NOT NULL DEFAULT 0 COMMENT '会话最大seq',
    min_seq          BIGINT       NOT NULL DEFAULT 0 COMMENT '会话最小seq',
    created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_owner_pinned (owner_id, is_pinned, updated_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### messages — 消息表

```sql
CREATE TABLE messages (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    conversation_id VARCHAR(128) NOT NULL COMMENT '会话ID',
    seq             BIGINT       NOT NULL COMMENT '消息序号',
    sender_id       VARCHAR(64)  NOT NULL COMMENT '发送者',
    msg_type        TINYINT      NOT NULL COMMENT '1-文本 2-图片 3-语音 4-视频 5-文件 6-系统通知',
    content         TEXT         NOT NULL COMMENT '消息内容(JSON)',
    client_msg_id   VARCHAR(64)  NOT NULL COMMENT '客户端消息ID(去重)',
    server_msg_id   VARCHAR(64)  NOT NULL COMMENT '服务端消息ID',
    is_read         TINYINT      NOT NULL DEFAULT 0,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_conv_seq (conversation_id, seq),
    UNIQUE KEY uk_client_msg (client_msg_id),
    INDEX idx_conv_seq (conversation_id, seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### user_conversation_seq — 用户会话已读位置

```sql
CREATE TABLE user_conversation_seq (
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id          VARCHAR(64)  NOT NULL,
    conversation_id  VARCHAR(128) NOT NULL,
    read_seq         BIGINT       NOT NULL DEFAULT 0 COMMENT '已读seq位置',
    UNIQUE KEY uk_user_conv (user_id, conversation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 第二阶段新增（MongoDB）

#### msg_docs — 消息文档（分片设计，参考 OpenIM）

```javascript
// 每个 conversation 每 100 条消息存为一个文档
{
    _id: "conv_userA_userB:0",  // DocID = conversationID:seqSuffix, seqSuffix = (seq-1)/100
    msgs: [
        {
            msg: {
                senderId: "userA",
                recvId: "userB",
                groupId: "",
                clientMsgId: "client-uuid-1",
                serverMsgId: "server-uuid-1",
                contentType: 1,       // 1-文本 2-图片 ...
                content: "hello",
                seq: 1,
                sendTime: 1714000000000,
                status: 1,            // 1-已发送 2-已撤回
                isRead: false,
                offlinePush: { title: "", content: "", ex: "" }
            },
            revoke: null,             // 撤回信息
            delList: []               // 删除列表(对某人不可见)
        }
        // ... 最多 100 条
    ]
}
```

#### groups — 群组表（MySQL）

```sql
CREATE TABLE groups (
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    group_id         VARCHAR(64)  NOT NULL UNIQUE,
    group_name       VARCHAR(64)  NOT NULL DEFAULT '',
    avatar_url       VARCHAR(512) NOT NULL DEFAULT '',
    notification     VARCHAR(512) NOT NULL DEFAULT '' COMMENT '群公告',
    introduction     VARCHAR(256) NOT NULL DEFAULT '' COMMENT '群简介',
    creator_id       VARCHAR(64)  NOT NULL,
    status           TINYINT      NOT NULL DEFAULT 1 COMMENT '1-正常 2-解散',
    max_members      INT          NOT NULL DEFAULT 500,
    created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### group_members — 群成员表（MySQL）

```sql
CREATE TABLE group_members (
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    group_id     VARCHAR(64)  NOT NULL,
    user_id      VARCHAR(64)  NOT NULL,
    nickname     VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '群内昵称',
    role         TINYINT      NOT NULL DEFAULT 0 COMMENT '0-普通成员 1-管理员 2-群主',
    join_source  TINYINT      NOT NULL DEFAULT 0 COMMENT '0-邀请 1-搜索 2-二维码',
    inviter_id   VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '邀请人',
    mute_end     DATETIME     NULL COMMENT '禁言截止时间',
    join_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_group_user (group_id, user_id),
    INDEX idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### group_requests — 入群申请表（MySQL）

```sql
CREATE TABLE group_requests (
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    group_id     VARCHAR(64)  NOT NULL,
    user_id      VARCHAR(64)  NOT NULL COMMENT '申请人',
    message      VARCHAR(256) NOT NULL DEFAULT '',
    status       TINYINT      NOT NULL DEFAULT 0 COMMENT '0-待处理 1-已同意 2-已拒绝',
    handler_id   VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '处理人',
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_group_status (group_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 第四阶段新增（MySQL）

```sql
-- 违规记录（内容审核 Agent）
CREATE TABLE violations (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    conversation_id VARCHAR(128) NOT NULL,
    client_msg_id   VARCHAR(64)  NOT NULL,
    sender_id       VARCHAR(64)  NOT NULL,
    category        VARCHAR(32)  NOT NULL COMMENT 'spam/abuse/politics/porn/ad/other',
    severity        VARCHAR(16)  NOT NULL COMMENT 'low/medium/high',
    reason          TEXT         NOT NULL,
    action          VARCHAR(32)  NOT NULL COMMENT 'mark/revoke/warn',
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_sender (sender_id),
    INDEX idx_category_time (category, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 敏感词库
CREATE TABLE sensitive_words (
    id        BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    word      VARCHAR(64) NOT NULL UNIQUE,
    category  VARCHAR(32) NOT NULL,
    level     TINYINT     NOT NULL DEFAULT 1 COMMENT '1-替换 2-拦截',
    created_at DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 群待办事项
CREATE TABLE group_todos (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    group_id    VARCHAR(64)  NOT NULL,
    content     VARCHAR(512) NOT NULL,
    assignee_id VARCHAR(64)  NOT NULL DEFAULT '',
    deadline    DATETIME     NULL,
    is_done     TINYINT      NOT NULL DEFAULT 0,
    source      VARCHAR(16)  NOT NULL DEFAULT 'ai' COMMENT 'manual/ai',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_group_done (group_id, is_done)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- AI 对话记录
CREATE TABLE ai_conversations (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id     VARCHAR(64)  NOT NULL,
    agent_type  VARCHAR(32)  NOT NULL COMMENT 'reply/moderation/admin/summary/todo/qa',
    messages    JSON         NOT NULL COMMENT '完整对话上下文',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_type (user_id, agent_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

> 第四阶段数据模型详细说明见 [docs/AI_AGENT.md §7.5](docs/AI_AGENT.md)

---

## 六、核心机制设计

### 6.1 消息投递与可靠性

采用 **Seq 序号机制** 保证消息可靠投递（参考 OpenIM），核心流程：

```
发送方客户端                   服务端                          接收方客户端
    │                           │                                │
    │─── 1. 发送消息(clientMsgId)──>│                             │
    │                           │─── 2. 分配Seq ──>              │
    │                           │─── 3. 持久化消息 ──>           │
    │                           │─── 4. 推送通知 ───────────────>│
    │<── 5. 返回Seq ─────────────│                               │
    │                           │                                │
    │                           │<── 6. 拉取消息(seq > localSeq)──│
    │                           │─── 7. 返回消息列表 ───────────>│
    │                           │                                │
    │                           │<── 8. 上报已读(readSeq) ────────│
    │<── 9. 已读回执通知 ────────│                                │
```

**关键设计原则**：

1. **消息不丢失**：消息先持久化再推通知，客户端通过 Seq 间隙检测可以补拉
2. **消息不重复**：clientMsgId 全局去重，serverMsgId 生成唯一 ID
3. **消息有序**：同一会话内 Seq 严格递增，客户端按 Seq 排序
4. **已读回执**：基于 readSeq，发送方对比对方 readSeq 与自己消息的 seq 判断已读状态

### 6.2 在线状态管理

```
┌─────────────────────────────────────────────────┐
│  Redis 数据结构                                   │
│                                                   │
│  Key: "online:{userId}"                          │
│  Value: JSON {platform: "web", connId: "xxx"}   │
│  TTL: 60s (心跳续期)                              │
│                                                   │
│  Key: "conn_map:{userId}"                        │
│  Value: Set {connId1, connId2, ...}             │
│  (支持同用户多端多连接)                             │
└─────────────────────────────────────────────────┘
```

- 客户端每 30s 发送心跳，服务端刷新 Redis TTL
- 连接断开时从 Set 移除 connId，Set 为空则删除 online key
- 第二阶段：多个 WS Gateway 实例各自维护本地连接映射，Redis 存全局在线状态

### 6.3 WebSocket 协议设计

消息格式统一为 JSON：

```json
// 客户端 -> 服务端
{
  "type": 1,           // 1-聊天消息 2-已读回执 3-心跳 4-拉取消息 5-输入状态
  "reqId": "uuid",     // 请求ID，用于响应匹配
  "data": {            // 根据 type 不同结构不同
    "conversationId": "single_userA_userB",
    "clientMsgId": "client-uuid",
    "contentType": 1,
    "content": "hello"
  }
}

// 服务端 -> 客户端
{
  "type": 101,         // 101-新消息通知 102-已读回执 103-心跳响应 104-消息拉取结果 105-输入状态通知
  "reqId": "uuid",     // 对应请求的reqId，推送消息无reqId
  "data": {
    "conversationId": "single_userA_userB",
    "seq": 42,
    "senderId": "userA",
    "content": "hello",
    "sendTime": 1714000000000
  }
}
```

### 6.4 会话 ID 规则

| 类型 | 格式 | 示例 |
|------|------|------|
| 单聊 | `single_{minUID}_{maxUID}` | `single_userA_userB`（按字典序排列，保证唯一性） |
| 群聊 | `group_{groupID}` | `group_g12345` |

### 6.5 Redis + MySQL 双写一致性

消息发送涉及 Redis 操作（Seq 分配、去重）和 MySQL 写入（消息持久化、会话更新），两者必须保持一致。

#### 失败场景分析

| 步骤 | 失败情况 | 影响 | 处理策略 |
|------|---------|------|---------|
| 1. Redis SETNX 去重 | 失败 | 无法去重，可能重复消息 | 允许，MySQL 层有 clientMsgId UNIQUE 约束兜底 |
| 2. Redis INCR 分配 Seq | 失败 | 无法分配序号 | 返回错误，客户端重试（新 clientMsgId） |
| 3. MySQL 写入消息 | 失败 | Seq 已分配但消息不存在（跳号） | **可接受**，Seq 允许间隙，客户端按 Seq 拉取时空号跳过即可 |
| 4. MySQL 更新会话 maxSeq | 失败 | 会话 maxSeq < 实际最大 Seq | 下次发消息时自动更新；客户端拉取时以 Redis Seq 为准 |

#### 核心设计原则

1. **以 Redis Seq 为权威源**：客户端消息拉取和已读回执都以 Redis 中的 `seq:conv:{conversationId}` 和 `readseq:{userId}:{conversationId}` 为准，MySQL 的 maxSeq 字段仅为辅助（加速会话列表查询）
2. **Seq 间隙可接受**：IM 系统中 Seq 不要求连续，只要求递增。跳号的原因可能是写入失败、消息被撤回/删除，客户端拉取时空号自动跳过
3. **去重双保险**：Redis SETNX 快速去重（第一道），MySQL clientMsgId UNIQUE 约束（第二道），即使 Redis 去重失败也不会产生重复消息

```
消息发送流程（含失败处理）：

  Redis SETNX(dedup:msg:{clientMsgId})
      │
      ├─ 失败 → 可能重复，继续（MySQL UNIQUE 兜底）
      │
      ▼
  Redis INCR(seq:conv:{conversationId})
      │
      ├─ 失败 → 返回错误，客户端用新 clientMsgId 重试
      │
      ▼
  MySQL INSERT messages
      │
      ├─ 失败(Duplicate) → 去重命中，返回已有消息的 Seq
      ├─ 失败(其他) → Seq 跳号，可接受，返回错误让客户端重试
      │
      ▼
  MySQL UPDATE conversations SET max_seq = {seq}
      │
      ├─ 失败 → 不影响消息正确性，下次发送时自动修正
      │
      ▼
  推送通知给接收方
```

### 6.6 客户端断连重连与消息补拉

#### 重连协议

WS 连接断开后，客户端必须能自动重连并补拉断连期间的消息，保证不丢失。

```
客户端                                  服务端
  │                                       │
  │── WS 连接断开 ───────────────────────│  (网络抖动/服务端重启/超时)
  │                                       │
  │── 1. 指数退避重连(1s,2s,4s,8s...)  │
  │── WS connect(token) ────────────────>│
  │                                       │
  │<── 2. 连接成功，返回 syncInfo ───────│
  │    {convMaxSeq, readSeq, ...}         │
  │                                       │
  │── 3. 对每个会话：                     │
  │    if localSeq < convMaxSeq:          │
  │      发送 type=4 拉取消息 ──────────>│
  │<── 4. 返回缺失消息 ─────────────────│
  │                                       │
  │── 5. 恢复正常收发 ──────────────────>│
```

#### WS 连接建立时的同步协议

客户端连接 WS 时，服务端返回各会话的最新状态，客户端对比本地状态决定是否补拉：

```json
// 服务端 -> 客户端（连接建立后立即推送）
{
  "type": 120,
  "reqId": "",
  "data": {
    "syncConversations": [
      {
        "conversationId": "single_alice_bob",
        "maxSeq": 45,
        "readSeq": 40
      },
      {
        "conversationId": "group_g123",
        "maxSeq": 102,
        "readSeq": 98
      }
    ]
  }
}
```

客户端逻辑：

```
对每个会话：
  if 本地 maxSeq < 服务端 maxSeq:
    发送 type=4 拉取 (localMaxSeq+1) 到 maxSeq 之间的消息
  if 本地 readSeq < 服务端 readSeq:
    更新本地已读状态（可能另一端已读了一些消息）
```

#### 重连参数

| 参数 | 值 | 说明 |
|------|-----|------|
| 初始重连间隔 | 1s | 首次重连等待 |
| 最大重连间隔 | 30s | 退避上限 |
| 退避因子 | 2x | 每次翻倍 |
| 最大重连次数 | 无限 | 持续尝试直到成功 |
| 重连抖动 | ±25% | 防止大量客户端同时重连（雷群效应） |

### 6.7 多端同步

同一用户可同时在多个设备/平台在线（Web + iOS + Android），各端状态必须同步。

#### 多端消息同步

```
alice 同时在线：Web + iOS

bob 发消息 "你好" (seq=43)
    │
    ├─→ WS 推送到 alice 的 Web 连接（type=101, seq=43）
    ├─→ WS 推送到 alice 的 iOS 连接（type=101, seq=43）
    │
    │  alice 在 iOS 上标记已读 (readSeq=43)
    │
    ├─→ 已读回执通知 bob（type=102, readSeq=43）
    └─→ 已读回执同步到 alice 的 Web 连接（type=102, readSeq=43）
         → Web 端也更新 UI 为"已读"
```

**实现要点**：

1. **Hub 的多连接管理**：`clients[userId]` 是一个 `map[*Client]struct{}`，同一用户的所有连接都在里面。推送时遍历该用户的所有连接
2. **已读回执自同步**：用户一端标记已读后，服务端不仅通知对方，也同步通知同用户的其他端（通过 Hub 遍历同用户的所有连接推送 type=102）
3. **输入状态自同步**：一端发送"正在输入"，也同步到同用户的其他端（避免另一端也显示"对方正在输入"的混淆）
4. **强制下线通知**：新设备登录踢掉同平台旧连接时，旧连接收到 type=109 强制下线通知，客户端清空本地状态跳转到登录页

#### 多端登录策略

| 策略 | 规则 | 本项目采用 |
|------|------|-----------|
| 单平台单实例 | 同一平台只允许一个连接，新连接踢旧 | **第一阶段采用** |
| 单平台多实例 | 同一平台允许最多 N 个连接 | 第二阶段扩展 |
| 全平台单实例 | 所有平台只能一个在线 | 不采用 |

### 6.8 WS Gateway 滚动更新与连接迁移

K8S 中每次部署新版本都会滚动更新 Pod，WS Gateway 的长连接必须优雅处理。

#### 滚动更新流程

```
kubectl rollout restart statefulset/gim-ws

  1. K8S 创建新 Pod (gim-ws-0-new)
  2. 新 Pod 就绪 → ReadinessProbe 通过
  3. K8S 向旧 Pod 发送 SIGTERM
  4. 旧 Pod 收到 SIGTERM：
     a. 停止接受新连接
     b. 向所有已连接客户端发送"服务器升级"通知（type=121）
     c. 等待客户端主动断开或超时（gracefulPeriod）
     d. 超时后强制关闭连接
  5. 旧 Pod 终止
  6. 客户端收到 type=121 或连接断开后，自动触发重连（§6.6）
  7. 客户端重连到新 Pod（通过 Service 负载均衡）
  8. 客户端补拉断连期间的消息
```

#### WS 协议扩展

```json
// 服务端 -> 客户端：服务器即将关闭通知
{
  "type": 121,
  "reqId": "",
  "data": {
    "reason": "server_rolling_update",
    "retryAfter": 1
  }
}
```

#### 关键配置

```yaml
# StatefulSet 配置
spec:
  podManagementPolicy: OrderedReady   # 逐个更新，不是同时
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      partition: 0                     # 从序号 0 开始更新
  template:
    spec:
      terminationGracePeriodSeconds: 60  # 给 60s 让客户端重连
      containers:
      - lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 5"]  # 等待 Service 摘除 Pod
```

### 6.9 消息发送原子性与失败处理

#### 第一阶段（同步写库）完整流程

```
SendMessage()
  │
  ├─ 1. Redis SETNX 去重
  │     失败 → 查 MySQL 是否已有，有则返回已有 Seq，无则继续
  │
  ├─ 2. 好友关系校验（单聊）
  │     失败 → 返回 ErrNotFriend
  │
  ├─ 3. Redis INCR 分配 Seq
  │     失败 → 返回 ErrInternal，客户端用新 clientMsgId 重试
  │
  ├─ 4. MySQL 事务：
  │     a. INSERT messages (seq, clientMsgId, ...)
  │        Duplicate → 返回已有消息的 Seq（去重命中）
  │        其他失败 → 事务回滚，返回错误，Seq 跳号可接受
  │     b. UPDATE conversations SET max_seq = seq
  │     c. UPSERT user_conversation_seq (初始化 readSeq=0)
  │
  ├─ 5. Redis SET readseq (如果用户首次在此会话)
  │     失败 → 不影响主流程，MySQL 有兜底
  │
  ├─ 6. WS 推送通知接收方
  │     失败 → 对方不在线或连接断开，上线后通过补拉获取
  │
  └─ 7. WS 确认发送方（type=101, reqId 匹配）
        失败 → 发送方超时后重连补拉
```

**关键保证**：步骤 4 的 MySQL 事务保证消息写入和会话更新的原子性。事务外部的失败（Redis/WS）不影响消息正确性——消息已持久化，客户端总是可以通过补拉获取。

#### 第二阶段（异步 Kafka）失败处理

```
SendMessage()
  │
  ├─ 1-3. 同第一阶段（去重 + 校验 + 分配 Seq）
  │
  ├─ 4. Kafka Produce（toMongo, toPush）
  │     失败 → 降级方案：直接同步写 MySQL（降级开关）
  │
  ├─ 5. 返回 Seq 给发送方
  │
  └─ 异步消费：
      MsgTransfer 写 MongoDB 失败 → Kafka 重试（at-least-once）
      消费超过 N 次仍失败 → 写死信队列 + 告警
```

---

## 七、进阶架构设计

> 以下内容标注 **[进阶]** 的为生产优化项，不影响核心功能运行，可在系统达到一定规模后逐步实施。

### 7.1 数据生命周期与消息归档 [进阶]

IM 消息只增不减，长期运行后单表可能上亿行，查询变慢。需要分层存储策略。

#### 数据分层

| 层级 | 存储 | 数据范围 | 访问频率 | 保留策略 |
|------|------|---------|---------|---------|
| 热数据 | Redis 缓存 | 最近 100 条/会话 | 高（每次打开聊天） | TTL 10 分钟 |
| 温数据 | MySQL/MongoDB | 近 90 天消息 | 中（翻历史记录） | 定期归档 |
| 冷数据 | 归档表/对象存储 | 90 天前消息 | 低（搜索/导出） | 永久保留 |

#### 归档方案

```sql
-- 归档表（与 messages 表结构相同）
CREATE TABLE messages_archive (
    -- 同 messages 表结构
    -- 不建高频索引，节省存储
    INDEX idx_conv_created (conversation_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**归档流程**（CronTask，每天凌晨执行）：

```
1. SELECT min(seq) FROM messages WHERE created_at < NOW() - INTERVAL 90 DAY
2. INSERT INTO messages_archive SELECT * FROM messages WHERE seq <= min_seq AND created_at < 90天前
3. UPDATE conversations SET min_seq = min_seq WHERE conversation_id IN (...)
4. DELETE FROM messages WHERE seq <= min_seq AND created_at < 90天前
5. 清理 Redis 缓存
```

> 拉取历史消息时，如果 `startSeq < minSeq`，则转查 `messages_archive` 表。

### 7.2 消息表分库分表 [进阶]

当单 messages 表日写入量超过千万行，单表性能不足时，按 conversationId 分表。

#### 分表策略

| 方案 | 规则 | 优点 | 缺点 |
|------|------|------|------|
| 按 conversationId 哈希 | `table = messages_{hash(convId) % 16}` | 分散均匀 | 跨会话查询困难 |
| 按 conversationId 范围 | 热门会话独立表，冷门会话共享表 | 灵活 | 需要动态路由 |
| 按 Seq 范围 + 会话 | 第二阶段 MongoDB 文档分片（已设计） | 天然支持 | 仅适用于 MongoDB |

**推荐路径**：第二阶段迁移到 MongoDB 后，其文档分片模型天然解决分表问题，无需额外分库分表。如果继续用 MySQL，采用按 conversationId 哈希分 16 表的方案。

#### 分表路由层

```go
// pkg/shard/shard.go
func MessageTable(conversationID string) string {
    h := fnv.New32a()
    h.Write([]byte(conversationID))
    return fmt.Sprintf("messages_%02d", h.Sum32()%16)
}
```

### 7.3 API 版本控制 [进阶]

当前 `/api/v1/` 前缀已预留版本空间，需要明确版本策略。

| 策略 | 规则 |
|------|------|
| URL 前缀 | `/api/v1/` → `/api/v2/`（不兼容变更时递增） |
| 兼容性 | v1 和 v2 至少共存 6 个月 |
| 客户端升级 | 客户端启动时检查最低版本号，低于要求提示升级 |
| 废弃通知 | HTTP Response Header: `Deprecation: true` + `Sunset: 2027-01-01` |
| WS 协议 | 连接建立时协商版本：`ws://host/ws?token=xxx&protoVersion=2` |

#### 版本协商（WS 连接建立时）

```json
// 服务端 -> 客户端（连接成功后）
{
  "type": 120,
  "reqId": "",
  "data": {
    "protoVersion": 2,
    "minSupportedVersion": 1,
    "features": ["typing", "ai_reply", "read_receipt"]
  }
}
```

### 7.4 安全纵深 [进阶]

在 JWT 鉴权和 CORS 基础上，补充以下安全层：

#### 7.4.1 接口安全

| 安全措施 | 应用场景 | 实现方式 |
|---------|---------|---------|
| 请求签名 | 防篡改（客户端伪造请求参数） | `HMAC-SHA256(timestamp + path + body, secret)` |
| 时间戳校验 | 防重放攻击 | 请求携带 timestamp，服务端校验 ±5 分钟 |
| Nonce 去重 | 防重放攻击 | Redis SETNX 存 nonce，TTL 10 分钟 |
| SQL 注入 | GORM 参数化查询已覆盖 | 禁止拼接 SQL，代码 Review 检查 |
| XSS | 消息内容可能含恶意脚本 | 服务端转义 HTML；客户端渲染时 sanitize |

#### 7.4.2 数据安全

| 安全措施 | 应用场景 | 实现方式 |
|---------|---------|---------|
| 手机号加密存储 | 隐私合规 | AES-GCM 加密后存库，查询时解密 |
| 密码哈希 | 已有 bcrypt | cost=10，足够 |
| TLS 传输加密 | 第二阶段 gRPC/HTTP | gRPC 启用 TLS，Ingress 启用 HTTPS |
| mTLS | 第二阶段服务间通信 | etcd 证书体系，RPC 双向验证 |
| 敏感操作二次验证 | 修改密码/注销账号 | 短信/邮件验证码 |

#### 7.4.3 WS 安全

| 安全措施 | 说明 |
|---------|------|
| Token 刷新 | WS 连接中 accessToken 过期后，客户端通过 type=12 发送 refreshToken，服务端验证后返回新 accessToken（type=122），无需断开重连 |
| 消息频率限制 | 每用户每秒最多 10 条消息，超限断连 |
| 消息大小限制 | maxMessageSize=4096 字节（已有） |
| 连接数限制 | 每用户最多 5 个连接（已有） |

WS Token 刷新协议：

```json
// 客户端 -> 服务端：刷新 Token
{
  "type": 12,
  "reqId": "req-refresh-001",
  "data": {
    "refreshToken": "eyJ..."
  }
}

// 服务端 -> 客户端：刷新成功
{
  "type": 122,
  "reqId": "req-refresh-001",
  "data": {
    "accessToken": "eyJ...(新)",
    "accessExpireAt": 1714070400
  }
}

// 刷新失败 → type=109 强制下线
```

### 7.5 限流体系 [进阶]

不同接口有不同的限流需求，不能一刀切：

| 接口类型 | 限流维度 | 窗口 | 阈值 | 理由 |
|---------|---------|------|------|------|
| 注册 | IP + 手机号 | 1h | 5次/IP, 3次/手机 | 防刷号 |
| 登录 | IP + UserID | 15min | 10次/IP, 5次/用户 | 防暴力破解 |
| 发消息(WS) | UserID + 会话 | 1s | 10条/用户, 5条/会话 | 防刷屏 |
| 拉历史(HTTP) | UserID | 1min | 30次/用户 | 防爬取 |
| 好友申请 | UserID | 1h | 20次/用户 | 防骚扰 |
| 文件上传 | UserID | 1day | 100次/用户 | 防滥用存储 |

```go
// 限流 Key 设计
ratelimit:register:ip:{ip}           → INCR, TTL=3600s
ratelimit:login:ip:{ip}              → INCR, TTL=900s
ratelimit:msg:user:{userId}          → INCR, TTL=1s
ratelimit:msg:conv:{conversationId}  → INCR, TTL=1s
ratelimit:history:user:{userId}      → INCR, TTL=60s
```

### 7.6 离线推送 [进阶]

第二阶段 Kafka `toOfflinePush` Topic 消费后，需要接入各厂商推送通道。

#### 推送架构

```
Kafka toOfflinePush
    │
    ▼
┌──────────────────────┐
│ OfflinePush Service  │
│                      │
│  1. 查用户注册平台    │
│  2. 查用户推送 Token  │
│  3. 聚合策略：        │
│     同一会话多条消息   │
│     合并为 1 条推送    │
│  4. 路由到厂商 SDK    │
└──────┬───────────────┘
       │
  ┌────┼──────────┬──────────┐
  ▼    ▼          ▼          ▼
APNs  FCM      华为Push    小米Push
(iOS) (Android) (Android)  (Android)
```

#### 推送聚合

5 秒窗口内同一用户的多条消息聚合为一条推送：

```
5秒内收到：
  bob: "你好"
  bob: "在吗"
  carol: "今天有空吗"

聚合推送：
  标题: "2个联系人发来3条新消息"
  内容: "bob: 你好 · carol: 今天有空吗"
```

#### 推送 Token 管理

```sql
CREATE TABLE push_tokens (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id    VARCHAR(64)  NOT NULL,
    platform   VARCHAR(16)  NOT NULL COMMENT 'ios/android/web',
    push_token VARCHAR(512) NOT NULL,
    bundle_id  VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '应用包名',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_platform (user_id, platform)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 7.7 微服务启动顺序与依赖管理 [进阶]

第二阶段微服务间有依赖，启动顺序错误会导致连接失败。

#### 服务依赖关系

```
启动顺序：
  1. etcd        （无依赖）
  2. MySQL/Redis/MongoDB/Kafka （无依赖，基础设施）
  3. rpc-auth    （依赖 etcd + MySQL + Redis）
  4. rpc-user    （依赖 etcd + MySQL + Redis）
  5. rpc-msg     （依赖 etcd + MySQL + Redis + Kafka）
  6. push        （依赖 etcd + Kafka）
  7. msgtransfer （依赖 etcd + Kafka + MongoDB）
  8. gim-ws      （依赖 etcd + rpc-msg + rpc-auth）
  9. gim-api     （依赖 etcd + rpc-auth + rpc-user + rpc-msg）
  10. gim-admin  （依赖 etcd + rpc-user + rpc-msg）
```

#### K8S 中的处理方式

**方案一：服务端重试（推荐）**。每个 RPC 服务启动时如果依赖的服务未就绪，不崩溃，而是重试连接（指数退避），直到依赖就绪。etcd 注册在所有依赖就绪后才执行。

```go
// 启动时连接依赖服务
func waitForDependency(serviceName, addr string, maxRetry int) {
    for i := 0; i < maxRetry; i++ {
        conn, err := grpc.Dial(addr, grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
        if err == nil {
            conn.Close()
            return
        }
        time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
    }
    log.Fatalf("Failed to connect to %s after %d retries", serviceName, maxRetry)
}
```

**方案二：Init Container（备选）**。在 Pod 启动前用 Init Container 检查依赖是否可用。

```yaml
initContainers:
- name: wait-for-etcd
  image: busybox
  command: ['sh', '-c', 'until nc -z etcd-headless 2379; do echo waiting; sleep 2; done;']
- name: wait-for-mysql
  image: busybox
  command: ['sh', '-c', 'until nc -z mysql 3306; do echo waiting; sleep 2; done;']
```

### 7.8 熔断与降级 [进阶]

当下游服务不可用时，需要有熔断和降级策略，避免级联故障。

#### 降级策略

| 故障场景 | 降级方案 | 影响 |
|---------|---------|------|
| Kafka 不可用 | 降级为直接同步写 MySQL | 吞吐量下降，但消息不丢 |
| MongoDB 不可用 | 降级为写 MySQL messages 表 | 回退到第一阶段存储方案 |
| etcd 不可用 | 使用本地缓存的服务地址列表 | 新服务实例无法发现，已有连接不受影响 |
| Redis 不可用 | Seq 降级为 MySQL 自增（加锁），在线状态降级为"全部在线" | 性能下降，功能降级 |
| Push 服务不可用 | 消息正常存储，上线后补拉 | 离线用户收不到推送通知 |

#### 熔断器设计

```go
// pkg/circuitbreaker/circuitbreaker.go
type State int

const (
    StateClosed    State = iota  // 正常，请求全部通过
    StateOpen                     // 熔断，请求全部拒绝
    StateHalfOpen                 // 半开，允许少量请求试探
)

type CircuitBreaker struct {
    name          string
    state         State
    failures      int
    threshold     int           // 连续失败阈值
    timeout       time.Duration // 熔断等待时间
    lastFailure   time.Time
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == StateOpen {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = StateHalfOpen  // 试探
        } else {
            return ErrCircuitOpen     // 快速失败
        }
    }

    err := fn()
    if err != nil {
        cb.failures++
        if cb.failures >= cb.threshold {
            cb.state = StateOpen
            cb.lastFailure = time.Now()
        }
        return err
    }

    cb.failures = 0
    cb.state = StateClosed
    return nil
}
```

### 7.9 分布式事务补偿 [进阶]

第二阶段微服务拆分后，好友同意等跨服务操作不再是本地事务。

#### 好友同意（跨服务版）— 最终一致性方案

```
AcceptFriendRequest()
  │
  ├─ 1. rpc-user: 更新申请状态(status=1) ─── 成功
  │
  ├─ 2. rpc-user: 创建双向好友关系 ─── 失败！
  │
  └─ 3. 补偿：回滚步骤 1（更新申请状态=status=0）
         或标记申请为"异常状态"，由定时任务重试
```

**推荐方案：本地消息表 + 定时补偿**

```sql
-- 本地消息表（每个服务各一张）
CREATE TABLE outbox (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    event_type  VARCHAR(64)  NOT NULL COMMENT '事件类型',
    payload     JSON         NOT NULL COMMENT '事件内容',
    status      TINYINT      NOT NULL DEFAULT 0 COMMENT '0-待发送 1-已发送 2-失败',
    retry_count INT          NOT NULL DEFAULT 0,
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**流程**：

1. 在同一个 MySQL 事务中：写入业务数据 + 写入 outbox 事件
2. 事务提交后，异步发送 outbox 事件到下游服务
3. 发送成功则标记 outbox status=1
4. 定时任务扫描 status=0 且 retry_count < 3 的记录，重新发送
5. 超过 3 次仍失败则标记 status=2，告警人工介入

### 7.10 配置热更新 [进阶]

部分配置需要运行时更新，不重启服务：

| 配置项 | 热更新方式 | 触发 |
|--------|----------|------|
| 敏感词库 | Redis 缓存 + 版本号，定时拉取 MySQL 刷新 | 管理后台修改词库 |
| 限流阈值 | Redis 存储阈值，服务启动加载，修改后 Pub 通知所有实例 | 运维调整 |
| 功能开关 | Redis Flag，代码中 `if flag.Enabled("ai_reply")` | 灰度发布/紧急关闭 |
| AI 模型切换 | Redis 存当前模型名，服务读取 Redis 而非配置文件 | 成本/质量调整 |

```go
// internal/config/hot.go
type HotConfig struct {
    rdb *redis.Client
    cache sync.Map  // 本地缓存
}

func (h *HotConfig) GetFeatureFlag(key string) bool {
    if v, ok := h.cache.Load(key); ok {
        return v.(bool)
    }
    val, _ := h.rdb.Get(context.Background(), "feature:"+key).Bool()
    h.cache.Store(key, val)
    return val
}

// 订阅 Redis Pub/Sub 刷新本地缓存
func (h *HotConfig) WatchUpdates() {
    sub := h.rdb.Subscribe(context.Background(), "config:reload")
    for range sub.Channel() {
        h.cache.Clear()  // 清空本地缓存，下次访问时从 Redis 重新读取
    }
}
```

### 7.11 Snowflake ID 时钟回拨防护 [进阶]

K8S 中 Pod 时钟不同步可能导致 Snowflake 生成的 ID 冲突或乱序。

```go
// pkg/snowflake/snowflake.go
type Node struct {
    node  *snowflake.Node
    lastMs int64  // 上次生成 ID 的时间戳
    mu    sync.Mutex
}

func (n *Node) Generate() snowflake.ID {
    n.mu.Lock()
    defer n.mu.Unlock()

    now := time.Now().UnixMilli()
    if now < n.lastMs {
        // 时钟回拨：等待追平
        waitMs := n.lastMs - now
        if waitMs > 500 {
            // 回拨超过 500ms，拒绝生成，报错
            return 0
        }
        time.Sleep(time.Duration(waitMs) * time.Millisecond)
        now = time.Now().UnixMilli()
    }
    n.lastMs = now
    return n.node.Generate()
}
```

> 生产环境应配置 NTP 时间同步（所有节点 `apt install ntp`），从根源避免时钟回拨。

### 7.12 测试策略 [进阶]

#### 测试层次

| 层次 | 工具 | 覆盖范围 | 运行时机 |
|------|------|---------|---------|
| 单元测试 | Go testing + testify | Service/Repository 逻辑 | 每次 commit |
| 集成测试 | Testcontainers + Docker | 真实 MySQL/Redis 交互 | PR 合并前 |
| API 测试 | httptest + WebSocket client | HTTP/WS 接口契约 | PR 合并前 |
| 压力测试 | k6 / ghz | 并发连接、消息吞吐 | 每个阶段末尾 |
| 混沌测试 | Chaos Mesh | 随机杀 Pod/网络延迟/Redis 宕机 | 第三阶段 |

#### 集成测试示例

```go
// internal/repository/user_test.go
func TestUserRepo_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Testcontainers 启动真实 MySQL
    ctx := context.Background()
    mysqlC, _ := testcontainers.GenericContainer(ctx,
        testcontainers.GenericContainerRequest{
            ContainerRequest: testcontainers.ContainerRequest{
                Image: "mysql:8.4",
                Env:   map[string]string{"MYSQL_ROOT_PASSWORD": "test", "MYSQL_DATABASE": "gim_test"},
                ExposedPorts: []string{"3306/tcp"},
            },
        })
    defer mysqlC.Terminate(ctx)

    // 用真实连接测试
    db := setupDB(t, mysqlC)
    repo := NewUserRepo(db)

    err := repo.Create(ctx, &model.User{UserID: "test", Nickname: "test"})
    assert.NoError(t, err)

    user, err := repo.GetByID(ctx, "test")
    assert.NoError(t, err)
    assert.Equal(t, "test", user.Nickname)
}
```

#### 压测基线

| 场景 | 目标指标 | 通过标准 |
|------|---------|---------|
| WS 并发连接 | 1 万连接/Gateway Pod | P99 延迟 < 100ms |
| 消息发送吞吐 | 5000 msg/s/集群 | P99 延迟 < 50ms |
| 历史消息拉取 | 100 QPS/用户 | P99 延迟 < 200ms |
| 群消息扇出 | 500 人群，1 msg/s | 全员收到 < 2s |
| 重连风暴 | 1000 并发重连 | 全部成功 < 30s |

---

## 八、API 设计

### 8.1 认证相关

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 用户注册 |
| POST | `/api/v1/auth/login` | 用户登录，返回 accessToken + refreshToken |
| POST | `/api/v1/auth/refresh` | 刷新 Token |
| POST | `/api/v1/auth/logout` | 退出登录 |

### 8.2 用户相关

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/user/profile` | 获取自己的资料 |
| PUT | `/api/v1/user/profile` | 修改自己的资料 |
| GET | `/api/v1/user/profile/{userId}` | 获取他人资料 |
| POST | `/api/v1/user/search` | 搜索用户(按手机号/昵称) |

### 8.3 好友相关

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/friend/request` | 发送好友申请 |
| GET | `/api/v1/friend/request/incoming` | 收到的好友申请列表 |
| POST | `/api/v1/friend/request/{id}/accept` | 同意好友申请 |
| POST | `/api/v1/friend/request/{id}/reject` | 拒绝好友申请 |
| DELETE | `/api/v1/friend/{userId}` | 删除好友 |
| GET | `/api/v1/friend/list` | 好友列表 |
| PUT | `/api/v1/friend/{userId}/remark` | 设置好友备注 |

### 8.4 消息相关

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/msg/history` | 拉取消息历史(?conversationId=&startSeq=&count=) |
| POST | `/api/v1/msg/read` | 标记已读(conversationId + readSeq) |
| POST | `/api/v1/msg/revoke` | 撤回消息 |

### 8.5 会话相关

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/conversation/list` | 会话列表(含未读计数) |
| PUT | `/api/v1/conversation/{id}/pin` | 置顶/取消置顶 |
| DELETE | `/api/v1/conversation/{id}` | 删除会话 |

### 8.6 群组相关（第二阶段）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/group/create` | 创建群组 |
| GET | `/api/v1/group/{groupId}` | 获取群信息 |
| POST | `/api/v1/group/{groupId}/invite` | 邀请入群 |
| POST | `/api/v1/group/{groupId}/join` | 申请入群 |
| POST | `/api/v1/group/{groupId}/leave` | 退群 |
| POST | `/api/v1/group/{groupId}/kick` | 踢人 |
| PUT | `/api/v1/group/{groupId}/info` | 修改群信息 |
| GET | `/api/v1/group/{groupId}/members` | 获取群成员列表 |
| POST | `/api/v1/group/{groupId}/mute` | 禁言成员 |

### 8.7 文件上传（第二阶段）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/file/upload` | 上传文件(返回URL) |

---

## 九、项目目录结构

```
gim/
├── cmd/
│   └── gim/                    # 第一阶段：单体入口
│       └── main.go
├── internal/                   # 私有代码（不可被外部导入）
│   ├── config/                 # 配置加载(Viper)
│   ├── handler/                # HTTP 处理器(Gin)
│   │   ├── auth.go
│   │   ├── user.go
│   │   ├── friend.go
│   │   ├── message.go
│   │   └── conversation.go
│   ├── ws/                     # WebSocket 网关
│   │   ├── server.go           # WS 服务器
│   │   ├── client.go           # 客户端连接管理
│   │   ├── hub.go              # 连接中心(路由消息)
│   │   └── protocol.go         # 协议定义
│   ├── service/                # 业务逻辑
│   │   ├── auth.go
│   │   ├── user.go
│   │   ├── friend.go
│   │   ├── message.go
│   │   └── conversation.go
│   ├── repository/             # 数据访问
│   │   ├── user.go
│   │   ├── friend.go
│   │   ├── message.go
│   │   └── conversation.go
│   ├── model/                  # 数据模型
│   │   ├── user.go
│   │   ├── friend.go
│   │   ├── message.go
│   │   └── conversation.go
│   └── middleware/             # Gin 中间件
│       ├── auth.go             # JWT 鉴权
│       ├── cors.go             # 跨域
│       └── ratelimit.go        # 限流
├── pkg/                        # 可复用公共包
│   ├── jwt/                    # JWT 工具
│   ├── snowflake/              # ID 生成
│   ├── resp/                   # 统一响应格式
│   └── errcode/                # 错误码定义
├── migrations/                 # SQL 迁移文件
├── configs/                    # 配置文件
│   └── config.yaml
├── deploy/                     # 部署相关
│   ├── docker/
│   │   └── Dockerfile
│   ├── docker-compose.yaml     # 开发环境
│   └── k8s/                    # 第三阶段 K8S 编排
│       └── helm/
├── scripts/                    # 脚本
│   └── init_db.sh
├── go.mod
├── go.sum
├── Makefile
├── CLAUDE.md
└── PLAN.md                     # 本文件
```

### 第二阶段目录扩展

```
cmd/
├── gim-api/                # API 网关入口
├── gim-ws/                 # WS Gateway 入口
├── gim-rpc-auth/           # Auth RPC 入口
├── gim-rpc-user/           # User RPC 入口
├── gim-rpc-msg/            # Msg RPC 入口
├── gim-push/               # Push 服务入口
├── gim-msgtransfer/        # MsgTransfer 入口
└── gim-admin/              # Admin API 入口(第三阶段)

internal/
├── rpc/                    # gRPC 生成代码和客户端
│   ├── auth/
│   ├── user/
│   └── msg/
├── kafka/                  # Kafka 生产者/消费者
└── mongo/                  # MongoDB 数据访问

api/                        # Protobuf 定义
├── auth/
│   └── auth.proto
├── user/
│   └── user.proto
└── msg/
    └── msg.proto
```

---

## 十、详细 TODO

> 详细实现方案见 [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md)，完整 API 文档见 [docs/API.md](docs/API.md)。

### 第一阶段：最小可运行单机版

#### Week 1-2：项目骨架与认证模块

- [ ] 初始化 Go Module，引入核心依赖(Gin, GORM, Redis, Viper, Zap, JWT)
- [ ] 配置管理：config.yaml + Viper 加载 → 实现: [IMPLEMENTATION.md §1.2-1.3](docs/IMPLEMENTATION.md)
- [ ] 统一响应格式与错误码体系 → 实现: [IMPLEMENTATION.md §9](docs/IMPLEMENTATION.md)
- [ ] MySQL 连接池 + GORM 初始化
- [ ] Redis 连接初始化
- [ ] 数据库迁移：users 表 → 实现: [IMPLEMENTATION.md §13](docs/IMPLEMENTATION.md)
- [ ] Snowflake ID 生成器
- [ ] JWT 工具包(RS256)：生成/验证 accessToken + refreshToken → 实现: [IMPLEMENTATION.md §2.2](docs/IMPLEMENTATION.md)
- [ ] API：POST /auth/register → 文档: [API.md §1.1](docs/API.md) | 实现: [IMPLEMENTATION.md §2.3-2.4](docs/IMPLEMENTATION.md)
- [ ] API：POST /auth/login → 文档: [API.md §1.2](docs/API.md)
- [ ] API：POST /auth/refresh → 文档: [API.md §1.3](docs/API.md)
- [ ] API：POST /auth/logout（Redis 黑名单机制） → 文档: [API.md §1.4](docs/API.md)
- [ ] Gin 中间件：JWT 鉴权、CORS、限流 → 实现: [IMPLEMENTATION.md §10](docs/IMPLEMENTATION.md)
- [ ] Makefile：build / run / migrate / lint 命令 → 实现: [IMPLEMENTATION.md §11](docs/IMPLEMENTATION.md)
- [ ] Docker Compose：MySQL + Redis 开发环境 → 实现: [IMPLEMENTATION.md §12](docs/IMPLEMENTATION.md)
- [ ] 单元测试：认证模块

#### Week 3-4：用户、好友、会话

- [ ] 数据库迁移：friends, friend_requests, conversations, user_conversation_seq 表
- [ ] API：用户资料 CRUD（GET/PUT /user/profile） → 文档: [API.md §2](docs/API.md) | 实现: [IMPLEMENTATION.md §3](docs/IMPLEMENTATION.md)
- [ ] API：搜索用户 → 文档: [API.md §2.4](docs/API.md)
- [ ] API：好友申请/同意/拒绝/删除/列表 → 文档: [API.md §3](docs/API.md) | 实现: [IMPLEMENTATION.md §4](docs/IMPLEMENTATION.md)
- [ ] 好友同意事务：更新申请+双向好友+双方会话 → 实现: [IMPLEMENTATION.md §4.2](docs/IMPLEMENTATION.md)
- [ ] API：会话列表（含未读计数 = maxSeq - readSeq） → 文档: [API.md §5](docs/API.md) | 实现: [IMPLEMENTATION.md §5](docs/IMPLEMENTATION.md)
- [ ] API：会话置顶/删除
- [ ] 好友关系校验（单聊发消息前检查）
- [ ] 单元测试：用户、好友、会话模块

#### Week 5-7：消息系统与 WebSocket

- [ ] 数据库迁移：messages 表
- [ ] WebSocket 服务器：基于 gorilla/websocket → 实现: [IMPLEMENTATION.md §7](docs/IMPLEMENTATION.md)
- [ ] WS Hub：用户-连接映射，消息路由 → 实现: [IMPLEMENTATION.md §7.1](docs/IMPLEMENTATION.md)
- [ ] WS Client：连接注册/注销/心跳/消息分发 → 实现: [IMPLEMENTATION.md §7.2-7.3](docs/IMPLEMENTATION.md)
- [ ] WS 协议实现：消息发送/心跳/拉取/已读回执/输入状态 → 文档: [API.md §8](docs/API.md)
- [ ] 消息发送流程（核心） → 实现: [IMPLEMENTATION.md §6.3](docs/IMPLEMENTATION.md)
  - [ ] 接收 WS 消息 -> 分配 Seq(Redis INCR) -> 写 MySQL -> 推送对方
  - [ ] clientMsgId 去重（Redis SETNX）
- [ ] 消息拉取：根据 Seq 范围查询 → 实现: [IMPLEMENTATION.md §6.4](docs/IMPLEMENTATION.md)
- [ ] 已读回执：更新 readSeq，通知发送方 → 实现: [IMPLEMENTATION.md §6.5](docs/IMPLEMENTATION.md)
- [ ] 离线消息：用户上线后主动拉取
- [ ] 消息撤回（2 分钟内） → 文档: [API.md §4.3](docs/API.md)
- [ ] Redis 在线状态管理 → 实现: [IMPLEMENTATION.md §8](docs/IMPLEMENTATION.md)
- [ ] 集成测试：消息收发完整流程
- [ ] 压测脚本：k6 发送 WS 消息测试并发

#### Week 8：收尾与文档

- [ ] 输入状态通知（"对方正在输入..."）
- [ ] 消息内容类型扩展：图片/文件（先上传到服务器本地目录）
- [ ] Dockerfile 多阶段构建 → 实现: [IMPLEMENTATION.md §12.1](docs/IMPLEMENTATION.md)
- [ ] docker-compose.yaml 完整编排 → 实现: [IMPLEMENTATION.md §12.2](docs/IMPLEMENTATION.md)
- [ ] 代码 Review 与重构

---

### 第二阶段：分布式架构与核心功能

#### Week 1-3：微服务拆分

- [ ] Protobuf 定义：auth.proto, user.proto, friend.proto, msg.proto → 实现: [IMPLEMENTATION.md §14.1](docs/IMPLEMENTATION.md)
- [ ] 代码生成：protoc + protoc-gen-go + protoc-gen-go-grpc
- [ ] etcd 服务注册与发现封装
- [ ] 拆分 gim-api：HTTP 网关，转发到各 RPC 服务
- [ ] 拆分 gim-rpc-auth：认证服务
- [ ] 拆分 gim-rpc-user：用户+好友服务
- [ ] 拆分 gim-rpc-msg：消息服务
- [ ] 修改 gim-ws：WS+gRPC 双协议 → 实现: [IMPLEMENTATION.md §14.4](docs/IMPLEMENTATION.md)
- [ ] gRPC 拦截器：日志/鉴权/限流/链路ID
- [ ] 确保功能与第一阶段一致

#### Week 4-6：Kafka + MongoDB + 消息流改造

- [ ] Kafka 集群部署（Docker Compose）
- [ ] Topic 设计：toMongo, toPush, toOfflinePush → 实现: [IMPLEMENTATION.md §14.2](docs/IMPLEMENTATION.md)
- [ ] MsgTransfer 服务：消费 Kafka -> 批量写 MongoDB → 实现: [IMPLEMENTATION.md §14.3](docs/IMPLEMENTATION.md)
- [ ] MongoDB 消息存储层（文档分片模型）
- [ ] 改造消息发送流程：rpc-msg -> Kafka -> MsgTransfer -> MongoDB
- [ ] Push 服务：消费 Kafka toPush topic -> 区分在线/离线
- [ ] 在线推送：Push -> gRPC -> WS Gateway -> 客户端
- [ ] Redis 改造：Seq 缓存、在线状态、本地缓存失效通知
- [ ] 消息查询迁移到 MongoDB（MySQL messages 表逐步停写）
- [ ] S3 兼容存储部署（MinIO/OSS） + 文件上传服务 → 文档: [API.md §7](docs/API.md)

#### Week 7-10：群聊功能

- [ ] 数据库迁移：groups, group_members, group_requests 表
- [ ] 群组 CRUD：创建群、修改群信息、解散群 → 文档: [API.md §6](docs/API.md)
- [ ] 群成员管理：邀请入群、申请入群、踢人、退群
- [ ] 群权限：群主/管理员/普通成员角色，权限校验
- [ ] 群聊消息：消息扇出 → 实现: [IMPLEMENTATION.md §14.5](docs/IMPLEMENTATION.md)
- [ ] 群消息推送优化：Push 服务批量推送给群成员
- [ ] 群禁言：全员禁言 + 单人禁言 → 文档: [API.md §6.9](docs/API.md)
- [ ] 群公告
- [ ] 会话列表支持群聊会话
- [ ] 集成测试：群聊完整流程
- [ ] 压测：群消息扇出性能

---

### 第三阶段：运营支撑与生产化

#### Week 1-3：后台管理系统

- [ ] Admin API（独立服务 gim-admin）
  - [ ] 管理员认证（独立于用户系统）
  - [ ] 用户管理：列表/搜索/禁用/启用
  - [ ] 群组管理：列表/搜索/解散/设置公告
  - [ ] 消息管理：消息查询/敏感词过滤
  - [ ] 统计数据：DAU/消息量/注册量
- [ ] Admin Web UI（React + Ant Design Pro）
  - [ ] 登录页
  - [ ] 仪表盘（统计图表）
  - [ ] 用户管理页
  - [ ] 群组管理页
  - [ ] 消息查询页
  - [ ] 系统配置页

#### Week 4-5：监控与可观测性

- [ ] Prometheus 指标埋点 → 实现: [IMPLEMENTATION.md §15.1](docs/IMPLEMENTATION.md)
  - [ ] HTTP 请求 QPS/延迟/错误率
  - [ ] WS 在线连接数
  - [ ] Kafka 消息积压量
  - [ ] gRPC 调用延迟
  - [ ] MySQL/Redis/MongoDB 连接池状态
- [ ] Grafana 仪表盘
- [ ] OpenTelemetry 链路追踪 → 实现: [IMPLEMENTATION.md §15.2](docs/IMPLEMENTATION.md)
  - [ ] HTTP -> gRPC -> Kafka -> MongoDB 全链路 Trace
  - [ ] Jaeger 部署与查询
- [ ] 日志聚合（Loki + Promtail）
- [ ] 告警规则（Grafana Alerting / Prometheus AlertManager）

#### Week 6-8：K8S 生产部署与优化

> 完整 K8S 集群搭建和服务部署步骤见 [docs/K8S_DEPLOY.md](docs/K8S_DEPLOY.md)

- [ ] Helm Chart → 结构: [IMPLEMENTATION.md §15.3](docs/IMPLEMENTATION.md)
  - [ ] 每个微服务一个 Deployment 模板
  - [ ] WS Gateway 使用 StatefulSet（稳定网络标识）
  - [ ] ConfigMap / Secret 管理
  - [ ] HPA 自动伸缩 → 配置: [IMPLEMENTATION.md §15.4](docs/IMPLEMENTATION.md)
- [ ] Ingress 配置（HTTP + WS 路径分流）
- [ ] 持久化存储：MySQL/MongoDB/Kafka StatefulSet + PVC
- [ ] 优雅关闭：WS 连接迁移、Kafka 消费者优雅退出
- [ ] 压力测试与调优（k6 + pprof）
- [ ] 安全加固（TLS / NetworkPolicy / SecurityContext）
- [ ] 灰度发布与滚动更新策略

---

### 第四阶段：AI Agent 集成

> 详细方案、架构设计、核心代码见 [docs/AI_AGENT.md](docs/AI_AGENT.md)

#### Week 1-2：智能回复助手

- [ ] 实现 AIProvider 统一接口（支持 Deepseek/Claude/本地模型切换）
- [ ] 集成 AI SDK：anthropic-sdk-go / openai-go（Deepseek 兼容 OpenAI 协议）
- [ ] 实现 AI Service 基础框架（配置、多 Provider 客户端初始化）
- [ ] 实现 ReplyAgent：构建上下文 + 调用 AI API + 流式输出
- [ ] WS 协议扩展：type=10（AI 请求）、type=110（AI 流式回复） → 定义: [AI_AGENT.md §4.3](docs/AI_AGENT.md)
- [ ] 指令解析：`@gim-bot 翻译/总结/润色`
- [ ] 前端展示 AI 流式回复（打字机效果）
- [ ] 上下文窗口管理（最近 N 条消息、Token 计数与截断）
- [ ] 测试：手动 @gim-bot 验证回复质量

#### Week 3-5：内容审核 Agent

- [ ] 数据库迁移：violations、sensitive_words 表
- [ ] 敏感词库管理（CRUD API + 批量导入）
- [ ] 规则引擎：关键词匹配（精确 + 模糊，第一层快速过滤）
- [ ] ModerationAgent：AI Tool Use 定义（通过 AIProvider 接口） → 定义: [AI_AGENT.md §5.3](docs/AI_AGENT.md)
  - [ ] mark_violation（记录违规，不撤回）
  - [ ] revoke_message（撤回消息+通知用户）
  - [ ] warn_user（警告用户）
  - [ ] pass（合规通过）
- [ ] Tool Use 多轮循环：LLM 调工具 → 看结果 → 继续思考
- [ ] Kafka 集成：新增 toModeration Topic，Agent 异步消费
- [ ] 审核结果持久化与查询 API
- [ ] 管理后台审核日志页面
- [ ] 测试：构造违规消息验证审核链路

#### Week 5-7：管理后台智能助手

- [ ] 数据库迁移：ai_conversations 表
- [ ] AdminAssistant 核心逻辑：Chat → Tool Use Loop
- [ ] 管理 Tool 定义 → 定义: [AI_AGENT.md §6.4](docs/AI_AGENT.md)
  - [ ] query_stats（查询统计数据）
  - [ ] ban_user（封禁用户）
  - [ ] search_knowledge_base（RAG 检索）
- [ ] RAG 管道搭建 → 详解: [AI_AGENT.md §6.5](docs/AI_AGENT.md)
  - [ ] Milvus / pgvector 部署（Docker Compose）
  - [ ] Embedding 封装（OpenAI API 或本地模型）
  - [ ] 文档切片与入库脚本（项目文档、运维手册）
  - [ ] 相似度检索实现
- [ ] 对话记忆管理（多轮上下文存取与截断）
- [ ] Admin Web UI：对话界面组件
- [ ] 测试：自然语言查询数据、执行操作、检索文档

#### Week 7-10：群聊多 Agent 协作

- [ ] 数据库迁移：group_todos 表
- [ ] AgentRouter：意图解析 → Agent 选择 → 并发调度 → 结果合并
- [ ] SummaryAgent：群消息摘要生成
- [ ] TodoAgent：待办提取与持久化
- [ ] RemindAgent：定时提醒（Cron + WS 推送）
- [ ] QAAgent：群内问答（RAG + 通用知识）
- [ ] WS 协议扩展：type=11（群 AI 请求）、type=111（群 AI 消息）
- [ ] AI 服务独立部署（gim-ai，gRPC 接入）
- [ ] Admin Web UI：待办管理页、AI 配置页、审核日志页
- [ ] 成本控制：规则引擎减少 AI 调用、路由用 Haiku、缓存 RAG 结果、每用户限流
- [ ] 压测：AI 请求对消息管道的影响
- [ ] 安全加固：Prompt 注入防护、AI 幻觉兜底、API Key 安全

---

## 十一、关键技术决策说明

### 为什么第一阶段用 MySQL 而非 MongoDB 存消息？

- 你熟悉关系型数据库，MySQL 的运维经验可以复用
- 单机阶段数据量小，MySQL 完全够用，避免同时学习新数据库
- GORM 对 MySQL 支持最成熟，开发效率高
- 第二阶段迁移到 MongoDB 时，可以对比两种方案的差异，学习效果更好

### 为什么用 Redis INCR 生成 Seq 而不是 MySQL 自增？

- Seq 必须是会话维度递增，MySQL 自增是表维度
- Redis INCR 原子操作，无锁竞争，高并发下性能远优于 MySQL
- Redis INCR 每秒可处理百万次操作，满足 IM 场景需求

### 为什么第二阶段引入 Kafka？

- 消息写入量随用户增长线性增加，同步写库会成为瓶颈
- Kafka 解耦后，消息发送立即返回，写入由 MsgTransfer 异步批量完成
- 推送与存储分离，各自可独立扩展
- 学习 Kafka 是后端高并发系统的核心技能

### 为什么 WS Gateway 设计为 WS + gRPC 双协议？

- 这是 OpenIM 的核心设计，也是生产环境的最佳实践
- 推送服务需要知道哪个 Gateway 实例持有目标用户的连接
- 通过 gRPC 调用 Gateway，推送服务无需关心连接分布
- Gateway 本地维护 UserMap，查找连接是 O(1)

### 为什么群聊消息扇出用 Push 服务而非 Gateway 直接推？

- 群消息需要推送给 N 个成员，Gateway 不应承担扇出逻辑
- Push 服务集中处理在线/离线判断，批量推送，性能可控
- Gateway 职责单一：管理连接 + 收发，不做业务逻辑

### 为什么 AI Agent 作为第四阶段而非一开始就集成？

- 前三阶段搭建 IM 基础设施（消息管道、推送、Kafka、K8S），AI Agent 复用这些设施而非从零搭建
- 同时学习 IM 架构 + AI Agent 认知负荷太高，先打好后端基础
- Agent 接入 IM 的方式是"消息消费者"，必须先有成熟的消息管道
- 第四阶段通过 AIProvider 接口统一多后端（Deepseek/Claude/本地模型），开发期用本地模型零成本验证，生产环境灵活切换

### 为什么支持多 AI Provider 而非绑定单一服务？

- Deepseek API 兼容 OpenAI 协议，性价比高，中文理解能力强
- 本地部署模型（Ollama/vLLM）在开发阶段零成本，离线可用，数据不出本机
- Claude API 在复杂推理和 Tool Use 方面表现最佳，适合高要求场景
- 统一的 AIProvider 接口允许按场景选择最优模型（路由用便宜模型，深度推理用强模型）

### 为什么内容审核用"规则引擎 + AI"混合而非纯 AI？

- 规则引擎（关键词匹配）是确定性的，零延迟，零成本，可拦截 80%+ 常见违规
- AI 审核处理规则无法覆盖的模糊场景（隐喻、变体写法），但延迟高、有成本
- 混合架构兼顾速度、成本和覆盖率，是生产环境的标准实践

---

## 十二、学习资源推荐

| 技术点 | 资源 |
|--------|------|
| Go 并发 | 《Concurrency in Go》+ Go 官方 blog 并发系列 |
| gRPC | 官方文档 grpc.io + protoc 实战 |
| Kafka | 《Kafka: The Definitive Guide》+ Confluent 文档 |
| K8S | 官方文档 + 《Kubernetes in Action》 |
| MongoDB | 官方文档 + 《MongoDB: The Definitive Guide》 |
| IM 系统 | OpenIM 源码 (github.com/openimsdk/open-im-server) |
| WebSocket | gorilla/websocket 源码 + RFC 6455 |

### AI Agent 学习资源

| 技术点 | 资源 |
|--------|------|
| Deepseek API | [Deepseek API 文档](https://platform.deepseek.com/docs)（兼容 OpenAI 协议） |
| Claude API | [Anthropic 官方文档](https://docs.anthropic.com/) + [Go SDK](https://github.com/anthropics/anthropic-sdk-go) |
| 本地部署 | [Ollama](https://ollama.com/) / [vLLM](https://docs.vllm.ai/)（GPU 推理加速） |
| OpenAI Go SDK | [openai-go](https://github.com/openai/openai-go)（Deepseek 兼容） |
| Tool Use | [Anthropic Tool Use 文档](https://docs.anthropic.com/en/docs/build-with-claude/tool-use) |
| RAG | [Anthropic RAG 教程](https://docs.anthropic.com/en/docs/build-with-claude/retrieval-augmented-generation) |
| Prompt Engineering | [Anthropic Prompt Engineering 指南](https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering/overview) |
| Agent 模式 | [Anthropic Agentic Systems](https://docs.anthropic.com/en/docs/build-with-claude/agentic-systems) |
| 向量数据库 | [Milvus 文档](https://milvus.io/docs) 或 [pgvector](https://github.com/pgvector/pgvector) |
| Multi-Agent | LangGraph / CrewAI 论文（理解概念，用 Go 自己实现） |

### 入门资源（零基础）

| 技术点 | 资源 |
|--------|------|
| Go 语言 | [Go 官方教程](https://go.dev/tour/) + [Go by Example](https://gobyexample.com/) |
| HTTP/API | [MDN HTTP 教程](https://developer.mozilla.org/zh-CN/docs/Web/HTTP) |
| MySQL | [MySQL 入门教程](https://dev.mysql.com/doc/refman/8.4/en/tutorial.html) |
| Redis | [Redis 入门](https://redis.io/docs/getting-started/) |
| Docker | [Docker 入门教程](https://docs.docker.com/get-started/) |
| Gin 框架 | [Gin 官方文档](https://gin-gonic.com/docs/quickstart/) |

> 零基础环境搭建与概念入门见 [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md)
