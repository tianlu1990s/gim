# GIM 第四阶段：AI Agent 集成计划

> 通过在 IM 系统中嵌入 AI Agent，实战学习 Agent 开发的核心技术栈：LLM API 调用、RAG、Tool Use、Multi-Agent 协作。

---

## 一、为什么 IM 项目是学习 AI Agent 的理想切入点？

| IM 特性 | 对应 Agent 能力 |
|---------|----------------|
| 消息流转管道天然存在 | Agent 可以作为"消息消费者"接入，零侵入 |
| 多种消息类型（文本/图片/文件） | 练习多模态处理 |
| 群聊场景 = 多角色交互 | Multi-Agent 协作的自然场景 |
| 后台管理系统 | Agent 监控面板、运营助手 |
| 用户行为数据 | RAG 的天然语料来源 |

**核心思路：** IM 系统已有消息管道（WS → Service → 推送），AI Agent 只是这条管道上的一个"智能消费者"——收到消息后，调 LLM 分析处理，再产生新的消息或操作。

---

## 二、第四阶段规划总览（预计 8-10 周）

**目标：** 在 gim 中集成 4 个 AI Agent，覆盖 Agent 开发的核心技术点。

| Agent | 核心学习点 | 难度 |
|-------|-----------|------|
| 1. 智能回复助手 | LLM API 调用、Prompt Engineering、流式输出 | ★★☆ |
| 2. 内容审核 Agent | Tool Use（函数调用）、规则+AI 混合审核 | ★★★ |
| 3. 管理后台智能助手 | RAG（知识库检索增强）、Multi-turn 对话 | ★★★ |
| 4. 群聊多 Agent 协作 | Multi-Agent、任务拆分与合并 | ★★★★ |

---

## 三、前置知识

### 3.1 什么是 AI Agent？

普通 LLM 调用：用户问 → LLM 答（只能生成文本）。
AI Agent：用户问 → LLM 思考 → 调用工具（查数据库/发消息/调 API）→ 根据工具结果继续思考 → 返回最终答案。

```
普通 LLM：
  用户："帮我查一下 alice 的注册时间"
  LLM："我无法访问数据库，无法查询。"（只会说话）

AI Agent：
  用户："帮我查一下 alice 的注册时间"
  LLM 思考：需要查数据库 → 调用 tool: query_user("alice")
  工具返回：{createdAt: "2026-04-26"}
  LLM：根据结果生成回答 → "alice 是在 2026年4月26日 注册的。"
```

**Agent = LLM + 工具 + 记忆 + 规划**

### 3.2 核心概念

| 概念 | 解释 | 本项目应用 |
|------|------|-----------|
| Tool Use / Function Calling | LLM 决定调用哪个函数、传什么参数 | 内容审核 Agent 调用"屏蔽消息""封禁用户"等工具 |
| RAG (Retrieval Augmented Generation) | 先从知识库检索相关内容，再让 LLM 基于检索结果回答 | 管理助手从文档/日志中检索信息 |
| Multi-Agent | 多个 Agent 各司其职，协作完成复杂任务 | 群聊中"总结 Agent"+"待办 Agent"+"提醒 Agent"分工 |
| Streaming | LLM 逐字输出，不等全部生成完才返回 | 聊天中的打字机效果 |
| Prompt Engineering | 设计 Prompt 让 LLM 输出符合预期格式 | 系统提示词、审核规则描述 |

### 3.3 技术选型

| 组件 | 选择 | 原因 |
|------|------|------|
| LLM API | Deepseek API / Claude API / 本地部署（Ollama/vLLM） | 多 Provider 可切换，开发用本地模型，生产按需选 API |
| AI 接口 | AIProvider 统一接口 | 解耦具体 LLM 实现，一行配置切换后端 |
| Go SDK | `anthropic-sdk-go` / `openai-go`（Deepseek 兼容 OpenAI 协议） | 按 Provider 选择 SDK，OpenAI 兼容协议覆盖多后端 |
| 向量数据库 | Milvus (自部署) 或 pgvector | RAG 场景存储文档向量 |
| Embedding | OpenAI text-embedding-3-small 或本地 Embedding | 文档向量化 |
| 消息队列 | 复用现有 Kafka | Agent 消息消费与业务消息共用管道 |

> **AIProvider 接口设计**：系统通过统一的 `AIProvider` 接口支持多后端切换。开发阶段默认使用本地部署模型（Ollama），生产环境可按需切换 Deepseek API 或 Claude API。Deepseek API 兼容 OpenAI 协议，迁移成本低。详见 PLAN.md 第四阶段技术栈。

---

## 四、Agent 1：智能回复助手（Week 1-2）

### 4.1 功能描述

- 用户在聊天中 @gim-bot，bot 基于上下文生成智能回复建议
- 支持流式输出（逐字显示，类 ChatGPT 体验）
- 支持多种指令：`@gim-bot 翻译` / `@gim-bot 总结` / `@gim-bot 润色`

### 4.2 架构设计

```
用户消息 "请帮我翻译一下这段话"
    │
    ▼
┌──────────────┐
│  WS Gateway   │  检测到 @gim-bot，路由到 AI Service
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  AI Service   │  1. 构建上下文（最近 N 条消息 + 系统提示词）
│               │  2. 调用 AIProvider（stream=true，Deepseek/Claude/本地）
│               │  3. 逐 chunk 通过 WS 推送给用户
└──────┬───────┘
       │
       ▼
  AIProvider 统一接口 → Deepseek/Claude/本地模型（流式返回）
       │
       ▼
┌──────────────┐
│  WS 推送      │  type=110 (AI 流式消息)
│  给客户端     │  {"chunk": "你", "isFinal": false}
│               │  {"chunk": "好", "isFinal": false}
│               │  {"chunk": "", "isFinal": true}
└──────────────┘
```

