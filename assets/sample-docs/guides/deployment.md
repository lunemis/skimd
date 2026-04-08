# Deployment Guide

## Docker

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o nexus ./cmd/nexus

FROM alpine:3.19
COPY --from=build /app/nexus /usr/local/bin/
COPY nexus.yml /etc/nexus/
EXPOSE 8080
CMD ["nexus", "serve", "--config", "/etc/nexus/nexus.yml"]
```

Build and run:

```bash
docker build -t nexus .
docker run -p 8080:8080 nexus
```

## Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nexus
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nexus
  template:
    metadata:
      labels:
        app: nexus
    spec:
      containers:
        - name: nexus
          image: ghcr.io/example/nexus:latest
          ports:
            - containerPort: 8080
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
          resources:
            requests:
              memory: "32Mi"
              cpu: "100m"
            limits:
              memory: "128Mi"
              cpu: "500m"
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXUS_PORT` | 8080 | Server port |
| `NEXUS_LOG_LEVEL` | info | Log verbosity |
| `NEXUS_REDIS_URL` | - | Redis for rate limiting |
| `NEXUS_TLS_CERT` | - | TLS certificate path |
| `NEXUS_TLS_KEY` | - | TLS key path |

## Monitoring

Prometheus metrics are available at `/metrics`:

- `nexus_requests_total` — total request count by route and status
- `nexus_request_duration_seconds` — request latency histogram
- `nexus_upstream_errors_total` — upstream error count
- `nexus_active_connections` — current active connections
