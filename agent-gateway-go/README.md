# Go Agent Gateway

[简体中文](./README.zh-CN.md)

`agent-gateway-go` is a placeholder for the planned Go implementation of the LobeHub Agent Gateway protocol.

The reference Agent Gateway is implemented with Cloudflare Workers and Durable Objects. This project will reimplement the same gateway role as a single-instance, platform-neutral Go service with no external runtime dependencies.

## Intended Scope

- Manage browser WebSocket sessions for Agent conversations.
- Route streaming Agent events from backend services to connected clients.
- Forward user input, tool confirmation, and interrupt messages from clients to backend services.
- Support service-token and JWT-based authentication where required by the protocol.
- Keep runtime state in memory for a single gateway instance.

## Planned Direction

The Go implementation is expected to follow the same repository principles as `device-gateway-go`:

- Standard Go HTTP/WebSocket server.
- No Cloudflare Workers or Durable Objects dependency.
- No Redis, PostgreSQL, NATS, or other external runtime services.
- Compatible public API and WebSocket behavior where practical.

## Status

Implementation has not started yet. This directory exists so the repository can reserve a stable location for the Agent Gateway Go service.