### 4.3 WS 协议扩展

```json
// 客户端 -> 服务端：请求 AI 回复
{
  "type": 10,
  "reqId": "req-ai-001",
  "data": {
    "conversationId": "single_alice_bob",
    "instruction": "翻译",       // 可选指令
    "triggerMsgId": "client-uuid-001"  // 触发的消息
  }
}

// 服务端 -> 客户端：AI 流式回复
{
  "type": 110,
  "reqId": "req-ai-001",
  "data": {
    "conversationId": "single_alice_bob",
    "chunk": "这是翻译结果",
    "isFinal": false,
    "messageId": "ai-msg-001"    // 所有 chunk 共享同一个 messageId
  }
}

// 最后一个 chunk
{
  "type": 110,
  "reqId": "req-ai-001",
  "data": {
    "conversationId": "single_alice_bob",
    "chunk": "",
    "isFinal": true,
    "messageId": "ai-msg-001",
    "totalContent": "这是翻译结果的完整内容"
  }
}
```

### 4.4 核心代码

```go
// internal/service/ai/reply.go
package ai

import (
    "context"
    "fmt"

    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
)

type ReplyAgent struct {
    client    *anthropic.Client
    msgRepo   repository.MessageRepo
    hub       *ws.Hub
}

func NewReplyAgent(apiKey string, msgRepo repository.MessageRepo, hub *ws.Hub) *ReplyAgent {
    client := anthropic.NewAnthropicClient(option.WithAPIKey(apiKey))
    return &ReplyAgent{client: client, msgRepo: msgRepo, hub: hub}
}

func (a *ReplyAgent) HandleReply(ctx context.Context, userID, convID, instruction string, reqID string) {
    // 1. 拉取最近 20 条消息作为上下文
    recentMsgs, _ := a.msgRepo.GetRecentN(ctx, convID, 20)

    // 2. 构建消息列表
    messages := a.buildMessages(recentMsgs, instruction)

    // 3. 流式调用 Claude
    stream := a.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(anthropic.ModelClaudeSonnet4_6),
        MaxTokens: anthropic.F(int64(1024)),
        System:    anthropic.F([]anthropic.TextBlockParam{{Text: a.buildSystemPrompt(instruction)}}),
        Messages:  anthropic.F(messages),
    })

    msgID := snowflake.Generate().String()
    var fullContent string

    for stream.Next() {
        event := stream.Current()
        // 处理内容块增量事件
        if event.Type == "content_block_delta" {
            if delta, ok := event.Delta.(anthropic.TextDelta); ok {
                fullContent += delta.Text
                a.hub.PushToUser(userID, &ws.WSMessage{
                    Type:  110,
                    ReqID: reqID,
                    Data: map[string]interface{}{
                        "conversationId": convID,
                        "chunk":          delta.Text,
                        "isFinal":        false,
                        "messageId":      msgID,
                    },
                })
            }
        }
    }

    // 发送结束标记
    a.hub.PushToUser(userID, &ws.WSMessage{
        Type:  110,
        ReqID: reqID,
        Data: map[string]interface{}{
            "conversationId": convID,
            "chunk":          "",
            "isFinal":        true,
            "messageId":      msgID,
            "totalContent":   fullContent,
        },
    })
}

func (a *ReplyAgent) buildSystemPrompt(instruction string) string {
    base := `你是 gim-bot，一个集成在即时通讯系统中的智能助手。
你可以帮助用户翻译、总结、润色文本，或回答问题。
回复时使用与用户相同的语言。保持简洁友好。`

    if instruction != "" {
        base += fmt.Sprintf("\n用户要求的操作是：%s。请据此处理。", instruction)
    }
    return base
}

func (a *ReplyAgent) buildMessages(recentMsgs []*model.Message, instruction string) []anthropic.MessageParam {
    var params []anthropic.MessageParam
    for _, msg := range recentMsgs {
        role := anthropic.MessageParamRoleUser
        if msg.SenderID == "gim-bot" {
            role = anthropic.MessageParamRoleAssistant
        }
        params = append(params, anthropic.MessageParam{
            Role: anthropic.F(role),
            Content: anthropic.F([]anthropic.ContentBlockParam{
                anthropic.NewTextBlock(msg.Content),
            }),
        })
    }
    // 追加当前指令
    if instruction != "" {
        params = append(params, anthropic.MessageParam{
            Role: anthropic.F(anthropic.MessageParamRoleUser),
            Content: anthropic.F([]anthropic.ContentBlockParam{
                anthropic.NewTextBlock(instruction),
            }),
        })
    }
    return params
}
```

### 4.5 学习要点

- **Prompt Engineering**：system prompt 如何影响输出风格和质量
- **流式输出**：SSE/Streaming API 的工作原理，逐 chunk 推送
- **上下文窗口管理**：最近 N 条消息的选择策略，Token 计数与截断
- **错误处理**：API 限流、超时、内容安全过滤的兜底方案

---

## 五、Agent 2：内容审核 Agent（Week 3-5）

### 5.1 功能描述

- 消息发送后经过 Agent 审核，判定是否违规
- 混合审核：规则引擎（关键词过滤）+ AI 判定
- AI 可调用工具：标记违规、撤回消息、警告用户、封禁用户
- 审核结果写入 Kafka，与消息管道集成

### 5.2 架构设计

