# GIM API Reference

Base URL: `http://localhost:8080/api/v1`

认证方式：除 `/auth/register` 和 `/auth/login` 外，所有请求需在 Header 中携带：

```
Authorization: Bearer <accessToken>
```

---

## 通用结构

### 响应格式

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

- `code=0` 表示成功，非零为错误码
- 分页响应的 `data` 结构：

```json
{
  "list": [],
  "total": 100,
  "page": 1,
  "pageSize": 20
}
```

### 错误码一览

| 错误码 | 含义 |
|--------|------|
| 0 | 成功 |
| 10001 | 参数错误 |
| 10002 | 未授权（Token 缺失/过期） |
| 10003 | 禁止访问（无权限） |
| 10004 | 资源不存在 |
| 10005 | 资源已存在（重复操作） |
| 10006 | 服务器内部错误 |
| 20001 | 用户名或密码错误 |
| 20002 | 用户已存在 |
| 20003 | 用户不存在 |
| 20004 | 用户被禁用 |
| 20005 | 好友关系已存在 |
| 20006 | 好友申请已存在 |
| 20007 | 非好友关系 |
| 20008 | 不能对自己操作 |
| 30001 | 会话不存在 |
| 30002 | 消息不存在 |
| 30003 | 消息已撤回 |
| 30004 | 消息超过可撤回时间 |
| 30005 | 非消息发送者 |
| 30006 | 非好友不能发消息 |
| 40001 | 群组不存在 |
| 40002 | 群组已解散 |
| 40003 | 已是群成员 |
| 40004 | 非群成员 |
| 40005 | 群已满员 |
| 40006 | 无群管理权限 |
| 40007 | 被禁言 |
| 50001 | 文件上传失败 |
| 50002 | 文件大小超限 |

---

## 1. 认证模块

### 1.1 POST /auth/register

注册新用户。

**请求体：**

```json
{
  "userId": "alice",
  "password": "P@ssw0rd123",
  "nickname": "爱丽丝",
  "phone": "13800138000",
  "email": "alice@example.com"
}
```

| 字段 | 类型 | 必填 | 规则 |
|------|------|------|------|
| userId | string | 是 | 4-32位，字母开头，仅字母数字下划线 |
| password | string | 是 | 8-64位，须含大小写字母和数字 |
| nickname | string | 否 | 1-64位，默认等于 userId |
| phone | string | 否 | 11位数字（中国手机号格式） |
| email | string | 否 | 合法邮箱格式 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "userId": "alice",
    "nickname": "爱丽丝",
    "avatarUrl": "",
    "createdAt": "2026-04-26T10:00:00Z"
  }
}
```

**错误场景：**

| code | 触发条件 |
|------|----------|
| 10001 | userId/password 格式不合规 |
| 20002 | userId 已被注册 |
| 20005 | phone 或 email 已被其他用户使用 |

---

### 1.2 POST /auth/login

登录，返回 Token 对。

**请求体：**

```json
{
  "userId": "alice",
  "password": "P@ssw0rd123",
  "platform": "web"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| userId | string | 是 | 用户ID |
| password | string | 是 | 密码 |
| platform | string | 否 | 登录平台：web/ios/android/desktop，默认 web |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "accessToken": "eyJhbGciOiJSUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJSUzI1NiIs...",
    "accessExpireAt": 1714070400,
    "refreshExpireAt": 1714675200,
    "userId": "alice"
  }
}
```

- accessToken 有效期：2 小时
- refreshToken 有效期：7 天
- 同 platform 踢掉旧连接（单平台单实例策略）

**错误场景：**

| code | 触发条件 |
|------|----------|
| 20001 | userId 或 password 错误 |
| 20004 | 用户被禁用 |

---

### 1.3 POST /auth/refresh

使用 refreshToken 换取新的 accessToken。

**请求体：**

```json
{
  "refreshToken": "eyJhbGciOiJSUzI1NiIs..."
}
```

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "accessToken": "eyJhbGciOiJSUzI1NiIs...",
    "accessExpireAt": 1714070400
  }
}
```

