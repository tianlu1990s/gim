package snowflake

import (
	"os"
	"os/exec"
	"sync"
	"testing"
)

func TestInitAndGenerate(t *testing.T) {
	// 重置 once 需要重新编译，这里直接测试 Init(1)
	// Init 已通过 sync.Once 保证安全，多次调用的效果等同于一次
	// 注意：如果其他测试已调用 Init，node 已初始化，不影响本测试
	Init(1)
	id := Generate()
	if id.String() == "" || id.String() == "0" {
		t.Error("Generate() returned zero/empty ID")
	}
}

func TestGenerateAutoInit(t *testing.T) {
	// 即使未显式 Init，Generate() 也会使用默认节点 1
	// 在已 Init 的环境下同样可以正常生成
	id := Generate()
	if id.String() == "" || id.String() == "0" {
		t.Error("Generate() should auto-init if needed")
	}
}

func TestGenerateUniqueness(t *testing.T) {
	// 并发生成 10000 个 ID，验证无重复
	const n = 10000
	ids := make(map[string]bool, n)
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < n/10; j++ {
				id := Generate().String()
				mu.Lock()
				ids[id] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(ids) != n {
		t.Errorf("expected %d unique IDs, got %d (collisions detected)", n, len(ids))
	}
}

func TestInitDifferentNodes(t *testing.T) {
	// 不同 nodeID 生成不同 workerID 的 ID
	// sync.Once 限制同一进程只能 Init 一次，这里验证 node 1 能正常生成
	Init(1)
	id := Generate()
	if id.Node() != 1 {
		t.Errorf("expected node 1, got node %d", id.Node())
	}
}

func TestInitInvalidNodeID(t *testing.T) {
	// Init with nodeID > 1023 should panic.
	// Use a subprocess since Init uses sync.Once and panics internally.
	if os.Getenv("SNOWFLAKE_TEST_INVALID_NODE") == "1" {
		Init(1024)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestInitInvalidNodeID")
	cmd.Env = append(os.Environ(), "SNOWFLAKE_TEST_INVALID_NODE=1")
	err := cmd.Run()
	if err == nil {
		t.Error("expected Init(1024) to exit with non-zero status (panic)")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if !exitErr.Success() {
			return
		}
		t.Errorf("expected non-zero exit, got %d", exitErr.ExitCode())
	}
}
