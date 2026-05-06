package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/tianlu1990s/gim/internal/config"
	"github.com/tianlu1990s/gim/pkg/slog"
)

// Repositories 聚合所有 Repository 接口，作为依赖注入的容器。
// 各 Service 通过 Repositories 获取所需的 Repo，而非直接持有 *gorm.DB。
type Repositories struct {
	User         UserRepo
	Friend       FriendRepo
	FriendReq    FriendRequestRepo
	Conversation ConversationRepo
	Message      MessageRepo
	db           *gorm.DB // 私有，通过 Transaction() 对外提供事务能力
}

// InitMySQL 初始化 MySQL 连接并配置连接池参数。
// 启动阶段失败直接 Fatal，因为数据库不可用意味着服务无法工作。
func InitMySQL(cfg config.MySQLConfig, logger *slog.Logger) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	logger.Info("MySQL connected", "host", cfg.Host, "port", cfg.Port, "db", cfg.DBName)
	return db
}

// InitRedis 初始化 Redis 客户端。
func InitRedis(cfg config.RedisConfig) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	return rdb
}

func NewRepositories(db *gorm.DB, rdb *redis.Client) *Repositories {
	return &Repositories{
		User:         newUserRepo(db),
		Friend:       newFriendRepo(db),
		FriendReq:    newFriendRequestRepo(db),
		Conversation: newConversationRepo(db),
		Message:      newMessageRepo(db, rdb),
		db:           db,
	}
}

// Transaction 提供事务支持。好友同意等场景涉及多表写入，必须在同一事务中完成。
// 使用示例：
//
//	err := repos.Transaction(ctx, func(tx *gorm.DB) error {
//	    // 多表写入...
//	    return nil
//	})
func (r *Repositories) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
