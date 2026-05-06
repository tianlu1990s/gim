package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/tianlu1990s/gim/internal/config"
	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/internal/repository"
	"github.com/tianlu1990s/gim/internal/ws"
	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/jwt"
	"github.com/tianlu1990s/gim/pkg/rediskey"
)

// AuthService 认证业务逻辑接口。
// 认证模块是整个系统的入口——注册、登录、Token 刷新、登出。
type AuthService interface {
	Register(ctx context.Context, req *model.RegisterReq) (*model.User, error)
	Login(ctx context.Context, req *model.LoginReq) (*model.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error)
	Logout(ctx context.Context, userID, platform, accessToken string) error
}

type authService struct {
	userRepo repository.UserRepo
	jwtMgr   *jwt.JWTManager
	rdb      *redis.Client
	hub      *ws.Hub
	cfg      *config.Config
}

func newAuthService(repos *repository.Repositories, jwtMgr *jwt.JWTManager, rdb *redis.Client, hub *ws.Hub, cfg *config.Config) AuthService {
	return &authService{
		userRepo: repos.User,
		jwtMgr:   jwtMgr,
		rdb:      rdb,
		hub:      hub,
		cfg:      cfg,
	}
}

// Register 用户注册。先校验参数格式（userId 格式、密码强度），再检查 userId 是否已存在，
// 密码使用 bcrypt 哈希后写入数据库。bcrypt 自动加盐，相同密码每次哈希结果不同，防止彩虹表攻击。
func (s *authService) Register(ctx context.Context, req *model.RegisterReq) (*model.User, error) {
	if err := validateRegisterReq(req); err != nil {
		return nil, err
	}
	exists, err := s.userRepo.ExistsByID(ctx, req.UserID)
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail(err.Error())
	}
	if exists {
		return nil, errcode.ErrUserAlreadyExists
	}
	// 检查手机号和邮箱唯一性
	if req.Phone != "" {
		phoneExists, _ := s.userRepo.ExistsByPhone(ctx, req.Phone, "")
		if phoneExists {
			return nil, errcode.ErrInvalidParam.WithDetail("手机号已被注册")
		}
	}
	if req.Email != "" {
		emailExists, _ := s.userRepo.ExistsByEmail(ctx, req.Email, "")
		if emailExists {
			return nil, errcode.ErrInvalidParam.WithDetail("邮箱已被注册")
		}
	}
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail("密码加密失败")
	}
	user := &model.User{
		UserID:   req.UserID,
		Nickname: req.Nickname,
		Password: string(hashedPwd),
		Phone:    req.Phone,
		Email:    req.Email,
	}
	if user.Nickname == "" {
		user.Nickname = req.UserID // 昵称默认为 userId
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errcode.ErrInternal.WithDetail(err.Error())
	}
	return user, nil
}

// Login 用户登录。验证 userId+password，检查用户状态，生成 accessToken 和 refreshToken 对。
// refreshToken 存入 Redis（用于后续刷新和吊销），同时通知 WS Hub 踢掉同平台旧连接（单点登录）。
// rdb 为 nil 时跳过 Redis 操作（测试环境兼容）。
func (s *authService) Login(ctx context.Context, req *model.LoginReq) (*model.TokenPair, error) {
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil || user == nil {
		return nil, errcode.ErrUserOrPasswordWrong
	}
	// bcrypt.CompareHashAndPassword 防时序攻击——不提前返回
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errcode.ErrUserOrPasswordWrong
	}
	if user.Status != 1 {
		return nil, errcode.ErrUserDisabled
	}
	accessToken, accessExp, err := s.jwtMgr.GenerateAccessToken(user.UserID, req.Platform)
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail("生成 accessToken 失败")
	}
	refreshToken, refreshExp, err := s.jwtMgr.GenerateRefreshToken(user.UserID, req.Platform)
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail("生成 refreshToken 失败")
	}
	// 存储 refreshToken 到 Redis（测试环境下 rdb 可能为 nil，跳过 Redis 操作）
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, rediskey.RefreshKey(user.UserID, req.Platform),
			refreshToken, s.cfg.JWT.RefreshTokenExpire).Err(); err != nil {
			return nil, errcode.ErrInternal.WithDetail("存储 refreshToken 失败")
		}
	}
	// 通知 Hub 踢掉同平台旧连接（单点登录）
	s.hub.PushToUser(user.UserID, &ws.WSMessage{
		Type: 110,
		Data: map[string]any{"reason": "kicked", "platform": req.Platform},
	})
	return &model.TokenPair{
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		AccessExpireAt:  accessExp,
		RefreshExpireAt: refreshExp,
		UserID:          user.UserID,
	}, nil
}

// Refresh 刷新 accessToken。验证 refreshToken 签名和 Redis 中存在性，检查用户状态，生成新 accessToken。
// refreshToken 本身不轮换（简化实现），仅返回新的 accessToken。
func (s *authService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	claims, err := s.jwtMgr.ParseToken(refreshToken)
	if err != nil {
		return nil, errcode.ErrUnauthorized.WithDetail("refreshToken 无效或已过期")
	}
	// 检查 refreshToken 在 Redis 中是否存在（logout 会删除）
	storedToken, err := s.rdb.Get(ctx, rediskey.RefreshKey(claims.UserID, claims.Platform)).Result()
	if err != nil || storedToken != refreshToken {
		return nil, errcode.ErrUnauthorized.WithDetail("refreshToken 已被吊销")
	}
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || user == nil || user.Status != 1 {
		return nil, errcode.ErrUserDisabled
	}
	newAccessToken, accessExp, err := s.jwtMgr.GenerateAccessToken(claims.UserID, claims.Platform)
	if err != nil {
		return nil, errcode.ErrInternal.WithDetail("生成 accessToken 失败")
	}
	return &model.TokenPair{
		AccessToken:     newAccessToken,
		RefreshToken:    refreshToken, // refreshToken 不变，仅刷新 accessToken
		AccessExpireAt:  accessExp,
		RefreshExpireAt: claims.ExpiresAt.Unix(),
		UserID:          claims.UserID,
	}, nil
}

// Logout 用户登出。将 accessToken 的 JTI 加入 Redis 黑名单（TTL = 剩余有效期），
// 删除 refreshToken，清除在线状态。即使 Token 未过期，鉴权中间件也会拒绝黑名单中的 Token。
func (s *authService) Logout(ctx context.Context, userID, platform, accessToken string) error {
	claims, err := s.jwtMgr.ParseToken(accessToken)
	if err == nil && s.rdb != nil {
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			// JTI 黑名单：TTL 设为 token 剩余有效期，到期自动清理
			s.rdb.Set(ctx, rediskey.BlacklistTokenKey(claims.ID), "1", ttl)
		}
		// 删除 refreshToken，撤销所有刷新能力
		s.rdb.Del(ctx, rediskey.RefreshKey(userID, platform))
		// 清除在线状态
		s.rdb.Del(ctx, rediskey.OnlineKey(userID))
		s.rdb.Del(ctx, rediskey.ConnMapKey(userID))
	}
	return nil
}