**错误场景：**

| code | 触发条件 |
|------|----------|
| 10002 | refreshToken 无效或已过期 |

---

### 1.4 POST /auth/logout

退出登录，accessToken 加入 Redis 黑名单，清除在线状态。

**请求体：**

```json
{
  "platform": "web"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| platform | string | 否 | 指定踢掉的平台，空则踢掉所有平台 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

---

## 2. 用户模块

### 2.1 GET /user/profile

获取自己的用户资料。

**请求：** 无参数，通过 Token 识别用户。

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "userId": "alice",
    "nickname": "爱丽丝",
    "avatarUrl": "https://cdn.example.com/avatar/alice.jpg",
    "phone": "138****8000",
    "email": "a***@example.com",
    "status": 1,
    "createdAt": "2026-04-26T10:00:00Z"
  }
}
```

> 注意：手机号和邮箱对本人脱敏展示（中间4位用星号替换）。

---

### 2.2 PUT /user/profile

修改自己的用户资料。

**请求体：**

```json
{
  "nickname": "新昵称",
  "avatarUrl": "https://cdn.example.com/avatar/new.jpg",
  "phone": "13900139000",
  "email": "new@example.com"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| nickname | string | 否 | 1-64位 |
| avatarUrl | string | 否 | 合法 URL，最大 512 字符 |
| phone | string | 否 | 更换手机号 |
| email | string | 否 | 更换邮箱 |

> userId 和 password 不可通过此接口修改，密码修改需单独接口。

**成功响应：** 返回更新后的完整 profile，同 2.1。

**错误场景：**

| code | 触发条件 |
|------|----------|
| 10001 | 字段格式不合规 |
| 20005 | 新手机号/邮箱已被其他用户使用 |

---

### 2.3 GET /user/profile/:userId

获取他人资料（公开信息）。

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| userId | string | 目标用户ID |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "userId": "bob",
    "nickname": "鲍勃",
    "avatarUrl": "https://cdn.example.com/avatar/bob.jpg",
    "isFriend": true,
    "remark": "老王"
  }
}
```

> isFriend 和 remark 仅在好友关系存在时返回。手机号/邮箱不对他人展示。

**错误场景：**

| code | 触发条件 |
|------|----------|
| 20003 | 目标用户不存在 |

---

### 2.4 POST /user/search

搜索用户（分页）。

**请求体：**

```json
{
  "keyword": "爱丽丝",
  "page": 1,
  "pageSize": 20
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 是 | 手机号精确匹配 或 昵称模糊匹配，2-64位 |
| page | int | 否 | 页码，默认 1 |
| pageSize | int | 否 | 每页条数，默认 20，最大 50 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "userId": "alice",
        "nickname": "爱丽丝",
        "avatarUrl": "https://cdn.example.com/avatar/alice.jpg",
        "isFriend": false
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

> keyword 纯数字时按手机号精确匹配，否则按昵称模糊匹配（LIKE %keyword%）。

---

## 3. 好友模块

### 3.1 POST /friend/request

发送好友申请。

**请求体：**

```json
{
  "toUserId": "bob",
  "message": "我是alice，加个好友吧"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| toUserId | string | 是 | 目标用户ID |
| message | string | 否 | 申请附言，最大 256 字符 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "requestId": 1001,
    "fromUserId": "alice",
    "toUserId": "bob",
    "message": "我是alice，加个好友吧",
    "status": 0,
    "createdAt": "2026-04-26T11:00:00Z"
  }
}
```

**错误场景：**

| code | 触发条件 |
|------|----------|
| 20003 | 目标用户不存在 |
| 20005 | 已经是好友 |
| 20006 | 已有待处理的申请（防止重复） |
| 20008 | 不能加自己为好友 |

---

### 3.2 GET /friend/request/incoming

收到的待处理好友申请列表（分页）。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 默认 1 |
| pageSize | int | 否 | 默认 20，最大 50 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "requestId": 1001,
        "fromUserId": "carol",
        "fromNickname": "卡罗尔",
        "fromAvatarUrl": "",
        "message": "加个好友",
        "status": 0,
        "createdAt": "2026-04-25T09:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

---

### 3.3 POST /friend/request/:id/accept

同意好友申请。

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | int | 好友申请ID |

**请求体：** 无

**处理逻辑：**
1. 验证申请存在且 status=0（待处理）
2. 验证申请的 toUserId 是当前用户
3. 事务内：更新申请状态 + 双向写入 friends 表 + 创建双方会话
4. 通过 WS 推送通知给申请方

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "friendUserId": "carol",
    "conversationId": "single_alice_carol"
  }
}
```

