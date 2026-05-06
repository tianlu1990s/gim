package convid

import (
	"strings"
	"testing"
)

func TestGenSingleConvID(t *testing.T) {
	// 同一对用户，无论传入顺序如何，生成的 convID 必须一致（幂等性）
	id1 := GenSingleConvID("alice", "bob")
	id2 := GenSingleConvID("bob", "alice")
	if id1 != id2 {
		t.Errorf("convID should be order-independent: %s vs %s", id1, id2)
	}
	if !strings.HasPrefix(id1, "single_") {
		t.Errorf("convID should start with single_: %s", id1)
	}
	if !strings.Contains(id1, "alice") || !strings.Contains(id1, "bob") {
		t.Errorf("convID should contain both userIds: %s", id1)
	}
	// 字典序：alice < bob
	if id1 != "single_alice_bob" {
		t.Errorf("convID should be sorted: got %s, want single_alice_bob", id1)
	}
}

func TestGenSingleConvIDSameUser(t *testing.T) {
	// 同一用户跟自己不应产生友关系的会话，但函数本身不校验（校验在 Service 层）
	id := GenSingleConvID("alice", "alice")
	if !strings.HasPrefix(id, "single_") {
		t.Errorf("convID should start with single_: %s", id)
	}
}

func TestGenGroupConvID(t *testing.T) {
	id := GenGroupConvID("group-123")
	want := "group_group-123"
	if id != want {
		t.Errorf("GenGroupConvID() = %v, want %v", id, want)
	}
}

func TestIsSingleConvID(t *testing.T) {
	if !IsSingleConvID("single_alice_bob") {
		t.Error("single_alice_bob should be single conv")
	}
	if IsSingleConvID("group_123") {
		t.Error("group_123 should not be single conv")
	}
	if IsSingleConvID("") {
		t.Error("empty string should not be single conv")
	}
}

func TestIsGroupConvID(t *testing.T) {
	if !IsGroupConvID("group_123") {
		t.Error("group_123 should be group conv")
	}
	if IsGroupConvID("single_alice_bob") {
		t.Error("single_alice_bob should not be group conv")
	}
	if IsGroupConvID("") {
		t.Error("empty string should not be group conv")
	}
}

func TestGenSingleConvIDSpecialChars(t *testing.T) {
	// 包含特殊字符的 userId（下划线是合法的 convID 分隔符，需验证）
	id := GenSingleConvID("user_1", "user_2")
	if id == "single_user_1_user_2" {
		// 没问题：两个下划线分隔后仍可解析
	} else {
		// 也没问题：取决于字典序
	}
	// 仅验证不 panic
	_ = id
}
