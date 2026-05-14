# Go Agent Gateway

[English](./README.md)

`agent-gateway-go` 是计划中的 LobeHub Agent Gateway 协议 Go 实现，目前为占位项目。

参考版 Agent Gateway 基于 Cloudflare Workers 与 Durable Objects 实现。本项目将用单实例、平台无关、无外部运行时依赖的 Go 服务重新实现相同的网关职责。

## 预期范围

- 管理 Agent 会话的浏览器 WebSocket 连接。
- 将后端服务产生的 Agent 流式事件路由到已连接客户端。
- 将用户输入、工具确认和中断消息从客户端转发给后端服务。
- 按协议需要支持 service-token 与 JWT 认证。
- 面向单网关实例，将运行时状态保存在内存中。

## 计划方向

Go 实现预计会沿用 `device-gateway-go` 的仓库原则：

- 标准 Go HTTP/WebSocket 服务。
- 不依赖 Cloudflare Workers 或 Durable Objects。
- 不依赖 Redis、PostgreSQL、NATS 或其他外部运行时服务。
- 在可行范围内保持公开 API 与 WebSocket 行为兼容。

## 状态

实现尚未开始。该目录用于为 Agent Gateway Go 服务保留稳定的位置。