**错误场景：**

| code | 触发条件 |
|------|----------|
| 10004 | 申请不存在 |
| 10003 | 非目标用户（无权处理） |
| 10005 | 申请已处理（非待处理状态） |

---

### 3.4 POST /friend/request/:id/reject

拒绝好友申请。

**路径参数：** 同 3.3

**请求体：** 无

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

---

### 3.5 DELETE /friend/:userId

删除好友。

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| userId | string | 好友的用户ID |

**处理逻辑：**
1. 事务内：双向删除 friends 表记录
2. 不删除会话和消息历史（用户仍可查看）
3. WS 推送通知对方好友关系解除

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

**错误场景：**

| code | 触发条件 |
|------|----------|
| 20007 | 非好友关系 |

---

### 3.6 GET /friend/list

好友列表（分页）。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 默认 1 |
| pageSize | int | 否 | 默认 20，最大 50 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "userId": "bob",
        "nickname": "鲍勃",
        "avatarUrl": "",
        "remark": "老王",
        "isPinned": false,
        "isOnline": true
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

> isOnline 从 Redis 实时读取。

---

### 3.7 PUT /friend/:userId/remark

设置好友备注名。

**请求体：**

```json
{
  "remark": "同事-小王"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| remark | string | 是 | 备注名，1-64位 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

**错误场景：**

| code | 触发条件 |
|------|----------|
| 20007 | 非好友关系 |

---

## 4. 消息模块

> 消息发送通过 WebSocket 进行（见 WS 协议章节），HTTP 接口仅用于历史消息拉取和消息操作。

### 4.1 GET /msg/history

拉取消息历史（分页，按 Seq 倒序）。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| conversationId | string | 是 | 会话ID |
| startSeq | int64 | 否 | 起始 Seq，默认 0 表示从最新开始 |
| count | int | 否 | 拉取条数，默认 20，最大 50 |

**拉取方向规则：**

| startSeq | 行为 |
|----------|------|
| 0 | 从会话 maxSeq 开始，拉取最新的 count 条（向前翻页） |
| N > 0 | 拉取 seq <= N 的 count 条消息（继续向前翻页） |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "conversationId": "single_alice_bob",
        "seq": 42,
        "senderId": "bob",
        "contentType": 1,
        "content": "{\"text\":\"你好\"}",
        "clientMsgId": "client-uuid-001",
        "serverMsgId": "server-uuid-001",
        "isRead": true,
        "sendTime": 1714000042000,
        "status": 1
      }
    ],
    "hasMore": true,
    "minSeq": 1,
    "maxSeq": 100
  }
}
```

| 字段 | 说明 |
|------|------|
| hasMore | 是否还有更早的消息可拉取 |
| minSeq | 该会话最小有效 Seq |
| maxSeq | 该会话最大 Seq |

---

### 4.2 POST /msg/read

标记消息已读。

**请求体：**

