package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/tianlu1990s/gim/internal/config"
	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/repository"
	"github.com/tianlu1990s/gim/internal/ws"
	"github.com/tianlu1990s/gim/pkg/jwt"
)

// ---------- mock repos (implement full interfaces) ----------

type mockUserRepo struct {
	users      map[string]*model.User
	phoneUsers map[string]*model.User
	emailUsers map[string]*model.User
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	m.users[user.UserID] = user
	if user.Phone != "" {
		m.phoneUsers[user.Phone] = user
	}
	if user.Email != "" {
		m.emailUsers[user.Email] = user
	}
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, userID string) (*model.User, error) {
	u, ok := m.users[userID]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserRepo) ExistsByID(_ context.Context, userID string) (bool, error) {
	_, ok := m.users[userID]
	return ok, nil
}

func (m *mockUserRepo) Update(_ context.Context, userID string, updates map[string]any) error {
	u, ok := m.users[userID]
	if !ok {
		return nil
	}
	if v, ok := updates["nickname"]; ok {
		u.Nickname = v.(string)
	}
	if v, ok := updates["avatar_url"]; ok {
		u.AvatarURL = v.(string)
	}
	if v, ok := updates["phone"]; ok {
		oldPhone := u.Phone
		u.Phone = v.(string)
		delete(m.phoneUsers, oldPhone)
		m.phoneUsers[u.Phone] = u
	}
	if v, ok := updates["email"]; ok {
		oldEmail := u.Email
		u.Email = v.(string)
		delete(m.emailUsers, oldEmail)
		m.emailUsers[u.Email] = u
	}
	return nil
}

func (m *mockUserRepo) Search(_ context.Context, keyword string, page, pageSize int) ([]*model.User, int64, error) {
	var results []*model.User
	for _, u := range m.users {
		if u.Nickname == keyword {
			results = append(results, u)
		}
	}
	total := int64(len(results))
	// Apply pagination
	start := (page - 1) * pageSize
	if start >= len(results) {
		return []*model.User{}, total, nil
	}
	end := start + pageSize
	if end > len(results) {
		end = len(results)
	}
	return results[start:end], total, nil
}

func (m *mockUserRepo) ExistsByPhone(_ context.Context, phone, excludeUserID string) (bool, error) {
	u, ok := m.phoneUsers[phone]
	if !ok {
		return false, nil
	}
	return u.UserID != excludeUserID, nil
}

func (m *mockUserRepo) ExistsByEmail(_ context.Context, email, excludeUserID string) (bool, error) {
	u, ok := m.emailUsers[email]
	if !ok {
		return false, nil
	}
	return u.UserID != excludeUserID, nil
}

type mockFriendRepo struct {
	friends  map[string]map[string]*model.Friend // ownerID -> friendID -> Friend
	mu       chan struct{}                        // simple lock
	callLog  []string                             // track method calls
}

func newMockFriendRepo() *mockFriendRepo {
	return &mockFriendRepo{
		friends: make(map[string]map[string]*model.Friend),
		mu:      make(chan struct{}, 1),
	}
}

func (m *mockFriendRepo) lock()   { m.mu <- struct{}{} }
func (m *mockFriendRepo) unlock() { <-m.mu }

func (m *mockFriendRepo) Create(_ context.Context, ownerID, friendID, remark string) error {
	m.lock()
	defer m.unlock()
	m.callLog = append(m.callLog, "Create")
	if m.friends[ownerID] == nil {
		m.friends[ownerID] = make(map[string]*model.Friend)
	}
	m.friends[ownerID][friendID] = &model.Friend{
		OwnerID:  ownerID,
		FriendID: friendID,
		Remark:   remark,
	}
	return nil
}

func (m *mockFriendRepo) CreateTx(_ context.Context, _ *gorm.DB, ownerID, friendID, remark string) error {
	m.lock()
	defer m.unlock()
	m.callLog = append(m.callLog, "CreateTx:"+ownerID+":"+friendID)
	if m.friends[ownerID] == nil {
		m.friends[ownerID] = make(map[string]*model.Friend)
	}
	m.friends[ownerID][friendID] = &model.Friend{
		OwnerID:  ownerID,
		FriendID: friendID,
		Remark:   remark,
	}
	return nil
}

func (m *mockFriendRepo) Delete(_ context.Context, ownerID, friendID string) error {
	m.lock()
	defer m.unlock()
	m.callLog = append(m.callLog, "Delete")
	if m.friends[ownerID] != nil {
		delete(m.friends[ownerID], friendID)
	}
	return nil
}

func (m *mockFriendRepo) IsFriend(_ context.Context, ownerID, friendID string) (bool, error) {
	m.lock()
	defer m.unlock()
	if m.friends[ownerID] == nil {
		return false, nil
	}
	_, ok := m.friends[ownerID][friendID]
	return ok, nil
}

func (m *mockFriendRepo) GetFriend(_ context.Context, ownerID, friendID string) (*model.Friend, error) {
	m.lock()
	defer m.unlock()
	if m.friends[ownerID] == nil {
		return nil, nil
	}
	f, ok := m.friends[ownerID][friendID]
	if !ok {
		return nil, nil
	}
	return f, nil
}

func (m *mockFriendRepo) List(_ context.Context, ownerID string, page, pageSize int) ([]*model.FriendVO, int64, error) {
	m.lock()
	defer m.unlock()
	m.callLog = append(m.callLog, "List")
	var result []*model.FriendVO
	if m.friends[ownerID] == nil {
		return result, 0, nil
	}
	for friendID, f := range m.friends[ownerID] {
		result = append(result, &model.FriendVO{
			FriendID:  friendID,
			Remark:    f.Remark,
		})
	}
	total := int64(len(result))
	return result, total, nil
}

func (m *mockFriendRepo) SetRemark(_ context.Context, ownerID, friendID, remark string) error {
	m.lock()
	defer m.unlock()
	m.callLog = append(m.callLog, "SetRemark")
	if m.friends[ownerID] == nil || m.friends[ownerID][friendID] == nil {
		return gorm.ErrRecordNotFound
	}
	m.friends[ownerID][friendID].Remark = remark
	return nil
}

type mockFriendRequestRepo struct {
	requests   map[int64]*model.FriendRequest
	nextID     int64
	hasPending bool
}

func newMockFriendRequestRepo() *mockFriendRequestRepo {
	return &mockFriendRequestRepo{
		requests: make(map[int64]*model.FriendRequest),
		nextID:   1,
	}
}

func (m *mockFriendRequestRepo) addRequest(fromID, toID, message string, status int8) int64 {
	id := m.nextID
	m.nextID++
	m.requests[id] = &model.FriendRequest{
		ID:         uint64(id),
		FromUserID: fromID,
		ToUserID:   toID,
		Message:    message,
		Status:     status,
	}
	return id
}

