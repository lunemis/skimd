# Getting Started with Nexus

Welcome to Nexus, a modern API gateway for microservices.

## Prerequisites

- Go 1.22+
- Docker (optional)
- Redis 7+ for rate limiting

## Installation

```bash
go install github.com/example/nexus@latest
```

Or build from source:

```bash
git clone https://github.com/example/nexus.git
cd nexus
make build
```

## Quick Start

### 1. Create a config file

```yaml
# nexus.yml
server:
  port: 8080
  host: 0.0.0.0

routes:
  - path: /api/users
    upstream: http://users-service:3000
    methods: [GET, POST]
    
  - path: /api/orders
    upstream: http://orders-service:3001
    methods: [GET, POST, PUT]
```

### 2. Start the gateway

```bash
nexus serve --config nexus.yml
```

### 3. Verify it's running

```bash
curl http://localhost:8080/health
# {"status": "ok", "uptime": "2s"}
```

## Next Steps

- Read the [API Reference](api-reference.md) for route configuration
- Check the [Architecture](architecture.md) overview
- See the [Changelog](changelog.md) for recent updates