```json
{
  "conversationId": "single_alice_bob",
  "readSeq": 42
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| conversationId | string | 是 | 会话ID |
| readSeq | int64 | 是 | 已读到此 Seq 位置（必须 <= maxSeq） |

**处理逻辑：**
1. 更新 user_conversation_seq 表的 readSeq
2. 同步更新 Redis 缓存
3. WS 通知对方用户已读回执

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

---

### 4.3 POST /msg/revoke

撤回消息（2 分钟内有效）。

**请求体：**

```json
{
  "conversationId": "single_alice_bob",
  "clientMsgId": "client-uuid-001"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|
| conversationId | string | 是 | 会话ID |
| clientMsgId | string | 是 | 要撤回的消息ID |

**处理逻辑：**
1. 查询消息，验证发送者是当前用户
2. 验证 sendTime + 120s > now
3. 更新消息 status=2（已撤回）
4. WS 通知对方消息已撤回

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

**错误场景：**

| code | 触发条件 |
|------|----------|
| 30002 | 消息不存在 |
| 30003 | 消息已被撤回 |
| 30004 | 超过 2 分钟可撤回时间 |
| 30005 | 非消息发送者 |

---

## 5. 会话模块

### 5.1 GET /conversation/list

获取当前用户的会话列表（含未读计数）。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 默认 1 |
| pageSize | int | 否 | 默认 20，最大 50 |

**排序规则：** 置顶会话优先，其次按最新消息时间倒序。

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "conversationId": "single_alice_bob",
        "type": 1,
        "targetId": "bob",
        "targetNickname": "鲍勃",
        "targetAvatarUrl": "",
        "isPinned": true,
        "unreadCount": 5,
        "lastMsg": {
          "senderId": "bob",
          "contentType": 1,
          "content": "{\"text\":\"在吗\"}",
          "sendTime": 1714000050000,
          "seq": 45
        },
        "maxSeq": 45,
        "readSeq": 40,
        "updatedAt": "2026-04-26T12:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

> unreadCount = maxSeq - readSeq（实时计算）

---

### 5.2 PUT /conversation/:id/pin

置顶/取消置顶会话。

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 会话ID（URL 编码） |

**请求体：**

```json
{
  "isPinned": true
}
```

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

---

### 5.3 DELETE /conversation/:id

删除会话（仅删除会话记录，不删除消息）。

**路径参数：** 同 5.2

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

> 消息历史仍可通过 /msg/history 拉取。下次收到该会话消息时自动重建会话记录。

---

## 6. 群组模块（第二阶段）

### 6.1 POST /group/create

创建群组。

**请求体：**

```json
{
  "groupName": "项目讨论组",
  "avatarUrl": "",
  "introduction": "日常工作讨论",
  "memberIds": ["bob", "carol"]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| groupName | string | 是 | 群名，1-64位 |
| avatarUrl | string | 否 | 群头像 URL |
| introduction | string | 否 | 群简介，最大 256 字符 |
| memberIds | []string | 否 | 初始成员列表（不含创建者），最多 50 人 |

**处理逻辑：**
1. 创建者自动成为群主（role=2）
2. memberIds 中的用户自动加入（role=0）
3. 创建群会话

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "groupId": "g1234567890",
    "groupName": "项目讨论组",
    "memberCount": 3,
    "createdAt": "2026-04-26T14:00:00Z"
  }
}
```

---

### 6.2 GET /group/:groupId

获取群组信息。

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "groupId": "g1234567890",
    "groupName": "项目讨论组",
    "avatarUrl": "",
    "notification": "",
    "introduction": "日常工作讨论",
    "creatorId": "alice",
    "memberCount": 3,
    "maxMembers": 500,
    "status": 1,
    "createdAt": "2026-04-26T14:00:00Z"
  }
}
```

---

### 6.3 POST /group/:groupId/invite

邀请入群。

**请求体：**

```json
{
  "userIds": ["dave"]
}
```

**权限：** 群主和管理员可邀请。

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "joinedUsers": ["dave"],
    "failedUsers": {}
  }
}
```

> 部分用户可能邀请失败（不存在/已是成员/被禁用），结果分 joinedUsers 和 failedUsers 返回。

---

### 6.4 POST /group/:groupId/join

申请入群（需要群主/管理员审批时）。

**请求体：**

