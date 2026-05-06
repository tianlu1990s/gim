package convutil

import "strings"

// ExtractTargetID 从单聊会话 ID 中提取对方的 userId。
// convID 格式为 "single_alice_bob"，若 senderID 为 "alice" 则返回 "bob"。
func ExtractTargetID(convID, senderID string) string {
	if !strings.HasPrefix(convID, "single_") {
		return convID
	}
	prefix := "single_"
	rest := strings.TrimPrefix(convID, prefix)
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == senderID {
		return parts[1]
	}
	if parts[1] == senderID {
		return parts[0]
	}
	return ""
}

// GetConversationMembers 返回会话中除 senderID 之外的所有成员。
// 单聊返回对方 ID，群聊返回完整 convID（群成员管理在 Phase 2 实现）。
func GetConversationMembers(convID, senderID string) []string {
	if !strings.HasPrefix(convID, "single_") {
		return []string{convID}
	}
	prefix := "single_"
	rest := strings.TrimPrefix(convID, prefix)
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) != 2 {
		return []string{}
	}
	return []string{parts[0], parts[1]}
}
