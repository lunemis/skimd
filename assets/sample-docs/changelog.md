# Changelog

## v0.4.0 (2026-04-01)

### Features
- Added WebSocket proxy support
- New `nexus validate` command to check config before starting
- Route-level timeout overrides
- Health check endpoint now includes upstream status

### Improvements
- Reduced memory allocation in hot path by 40%
- Faster route matching with radix tree (was linear scan)
- Better error messages for invalid config

### Bug Fixes
- Fixed connection leak when upstream returns 502
- Fixed race condition in rate limiter reset
- Corrected Content-Length header on retry

## v0.3.0 (2026-03-15)

### Features
- JWT authentication middleware
- Redis-backed rate limiting
- Request/response body logging (opt-in)
- Prometheus metrics endpoint at `/metrics`

### Improvements
- Config hot-reload via SIGHUP
- Graceful shutdown with in-flight request draining
- Added `--dry-run` flag to `nexus serve`

### Bug Fixes
- Fixed panic on nil upstream response
- Fixed incorrect CORS headers on preflight

## v0.2.0 (2026-02-28)

### Features
- HTTPS/TLS termination
- Retry with exponential backoff
- Request timeout configuration
- CORS middleware

### Improvements
- Structured JSON logging
- Docker image published to GHCR

## v0.1.0 (2026-02-01)

### Features
- Basic HTTP reverse proxy
- YAML-based route configuration
- Path prefix matching
- Health check endpoint
- Graceful shutdown

Initial release.
