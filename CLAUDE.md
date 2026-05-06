# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

gim — A cloud-native Go IM (Instant Messaging) system. Four-phase development: (1) minimal single-node, (2) microservices + Kafka + MongoDB, (3) admin + monitoring + K8S production, (4) AI Agent integration. See PLAN.md for full details.

## Architecture

- Phase 1: Monolith (Gin HTTP + gorilla/websocket + MySQL + Redis)
- Phase 2: Microservices (gRPC + etcd + Kafka + MongoDB + S3/MinIO/OSS)
- Phase 3: K8S (Helm + Prometheus + Grafana + OpenTelemetry)
- Phase 4: AI Agents (Deepseek/Claude/Local + Tool Use + RAG + Multi-Agent)

## Tech Stack

Go 1.26+, Gin, gorilla/websocket, GORM, golang-migrate, Redis, JWT (RS256), Viper, Zap. Phase 2 adds: gRPC, etcd, Kafka, MongoDB, S3/MinIO/OSS. Phase 4 adds: AI Provider (Deepseek/Claude/Local, anthropic-sdk-go / openai-go), Milvus/pgvector.

## Build & Run

```bash
make build          # Build binary
make run            # Run server
make migrate        # Run DB migrations
make lint           # golangci-lint
make test           # All tests
make test-single    # go test -run TestName ./path/to/package
```

## Documentation

- `PLAN.md` — Full project plan, architecture, data models, TODOs
- `docs/API.md` — Complete API reference (30+ endpoints, WS protocol, error codes)
- `docs/IMPLEMENTATION.md` — Implementation guide with Go code outlines
- `docs/GETTING_STARTED.md` — Zero-basics setup guide and concept primer
- `docs/AI_AGENT.md` — Phase 4 AI Agent integration plan
- `docs/K8S_DEPLOY.md` — Step-by-step K8S cluster setup (1-master 2-worker) and gim deployment guide

## Conventions

- Config: `configs/config.yaml` (Viper)
- Migrations: `migrations/` (sequential SQL files)
- Private code: `internal/` — handlers, services, repositories, models, middleware, ws
- Reusable packages: `pkg/` — jwt, snowflake, resp, errcode
- Entry point: `cmd/gim/main.go`
- Chinese preferred for project discussions
