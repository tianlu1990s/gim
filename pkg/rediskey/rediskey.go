package rediskey

import "fmt"

// Redis Key 统一管理，避免各模块各自拼 Key 造成命名冲突。
// 所有 Key 生成函数集中在此，便于全局搜索和修改前缀。

const (
	// Token 黑名单 — logout 时加入，TTL = token 剩余有效期
	BlacklistToken = "blacklist:token:%s"

	// Refresh Token 存储 — 用于刷新和吊销
	Refresh = "refresh:%s:%s" // key: userId:platform

	// 消息 Seq，会话维度递增（Redis INCR 代替 MySQL 自增，避免表级锁瓶颈）
	SeqConv = "seq:conv:%s"

	// 消息去重 — SETNX，TTL=5min，防止网络重试导致重复写入
	DedupMsg = "dedup:msg:%s"

	// 用户已读位置 — 记录每个用户在每个会话中已读到哪条
	ReadSeq = "readseq:%s:%s" // key: userId:conversationId

	// 消息内容缓存（可选），减少数据库查询
	MsgCache = "msg:cache:%s:%d"

	// 在线状态 — Hash，存平台和连接信息
	Online = "online:%s"

	// 连接映射 — Set，存用户的所有 connID
	ConnMap = "conn_map:%s"

	// 限流计数
	RateLimit = "ratelimit:%s"
)

func BlacklistTokenKey(jti string) string {
	return fmt.Sprintf(BlacklistToken, jti)
}

func RefreshKey(userID, platform string) string {
	return fmt.Sprintf(Refresh, userID, platform)
}

func SeqConvKey(convID string) string {
	return fmt.Sprintf(SeqConv, convID)
}

func DedupMsgKey(clientMsgID string) string {
	return fmt.Sprintf(DedupMsg, clientMsgID)
}

func ReadSeqKey(userID, convID string) string {
	return fmt.Sprintf(ReadSeq, userID, convID)
}

func OnlineKey(userID string) string {
	return fmt.Sprintf(Online, userID)
}

func ConnMapKey(userID string) string {
	return fmt.Sprintf(ConnMap, userID)
}

func RateLimitKey(userID string) string {
	return fmt.Sprintf(RateLimit, userID)
}