func (m *mockFriendRequestRepo) Create(_ context.Context, fromID, toID, message string) (int64, error) {
	id := m.addRequest(fromID, toID, message, 0)
	return id, nil
}

func (m *mockFriendRequestRepo) GetByID(_ context.Context, id int64) (*model.FriendRequest, error) {
	req, ok := m.requests[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return req, nil
}

func (m *mockFriendRequestRepo) ListIncoming(_ context.Context, toID string, _, _ int) ([]*model.FriendRequestVO, int64, error) {
	var result []*model.FriendRequestVO
	for _, req := range m.requests {
		if req.ToUserID == toID {
			result = append(result, &model.FriendRequestVO{
				ID:         int64(req.ID),
				FromUserID: req.FromUserID,
				Message:    req.Message,
				Status:     req.Status,
			})
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockFriendRequestRepo) UpdateStatus(_ context.Context, id int64, status int) error {
	req, ok := m.requests[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	req.Status = int8(status)
	return nil
}

func (m *mockFriendRequestRepo) UpdateStatusTx(_ context.Context, _ *gorm.DB, id int64, status int) error {
	return m.UpdateStatus(context.Background(), id, status)
}

func (m *mockFriendRequestRepo) HasPendingRequest(_ context.Context, fromID, toID string) (bool, error) {
	return m.hasPending, nil
}

type mockConversationRepo struct {
	convs map[string]*model.Conversation // key: ownerID:convID
}

func newMockConversationRepo() *mockConversationRepo {
	return &mockConversationRepo{
		convs: make(map[string]*model.Conversation),
	}
}

func (m *mockConversationRepo) key(ownerID, convID string) string {
	return ownerID + ":" + convID
}

func (m *mockConversationRepo) CreateIfNotExistTx(_ context.Context, _ *gorm.DB, ownerID, convID string, convType int, targetID string) error {
	k := m.key(ownerID, convID)
	if _, ok := m.convs[k]; ok {
		return nil
	}
	m.convs[k] = &model.Conversation{
		OwnerID:        ownerID,
		ConversationID: convID,
		ConvType:       convType,
		TargetID:       targetID,
	}
	return nil
}

func (m *mockConversationRepo) List(_ context.Context, ownerID string, _, _ int) ([]*model.ConversationVO, int64, error) {
	var result []*model.ConversationVO
	for k, conv := range m.convs {
		if k[:len(ownerID)] == ownerID {
			result = append(result, &model.ConversationVO{
				ConversationID: conv.ConversationID,
				ConvType:       conv.ConvType,
				TargetID:       conv.TargetID,
				MaxSeq:         conv.MaxSeq,
				IsPinned:       conv.IsPinned,
			})
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockConversationRepo) UpdatePin(_ context.Context, ownerID, convID string, isPinned bool) error {
	k := m.key(ownerID, convID)
	if conv, ok := m.convs[k]; ok {
		conv.IsPinned = isPinned
	}
	return nil
}

func (m *mockConversationRepo) Delete(_ context.Context, ownerID, convID string) error {
	k := m.key(ownerID, convID)
	delete(m.convs, k)
	return nil
}

func (m *mockConversationRepo) UpdateMaxSeq(_ context.Context, convID string, seq int64) error {
	for _, conv := range m.convs {
		if conv.ConversationID == convID {
			conv.MaxSeq = seq
		}
	}
	return nil
}

func (m *mockConversationRepo) GetByID(_ context.Context, ownerID, convID string) (*model.Conversation, error) {
	k := m.key(ownerID, convID)
	conv, ok := m.convs[k]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return conv, nil
}

func (m *mockConversationRepo) ListByOwner(_ context.Context, ownerID string) ([]*model.Conversation, error) {
	var result []*model.Conversation
	for k, conv := range m.convs {
		if k[:len(ownerID)] == ownerID {
			result = append(result, conv)
		}
	}
	return result, nil
}

type mockMessageRepo struct {
	msgs   []*model.Message // stored in order
	seqMap map[string]int64 // conversationID -> current seq
	maxSeq map[string]int64 // conversationID -> max seq
	minSeq map[string]int64 // conversationID -> min seq
}

func newMockMessageRepo() *mockMessageRepo {
	return &mockMessageRepo{
		seqMap: make(map[string]int64),
		maxSeq: make(map[string]int64),
		minSeq: make(map[string]int64),
	}
}

func (m *mockMessageRepo) Create(_ context.Context, msg *model.Message) error {
	m.msgs = append(m.msgs, msg)
	if msg.Seq > m.maxSeq[msg.ConversationID] {
		m.maxSeq[msg.ConversationID] = msg.Seq
	}
	if m.minSeq[msg.ConversationID] == 0 || msg.Seq < m.minSeq[msg.ConversationID] {
		m.minSeq[msg.ConversationID] = msg.Seq
	}
	return nil
}

func (m *mockMessageRepo) GetBySeqRange(_ context.Context, conversationID string, startSeq, endSeq int64, limit int) ([]*model.Message, error) {
	var result []*model.Message
	for _, msg := range m.msgs {
		if msg.ConversationID != conversationID {
			continue
		}
		if startSeq > 0 && msg.Seq < startSeq {
			continue
		}
		if endSeq > 0 && msg.Seq > endSeq {
			continue
		}
		result = append(result, msg)
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockMessageRepo) GetByClientMsgID(_ context.Context, clientMsgID string) (*model.Message, error) {
	for _, msg := range m.msgs {
		if msg.ClientMsgID == clientMsgID {
			return msg, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockMessageRepo) Revoke(_ context.Context, conversationID, clientMsgID string) error {
	for _, msg := range m.msgs {
		if msg.ConversationID == conversationID && msg.ClientMsgID == clientMsgID {
			msg.IsRevoked = true
			return nil
		}
	}
	return nil
}

func (m *mockMessageRepo) GetLastMsg(_ context.Context, conversationID string) (*model.Message, error) {
	var last *model.Message
	for _, msg := range m.msgs {
		if msg.ConversationID == conversationID && msg.IsRevoked == false {
			if last == nil || msg.Seq > last.Seq {
				last = msg
			}
		}
	}
	if last == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return last, nil
}

func (m *mockMessageRepo) GetUserReadSeq(_ context.Context, _, _ string) (int64, error) { return 0, nil }
func (m *mockMessageRepo) SetUserReadSeq(_ context.Context, _, _ string, _ int64) error { return nil }
func (m *mockMessageRepo) UpdateUserReadSeqDB(_ context.Context, _, _ string, _ int64) error {
	return nil
}

func (m *mockMessageRepo) GetMaxSeq(_ context.Context, conversationID string) (int64, error) {
	return m.maxSeq[conversationID], nil
}

func (m *mockMessageRepo) GetMinSeq(_ context.Context, conversationID string) (int64, error) {
	return m.minSeq[conversationID], nil
}

func (m *mockMessageRepo) IncrSeq(_ context.Context, conversationID string) (int64, error) {
	m.seqMap[conversationID]++
	seq := m.seqMap[conversationID]
	if seq > m.maxSeq[conversationID] {
		m.maxSeq[conversationID] = seq
	}
	return seq, nil
}

// ---------- helpers ----------

func newMockRepos() *repository.Repositories {
	return &repository.Repositories{
		User: &mockUserRepo{
			users:      make(map[string]*model.User),
			phoneUsers: make(map[string]*model.User),
			emailUsers: make(map[string]*model.User),
		},
		Friend:       newMockFriendRepo(),
		FriendReq:    newMockFriendRequestRepo(),
		Conversation: newMockConversationRepo(),
		Message:      newMockMessageRepo(),
	}
}

func newTestJWTServiceMgr(t *testing.T) *jwt.JWTManager {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return jwt.NewJWTManagerFromKeys(privKey, &privKey.PublicKey, 15*time.Minute, 168*time.Hour)
}

func newTestServices(repos *repository.Repositories) (*config.Config, *jwt.JWTManager, *ws.Hub) {
	cfg := &config.Config{
		JWT: config.JWTConfig{AccessTokenExpire: 15 * time.Minute, RefreshTokenExpire: 168 * time.Hour},
	}
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwtMgr := jwt.NewJWTManagerFromKeys(privKey, &privKey.PublicKey, 15*time.Minute, 168*time.Hour)
	hub := ws.NewHub(nil, config.WebSocketConfig{MaxConnPerUser: 5, MaxMessageSize: 4096})
	go hub.Run()
	return cfg, jwtMgr, hub
}

// newTestHub creates a ws.Hub for tests that don't need full services.
func newTestHub() *ws.Hub {
	hub := ws.NewHub(nil, config.WebSocketConfig{MaxConnPerUser: 5, MaxMessageSize: 4096})
	go hub.Run()
	return hub
}

// newTestDB connects to MySQL for tests that need a real *gorm.DB (e.g. Transaction).
// Returns nil if MySQL is not available (test will be skipped).
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "gim:gim_pass@tcp(127.0.0.1:3306)/gim?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil
	}
	return db
}

// startTestRedis starts a miniredis server and returns the client and server for cleanup.
func startTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		rdb.Close()
		mr.Close()
	})
	return rdb, mr
}

// ---------- register tests ----------

func TestRegisterSuccess(t *testing.T) {
	repos := newMockRepos()
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	user, err := svc.Register(context.Background(), &model.RegisterReq{
		UserID:   "alice123",
		Password: "Pass1234",
		Nickname: "Alice",
	})
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if user.UserID != "alice123" {
		t.Errorf("UserID = %s, want alice123", user.UserID)
	}
	if user.Nickname != "Alice" {
		t.Errorf("Nickname = %s, want Alice", user.Nickname)
	}
}

func TestRegisterDefaultNickname(t *testing.T) {
	repos := newMockRepos()
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	user, err := svc.Register(context.Background(), &model.RegisterReq{
		UserID:   "bob456",
		Password: "Pass1234",
	})
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if user.Nickname != "bob456" {
		t.Errorf("default nickname = %s, want bob456", user.Nickname)
	}
}

func TestRegisterExistingUser(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123"}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	_, err := svc.Register(context.Background(), &model.RegisterReq{
		UserID:   "alice123",
		Password: "Pass1234",
	})
	if err == nil {
		t.Error("should return error for existing user")
	}
}

func TestRegisterInvalidUserID(t *testing.T) {
	repos := newMockRepos()
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	tests := []struct{ name, userID string }{
		{"too short", "ab"},
		{"starts with digit", "1alice"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), &model.RegisterReq{UserID: tt.userID, Password: "Pass1234"})
			if err == nil {
				t.Error("should return error")
			}
		})
	}
}