```
消息发送
    │
    ├──→ 正常流程：分配Seq → 写库 → 推送（不阻塞）
    │
    └──→ 审核流程（异步，Kafka）：
             │
             ▼
        ┌─────────────────┐
        │  Moderation      │  1. 规则引擎（关键词/正则）快速过滤
        │  Agent            │  2. 规则未命中 → 调用 AIProvider + Tool Use
        │                   │  3. LLM 决定是否违规，调用对应工具
        └──────┬───────────┘
               │
               ▼
        ┌─────────────────┐
        │  AIProvider      │  Tool Use（函数调用）
        │  + Tools 定义    │
        └──────┬───────────┘
               │
        ┌──────┴───────────────────┐
        │                          │
        ▼                          ▼
  tool: mark_violation     tool: revoke_message
  (记录违规，不撤回)        (撤回消息+通知用户)
        │                          │
        ▼                          ▼
  MySQL: violations 表     WS 推送: 消息被撤回通知
```

### 5.3 Tool 定义

```go
// internal/service/ai/moderation.go
package ai

var moderationTools = []anthropic.ToolParam{
    {
        Name:        anthropic.F("mark_violation"),
        Description: anthropic.F("标记消息为违规但不撤回，仅记录。用于轻度违规。"),
        InputSchema: anthropic.F(anthropic.ToolInputSchema{
            Type: anthropic.F("object"),
            Properties: anthropic.F(map[string]interface{}{
                "conversationId": map[string]interface{}{"type": "string", "description": "会话ID"},
                "clientMsgId":    map[string]interface{}{"type": "string", "description": "消息ID"},
                "category":       map[string]interface{}{"type": "string", "enum": []string{"spam", "abuse", "politics", "porn", "ad", "other"}},
                "severity":       map[string]interface{}{"type": "string", "enum": []string{"low", "medium", "high"}},
                "reason":         map[string]interface{}{"type": "string", "description": "判定理由"},
            }),
            Required: anthropic.F([]string{"conversationId", "clientMsgId", "category", "severity"}),
        }),
    },
    {
        Name:        anthropic.F("revoke_message"),
        Description: anthropic.F("撤回违规消息并通知发送者。用于严重违规。"),
        InputSchema: anthropic.F(anthropic.ToolInputSchema{
            Type: anthropic.F("object"),
            Properties: anthropic.F(map[string]interface{}{
                "conversationId": map[string]interface{}{"type": "string"},
                "clientMsgId":    map[string]interface{}{"type": "string"},
                "category":       map[string]interface{}{"type": "string"},
                "reason":         map[string]interface{}{"type": "string", "description": "撤回理由，将展示给用户"},
            }),
            Required: anthropic.F([]string{"conversationId", "clientMsgId", "category", "reason"}),
        }),
    },
    {
        Name:        anthropic.F("warn_user"),
        Description: anthropic.F("向用户发送警告通知。用于多次违规。"),
        InputSchema: anthropic.F(anthropic.ToolInputSchema{
            Type: anthropic.F("object"),
            Properties: anthropic.F(map[string]interface{}{
                "userId":  map[string]interface{}{"type": "string"},
                "reason":  map[string]interface{}{"type": "string"},
                "message": map[string]interface{}{"type": "string", "description": "警告消息内容"},
            }),
            Required: anthropic.F([]string{"userId", "reason", "message"}),
        }),
    },
    {
        Name:        anthropic.F("pass"),
        Description: anthropic.F("消息内容合规，无需任何操作。"),
        InputSchema: anthropic.F(anthropic.ToolInputSchema{
            Type: anthropic.F("object"),
            Properties: anthropic.F(map[string]interface{}{}),
        }),
    },
}
```

### 5.4 核心审核逻辑

```go
// internal/service/ai/moderation.go

type ModerationAgent struct {
    client      *anthropic.Client
    keywordRepo repository.KeywordRepo   // 敏感词库
    violationRepo repository.ViolationRepo
    msgSvc      service.MessageService
    hub         *ws.Hub
}

func (a *ModerationAgent) Moderate(ctx context.Context, msg *model.Message) error {
    // 第一层：规则引擎（关键词快速过滤）
    if hit := a.keywordRepo.Check(ctx, msg.Content); hit {
        return a.handleViolation(ctx, msg, "keyword", "high", "命中敏感词: "+hit.Word)
    }

    // 第二层：AI 审核（规则未命中时调用 LLM）
    resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(anthropic.ModelClaudeSonnet4_6),
        MaxTokens: anthropic.F(int64(512)),
        System: anthropic.F([]anthropic.TextBlockParam{{
            Text: `你是内容审核员。判断消息是否违规，违规则调用对应工具处理，合规则调用 pass。
违规类别：垃圾广告(spam)、辱骂人身攻击(abuse)、政治敏感(politics)、色情(porn)、欺诈广告(ad)。
判定标准：宁可放过，不可错杀。只有明确违规才判定。`,
        }}),
        Messages: anthropic.F([]anthropic.MessageParam{{
            Role: anthropic.F(anthropic.MessageParamRoleUser),
            Content: anthropic.F([]anthropic.ContentBlockParam{
                anthropic.NewTextBlock(fmt.Sprintf(
                    "发送者：%s\n消息内容：%s", msg.SenderID, msg.Content,
                )),
            }),
        }}),
        Tools: anthropic.F(moderationTools),
    })

    if err != nil {
        return err
    }

    // 处理 Tool Use 响应
    for _, block := range resp.Content {
        if block.Type == "tool_use" {
            a.executeTool(ctx, block.Name, block.Input)
        }
    }
    return nil
}

func (a *ModerationAgent) executeTool(ctx context.Context, toolName string, input map[string]interface{}) {
    switch toolName {
    case "mark_violation":
        convID := input["conversationId"].(string)
        clientMsgID := input["clientMsgId"].(string)
        category := input["category"].(string)
        severity := input["severity"].(string)
        reason, _ := input["reason"].(string)
        a.violationRepo.Create(ctx, &model.Violation{
            ConversationID: convID,
            ClientMsgID:    clientMsgID,
            Category:       category,
            Severity:       severity,
            Reason:         reason,
        })

    case "revoke_message":
        convID := input["conversationId"].(string)
        clientMsgID := input["clientMsgId"].(string)
        reason := input["reason"].(string)
        a.msgSvc.Revoke(ctx, convID, clientMsgID)
        // 通知用户
        // a.hub.PushToUser(...)

    case "warn_user":
        userID := input["userId"].(string)
        message := input["message"].(string)
        a.hub.PushToUser(userID, &ws.WSMessage{
            Type: 109,
            Data: map[string]interface{}{"reason": message},
        })

    case "pass":
        // 合规，无需操作
    }
}
```

