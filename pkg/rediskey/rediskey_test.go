package rediskey

import (
	"testing"
)

func TestBlacklistTokenKey(t *testing.T) {
	got := BlacklistTokenKey("jti-abc123")
	want := "blacklist:token:jti-abc123"
	if got != want {
		t.Errorf("BlacklistTokenKey() = %v, want %v", got, want)
	}
}

func TestRefreshKey(t *testing.T) {
	got := RefreshKey("alice", "ios")
	want := "refresh:alice:ios"
	if got != want {
		t.Errorf("RefreshKey() = %v, want %v", got, want)
	}
}

func TestSeqConvKey(t *testing.T) {
	got := SeqConvKey("single_alice_bob")
	want := "seq:conv:single_alice_bob"
	if got != want {
		t.Errorf("SeqConvKey() = %v, want %v", got, want)
	}
}

func TestDedupMsgKey(t *testing.T) {
	got := DedupMsgKey("client-msg-uuid-123")
	want := "dedup:msg:client-msg-uuid-123"
	if got != want {
		t.Errorf("DedupMsgKey() = %v, want %v", got, want)
	}
}

func TestReadSeqKey(t *testing.T) {
	got := ReadSeqKey("alice", "single_alice_bob")
	want := "readseq:alice:single_alice_bob"
	if got != want {
		t.Errorf("ReadSeqKey() = %v, want %v", got, want)
	}
}

func TestOnlineKey(t *testing.T) {
	got := OnlineKey("alice")
	want := "online:alice"
	if got != want {
		t.Errorf("OnlineKey() = %v, want %v", got, want)
	}
}

func TestConnMapKey(t *testing.T) {
	got := ConnMapKey("alice")
	want := "conn_map:alice"
	if got != want {
		t.Errorf("ConnMapKey() = %v, want %v", got, want)
	}
}

func TestRateLimitKey(t *testing.T) {
	got := RateLimitKey("alice")
	want := "ratelimit:alice"
	if got != want {
		t.Errorf("RateLimitKey() = %v, want %v", got, want)
	}
}

func TestAllKeysUnique(t *testing.T) {
	// 所有 Key 函数应生成不同的格式，避免不同用途间的 Key 碰撞
	keys := []string{
		BlacklistTokenKey("test1"),
		RefreshKey("test1", "ios"),
		SeqConvKey("test1"),
		DedupMsgKey("test1"),
		ReadSeqKey("test1", "test2"),
		OnlineKey("test1"),
		ConnMapKey("test1"),
		RateLimitKey("test1"),
	}
	seen := map[string]string{}
	for i, key := range keys {
		if name, exists := seen[key]; exists {
			t.Errorf("key collision: index %d key %q collides with %s", i, key, name)
		}
		seen[key] = ""
	}
}