func TestRegisterWeakPassword(t *testing.T) {
	repos := newMockRepos()
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	_, err := svc.Register(context.Background(), &model.RegisterReq{
		UserID:   "alice123",
		Password: "abc",
	})
	if err == nil {
		t.Error("should return error for weak password")
	}
}

func TestRegisterDuplicatePhone(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{},
		phoneUsers: map[string]*model.User{"13800138000": {UserID: "existing"}},
		emailUsers: map[string]*model.User{},
	}
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	_, err := svc.Register(context.Background(), &model.RegisterReq{
		UserID:   "alice123",
		Password: "Pass1234",
		Phone:    "13800138000",
	})
	if err == nil {
		t.Error("should return error for duplicate phone")
	}
}

// ---------- login tests ----------

func TestLoginSuccess(t *testing.T) {
	repos := newMockRepos()
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("Pass1234"), bcrypt.DefaultCost)
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Password: string(hashedPwd), Status: 1, Nickname: "Alice"}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	pair, err := svc.Login(context.Background(), &model.LoginReq{UserID: "alice123", Password: "Pass1234", Platform: "web"})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Error("tokens should not be empty")
	}
	if pair.UserID != "alice123" {
		t.Errorf("UserID = %s, want alice123", pair.UserID)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	repos := newMockRepos()
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("CorrectPass1"), bcrypt.DefaultCost)
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Password: string(hashedPwd), Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	_, err := svc.Login(context.Background(), &model.LoginReq{UserID: "alice123", Password: "WrongPass1", Platform: "web"})
	if err == nil {
		t.Error("should return error for wrong password")
	}
}

func TestLoginUserNotFound(t *testing.T) {
	repos := newMockRepos()
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	_, err := svc.Login(context.Background(), &model.LoginReq{UserID: "ghost", Password: "Pass1234", Platform: "web"})
	if err == nil {
		t.Error("should return error for non-existent user")
	}
}

func TestLoginDisabled(t *testing.T) {
	repos := newMockRepos()
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("Pass1234"), bcrypt.DefaultCost)
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Password: string(hashedPwd), Status: 2}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	_, err := svc.Login(context.Background(), &model.LoginReq{UserID: "alice123", Password: "Pass1234", Platform: "web"})
	if err == nil {
		t.Error("should return error for disabled user")
	}
}

// ---------- refresh tests ----------

func TestRefreshInvalidToken(t *testing.T) {
	repos := newMockRepos()
	rdb, _ := startTestRedis(t)
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, rdb, hub, cfg)

	_, err := svc.Refresh(context.Background(), "invalid-token")
	if err == nil {
		t.Error("should return error for invalid token")
	}
}