### 5.5 与 Kafka 集成

第二阶段已有 Kafka 消息管道，审核 Agent 只需新增一个 Topic：

```
现有 Topic:
  toMongo        → MsgTransfer 消费
  toPush         → Push 服务消费
  toOfflinePush  → 离线推送消费

新增 Topic:
  toModeration   → Moderation Agent 消费
```

rpc-msg 发送消息时同时写 `toModeration` Topic，审核 Agent 异步消费处理。

### 5.6 学习要点

- **Tool Use / Function Calling**：LLM 如何决定调用哪个工具、解析参数、处理调用结果
- **混合架构**：规则引擎（确定性、快）+ AI（模糊判断、慢）的分层设计
- **异步审核**：消息先放行再审核 vs 先审核再放行的权衡
- **安全边界**：AI 审核的误判处理、申诉机制、人工复核

---

## 六、Agent 3：管理后台智能助手（Week 5-7）

### 6.1 功能描述

- 在管理后台提供一个对话式界面，管理员用自然语言查询数据、执行操作
- 示例对话：
  - "今天有多少新注册用户？" → 查数据库，返回数字
  - "最近 1 小时消息量是多少？" → 查统计表
  - "把用户 alice 封禁" → 调用管理 API
  - " gim 项目的部署流程是什么？" → 从知识库检索文档回答
- 支持 RAG：从项目文档、运维手册中检索信息辅助回答

### 6.2 架构设计

```
管理员输入 "今天有多少新用户？"
    │
    ▼
┌──────────────────────────────┐
│  Admin Assistant Agent        │
│                               │
│  1. 判断意图：数据查询 or     │
│     知识查询 or 管理操作       │
│                               │
│  ┌──────────┐ ┌──────────┐   │
│  │ Tool Use  │ │   RAG    │   │
│  │ 查数据库  │ │ 检索文档  │   │
│  │ 调管理API │ │          │   │
│  └──────────┘ └──────────┘   │
│                               │
│  2. 生成回答                  │
└──────────────────────────────┘
```

### 6.3 RAG 实现

```
知识文档 (Markdown/PDF)
    │
    ▼
┌──────────────┐
│  文档切片     │  将长文档切成 500-1000 字的片段
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Embedding    │  调用 Embedding API 将每个片段转为向量
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  向量数据库   │  Milvus / pgvector 存储向量
│  (Milvus)     │
└──────┬───────┘
       │
       │  查询时
       ▼
┌──────────────┐
│  相似度检索   │  用户问题 → Embedding → 在向量库中找 Top-K 最相似的文档片段
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  增强 Prompt  │  将检索到的文档片段注入 Prompt，LLM 基于这些内容回答
└──────────────┘
```

### 6.4 核心代码

```go
// internal/service/ai/admin_assistant.go
package ai

type AdminAssistant struct {
    client    *anthropic.Client
    adminSvc  service.AdminService
    vectorDB  *milvus.Client
    embedder  *Embedder
}

// Admin 工具定义
var adminTools = []anthropic.ToolParam{
    {
        Name:        anthropic.F("query_stats"),
        Description: anthropic.F("查询系统统计数据，如用户数、消息量、在线数等"),
        InputSchema: anthropic.F(anthropic.ToolInputSchema{
            Type: anthropic.F("object"),
            Properties: anthropic.F(map[string]interface{}{
                "metric":    map[string]interface{}{"type": "string", "enum": []string{"new_users", "total_users", "messages_today", "online_users", "groups_count"}},
                "startDate": map[string]interface{}{"type": "string", "description": "开始日期 YYYY-MM-DD"},
                "endDate":   map[string]interface{}{"type": "string", "description": "结束日期 YYYY-MM-DD"},
            }),
            Required: anthropic.F([]string{"metric"}),
        }),
    },
    {
        Name:        anthropic.F("ban_user"),
        Description: anthropic.F("封禁用户账号"),
        InputSchema: anthropic.F(anthropic.ToolInputSchema{
            Type: anthropic.F("object"),
            Properties: anthropic.F(map[string]interface{}{
                "userId": map[string]interface{}{"type": "string", "description": "要封禁的用户ID"},
                "reason": map[string]interface{}{"type": "string", "description": "封禁理由"},
            }),
            Required: anthropic.F([]string{"userId", "reason"}),
        }),
    },
    {
        Name:        anthropic.F("search_knowledge_base"),
        Description: anthropic.F("从项目知识库中检索文档，用于回答关于项目本身的问题"),
        InputSchema: anthropic.F(anthropic.ToolInputSchema{
            Type: anthropic.F("object"),
            Properties: anthropic.F(map[string]interface{}{
                "query": map[string]interface{}{"type": "string", "description": "检索关键词"},
            }),
            Required: anthropic.F([]string{"query"}),
        }),
    },
}

func (a *AdminAssistant) Chat(ctx context.Context, adminID string, messages []ChatMessage) (string, error) {
    // 构建 Anthropic 消息格式
    msgParams := a.buildMessageParams(messages)

    // 调用 Claude（带 Tools）
    resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(anthropic.ModelClaudeSonnet4_6),
        MaxTokens: anthropic.F(int64(2048)),
        System: anthropic.F([]anthropic.TextBlockParam{{
            Text: `你是 gim 管理后台的智能助手。管理员可以通过你查询系统数据、执行管理操作、查询项目文档。
