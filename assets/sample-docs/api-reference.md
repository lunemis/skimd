# API Reference

## Authentication

All endpoints require a Bearer token in the `Authorization` header.

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/users
```

## Endpoints

### Users

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/users` | List all users |
| GET | `/api/users/:id` | Get user by ID |
| POST | `/api/users` | Create new user |
| PUT | `/api/users/:id` | Update user |
| DELETE | `/api/users/:id` | Delete user |

### Orders

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/orders` | List orders |
| GET | `/api/orders/:id` | Get order detail |
| POST | `/api/orders` | Create order |
| PUT | `/api/orders/:id/status` | Update status |

## Request Examples

### Create a user

```json
POST /api/users
Content-Type: application/json

{
  "name": "Alice Kim",
  "email": "alice@example.com",
  "role": "admin"
}
```

Response:

```json
{
  "id": "usr_a1b2c3",
  "name": "Alice Kim",
  "email": "alice@example.com",
  "role": "admin",
  "created_at": "2026-04-08T12:00:00Z"
}
```

### List orders with filters

```bash
GET /api/orders?status=pending&limit=10&offset=0
```

## Rate Limiting

| Tier | Requests/min | Burst |
|------|-------------|-------|
| Free | 60 | 10 |
| Pro | 600 | 50 |
| Enterprise | 6000 | 200 |

Rate limit headers are included in every response:

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1712600000
```

## Error Codes

| Code | Meaning |
|------|---------|
| 400 | Bad request — check your request body |
| 401 | Unauthorized — invalid or missing token |
| 403 | Forbidden — insufficient permissions |
| 404 | Not found |
| 429 | Too many requests — rate limited |
| 500 | Internal server error |
