FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY apps/device-gateway-go/go.mod ./
COPY apps/device-gateway-go/cmd ./cmd
COPY apps/device-gateway-go/internal ./internal
RUN go build -trimpath -ldflags='-s -w' -o /out/device-gateway-go ./cmd/device-gateway-go

FROM alpine:3.20
RUN adduser -D -H -u 10001 appuser
USER appuser
EXPOSE 8787
ENV PORT=8787
COPY --from=builder /out/device-gateway-go /usr/local/bin/device-gateway-go
ENTRYPOINT ["device-gateway-go"]