```json
{
  "message": "我想加入这个群"
}
```

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "requestId": 2001,
    "status": 0
  }
}
```

---

### 6.5 POST /group/:groupId/leave

退群。群主不可退群（需先转让群主或解散群）。

**请求体：** 无

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

---

### 6.6 POST /group/:groupId/kick

踢人。

**请求体：**

```json
{
  "userIds": ["dave"]
}
```

**权限：** 群主和管理员可踢人，不可踢群主。

---

### 6.7 PUT /group/:groupId/info

修改群信息。

**请求体：**

```json
{
  "groupName": "新群名",
  "notification": "群公告内容",
  "introduction": "新群简介"
}
```

**权限：** 群主和管理员可修改。

---

### 6.8 GET /group/:groupId/members

获取群成员列表（分页）。

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 默认 1 |
| pageSize | int | 否 | 默认 50，最大 100 |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "userId": "alice",
        "nickname": "爱丽丝",
        "avatarUrl": "",
        "groupNickname": "",
        "role": 2,
        "joinAt": "2026-04-26T14:00:00Z"
      },
      {
        "userId": "bob",
        "nickname": "鲍勃",
        "avatarUrl": "",
        "groupNickname": "",
        "role": 0,
        "joinAt": "2026-04-26T14:00:00Z"
      }
    ],
    "total": 3,
    "page": 1,
    "pageSize": 50
  }
}
```

---

### 6.9 POST /group/:groupId/mute

禁言/取消禁言成员。

**请求体：**

```json
{
  "userId": "bob",
  "duration": 3600
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| userId | string | 是 | 被禁言的用户ID |
| duration | int | 是 | 禁言时长（秒），0 表示取消禁言 |

**权限：** 群主和管理员可操作。

---

## 7. 文件上传模块（第二阶段）

### 7.1 POST /file/upload

上传文件到 S3 兼容存储（MinIO/OSS）。

**请求格式：** `multipart/form-data`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | 文件内容 |
| type | string | 否 | 文件类型：image/video/audio/file，默认 file |

**限制：**

| 类型 | 大小上限 |
|------|----------|
| image | 10 MB |
| video | 100 MB |
| audio | 20 MB |
| file | 50 MB |

**成功响应：**

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "url": "https://minio.example.com/gim/image/2026/04/26/uuid.jpg",
    "fileType": "image",
    "fileSize": 102400
  }
}
```

---

## 8. WebSocket 协议

连接地址：`ws://localhost:8081/ws?token=<accessToken>&platform=web`

### 8.1 客户端 -> 服务端消息类型

| type | 名称 | data 结构 |
|------|------|-----------|
| 1 | 发送聊天消息 | `{conversationId, clientMsgId, contentType, content}` |
| 2 | 已读回执 | `{conversationId, readSeq}` |
| 3 | 心跳 | `{}` |
| 4 | 拉取消息 | `{conversationId, startSeq, count}` |
| 5 | 输入状态 | `{conversationId, isTyping: bool}` |

### 8.2 服务端 -> 客户端消息类型

| type | 名称 | data 结构 | 触发 |
|------|------|-----------|------|
| 101 | 新消息通知 | 见下方 | 收到他人消息 |
| 102 | 已读回执通知 | `{conversationId, readSeq, readUserId}` | 对方标记已读 |
| 103 | 心跳响应 | `{}` | 心跳回复 |
| 104 | 消息拉取结果 | 同 /msg/history 的 list 结构 | 拉取消息回复 |
| 105 | 输入状态通知 | `{conversationId, userId, isTyping}` | 对方正在输入 |
| 106 | 消息撤回通知 | `{conversationId, clientMsgId}` | 消息被撤回 |
| 107 | 好友变更通知 | `{type: "added"/"deleted", userId}` | 好友关系变更 |
| 108 | 群组变更通知 | `{type: "invited"/"kicked"/"dismissed", groupId}` | 群成员变更 |
| 109 | 强制下线 | `{reason: string}` | 被踢下线 |

### 8.3 各类型详细定义

#### type=1 发送聊天消息

```json
{
  "type": 1,
  "reqId": "req-uuid-001",
  "data": {
    "conversationId": "single_alice_bob",
    "clientMsgId": "client-uuid-001",
    "contentType": 1,
    "content": "{\"text\":\"你好\"}"
  }
}
```

