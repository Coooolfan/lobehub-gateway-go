# LobeHub Gateway Go

[简体中文](./README.zh-CN.md)

`lobehub-gateway-go` is a Go reimplementation of the LobeHub gateway services. It is intended for self-hosted deployments that need a simple, single-instance gateway without Cloudflare platform bindings or external infrastructure dependencies.

The original gateway implementations remain the protocol reference. This repository focuses on keeping the public HTTP and WebSocket behavior compatible while making the runtime easier to run as a normal Go service.

## Goals

- Go implementation of LobeHub gateway protocols.
- Single-instance design for straightforward self-hosting.
- No Cloudflare Workers or Durable Objects dependency.
- No Redis, PostgreSQL, NATS, or other external runtime services.
- Keep protocol behavior compatible with the reference implementations where practical.

## Projects

| Project | Status | Description |
| --- | --- | --- |
| [`device-gateway-go`](./device-gateway-go/README.md) | Implemented | Device gateway for routing LobeHub device connections, status queries, tool calls, system information, and agent-run requests. |
| [`agent-gateway-go`](./agent-gateway-go/README.md) | Placeholder | Planned Go implementation of the Agent Gateway protocol. |

## Architecture

This repository is organized as a collection of gateway services. Each gateway lives in its own directory so more gateway implementations can be added without coupling them to the existing services.

```text
lobehub-gateway-go/
  device-gateway-go/
  agent-gateway-go/
```

The current implementation style is intentionally minimal: state is held in memory, services run as ordinary HTTP/WebSocket servers, and deployment can be handled by systemd, Docker, Kubernetes, or any other process manager.

## Relationship To LobeHub Gateway

Compared with the original LobeHub gateway projects, this repository has several deliberate differences:

| Area | Original gateway | This repository |
| --- | --- | --- |
| Language | TypeScript | Go |
| Runtime | Cloudflare Workers | Standard Go process |
| Coordination | Durable Objects | In-memory single instance |
| Platform binding | Cloudflare-specific | Platform-neutral |
| Runtime dependencies | Cloudflare services | No external services required |

Because this repository targets a single gateway instance, it does not provide distributed session coordination by default. If you need horizontal scaling, run one active gateway instance per routing domain or add a shared coordination layer explicitly.

## Development

Enter a gateway directory before running commands:

```bash
cd device-gateway-go
go test ./...
```

See each gateway README for service-specific configuration, API routes, and local run instructions.
