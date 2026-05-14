# LobeHub Gateway Go

[English](./README.md)

`lobehub-gateway-go` 是 LobeHub 网关服务的 Go 语言重新实现版本。它面向自托管部署场景，目标是在不绑定 Cloudflare 平台、也不依赖外部基础设施的前提下，提供简单的单实例网关服务。

原始网关实现仍然是协议参考。本仓库关注在尽量保持公开 HTTP 与 WebSocket 行为兼容的同时，让网关可以作为普通 Go 服务运行。

## 目标

- 使用 Go 实现 LobeHub 网关协议。
- 采用单实例设计，便于自托管部署。
- 不依赖 Cloudflare Workers 或 Durable Objects。
- 不依赖 Redis、PostgreSQL、NATS 或其他外部运行时服务。
- 在可行范围内保持与参考实现的协议行为兼容。

## 项目

| 项目 | 状态 | 说明 |
| --- | --- | --- |
| [`device-gateway-go`](./device-gateway-go/README.md) | 已实现 | 设备网关，用于路由 LobeHub 设备连接、状态查询、工具调用、系统信息和 agent 运行请求。 |
| [`agent-gateway-go`](./agent-gateway-go/README.md) | 占位 | 计划中的 Agent Gateway 协议 Go 实现。 |

## 架构

本仓库按多个 gateway 服务组织。每个 gateway 都位于独立目录中，便于后续继续添加其他 gateway，而不会和现有服务强耦合。

```text
lobehub-gateway-go/
  device-gateway-go/
  agent-gateway-go/
```

当前实现风格刻意保持最小化：状态保存在内存中，服务作为普通 HTTP/WebSocket 服务器运行，部署方式可以是 systemd、Docker、Kubernetes 或任意进程管理器。

## 与 LobeHub Gateway 的关系

相比原始 LobeHub gateway 项目，本仓库有几个有意的差异：

| 维度 | 原始 gateway | 本仓库 |
| --- | --- | --- |
| 编写语言 | TypeScript | Go |
| 运行时 | Cloudflare Workers | 标准 Go 进程 |
| 协调机制 | Durable Objects | 单实例内存状态 |
| 平台绑定 | Cloudflare 相关 | 平台无关 |
| 运行时依赖 | Cloudflare 服务 | 无外部服务依赖 |

由于本仓库面向单实例网关设计，默认不提供分布式会话协调。如果需要水平扩展，请为每个路由域运行一个活动网关实例，或显式引入共享协调层。

## 开发

运行命令前先进入具体 gateway 目录：

```bash
cd device-gateway-go
go test ./...
```

各 gateway 的配置、API 路由和本地运行方式请查看对应 README。