执行操作前务必确认，涉及封禁用户等不可逆操作时需特别谨慎。
如果不确定答案，如实说明，不要编造数据。`,
        }}),
        Messages: anthropic.F(msgParams),
        Tools:   anthropic.F(adminTools),
    })
    if err != nil {
        return "", err
    }

    // 处理 Tool Use（可能多轮）
    return a.handleToolUseLoop(ctx, resp, msgParams)
}

func (a *AdminAssistant) handleToolUseLoop(ctx context.Context, resp *anthropic.Message, msgParams []anthropic.MessageParam) (string, error) {
    for {
        var toolResults []anthropic.ToolResultBlockParam
        var hasToolUse bool

        // 追加 assistant 回复
        msgParams = append(msgParams, anthropic.MessageParam{
            Role:    anthropic.F(anthropic.MessageParamRoleAssistant),
            Content: anthropic.F(resp.Content),
        })

        for _, block := range resp.Content {
            if block.Type == "tool_use" {
                hasToolUse = true
                result := a.executeAdminTool(ctx, block.Name, block.Input)
                toolResults = append(toolResults, anthropic.ToolResultBlockParam{
                    ToolUseID: anthropic.F(block.ID),
                    Content:   anthropic.F([]anthropic.TextBlockParam{{Text: result}}),
                })
            }
            if block.Type == "text" {
                // LLM 的文本回复
            }
        }

        if !hasToolUse {
            // 没有工具调用，提取最终文本回复
            return a.extractText(resp), nil
        }

        // 追加工具结果，继续对话
        msgParams = append(msgParams, anthropic.MessageParam{
            Role:    anthropic.F(anthropic.MessageParamRoleUser),
            Content: anthropic.F(toolResults),
        })

        resp, _ = a.client.Messages.New(ctx, anthropic.MessageNewParams{
            Model:     anthropic.F(anthropic.ModelClaudeSonnet4_6),
            MaxTokens: anthropic.F(int64(2048)),
            System: anthropic.F([]anthropic.TextBlockParam{{Text: "继续处理。"}}),
            Messages: anthropic.F(msgParams),
            Tools:   anthropic.F(adminTools),
        })
    }
}

func (a *AdminAssistant) executeAdminTool(ctx context.Context, toolName string, input map[string]interface{}) string {
    switch toolName {
    case "query_stats":
        metric, _ := input["metric"].(string)
        start, _ := input["startDate"].(string)
        end, _ := input["endDate"].(string)
        result, err := a.adminSvc.QueryStats(ctx, metric, start, end)
        if err != nil {
            return fmt.Sprintf("查询失败: %v", err)
        }
        return result

    case "ban_user":
        userID, _ := input["userId"].(string)
        reason, _ := input["reason"].(string)
        err := a.adminSvc.BanUser(ctx, userID, reason)
        if err != nil {
            return fmt.Sprintf("封禁失败: %v", err)
        }
        return fmt.Sprintf("用户 %s 已被封禁，理由: %s", userID, reason)

    case "search_knowledge_base":
        query, _ := input["query"].(string)
        docs, _ := a.ragSearch(ctx, query)
        if len(docs) == 0 {
            return "未找到相关文档。"
        }
        // 将检索结果格式化为文本供 LLM 引用
        var sb strings.Builder
        for i, doc := range docs {
            sb.WriteString(fmt.Sprintf("【文档%d】%s\n", i+1, doc.Content))
        }
        return sb.String()

    default:
        return "未知工具"
    }
}

// RAG 检索
func (a *AdminAssistant) ragSearch(ctx context.Context, query string) ([]Document, error) {
    // 1. 将查询向量化
    queryVec, err := a.embedder.Embed(ctx, query)
    if err != nil {
        return nil, err
    }

    // 2. 在 Milvus 中搜索 Top-5 最相似文档
    results, err := a.vectorDB.Search(ctx, "gim_docs", queryVec, 5)
    if err != nil {
        return nil, err
    }

    return results, nil
}
```

### 6.5 知识库构建

```go
// internal/service/ai/knowledge.go
package ai

// 文档切片与入库
func (a *AdminAssistant) IngestDocument(ctx context.Context, title, content string) error {
    // 1. 切片（按段落，每片 500-1000 字）
    chunks := splitByParagraph(content, 800)

    // 2. 批量 Embedding
    vectors, err := a.embedder.EmbedBatch(ctx, chunks)
    if err != nil {
        return err
    }

    // 3. 写入 Milvus
    for i, vec := range vectors {
        a.vectorDB.Insert(ctx, "gim_docs", &Document{
            ID:      snowflake.Generate().String(),
            Title:   title,
            Content: chunks[i],
            Vector:  vec,
        })
    }
    return nil
}

func splitByParagraph(content string, maxLen int) []string {
    paragraphs := strings.Split(content, "\n\n")
    var chunks []string
    var current string
    for _, p := range paragraphs {
        if len(current)+len(p) > maxLen && current != "" {
            chunks = append(chunks, current)
            current = p
        } else {
            if current != "" {
                current += "\n\n"
            }
            current += p
        }
    }
    if current != "" {
        chunks = append(chunks, current)
    }
    return chunks
}
```

