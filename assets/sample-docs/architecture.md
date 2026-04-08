# Architecture

## Overview

Nexus sits between clients and backend services, handling routing, auth, and rate limiting.

```
                    ┌─────────────┐
                    │   Clients   │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │    Nexus    │
                    │  (Gateway)  │
                    └──┬───┬───┬──┘
                       │   │   │
              ┌────────┘   │   └────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │  Users   │ │  Orders  │ │ Payments │
        │ Service  │ │ Service  │ │ Service  │
        └──────────┘ └──────────┘ └──────────┘
```

## Components

### Router

The router matches incoming requests to upstream services using path prefix matching. Routes are evaluated in order — first match wins.

### Middleware Pipeline

Each request passes through a configurable middleware chain:

1. **Logger** — structured request/response logging
2. **Auth** — JWT validation and claim extraction
3. **RateLimiter** — token bucket per client IP
4. **Timeout** — upstream request timeout (default 30s)
5. **Retry** — automatic retry with exponential backoff

### Configuration

Config is loaded from YAML at startup and can be hot-reloaded via SIGHUP.

```go
type Config struct {
    Server  ServerConfig  `yaml:"server"`
    Routes  []Route       `yaml:"routes"`
    Auth    AuthConfig    `yaml:"auth"`
    Limits  LimitConfig   `yaml:"limits"`
}
```

## Data Flow

1. Client sends request to Nexus
2. Router matches path to upstream
3. Middleware pipeline processes request
4. Request is forwarded to upstream service
5. Response flows back through middleware
6. Client receives final response

## Performance

Benchmarks on M1 MacBook Pro:

| Metric | Value |
|--------|-------|
| Latency (p50) | 0.3ms |
| Latency (p99) | 2.1ms |
| Throughput | 45k req/s |
| Memory | 12MB idle |
