# Troubleshooting

## Common Issues

### Gateway returns 502

**Symptom**: All requests return `502 Bad Gateway`

**Cause**: Upstream service is unreachable.

**Fix**:
1. Check if the upstream is running: `curl http://upstream-host:port/health`
2. Verify the route config points to the correct address
3. Check network connectivity between Nexus and upstream

### High latency

**Symptom**: p99 latency exceeds 500ms

**Possible causes**:
- Upstream is slow — check upstream metrics
- Rate limiter Redis is remote — use local Redis or in-memory limiter
- Too many middleware layers — disable unused middleware

### Memory growing over time

**Symptom**: RSS increases steadily

**Fix**: This was a known issue in v0.2.x. Upgrade to v0.3.0+ which fixed the connection pool leak.

```bash
nexus --version   # should be >= 0.3.0
```

## Debug Mode

Enable verbose logging:

```bash
NEXUS_LOG_LEVEL=debug nexus serve --config nexus.yml
```

This logs every request/response including headers and timing.
