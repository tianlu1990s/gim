package snowflake

import (
	"sync"

	sf "github.com/bwmarrin/snowflake"
)

var (
	node *sf.Node
	once sync.Once // 保证 Init 只执行一次，多次调用并发安全
)

// Init 初始化 Snowflake 节点。nodeID 范围 0~1023，K8S 多副本部署时
// 每个 Pod 需分配不同 nodeID 以避免 ID 碰撞。
func Init(nodeID int64) {
	once.Do(func() {
		var err error
		node, err = sf.NewNode(nodeID)
		if err != nil {
			panic("failed to init snowflake node: " + err.Error())
		}
	})
}

// Generate 生成全局唯一 ID。Snowflake ID 天然包含时间戳，数据库索引友好
//（有序递增，不会像 UUID 那样导致 B+树页分裂）。
// 首次调用时若未显式 Init，自动使用节点 1 兜底。
func Generate() sf.ID {
	if node == nil {
		Init(1)
	}
	return node.Generate()
}