func TestRefreshRevokedToken(t *testing.T) {
	repos := newMockRepos()
	// User must exist with status=1 for the code path to reach Redis check
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("Pass1234"), bcrypt.DefaultCost)
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Password: string(hashedPwd), Status: 1, Nickname: "Alice"}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	rdb, _ := startTestRedis(t)
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, rdb, hub, cfg)

	// Generate a valid refresh token but don't store it in Redis
	refreshToken, _, err := jwtMgr.GenerateRefreshToken("alice123", "web")
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	// Token is valid but not in Redis -> "revoked"
	_, err = svc.Refresh(context.Background(), refreshToken)
	if err == nil {
		t.Error("should return error for revoked token")
	}
}

func TestRefreshExpiredToken(t *testing.T) {
	repos := newMockRepos()
	rdb, _ := startTestRedis(t)
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, rdb, hub, cfg)

	// Create a JWT manager with zero-expiry tokens
	zeroMgr := jwt.NewJWTManagerFromKeys(
		func() *rsa.PrivateKey {
			k, _ := rsa.GenerateKey(rand.Reader, 2048)
			return k
		}(),
		func() *rsa.PublicKey {
			k, _ := rsa.GenerateKey(rand.Reader, 2048)
			return &k.PublicKey
		}(),
		-time.Hour, // negative = expired
		-time.Hour,
	)
	refreshToken, _, err := zeroMgr.GenerateRefreshToken("alice123", "web")
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}

	// Token should be expired
	_, err = svc.Refresh(context.Background(), refreshToken)
	if err == nil {
		t.Error("should return error for expired token")
	}
}

func TestRefreshSuccess(t *testing.T) {
	repos := newMockRepos()
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("Pass1234"), bcrypt.DefaultCost)
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Password: string(hashedPwd), Status: 1, Nickname: "Alice"}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	rdb, mr := startTestRedis(t)
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, rdb, hub, cfg)

	// Generate refresh token and store it in Redis (as Login would do)
	refreshToken, _, err := jwtMgr.GenerateRefreshToken("alice123", "web")
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	mr.Set("refresh:alice123:web", refreshToken)

	pair, err := svc.Refresh(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("Refresh error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("access token should not be empty")
	}
	if pair.RefreshToken != refreshToken {
		t.Error("refresh token should remain unchanged")
	}
	if pair.UserID != "alice123" {
		t.Errorf("UserID = %s, want alice123", pair.UserID)
	}
}

// ---------- logout tests ----------

func TestLogout(t *testing.T) {
	repos := newMockRepos()
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, nil, hub, cfg)

	token, _, err := jwtMgr.GenerateAccessToken("alice123", "web")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	err = svc.Logout(context.Background(), "alice123", "web", token)
	if err != nil {
		t.Errorf("Logout error: %v", err)
	}
}

func TestLogoutWithRedis(t *testing.T) {
	repos := newMockRepos()
	rdb, mr := startTestRedis(t)
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, rdb, hub, cfg)

	// Pre-populate Redis with refresh and online keys
	mr.Set("refresh:alice123:web", "some-refresh-token")
	mr.Set("online:alice123", "1")
	mr.SAdd("conn_map:alice123", "conn-1")

	token, _, err := jwtMgr.GenerateAccessToken("alice123", "web")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	err = svc.Logout(context.Background(), "alice123", "web", token)
	if err != nil {
		t.Errorf("Logout error: %v", err)
	}

	// Verify Redis keys were cleaned up
	if mr.Exists("refresh:alice123:web") {
		t.Error("refresh token should be deleted")
	}
	if mr.Exists("online:alice123") {
		t.Error("online status should be deleted")
	}
	if mr.Exists("conn_map:alice123") {
		t.Error("conn map should be deleted")
	}
}

func TestLogoutParseFail(t *testing.T) {
	repos := newMockRepos()
	rdb, _ := startTestRedis(t)
	cfg, jwtMgr, hub := newTestServices(repos)
	svc := newAuthService(repos, jwtMgr, rdb, hub, cfg)

	// Logout with an invalid token should still return nil (best-effort cleanup)
	err := svc.Logout(context.Background(), "alice123", "web", "invalid-token")
	if err != nil {
		t.Errorf("Logout should not return error for invalid token: %v", err)
	}
}

// ---------- user profile tests ----------

func TestGetProfile(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Nickname: "Alice", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	svc := newUserService(repos)

	vo, err := svc.GetProfile(context.Background(), "alice123")
	if err != nil {
		t.Fatalf("GetProfile error: %v", err)
	}
	if vo.UserID != "alice123" {
		t.Errorf("UserID = %s, want alice123", vo.UserID)
	}
	if vo.Nickname != "Alice" {
		t.Errorf("Nickname = %s, want Alice", vo.Nickname)
	}
}

func TestGetProfileNotFound(t *testing.T) {
	repos := newMockRepos()
	svc := newUserService(repos)

	_, err := svc.GetProfile(context.Background(), "ghost")
	if err == nil {
		t.Error("should return error for non-existent user")
	}
}

// ---------- GetOtherProfile tests ----------

func TestGetOtherProfileFriend(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"bob456": {UserID: "bob456", Nickname: "Bob", AvatarURL: "https://example.com/bob.png", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	friendRepo := newMockFriendRepo()
	// Add friend relationship: alice -> bob with remark "Bobby"
	friendRepo.Create(context.Background(), "alice123", "bob456", "Bobby")
	repos.Friend = friendRepo
	svc := newUserService(repos)

	vo, err := svc.GetOtherProfile(context.Background(), "alice123", "bob456")
	if err != nil {
		t.Fatalf("GetOtherProfile error: %v", err)
	}
	if vo.UserID != "bob456" {
		t.Errorf("UserID = %s, want bob456", vo.UserID)
	}
	if !vo.IsFriend {
		t.Error("should be friend")
	}
	if vo.Remark != "Bobby" {
		t.Errorf("Remark = %s, want Bobby", vo.Remark)
	}
	if vo.Nickname != "Bob" {
		t.Errorf("Nickname = %s, want Bob", vo.Nickname)
	}
}

func TestGetOtherProfileNonFriend(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"bob456": {UserID: "bob456", Nickname: "Bob", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	svc := newUserService(repos)

	vo, err := svc.GetOtherProfile(context.Background(), "alice123", "bob456")
	if err != nil {
		t.Fatalf("GetOtherProfile error: %v", err)
	}
	if vo.UserID != "bob456" {
		t.Errorf("UserID = %s, want bob456", vo.UserID)
	}
	if vo.IsFriend {
		t.Error("should NOT be friend")
	}
	if vo.Remark != "" {
		t.Errorf("Remark should be empty for non-friend, got %s", vo.Remark)
	}
}

func TestGetOtherProfileNotFound(t *testing.T) {
	repos := newMockRepos()
	svc := newUserService(repos)

	_, err := svc.GetOtherProfile(context.Background(), "alice123", "ghost")
	if err == nil {
		t.Error("should return error for non-existent user")
	}
}

// ---------- UpdateProfile tests ----------

func TestUpdateProfileNickname(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Nickname: "Alice", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	svc := newUserService(repos)

	vo, err := svc.UpdateProfile(context.Background(), "alice123", &model.UpdateProfileReq{Nickname: "AliceNew"})
	if err != nil {
		t.Fatalf("UpdateProfile error: %v", err)
	}
	if vo == nil {
		t.Fatal("result should not be nil")
	}
	if vo.Nickname != "AliceNew" {
		t.Errorf("Nickname = %s, want AliceNew", vo.Nickname)
	}
}

func TestUpdateProfilePhoneUnique(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Nickname: "Alice", Status: 1}},
		phoneUsers: map[string]*model.User{"13800138001": {UserID: "other"}},
		emailUsers: map[string]*model.User{},
	}
	svc := newUserService(repos)

	_, err := svc.UpdateProfile(context.Background(), "alice123", &model.UpdateProfileReq{Phone: "13800138001"})
	if err == nil {
		t.Error("should return error for duplicate phone")
	}
}

