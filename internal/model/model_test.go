package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserToVO(t *testing.T) {
	now := time.Now().Truncate(time.Second) // truncate to avoid sub-second precision issues
	user := &User{
		ID:        1,
		UserID:    "testuser",
		Nickname:  "Test Nick",
		AvatarURL: "https://example.com/avatar.png",
		Password:  "should-not-appear",
		Phone:     "13800138000",
		Email:     "test@example.com",
		Status:    1,
		CreatedAt: now,
	}

	vo := user.ToVO()

	if vo.UserID != "testuser" {
		t.Errorf("UserID = %s, want testuser", vo.UserID)
	}
	if vo.Nickname != "Test Nick" {
		t.Errorf("Nickname = %s, want Test Nick", vo.Nickname)
	}
	if vo.AvatarURL != "https://example.com/avatar.png" {
		t.Errorf("AvatarURL = %s, want https://example.com/avatar.png", vo.AvatarURL)
	}
	if vo.Phone != "13800138000" {
		t.Errorf("Phone = %s, want 13800138000", vo.Phone)
	}
	if vo.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", vo.Email)
	}
	if vo.Status != 1 {
		t.Errorf("Status = %d, want 1", vo.Status)
	}
	if vo.CreatedAt != now.Format(time.RFC3339) {
		t.Errorf("CreatedAt = %s, want %s", vo.CreatedAt, now.Format(time.RFC3339))
	}
}

func TestPageReqGetPage(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"zero", 0, 1},
		{"negative", -5, 1},
		{"one", 1, 1},
		{"positive", 10, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PageReq{Page: tt.input}
			if got := r.GetPage(); got != tt.want {
				t.Errorf("GetPage() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPageReqGetPageSize(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"zero", 0, 20},
		{"negative", -5, 20},
		{"ten", 10, 10},
		{"fifty", 50, 50},
		{"over_max", 100, 100}, // GetPageSize does not cap at 50; binding does
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PageReq{PageSize: tt.input}
			if got := r.GetPageSize(); got != tt.want {
				t.Errorf("GetPageSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTableNames(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{ TableName() string }
		want string
	}{
		{"User", User{}, "users"},
		{"Friend", Friend{}, "friends"},
		{"FriendRequest", FriendRequest{}, "friend_requests"},
		{"Conversation", Conversation{}, "conversations"},
		{"Message", Message{}, "messages"},
		{"UserConversationSeq", UserConversationSeq{}, "user_conversation_seqs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.obj.TableName(); got != tt.want {
				t.Errorf("TableName() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTokenPairJSON(t *testing.T) {
	tp := TokenPair{
		AccessToken:     "access-token-123",
		RefreshToken:    "refresh-token-456",
		AccessExpireAt:  1700000000,
		RefreshExpireAt: 1700000001,
		UserID:          "alice",
	}

	data, err := json.Marshal(tp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded TokenPair
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.AccessToken != "access-token-123" {
		t.Errorf("AccessToken = %s, want access-token-123", decoded.AccessToken)
	}
	if decoded.RefreshToken != "refresh-token-456" {
		t.Errorf("RefreshToken = %s, want refresh-token-456", decoded.RefreshToken)
	}
	if decoded.AccessExpireAt != 1700000000 {
		t.Errorf("AccessExpireAt = %d, want 1700000000", decoded.AccessExpireAt)
	}
	if decoded.RefreshExpireAt != 1700000001 {
		t.Errorf("RefreshExpireAt = %d, want 1700000001", decoded.RefreshExpireAt)
	}
	if decoded.UserID != "alice" {
		t.Errorf("UserID = %s, want alice", decoded.UserID)
	}
}

func TestSendMsgRespJSON(t *testing.T) {
	resp := SendMsgResp{
		Seq:         42,
		ServerMsgID: "srv-msg-001",
		SendTime:    1700000000000,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded SendMsgResp
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Seq != 42 {
		t.Errorf("Seq = %d, want 42", decoded.Seq)
	}
	if decoded.ServerMsgID != "srv-msg-001" {
		t.Errorf("ServerMsgID = %s, want srv-msg-001", decoded.ServerMsgID)
	}
	if decoded.SendTime != 1700000000000 {
		t.Errorf("SendTime = %d, want 1700000000000", decoded.SendTime)
	}
}
