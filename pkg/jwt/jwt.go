package jwt

import (
	"crypto/rsa"
	"errors"
	"os"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"

	"github.com/tianlu1990s/gim/pkg/snowflake"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims 自定义 JWT Claims，在标准 RegisteredClaims 基础上追加业务字段。
// JTI（ID 字段）用于 Token 黑名单机制：logout 时将 JTI 存入 Redis，后续请求校验黑名单。
type Claims struct {
	UserID   string `json:"userId"`
	Platform string `json:"platform"`
	jwtv5.RegisteredClaims
}

// JWTManager 管理 Token 的生成与验证。
// 使用 RS256 非对称加密：私钥签名（仅服务端持有）、公钥验证（可分发到其他微服务）。
// 相比 HS256（对称加密），RS256 更安全——私钥泄露不会影响历史 Token，公钥可安全分发。
type JWTManager struct {
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
	accessExpire  time.Duration
	refreshExpire time.Duration
}

func NewJWTManager(privateKeyPath, publicKeyPath string, accessExp, refreshExp time.Duration) *JWTManager {
	privBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		panic("failed to read private key: " + err.Error())
	}
	pubBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		panic("failed to read public key: " + err.Error())
	}
	privKey, err := jwtv5.ParseRSAPrivateKeyFromPEM(privBytes)
	if err != nil {
		panic("failed to parse private key: " + err.Error())
	}
	pubKey, err := jwtv5.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		panic("failed to parse public key: " + err.Error())
	}
	return &JWTManager{
		privateKey:    privKey,
		publicKey:     pubKey,
		accessExpire:  accessExp,
		refreshExpire: refreshExp,
	}
}

// GenerateAccessToken 生成短期访问 Token（默认 24h）。
// 返回 token 字符串 + 过期 Unix 时间戳。
func (m *JWTManager) GenerateAccessToken(userID, platform string) (string, int64, error) {
	now := time.Now()
	expireAt := now.Add(m.accessExpire).Unix()
	claims := &Claims{
		UserID:   userID,
		Platform: platform,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ID:        snowflake.Generate().String(), // JTI — 全局唯一，用于黑名单
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(m.accessExpire)),
		},
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(m.privateKey)
	return tokenStr, expireAt, err
}

// GenerateRefreshToken 生成长期刷新 Token（默认 7 天），用于无感续期 accessToken。
func (m *JWTManager) GenerateRefreshToken(userID, platform string) (string, int64, error) {
	now := time.Now()
	expireAt := now.Add(m.refreshExpire).Unix()
	claims := &Claims{
		UserID:   userID,
		Platform: platform,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ID:        snowflake.Generate().String(),
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(m.refreshExpire)),
		},
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims)
	tokenStr, err := token.SignedString(m.privateKey)
	return tokenStr, expireAt, err
}

// NewJWTManagerFromKeys 使用已生成的 RSA 密钥对创建 JWTManager，用于测试。
func NewJWTManagerFromKeys(privKey *rsa.PrivateKey, pubKey *rsa.PublicKey, accessExp, refreshExp time.Duration) *JWTManager {
	return &JWTManager{
		privateKey:    privKey,
		publicKey:     pubKey,
		accessExpire:  accessExp,
		refreshExpire: refreshExp,
	}
}

// ParseToken 解析并验证 Token，返回 Claims。
// 验证包括：签名校验、过期时间、JWT 格式。
func (m *JWTManager) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwtv5.ParseWithClaims(tokenStr, &Claims{}, func(t *jwtv5.Token) (any, error) {
		return m.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ValidateToken 仅验证签名，不提取 Claims。用于中间件快速校验。
func (m *JWTManager) ValidateToken(tokenStr string) (*jwtv5.Token, error) {
	return jwtv5.Parse(tokenStr, func(t *jwtv5.Token) (any, error) {
		return m.publicKey, nil
	})
}