需要入库的文档：
- PLAN.md、API.md、IMPLEMENTATION.md（项目文档）
- 运维手册（部署步骤、常见故障处理）
- 配置说明

### 6.6 学习要点

- **RAG 全流程**：文档切片 → Embedding → 向量存储 → 相似度检索 → 增强生成
- **Tool Use 多轮循环**：LLM 调工具 → 看结果 → 继续思考 → 可能再调工具
- **对话记忆管理**：多轮对话上下文的保存与截断策略
- **安全控制**：管理操作的确认机制、权限边界

---

## 七、Agent 4：群聊多 Agent 协作（Week 7-10）

### 7.1 功能描述

在群聊场景中部署多个专职 Agent，各司其职，协作完成复杂任务：

| Agent | 职责 | 触发方式 |
|-------|------|---------|
| 总结 Agent | 生成群消息摘要、会议纪要 | `@gim-bot 总结最近 50 条消息` |
| 待办 Agent | 从群聊中提取待办事项 | `@gim-bot 提取待办` 或自动检测 |
| 提醒 Agent | 定时提醒群成员 | `@gim-bot 提醒大家明天 10 点开会` |
| 问答 Agent | 回答群内问题 | `@gim-bot 什么是...` |

### 7.2 架构设计

```
@gim-bot 总结最近50条消息并提取待办
    │
    ▼
┌──────────────────────────────┐
│  Agent Router（路由器）        │
│                               │
│  1. 解析意图：需要总结 + 提待办 │
│  2. 拆分任务：                 │
│     → 总结 Agent：处理总结     │
│     → 待办 Agent：提取待办     │
│  3. 并发调用两个 Agent         │
│  4. 合并结果，统一回复          │
└──────────────────────────────┘
       │              │
       ▼              ▼
┌─────────────┐ ┌─────────────┐
│ Summary      │ │ Todo         │
│ Agent        │ │ Agent        │
│              │ │              │
│ 读50条消息   │ │ 读50条消息   │
│ LLM 生成摘要 │ │ LLM 提取待办 │
└──────┬──────┘ └──────┬──────┘
       │              │
       ▼              ▼
    "群聊摘要：..."   "待办事项：1. ... 2. ..."
       │              │
       └──────┬───────┘
              ▼
     合并输出，推送至群聊
```

### 7.3 Agent Router 核心代码

```go
// internal/service/ai/router.go
package ai

type AgentRouter struct {
    client       *anthropic.Client
    summaryAgent *SummaryAgent
    todoAgent    *TodoAgent
    remindAgent  *RemindAgent
    qaAgent      *QAAgent
}

func (r *AgentRouter) Handle(ctx context.Context, msg *GroupMessage) error {
    // 1. 路由判断：调用 LLM 决定需要哪些 Agent
    plan := r.planAgents(ctx, msg.Content)

    // 2. 并发调用各 Agent
    type agentResult struct {
        name    string
        content string
        err     error
    }
    resultCh := make(chan agentResult, len(plan.Agents))

    for _, agentName := range plan.Agents {
        go func(name string) {
            var content string
            var err error
            switch name {
            case "summary":
                content, err = r.summaryAgent.Run(ctx, msg.GroupID, plan.Params)
            case "todo":
                content, err = r.todoAgent.Run(ctx, msg.GroupID, plan.Params)
            case "remind":
                content, err = r.remindAgent.Run(ctx, msg.GroupID, plan.Params)
            case "qa":
                content, err = r.qaAgent.Run(ctx, msg.GroupID, plan.Params)
            }
            resultCh <- agentResult{name: name, content: content, err: err}
        }(agentName)
    }

    // 3. 收集结果
    var results []agentResult
    for i := 0; i < len(plan.Agents); i++ {
        results = append(results, <-resultCh)
    }

    // 4. 合并结果并推送
    finalContent := r.mergeResults(results)
    r.hub.PushToGroup(msg.GroupID, &ws.WSMessage{
        Type: 111, // 群聊 AI 消息
        Data: map[string]interface{}{
            "groupId":   msg.GroupID,
            "agentName": "gim-bot",
            "content":   finalContent,
        },
    })
    return nil
}

func (r *AgentRouter) planAgents(ctx context.Context, userMessage string) *AgentPlan {
    resp, _ := r.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(anthropic.ModelClaudeHaiku4_5_20251001), // 路由用轻量模型
        MaxTokens: anthropic.F(int64(256)),
        System: anthropic.F([]anthropic.TextBlockParam{{Text: `分析用户消息，决定需要调用哪些 Agent。
可用 Agent：
- summary：总结群聊消息、生成摘要
- todo：从消息中提取待办事项
- remind：设置定时提醒
- qa：回答知识性问题

返回 JSON 格式：{"agents": ["summary", "todo"], "params": {"count": 50}}
只返回 JSON，不要其他内容。`}}),
        Messages: anthropic.F([]anthropic.MessageParam{{
            Role:    anthropic.F(anthropic.MessageParamRoleUser),
            Content: anthropic.F([]anthropic.ContentBlockParam{anthropic.NewTextBlock(userMessage)}),
        }}),
    })

    var plan AgentPlan
    text := resp.Content[0].Text
    json.Unmarshal([]byte(text), &plan)
    return &plan
}
```

### 7.4 待办 Agent 示例

