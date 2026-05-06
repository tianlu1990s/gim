package convutil

import (
	"testing"
)

func TestExtractTargetID(t *testing.T) {
	tests := []struct {
		name     string
		convID   string
		senderID string
		want     string
	}{
		{
			name:     "alice sends to bob",
			convID:   "single_alice_bob",
			senderID: "alice",
			want:     "bob",
		},
		{
			name:     "bob sends to alice",
			convID:   "single_alice_bob",
			senderID: "bob",
			want:     "alice",
		},
		{
			name:     "non-single conv returns convID directly",
			convID:   "group_123",
			senderID: "alice",
			want:     "group_123",
		},
		{
			name:     "empty convID",
			convID:   "",
			senderID: "alice",
			want:     "",
		},
		{
			name:     "sender not in convID",
			convID:   "single_alice_bob",
			senderID: "charlie",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTargetID(tt.convID, tt.senderID)
			if got != tt.want {
				t.Errorf("ExtractTargetID(%q, %q) = %q, want %q", tt.convID, tt.senderID, got, tt.want)
			}
		})
	}
}

func TestGetConversationMembers(t *testing.T) {
	tests := []struct {
		name     string
		convID   string
		senderID string
		wantLen  int
	}{
		{
			name:     "single chat: extract the other user",
			convID:   "single_alice_bob",
			senderID: "alice",
			wantLen:  2, // returns both user IDs for single chat
		},
		{
			name:     "group chat: returns convID as single member",
			convID:   "group_123",
			senderID: "alice",
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetConversationMembers(tt.convID, tt.senderID)
			if len(got) != tt.wantLen {
				t.Errorf("GetConversationMembers() returned %d members, want %d", len(got), tt.wantLen)
			}
		})
	}
}
