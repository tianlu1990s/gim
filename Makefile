.PHONY: build run test lint migrate clean docker docker-down docker-build deps gen swagger help

APP_NAME := gim
BUILD_DIR := bin
GO ?= go
MAIN := cmd/gim/main.go

# 数据库连接配置（可在命令行覆盖：make migrate-up DB_DSN="..."）
DB_USER ?= gim
DB_PASSWORD ?= gim_pass
DB_HOST ?= localhost
DB_PORT ?= 3306
DB_NAME ?= gim
DB_DSN := $(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?charset=utf8mb4&parseTime=True&loc=Local

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

run: build
	@echo "Running $(APP_NAME)..."
	$(BUILD_DIR)/$(APP_NAME)

test:
	$(GO) test -v -count=1 ./...

test-single:
	@echo "Usage: make test-single TEST=TestName PKG=./path/to/package"
	@echo "Example: make test-single TEST=TestRegister PKG=./internal/service"
	$(GO) test -v -count=1 -run $(TEST) $(PKG)

lint:
	golangci-lint run ./...

# 数据库迁移（需先启动 MySQL）
migrate-up:
	@echo "Running migrations up..."
	migrate -path migrations -database "mysql://$(DB_DSN)" up

migrate-down:
	@echo "Running migrations down (one version)..."
	migrate -path migrations -database "mysql://$(DB_DSN)" down 1

migrate-create:
	@echo "Usage: make migrate-create NAME=create_users_table"
	@echo "Creating migration file..."
	migrate create -ext sql -dir migrations -seq $(NAME)

# Docker 操作
docker:
	docker compose -f deploy/docker-compose.yaml up -d

docker-down:
	docker compose -f deploy/docker-compose.yaml down

docker-logs:
	docker compose -f deploy/docker-compose.yaml logs -f

docker-build:
	docker build -f deploy/docker/Dockerfile -t $(APP_NAME):latest .

# 代码生成（第二阶段使用）
gen:
	@echo "Generating gRPC code from protobuf..."
	protoc --go_out=. --go-grpc_out=. api/**/*.proto

swagger:
	@echo "Generating Swagger documentation..."
	swag init -g cmd/gim/main.go -o docs/swagger

# 清理
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

# 依赖管理
deps:
	$(GO) mod tidy
	$(GO) mod download

deps-check:
	$(GO) mod verify

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  make build          - Build the application"
	@echo "  make run            - Build and run the application"
	@echo "  make test           - Run all tests"
	@echo "  make test-single    - Run specific test (TEST=TestName PKG=./path/to/package)"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make migrate-up     - Run database migrations up"
	@echo "  make migrate-down   - Rollback one migration"
	@echo "  make migrate-create NAME=name - Create new migration file"
	@echo "  make docker         - Start Docker Compose services"
	@echo "  make docker-down    - Stop Docker Compose services"
	@echo "  make docker-logs    - View Docker Compose logs"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make deps           - Tidy and download dependencies"
	@echo "  make deps-check     - Verify dependencies"
	@echo "  make gen            - Generate gRPC code (Phase 2)"
	@echo "  make swagger        - Generate Swagger docs"
