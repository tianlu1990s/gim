# gim
A Go IM Project.

# GIM (Generic Instant Messaging)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8)](https://golang.org/)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()

> **GIM** 是一个高性能、分布式、可扩展的通用即时通讯系统后端架构。它致力于提供稳定可靠的消息收发服务，支持单聊、群聊、系统通知等多种场景，旨在为开发者提供一套开箱即用的 IM 基础设施解决方案。

## 🚀 核心特性

- **高性能架构**: 基于 Go 语言的高并发特性，单机可支撑十万级长连接。
- **分布式设计**: 支持水平扩展，无单点故障，利用 Redis 和 Kafka 进行状态管理和流量削峰。
- **多端同步**: 完善的序列号机制，支持移动端、PC 端、Web 端消息实时同步与漫游。
- **丰富消息类型**: 支持文本、图片、语音、视频、自定义信令等消息格式。
- **安全性**: 集成鉴权机制，支持消息内容安全审计接口。
- **协议灵活**: 支持 WebSocket 作为主要传输协议，预留 TCP 私有协议扩展接口。

## 🛠️ 技术栈

- **语言**: Go 1.21+
- **网关层**: Gorilla WebSocket, Netpoll
- **存储层**: MySQL (元数据), MongoDB (消息内容), Redis (缓存/会话状态)
- **消息队列**: Apache Kafka / Pulsar
- **服务发现**: Etcd / Consul
- **RPC**: gRPC

## 🏃 快速开始

### 环境依赖

确保你的机器安装了以下软件：
- Go 1.21+
- Docker & Docker Compose
- MySQL 8.0+
- Redis 7.0+

### 安装与运行

1. **克隆项目**
   ```bash
   git clone https://github.com/your-org/gim.git
   cd gim
   ```

2. 启动基础设施
    使用 Docker Compose 一键启动依赖服务（MySQL, Redis, Kafka, Mongo）：
    ```bash
    docker-compose up -d
    ```

3. 编译并运行
    ```bash
    make build
    ./bin/gim-server -c config/config.yaml
    ```

4. 验证服务
    服务启动后，监听 0.0.0.0:8080 端口。你可以使用项目自带的 client 目录下的测试脚本进行连接测试。

### 目录结构

```
gim
├── api          # 接口定义 (Protobuf/HTTP)
├── cmd          # 程序入口 (main.go)
├── internal     # 核心业务逻辑
│   ├── connect  # 连接层 (WebSocket/TCP管理)
│   ├── logic    # 逻辑层 (消息处理/路由)
│   └── storage  # 存储层 (DAO)
├── pkg          # 公共库 (协议编解码/工具类)
└── config       # 配置文件
```