```go
// internal/service/ai/todo_agent.go
package ai

type TodoAgent struct {
    client   *anthropic.Client
    msgRepo  repository.MessageRepo
    todoRepo repository.TodoRepo
}

func (t *TodoAgent) Run(ctx context.Context, groupID string, params map[string]interface{}) (string, error) {
    // 1. 拉取群消息
    count := 50
    if c, ok := params["count"].(float64); ok {
        count = int(c)
    }
    msgs, _ := t.msgRepo.GetRecentGroupMsgs(ctx, groupID, count)

    // 2. 构建 Prompt
    var msgTexts []string
    for _, m := range msgs {
        msgTexts = append(msgTexts, fmt.Sprintf("[%s] %s: %s", m.SendTime.Format("15:04"), m.SenderID, m.Content))
    }

    // 3. 调用 LLM 提取待办
    resp, err := t.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(anthropic.ModelClaudeSonnet4_6),
        MaxTokens: anthropic.F(int64(1024)),
        System: anthropic.F([]anthropic.TextBlockParam{{Text: `从群聊消息中提取待办事项。
输出格式：
📌 待办事项：
1. [负责人] 任务描述 (截止时间)
2. [负责人] 任务描述
如果没有明确的待办，回复"未发现待办事项"。`}}),
        Messages: anthropic.F([]anthropic.MessageParam{{
            Role:    anthropic.F(anthropic.MessageParamRoleUser),
            Content: anthropic.F([]anthropic.ContentBlockParam{anthropic.NewTextBlock(strings.Join(msgTexts, "\n"))}),
        }}),
    })
    if err != nil {
        return "", err
    }

    result := resp.Content[0].Text

    // 4. 持久化待办（可选）
    t.todoRepo.CreateFromAI(ctx, groupID, result)

    return result, nil
}
```

### 7.5 数据模型

```sql
-- AI 相关表

-- 违规记录
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

### 7.6 学习要点

- **Multi-Agent 协作**：路由拆分 → 并发执行 → 结果合并
- **Agent 路由**：如何用 LLM 做任务理解和分发
- **Agent 记忆**：长期记忆（数据库）vs 短期记忆（对话上下文）
- **生产考量**：并发控制、失败重试、成本控制（Token 计费）

---

## 八、目录结构扩展

第四阶段新增的目录和文件：

```
gim/
├── internal/
│   └── service/
│       └── ai/                    ← 新增 AI Agent 服务
│           ├── router.go          ← Agent 路由器
│           ├── reply.go           ← 智能回复 Agent
│           ├── moderation.go      ← 内容审核 Agent
│           ├── admin_assistant.go ← 管理助手 Agent
│           ├── summary_agent.go   ← 总结 Agent
│           ├── todo_agent.go      ← 待办 Agent
│           ├── remind_agent.go    ← 提醒 Agent
│           ├── qa_agent.go        ← 问答 Agent
│           ├── knowledge.go       ← RAG 知识库管理
│           └── embedder.go        ← Embedding 封装
├── cmd/
│   └── gim-ai/                    ← AI 服务入口（独立部署）
│       └── main.go
├── api/
│   └── ai/
│       └── ai.proto               ← AI 服务 gRPC 定义
└── deploy/
    └── k8s/
        └── helm/
            └── templates/
                └── ai-deployment.yaml
```

---

## 九、第四阶段详细 TODO

### Week 1-2：智能回复助手

- [ ] 实现 AIProvider 统一接口（支持 Deepseek/Claude/本地模型切换）
- [ ] 集成 AI SDK：anthropic-sdk-go / openai-go（Deepseek 兼容 OpenAI 协议）
- [ ] 实现 AI Service 基础框架（配置、多 Provider 客户端初始化）
- [ ] 实现 ReplyAgent：构建上下文 + 调用 AIProvider + 流式输出
- [ ] WS 协议扩展：type=10（AI 请求）、type=110（AI 流式回复）
- [ ] WS Client 处理 type=10 消息，路由到 ReplyAgent
- [ ] 指令解析：`@gim-bot 翻译/总结/润色`
- [ ] 前端展示 AI 流式回复（打字机效果）
- [ ] 测试：手动 @gim-bot 验证回复质量
- [ ] 学习产出：写一篇笔记记录 Prompt Engineering 心得

### Week 3-5：内容审核 Agent

- [ ] 数据库迁移：violations、sensitive_words 表
- [ ] 敏感词库管理（CRUD API + 导入/导出）
- [ ] 规则引擎：关键词匹配（精确 + 模糊）
- [ ] ModerationAgent：Tool Use 定义 + 调用逻辑
- [ ] Tool 执行器：mark_violation / revoke_message / warn_user / pass
- [ ] Kafka 集成：新增 toModeration Topic，Agent 作为消费者
- [ ] 审核结果持久化与查询 API
- [ ] 管理后台审核日志页面
- [ ] 测试：构造违规消息验证审核链路
- [ ] 学习产出：写一篇笔记记录 Tool Use 工作原理

### Week 5-7：管理后台智能助手

- [ ] 数据库迁移：ai_conversations 表
- [ ] AdminAssistant 核心逻辑：Chat → Tool Use Loop
- [ ] 管理 Tool 定义：query_stats / ban_user / search_knowledge_base
- [ ] RAG 管道搭建：
  - [ ] Milvus / pgvector 部署（Docker Compose）
  - [ ] Embedding 封装（OpenAI API 或本地模型）
  - [ ] 文档切片与入库脚本
  - [ ] 相似度检索实现
- [ ] 对话记忆管理（ai_conversations 表存取）
- [ ] Admin Web UI：对话界面组件
- [ ] 测试：自然语言查询数据、执行操作、检索文档
- [ ] 学习产出：写一篇笔记记录 RAG 全流程与调优经验

### Week 7-10：群聊多 Agent 协作

- [ ] 数据库迁移：group_todos 表
- [ ] AgentRouter：意图解析 → Agent 选择 → 并发调度
- [ ] SummaryAgent：群消息摘要生成
- [ ] TodoAgent：待办提取与持久化
- [ ] RemindAgent：定时提醒（Cron + WS 推送）
- [ ] QAAgent：群内问答（RAG + 通用知识）
- [ ] WS 协议扩展：type=11（群 AI 请求）、type=111（群 AI 消息）
- [ ] 结果合并与格式化输出
- [ ] AI 服务独立部署（gim-ai，gRPC 接入）
- [ ] Admin Web UI：待办管理页、AI 配置页
- [ ] 压测：AI 请求对消息管道的影响
- [ ] 学习产出：写一篇笔记记录 Multi-Agent 架构设计

---

## 十、AI Agent 学习路径

按以下顺序学习，每个知识点都在项目中有实战场景：

```
LLM API 基础调用（回复助手）
    │
    ▼