func TestUpdateProfilePhoneSuccess(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Nickname: "Alice", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	svc := newUserService(repos)

	vo, err := svc.UpdateProfile(context.Background(), "alice123", &model.UpdateProfileReq{Phone: "13800138000"})
	if err != nil {
		t.Fatalf("UpdateProfile error: %v", err)
	}
	if vo == nil {
		t.Fatal("result should not be nil")
	}
}

func TestUpdateProfileEmail(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Nickname: "Alice", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	svc := newUserService(repos)

	vo, err := svc.UpdateProfile(context.Background(), "alice123", &model.UpdateProfileReq{Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("UpdateProfile error: %v", err)
	}
	if vo == nil {
		t.Fatal("result should not be nil")
	}
}

// ---------- Search tests ----------

func TestSearch(t *testing.T) {
	repos := newMockRepos()
	mockUser := &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Nickname: "Alice", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	repos.User = mockUser
	svc := newUserService(repos)

	result, err := svc.Search(context.Background(), "alice123", &model.SearchReq{Keyword: "Alice", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if len(result.List) != 1 {
		t.Errorf("len(List) = %d, want 1", len(result.List))
	}
	if result.List[0].UserID != "alice123" {
		t.Errorf("List[0].UserID = %s, want alice123", result.List[0].UserID)
	}
	if result.Page != 1 {
		t.Errorf("Page = %d, want 1", result.Page)
	}
	if result.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", result.PageSize)
	}
}

func TestSearchCustomPagination(t *testing.T) {
	repos := newMockRepos()
	mockUser := &mockUserRepo{
		users:      map[string]*model.User{"alice123": {UserID: "alice123", Nickname: "Alice", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	repos.User = mockUser
	svc := newUserService(repos)

	result, err := svc.Search(context.Background(), "alice123", &model.SearchReq{Keyword: "Alice", Page: 2, PageSize: 10})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if result.Page != 2 {
		t.Errorf("Page = %d, want 2", result.Page)
	}
	if result.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", result.PageSize)
	}
}

func TestSearchEmpty(t *testing.T) {
	repos := newMockRepos()
	svc := newUserService(repos)

	result, err := svc.Search(context.Background(), "alice123", &model.SearchReq{Keyword: "Nonexistent", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if len(result.List) != 0 {
		t.Errorf("len(List) = %d, want 0", len(result.List))
	}
}

// ---------- FriendService tests ----------

func TestSendFriendRequestSuccess(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"bob456": {UserID: "bob456", Nickname: "Bob", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	id, err := svc.SendRequest(context.Background(), "alice123", &model.SendFriendRequestReq{ToUserID: "bob456", Message: "Hello!"})
	if err != nil {
		t.Fatalf("SendRequest error: %v", err)
	}
	if id <= 0 {
		t.Errorf("request ID should be > 0, got %d", id)
	}
}

func TestSendFriendRequestSelf(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	_, err := svc.SendRequest(context.Background(), "alice123", &model.SendFriendRequestReq{ToUserID: "alice123"})
	if err == nil {
		t.Error("should return error for self-friend request")
	}
}

func TestSendFriendRequestTargetNotFound(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	_, err := svc.SendRequest(context.Background(), "alice123", &model.SendFriendRequestReq{ToUserID: "ghost"})
	if err == nil {
		t.Error("should return error for non-existent target")
	}
}

func TestSendFriendRequestAlreadyFriend(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"bob456": {UserID: "bob456", Nickname: "Bob", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	friendRepo := newMockFriendRepo()
	friendRepo.Create(context.Background(), "alice123", "bob456", "")
	repos.Friend = friendRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	_, err := svc.SendRequest(context.Background(), "alice123", &model.SendFriendRequestReq{ToUserID: "bob456"})
	if err == nil {
		t.Error("should return error for already-friend")
	}
}

func TestSendFriendRequestPendingExists(t *testing.T) {
	repos := newMockRepos()
	repos.User = &mockUserRepo{
		users:      map[string]*model.User{"bob456": {UserID: "bob456", Nickname: "Bob", Status: 1}},
		phoneUsers: map[string]*model.User{},
		emailUsers: map[string]*model.User{},
	}
	friendReqRepo := newMockFriendRequestRepo()
	friendReqRepo.hasPending = true
	repos.FriendReq = friendReqRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	_, err := svc.SendRequest(context.Background(), "alice123", &model.SendFriendRequestReq{ToUserID: "bob456"})
	if err == nil {
		t.Error("should return error when pending request exists")
	}
}

// ---------- ListRequests tests ----------

func TestListRequestsEmpty(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	requests, total, err := svc.ListRequests(context.Background(), "alice123", 1, 20)
	if err != nil {
		t.Fatalf("ListRequests error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(requests) != 0 {
		t.Errorf("len(requests) = %d, want 0", len(requests))
	}
}

func TestListRequestsWithItems(t *testing.T) {
	repos := newMockRepos()
	friendReqRepo := newMockFriendRequestRepo()
	friendReqRepo.addRequest("bob456", "alice123", "Hello!", 0)
	friendReqRepo.addRequest("charlie", "alice123", "Hi!", 0)
	repos.FriendReq = friendReqRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	requests, total, err := svc.ListRequests(context.Background(), "alice123", 1, 20)
	if err != nil {
		t.Fatalf("ListRequests error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(requests) != 2 {
		t.Errorf("len(requests) = %d, want 2", len(requests))
	}
}

// ---------- AcceptRequest tests ----------

func TestAcceptRequestNotFound(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.AcceptRequest(context.Background(), "alice123", 999)
	if err == nil {
		t.Error("should return error for non-existent request")
	}
}

func TestAcceptRequestWrongUser(t *testing.T) {
	repos := newMockRepos()
	friendReqRepo := newMockFriendRequestRepo()
	friendReqRepo.addRequest("charlie", "bob456", "Be my friend!", 0)
	repos.FriendReq = friendReqRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.AcceptRequest(context.Background(), "alice123", 1)
	if err == nil {
		t.Error("should return error for wrong to-user")
	}
}

func TestAcceptRequestAlreadyProcessed(t *testing.T) {
	repos := newMockRepos()
	friendReqRepo := newMockFriendRequestRepo()
	friendReqRepo.addRequest("charlie", "alice123", "Be my friend!", 1) // already accepted
	repos.FriendReq = friendReqRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.AcceptRequest(context.Background(), "alice123", 1)
	if err == nil {
		t.Error("should return error for already processed request")
	}
}

func TestAcceptRequestSuccess(t *testing.T) {
	db := newTestDB(t)
	if db == nil {
		t.Skip("MySQL not available, skipping AcceptRequest success test")
	}

	// Create mock repos
	mockUser := newMockRepos().User
	mockFriend := newMockFriendRepo()
	mockFriendReq := newMockFriendRequestRepo()
	mockConv := newMockConversationRepo()
	mockMsg := newMockMessageRepo()
	mockFriendReq.addRequest("charlie", "alice123", "Be my friend!", 0)

	// Create full Repositories with real DB for Transaction support
	fullRepos := repository.NewRepositories(db, nil)
	fullRepos.User = mockUser
	fullRepos.Friend = mockFriend
	fullRepos.FriendReq = mockFriendReq
	fullRepos.Conversation = mockConv
	fullRepos.Message = mockMsg

	hub := newTestHub()
	svc := newFriendService(fullRepos, hub, nil)

	err := svc.AcceptRequest(context.Background(), "alice123", 1)
	if err != nil {
		t.Fatalf("AcceptRequest error: %v", err)
	}

	// Verify friendship was created both ways
	isFriend, _ := mockFriend.IsFriend(context.Background(), "alice123", "charlie")
	if !isFriend {
		t.Error("alice123 should be friend of charlie")
	}
	isFriend2, _ := mockFriend.IsFriend(context.Background(), "charlie", "alice123")
	if !isFriend2 {
		t.Error("charlie should be friend of alice123")
	}

	// Verify request status updated
	req, _ := mockFriendReq.GetByID(context.Background(), 1)
	if req == nil || req.Status != 1 {
		t.Error("request status should be accepted (1)")
	}

	// Verify conversations were created
	convID := "single_alice123_charlie"
	_, err = mockConv.GetByID(context.Background(), "alice123", convID)
	if err != nil {
		t.Errorf("conversation for alice123 should exist: %v", err)
	}
	_, err = mockConv.GetByID(context.Background(), "charlie", convID)
	if err != nil {
		t.Errorf("conversation for charlie should exist: %v", err)
	}
}

// ---------- RejectRequest tests ----------

func TestRejectRequestNotFound(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.RejectRequest(context.Background(), "alice123", 999)
	if err == nil {
		t.Error("should return error for non-existent request")
	}
}

func TestRejectRequestWrongUser(t *testing.T) {
	repos := newMockRepos()
	friendReqRepo := newMockFriendRequestRepo()
	friendReqRepo.addRequest("charlie", "bob456", "Be my friend!", 0)
	repos.FriendReq = friendReqRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.RejectRequest(context.Background(), "alice123", 1)
	if err == nil {
		t.Error("should return error for wrong to-user")
	}
}

func TestRejectRequestAlreadyProcessed(t *testing.T) {
	repos := newMockRepos()
	friendReqRepo := newMockFriendRequestRepo()
	friendReqRepo.addRequest("charlie", "alice123", "Be my friend!", 1) // already accepted
	repos.FriendReq = friendReqRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.RejectRequest(context.Background(), "alice123", 1)
	if err == nil {
		t.Error("should return error for already processed request")
	}
}

func TestRejectRequestSuccess(t *testing.T) {
	repos := newMockRepos()
	friendReqRepo := newMockFriendRequestRepo()
	friendReqRepo.addRequest("charlie", "alice123", "Be my friend!", 0)
	repos.FriendReq = friendReqRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.RejectRequest(context.Background(), "alice123", 1)
	if err != nil {
		t.Fatalf("RejectRequest error: %v", err)
	}

	// Verify request status updated
	req, _ := friendReqRepo.GetByID(context.Background(), 1)
	if req == nil || req.Status != 2 {
		t.Error("request status should be rejected (2)")
	}
}

// ---------- Delete friend tests ----------

func TestDeleteFriendSuccess(t *testing.T) {
	repos := newMockRepos()
	friendRepo := newMockFriendRepo()
	friendRepo.Create(context.Background(), "alice123", "bob456", "")
	repos.Friend = friendRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.Delete(context.Background(), "alice123", "bob456")
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	// Verify friend was deleted (only one direction)
	isFriend, _ := friendRepo.IsFriend(context.Background(), "alice123", "bob456")
	if isFriend {
		t.Error("friend should be deleted")
	}
}

func TestDeleteFriendNotFriend(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.Delete(context.Background(), "alice123", "bob456")
	if err == nil {
		t.Error("should return error for non-friend")
	}
}

// ---------- List friends tests ----------

func TestListFriendsSuccess(t *testing.T) {
	repos := newMockRepos()
	friendRepo := newMockFriendRepo()
	friendRepo.Create(context.Background(), "alice123", "bob456", "Bobby")
	friendRepo.Create(context.Background(), "alice123", "charlie", "")
	repos.Friend = friendRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	friends, total, err := svc.List(context.Background(), "alice123", 1, 20)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(friends) != 2 {
		t.Errorf("len(friends) = %d, want 2", len(friends))
	}
}

func TestListFriendsEmpty(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	friends, total, err := svc.List(context.Background(), "alice123", 1, 20)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(friends) != 0 {
		t.Errorf("len(friends) = %d, want 0", len(friends))
	}
}

// ---------- SetRemark tests ----------

func TestSetRemarkSuccess(t *testing.T) {
	repos := newMockRepos()
	friendRepo := newMockFriendRepo()
	friendRepo.Create(context.Background(), "alice123", "bob456", "")
	repos.Friend = friendRepo
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.SetRemark(context.Background(), "alice123", "bob456", "Bestie")
	if err != nil {
		t.Fatalf("SetRemark error: %v", err)
	}

	// Verify remark was updated
	f, _ := friendRepo.GetFriend(context.Background(), "alice123", "bob456")
	if f == nil || f.Remark != "Bestie" {
		t.Errorf("Remark = %s, want Bestie", f.Remark)
	}
}

func TestSetRemarkNotFriend(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newFriendService(repos, hub, nil)

	err := svc.SetRemark(context.Background(), "alice123", "bob456", "Bestie")
	if err == nil {
		t.Error("should return error for non-friend")
	}
}

// ---------- MessageService tests ----------

func TestSendMessageSuccess(t *testing.T) {
	repos := newMockRepos()
	rdb, _ := startTestRedis(t)
	friendRepo := newMockFriendRepo()
	// Add friend relationship so friend check passes
	friendRepo.Create(context.Background(), "alice123", "bob456", "")
	repos.Friend = friendRepo
	repos.Message = newMockMessageRepo()
	hub := newTestHub()
	svc := newMessageService(repos, hub, rdb)

	// Use single_ prefixed conversation ID for friend check
	convID := "single_alice123_bob456"
	resp, err := svc.SendMessage(context.Background(), "alice123", &model.SendMsgReq{
		ConversationID: convID,
		ClientMsgID:    "msg-001",
		ContentType:    1,
		Content:        "Hello Bob!",
	})
	if err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if resp.Seq <= 0 {
		t.Errorf("Seq = %d, want > 0", resp.Seq)
	}
	if resp.ServerMsgID == "" {
		t.Error("ServerMsgID should not be empty")
	}
	if resp.SendTime <= 0 {
		t.Error("SendTime should be > 0")
	}
}

func TestSendMessageDedupHit(t *testing.T) {
	repos := newMockRepos()
	rdb, mr := startTestRedis(t)
	friendRepo := newMockFriendRepo()
	friendRepo.Create(context.Background(), "alice123", "bob456", "")
	repos.Friend = friendRepo
	msgRepo := newMockMessageRepo()
	repos.Message = msgRepo
	hub := newTestHub()
	svc := newMessageService(repos, hub, rdb)

	convID := "single_alice123_bob456"

	// Pre-set the dedup key so SETNX fails
	mr.Set("dedup:msg:msg-001", "1")

	// Pre-create a message with this clientMsgID
	msgRepo.msgs = append(msgRepo.msgs, &model.Message{
		ConversationID: convID,
		Seq:            5,
		ClientMsgID:    "msg-001",
		ServerMsgID:    "srv-001",
	})

	resp, err := svc.SendMessage(context.Background(), "alice123", &model.SendMsgReq{
		ConversationID: convID,
		ClientMsgID:    "msg-001",
		ContentType:    1,
		Content:        "Hello Bob!",
	})
	if err != nil {
		t.Fatalf("SendMessage error (dedup): %v", err)
	}
	if resp.Seq != 5 {
		t.Errorf("Seq = %d, want 5 (existing seq)", resp.Seq)
	}
	if resp.ServerMsgID != "srv-001" {
		t.Errorf("ServerMsgID = %s, want srv-001", resp.ServerMsgID)
	}
}

func TestSendMessageNotFriend(t *testing.T) {
	repos := newMockRepos()
	rdb, _ := startTestRedis(t)
	hub := newTestHub()
	svc := newMessageService(repos, hub, rdb)

	_, err := svc.SendMessage(context.Background(), "alice123", &model.SendMsgReq{
		ConversationID: "single_alice123_bob456",
		ClientMsgID:    "msg-001",
		ContentType:    1,
		Content:        "Hello Bob!",
	})
	if err == nil {
		t.Error("should return error when not friends")
	}
}

func TestSendMessageEmptyConversation(t *testing.T) {
	repos := newMockRepos()
	rdb, _ := startTestRedis(t)
	// Non-single conversation skips friend check
	hub := newTestHub()
	svc := newMessageService(repos, hub, rdb)

	// Use a non-single conv ID to bypass friend check path
	resp, err := svc.SendMessage(context.Background(), "alice123", &model.SendMsgReq{
		ConversationID: "group_chat_room_1",
		ClientMsgID:    "msg-group-001",
		ContentType:    1,
		Content:        "Hello everyone!",
	})
	if err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if resp.Seq <= 0 {
		t.Errorf("Seq = %d, want > 0", resp.Seq)
	}
}

// ---------- History tests ----------

func TestHistoryEmptyConversation(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	resp, err := svc.History(context.Background(), "alice123", &model.HistoryReq{
		ConversationID: "conv-empty",
		Count:          20,
	})
	if err != nil {
		t.Fatalf("History error: %v", err)
	}
	if resp.HasMore {
		t.Error("HasMore should be false for empty conversation")
	}
	if len(resp.List) != 0 {
		t.Errorf("len(List) = %d, want 0", len(resp.List))
	}
	if resp.MaxSeq != 0 {
		t.Errorf("MaxSeq = %d, want 0", resp.MaxSeq)
	}
}

func TestHistoryNormal(t *testing.T) {
	repos := newMockRepos()
	msgRepo := newMockMessageRepo()
	repos.Message = msgRepo
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	// Add messages to the conversation
	convID := "conv-001"
	for i := int64(1); i <= 5; i++ {
		msgRepo.msgs = append(msgRepo.msgs, &model.Message{
			ConversationID: convID,
			Seq:            i,
			SenderID:       "alice123",
			MsgType:        1,
			Content:        "msg " + string(rune('0'+i)),
		})
	}
	msgRepo.maxSeq[convID] = 5
	msgRepo.minSeq[convID] = 1

	resp, err := svc.History(context.Background(), "alice123", &model.HistoryReq{
		ConversationID: convID,
		StartSeq:       0,
		Count:          3,
	})
	if err != nil {
		t.Fatalf("History error: %v", err)
	}
	// Should return 3 most recent messages in reverse order
	if len(resp.List) != 3 {
		t.Errorf("len(List) = %d, want 3", len(resp.List))
	}
	if resp.MaxSeq != 5 {
		t.Errorf("MaxSeq = %d, want 5", resp.MaxSeq)
	}
	if resp.MinSeq != 1 {
		t.Errorf("MinSeq = %d, want 1", resp.MinSeq)
	}
	// Messages should be in reverse order (newest first)
	if len(resp.List) > 0 && resp.List[0].Seq < resp.List[len(resp.List)-1].Seq {
		t.Error("messages should be in reverse order (newest first)")
	}
}

func TestHistoryHasMore(t *testing.T) {
	repos := newMockRepos()
	msgRepo := newMockMessageRepo()
	repos.Message = msgRepo
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	convID := "conv-002"
	for i := int64(1); i <= 5; i++ {
		msgRepo.msgs = append(msgRepo.msgs, &model.Message{
			ConversationID: convID,
			Seq:            i,
			SenderID:       "alice123",
			Content:        "msg " + string(rune('0'+i)),
		})
	}
	msgRepo.maxSeq[convID] = 5

	// Request count=2 with startSeq=0 -> fetches 3 items (count+1) from latest, hasMore=true
	resp, err := svc.History(context.Background(), "alice123", &model.HistoryReq{
		ConversationID: convID,
		StartSeq:       0,
		Count:          2,
	})
	if err != nil {
		t.Fatalf("History error: %v", err)
	}
	if !resp.HasMore {
		t.Error("HasMore should be true when there are more messages")
	}
	if len(resp.List) != 2 {
		t.Errorf("len(List) = %d, want 2 (truncated to count)", len(resp.List))
	}
}

func TestHistoryNoMore(t *testing.T) {
	repos := newMockRepos()
	msgRepo := newMockMessageRepo()
	repos.Message = msgRepo
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	convID := "conv-003"
	for i := int64(1); i <= 3; i++ {
		msgRepo.msgs = append(msgRepo.msgs, &model.Message{
			ConversationID: convID,
			Seq:            i,
			SenderID:       "alice123",
			Content:        "msg " + string(rune('0'+i)),
		})
	}
	msgRepo.maxSeq[convID] = 3

	// Request count=5, but only 3 messages exist => hasMore=false
	resp, err := svc.History(context.Background(), "alice123", &model.HistoryReq{
		ConversationID: convID,
		StartSeq:       0,
		Count:          5,
	})
	if err != nil {
		t.Fatalf("History error: %v", err)
	}
	if resp.HasMore {
		t.Error("HasMore should be false when all messages are returned")
	}
	if len(resp.List) != 3 {
		t.Errorf("len(List) = %d, want 3", len(resp.List))
	}
}

// ---------- MarkRead tests ----------

func TestMarkReadSuccess(t *testing.T) {
	repos := newMockRepos()
	msgRepo := newMockMessageRepo()
	msgRepo.maxSeq["conv-read"] = 10
	repos.Message = msgRepo
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	err := svc.MarkRead(context.Background(), "alice123", &model.MarkReadReq{
		ConversationID: "conv-read",
		ReadSeq:        5,
	})
	if err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}
}

func TestMarkReadSeqExceedsMax(t *testing.T) {
	repos := newMockRepos()
	msgRepo := newMockMessageRepo()
	msgRepo.maxSeq["conv-read"] = 10
	repos.Message = msgRepo
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	err := svc.MarkRead(context.Background(), "alice123", &model.MarkReadReq{
		ConversationID: "conv-read",
		ReadSeq:        20, // > maxSeq 10
	})
	if err == nil {
		t.Error("should return error when readSeq > maxSeq")
	}
}

// ---------- Revoke tests ----------

func TestRevokeSuccess(t *testing.T) {
	repos := newMockRepos()
	msgRepo := newMockMessageRepo()
	repos.Message = msgRepo
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	convID := "conv-revoke"
	// Add a message sent by alice123
	msgRepo.msgs = append(msgRepo.msgs, &model.Message{
		ConversationID: convID,
		Seq:            1,
		SenderID:       "alice123",
		ClientMsgID:    "msg-revoke-001",
		Content:        "Oops!",
	})

	err := svc.Revoke(context.Background(), "alice123", &model.RevokeMsgReq{
		ConversationID: convID,
		ClientMsgID:    "msg-revoke-001",
	})
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	// Verify message is revoked
	msg, _ := msgRepo.GetByClientMsgID(context.Background(), "msg-revoke-001")
	if msg == nil || !msg.IsRevoked {
		t.Error("message should be revoked")
	}
}

func TestRevokeNotOwnMessage(t *testing.T) {
	repos := newMockRepos()
	msgRepo := newMockMessageRepo()
	repos.Message = msgRepo
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	convID := "conv-revoke"
	msgRepo.msgs = append(msgRepo.msgs, &model.Message{
		ConversationID: convID,
		Seq:            1,
		SenderID:       "bob456", // different sender
		ClientMsgID:    "msg-revoke-002",
		Content:        "Bob's message",
	})

	err := svc.Revoke(context.Background(), "alice123", &model.RevokeMsgReq{
		ConversationID: convID,
		ClientMsgID:    "msg-revoke-002",
	})
	if err == nil {
		t.Error("should return error when revoking someone else's message")
	}
}

func TestRevokeMessageNotFound(t *testing.T) {
	repos := newMockRepos()
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	err := svc.Revoke(context.Background(), "alice123", &model.RevokeMsgReq{
		ConversationID: "conv-revoke",
		ClientMsgID:    "non-existent",
	})
	if err == nil {
		t.Error("should return error for non-existent message")
	}
}

func TestRevokeAlreadyRevoked(t *testing.T) {
	repos := newMockRepos()
	msgRepo := newMockMessageRepo()
	repos.Message = msgRepo
	hub := newTestHub()
	svc := newMessageService(repos, hub, nil)

	convID := "conv-revoke"
	msgRepo.msgs = append(msgRepo.msgs, &model.Message{
		ConversationID: convID,
		Seq:            1,
		SenderID:       "alice123",
		ClientMsgID:    "msg-revoke-003",
		Content:        "Already revoked",
		IsRevoked:      true,
	})

	// Revoking an already revoked message should succeed (idempotent)
	err := svc.Revoke(context.Background(), "alice123", &model.RevokeMsgReq{
		ConversationID: convID,
		ClientMsgID:    "msg-revoke-003",
	})
	if err != nil {
		t.Fatalf("Revoke already-revoked message error: %v", err)
	}
}

// ---------- ConversationService tests ----------

func TestConversationList(t *testing.T) {
	repos := newMockRepos()
	svc := newConversationService(repos)

	result, err := svc.List(context.Background(), "alice123", 1, 20)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if result == nil {
		t.Error("result should not be nil (empty page expected)")
	}
}

func TestConversationPinSuccess(t *testing.T) {
	repos := newMockRepos()
	convRepo := newMockConversationRepo()
	convRepo.CreateIfNotExistTx(context.Background(), nil, "alice123", "conv-001", 1, "bob456")
	repos.Conversation = convRepo
	svc := newConversationService(repos)

	err := svc.Pin(context.Background(), "alice123", "conv-001", true)
	if err != nil {
		t.Fatalf("Pin error: %v", err)
	}

	// Verify conversation is pinned
	conv, _ := convRepo.GetByID(context.Background(), "alice123", "conv-001")
	if conv == nil || !conv.IsPinned {
		t.Error("conversation should be pinned")
	}
}

func TestConversationDeleteSuccess(t *testing.T) {
	repos := newMockRepos()
	convRepo := newMockConversationRepo()
	convRepo.CreateIfNotExistTx(context.Background(), nil, "alice123", "conv-001", 1, "bob456")
	repos.Conversation = convRepo
	svc := newConversationService(repos)

	err := svc.Delete(context.Background(), "alice123", "conv-001")
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	// Verify conversation is deleted
	_, err = convRepo.GetByID(context.Background(), "alice123", "conv-001")
	if err == nil {
		t.Error("conversation should be deleted")
	}
}

func TestConversationDeleteNotFound(t *testing.T) {
	repos := newMockRepos()
	svc := newConversationService(repos)

	err := svc.Delete(context.Background(), "alice123", "non-existent")
	if err == nil {
		t.Error("should return error for non-existent conversation")
	}
}
