package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
server:
  httpPort: 8080
  readTimeout: 10s
  writeTimeout: 10s

mysql:
  host: 127.0.0.1
  port: 3306
  user: root
  password: secret
  dbname: gim
  maxOpenConns: 100
  maxIdleConns: 10
  connMaxLifetime: 3600s

redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  db: 0
  poolSize: 10

jwt:
  accessTokenExpire: 24h
  refreshTokenExpire: 168h
  privateKeyPath: /tmp/private.pem
  publicKeyPath: /tmp/public.pem

websocket:
  port: 8081
  maxConnPerUser: 5
  maxMessageSize: 4096
  writeWait: 10s
  pongWait: 60s
  pingPeriod: 30s

log:
  level: info
  format: text
  output: stdout

snowflake:
  nodeID: 1
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	cfg := Load()
	if cfg == nil {
		t.Fatal("Load() returned nil")
	}

	if cfg.Server.HTTPPort != 8080 {
		t.Errorf("Server.HTTPPort = %d, want 8080", cfg.Server.HTTPPort)
	}
	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Errorf("Server.ReadTimeout = %v, want 10s", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 10*time.Second {
		t.Errorf("Server.WriteTimeout = %v, want 10s", cfg.Server.WriteTimeout)
	}

	if cfg.MySQL.Host != "127.0.0.1" {
		t.Errorf("MySQL.Host = %s, want 127.0.0.1", cfg.MySQL.Host)
	}
	if cfg.MySQL.Port != 3306 {
		t.Errorf("MySQL.Port = %d, want 3306", cfg.MySQL.Port)
	}
	if cfg.MySQL.User != "root" {
		t.Errorf("MySQL.User = %s, want root", cfg.MySQL.User)
	}
	if cfg.MySQL.DBName != "gim" {
		t.Errorf("MySQL.DBName = %s, want gim", cfg.MySQL.DBName)
	}
	if cfg.MySQL.MaxOpenConns != 100 {
		t.Errorf("MySQL.MaxOpenConns = %d, want 100", cfg.MySQL.MaxOpenConns)
	}

	if cfg.Redis.Host != "127.0.0.1" {
		t.Errorf("Redis.Host = %s, want 127.0.0.1", cfg.Redis.Host)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port = %d, want 6379", cfg.Redis.Port)
	}
	if cfg.Redis.DB != 0 {
		t.Errorf("Redis.DB = %d, want 0", cfg.Redis.DB)
	}

	if cfg.JWT.AccessTokenExpire != 24*time.Hour {
		t.Errorf("JWT.AccessTokenExpire = %v, want 24h", cfg.JWT.AccessTokenExpire)
	}
	if cfg.JWT.RefreshTokenExpire != 168*time.Hour {
		t.Errorf("JWT.RefreshTokenExpire = %v, want 168h", cfg.JWT.RefreshTokenExpire)
	}

	if cfg.WebSocket.Port != 8081 {
		t.Errorf("WebSocket.Port = %d, want 8081", cfg.WebSocket.Port)
	}
	if cfg.WebSocket.MaxConnPerUser != 5 {
		t.Errorf("WebSocket.MaxConnPerUser = %d, want 5", cfg.WebSocket.MaxConnPerUser)
	}
	if cfg.WebSocket.MaxMessageSize != 4096 {
		t.Errorf("WebSocket.MaxMessageSize = %d, want 4096", cfg.WebSocket.MaxMessageSize)
	}
	if cfg.WebSocket.WriteWait != 10*time.Second {
		t.Errorf("WebSocket.WriteWait = %v, want 10s", cfg.WebSocket.WriteWait)
	}
	if cfg.WebSocket.PongWait != 60*time.Second {
		t.Errorf("WebSocket.PongWait = %v, want 60s", cfg.WebSocket.PongWait)
	}
	if cfg.WebSocket.PingPeriod != 30*time.Second {
		t.Errorf("WebSocket.PingPeriod = %v, want 30s", cfg.WebSocket.PingPeriod)
	}

	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %s, want info", cfg.Log.Level)
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %s, want text", cfg.Log.Format)
	}
	if cfg.Log.Output != "stdout" {
		t.Errorf("Log.Output = %s, want stdout", cfg.Log.Output)
	}

	if cfg.Snowflake.NodeID != 1 {
		t.Errorf("Snowflake.NodeID = %d, want 1", cfg.Snowflake.NodeID)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if os.Getenv("CONFIG_TEST_MISSING") == "1" {
		Load()
		return
	}
	tmpDir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestLoadMissingFile")
	cmd.Env = append(os.Environ(), "CONFIG_TEST_MISSING=1")
	cmd.Dir = tmpDir

	err := cmd.Run()
	if err == nil {
		t.Error("expected Load() to exit with non-zero status when config file is missing")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if !exitErr.Success() {
			return
		}
		t.Errorf("expected non-zero exit, got %d", exitErr.ExitCode())
	}
}