Prompt Engineering（优化回复质量）
    │
    ▼
流式输出（用户体验）
    │
    ▼
Tool Use / Function Calling（审核 Agent）
    │
    ▼
RAG 检索增强（管理助手）
    │
    ▼
Multi-Agent 协作（群聊 Agent）
    │
    ▼
生产化：成本控制 / 限流 / 缓存 / 监控
```

### 推荐学习资源

| 主题 | 资源 |
|------|------|
| Deepseek API | [Deepseek API 文档](https://platform.deepseek.com/api-docs/)（OpenAI 兼容协议） |
| Claude API | [Anthropic 官方文档](https://docs.anthropic.com/) + [Go SDK](https://github.com/anthropics/anthropic-sdk-go) |
| 本地部署 | [Ollama](https://ollama.com/) / [vLLM](https://docs.vllm.ai/)（开发环境本地运行） |
| OpenAI Go SDK | [openai-go](https://github.com/openai/openai-go)（Deepseek 兼容） |
| Tool Use | [Anthropic Tool Use 文档](https://docs.anthropic.com/en/docs/build-with-claude/tool-use) |
| RAG | [Anthropic RAG 教程](https://docs.anthropic.com/en/docs/build-with-claude/retrieval-augmented-generation) |
| Prompt Engineering | [Anthropic Prompt Engineering 指南](https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering/overview) |
| Agent 模式 | [Anthropic Agent 模式](https://docs.anthropic.com/en/docs/build-with-claude/agentic-systems) |
| Multi-Agent | LangGraph / CrewAI 论文和文档（理解概念，用 Go 实现自己的版本） |
| 向量数据库 | [Milvus 文档](https://milvus.io/docs) 或 [pgvector](https://github.com/pgvector/pgvector) |

---

## 十一、成本与风险控制

### 11.1 API 调用成本

多 Provider 成本对比（以 1000 次调用估算）：

| Agent | Deepseek API | Claude API | 本地模型（Ollama） |
|-------|------------|------------|-------------------|
| 智能回复 | ~$0.5 | ~$5 | 免费（需 GPU） |
| 内容审核 | ~$2 | ~$15 | 免费（需 GPU） |
| 管理助手 | ~$0.1 | ~$1 | 免费（需 GPU） |
| 路由判断 | ~$0.05 | ~$0.1 | 免费（需 GPU） |

> **开发建议**：开发阶段使用本地模型（Ollama）零成本调试；生产环境按预算选择 Deepseek API（高性价比）或 Claude API（复杂推理场景）。

**节省策略：**
- 规则引擎拦截大部分消息，只有规则未命中的才调 AI（审核 Agent 可减少 80%+ API 调用）
- 路由判断用轻量模型/本地模型
- 缓存常见问题的 RAG 检索结果
- 设置每用户每日 AI 调用次数上限
- 通过 AIProvider 接口灵活切换后端，按需选择成本最优方案

### 11.2 安全风险

| 风险 | 应对 |
|------|------|
| Prompt 注入（用户构造消息欺骗 AI） | 输入清洗、System Prompt 明确边界、审核结果人工复核 |
| AI 幻觉（编造不存在的数据） | 数据查询走 Tool（而非让 LLM 直接回答数字），RAG 限定只基于检索结果回答 |
| 审核误判 | 规则引擎 + AI 双重审核，高 severity 操作需人工确认 |
| API Key 泄露 | 存 K8S Secret，代码中不硬编码，日志中不打印 |

---

## 十二、与前三阶段的关系

```
第一阶段 ─── 第二阶段 ─── 第三阶段 ─── 第四阶段
  │             │             │             │
  │             │             │         AI Agent 接入
  │             │             │         ├─ 复用 WS Gateway（推送 AI 回复）
  │             │             │         ├─ 复用 Kafka（toModeration Topic）
  │             │             │         ├─ 复用 Admin API（管理助手）
  │             │             │         ├─ 复用 K8S 部署（新增 gim-ai 服务）
  │             │             │         └─ 复用 Prometheus（AI 调用量/延迟监控）
  │             │             │
  │             │          监控+K8S+Admin     ← 第三阶段建好的基础设施，Agent 直接用
  │             │
  │          微服务+Kafka+MongoDB             ← 第二阶段的 Kafka 管道是 Agent 消息的入口
  │
单体+MySQL+Redis                           ← 第一阶段的消息模型（Seq/会话）Agent 同样遵守
```

**第四阶段不是孤立的，它复用前三阶段的所有基础设施。** 这正是 IM 项目学习 AI Agent 的优势——你不需要从零搭建消息管道、推送系统、部署方案，只需在已有管道上插入一个"智能消费者"。
