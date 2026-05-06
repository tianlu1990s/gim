package convid

import (
	"fmt"
	"sort"
	"strings"
)

// GenSingleConvID 生成单聊会话 ID。两个 userId 按字典序排列后拼接，
// 保证 Alice+Bob 和 Bob+Alice 生成相同的会话 ID（两人共享同一会话）。
func GenSingleConvID(uid1, uid2 string) string {
	ids := []string{uid1, uid2}
	sort.Strings(ids)
	return fmt.Sprintf("single_%s_%s", ids[0], ids[1])
}

// GenGroupConvID 生成群聊会话 ID（群功能 Phase 2 预留）。
func GenGroupConvID(groupID string) string {
	return fmt.Sprintf("group_%s", groupID)
}

func IsSingleConvID(convID string) bool {
	return strings.HasPrefix(convID, "single_")
}

func IsGroupConvID(convID string) bool {
	return strings.HasPrefix(convID, "group_")
}