contentType 枚举：

| 值 | 类型 | content 结构 |
|----|------|-------------|
| 1 | 文本 | `{"text": "消息内容"}` |
| 2 | 图片 | `{"url": "https://...", "width": 800, "height": 600, "size": 102400}` |
| 3 | 语音 | `{"url": "https://...", "duration": 30, "size": 204800}` |
| 4 | 视频 | `{"url": "https://...", "duration": 60, "size": 5242880, "thumbnailUrl": "https://..."}` |
| 5 | 文件 | `{"url": "https://...", "fileName": "doc.pdf", "fileSize": 1048576}` |
| 6 | 系统通知 | `{"text": "系统消息文本"}` |

服务端回复（发送确认）：

```json
{
  "type": 101,
  "reqId": "req-uuid-001",
  "data": {
    "conversationId": "single_alice_bob",
    "seq": 43,
    "serverMsgId": "server-uuid-001",
    "clientMsgId": "client-uuid-001",
    "sendTime": 1714000043000
  }
}
```

> reqId 匹配：客户端通过 reqId 关联发送请求与服务端确认。

#### type=101 新消息通知（推送）

```json
{
  "type": 101,
  "reqId": "",
  "data": {
    "conversationId": "single_alice_bob",
    "seq": 43,
    "senderId": "bob",
    "contentType": 1,
    "content": "{\"text\":\"你好\"}",
    "serverMsgId": "server-uuid-001",
    "clientMsgId": "client-uuid-001",
    "sendTime": 1714000043000
  }
}
```

#### type=2 已读回执

```json
{
  "type": 2,
  "reqId": "req-uuid-002",
  "data": {
    "conversationId": "single_alice_bob",
    "readSeq": 43
  }
}
```

#### type=102 已读回执通知（推送给对方）

```json
{
  "type": 102,
  "reqId": "",
  "data": {
    "conversationId": "single_alice_bob",
    "readUserId": "bob",
    "readSeq": 43
  }
}
```

#### type=3 心跳

```json
{
  "type": 3,
  "reqId": "heartbeat-001",
  "data": {}
}
```

回复：

```json
{
  "type": 103,
  "reqId": "heartbeat-001",
  "data": {}
}
```

> 客户端每 30s 发送一次心跳，服务端 60s 未收到心跳则断开连接。

#### type=4 拉取消息

```json
{
  "type": 4,
  "reqId": "req-uuid-003",
  "data": {
    "conversationId": "single_alice_bob",
    "startSeq": 0,
    "count": 20
  }
}
```

回复 type=104，数据结构同 HTTP /msg/history。

#### type=5 输入状态

```json
{
  "type": 5,
  "reqId": "req-uuid-004",
  "data": {
    "conversationId": "single_alice_bob",
    "isTyping": true
  }
}
```

对方收到 type=105：

```json
{
  "type": 105,
  "reqId": "",
  "data": {
    "conversationId": "single_alice_bob",
    "userId": "alice",
    "isTyping": true
  }
}
```

> isTyping=false 在停止输入 3 秒后自动发送，或客户端主动发送。

---

## 9. 配置文件参考

```yaml
server:
  httpPort: 8080
  wsPort: 8081
  readTimeout: 10s
  writeTimeout: 10s

mysql:
  host: 127.0.0.1
  port: 3306
  user: gim
  password: gim_pass
  dbname: gim
  maxOpenConns: 100
  maxIdleConns: 20
  connMaxLifetime: 300s

redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  db: 0
  poolSize: 100

jwt:
  accessTokenExpire: 2h
  refreshTokenExpire: 168h
  privateKeyPath: configs/private.pem
  publicKeyPath: configs/public.pem

websocket:
  maxConnPerUser: 5
  heartbeatInterval: 30s
  heartbeatTimeout: 60s
  maxMessageSize: 4096
  writeWait: 10s
  pongWait: 60s
  pingPeriod: 30s

log:
  level: info
  format: json
  output: stdout
```
