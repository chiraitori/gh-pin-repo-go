# gh-pin-repo-go

REST API for querying pinned GitHub repositories of any user or organization without requiring a personal access token. Includes interactive Swagger/OpenAPI documentation and an in-memory caching engine designed for high throughput.

---

## Technical Overview

GitHub's official REST API v3 does not expose an endpoint for pinned repositories. The v4 GraphQL API requires authentication tokens with strict rate limits. 

This service scrapes profile markup directly using `goquery` and wraps it in a production-ready HTTP interface:

```
                          +------------------------------------------+
                          |              gh-pin-repo-go              |
                          |                                          |
Client Request ---------> |  [ Gin Engine ]                          |
                          |        |                                 |
                          |  [ 32-Shard In-Memory Cache ]            |
                          |        |                                 |
                          |     (miss)                               |
                          |        v                                 |
                          |  [ singleflight.Group ]                  |
                          |        | (1 outbound request per key)    |
                          +--------|---------------------------------+
                                   v
                       GitHub (https://github.com/:username)
```

### High-Throughput Design

- **Sharded In-Memory Cache**: The cache splits keys across 32 independent buckets (`cacheShard`), each protected by its own `sync.RWMutex`. This prevents lock contention under concurrent reads and keeps cache hits under 0.2ms.
- **Thundering Herd Protection**: Uses `golang.org/x/sync/singleflight`. When a cache miss occurs under high concurrency, all incoming requests for the same username wait on a single outbound HTTP call to GitHub instead of firing duplicate requests that trigger upstream 429 rate limits.
- **Connection Pooling**: Customized `http.Transport` with 1,000 idle connections and persistent keep-alive.

---

## API Reference

Base URL: `http://localhost:8344`

### Endpoints

| Method | Path | Description |
| :--- | :--- | :--- |
| `GET` | `/api/v1/pinned/:username` | Fetch pinned repositories for a user or organization |
| `GET` | `/health` | Health check probe |
| `GET` | `/swagger/*any` | Interactive Swagger UI |
| `GET` | `/` | Redirects to `/swagger/index.html` |

---

### `GET /api/v1/pinned/:username`

#### Parameters

| Name | In | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `username` | path | `string` | Yes | - | GitHub username or organization name |
| `refresh` | query | `boolean` | No | `false` | Bypass cache and fetch fresh data from GitHub |
| `raw` | query | `boolean` | No | `false` | Return flat JSON array directly (100% `gh-pinned-repos` drop-in compatible) |

#### Response `200 OK` (Standard)

```json
{
  "success": true,
  "message": "Successfully retrieved pinned repositories",
  "total": 4,
  "cached": false,
  "data": [
    {
      "owner": "chiraitori",
      "repo": "endfield-daily-helper",
      "description": "Automatically claim daily login rewards for Arknights: Endfield.",
      "language": "Python",
      "languageColor": "#3572A5",
      "stars": 15,
      "starsDisplay": "15",
      "forks": 2,
      "forksDisplay": "2",
      "link": "https://github.com/chiraitori/endfield-daily-helper",
      "image": "https://opengraph.githubassets.com/1/chiraitori/endfield-daily-helper",
      "isFork": false
    }
  ]
}
```

#### Response `200 OK` (With `?raw=true`)

```json
[
  {
    "owner": "chiraitori",
    "repo": "endfield-daily-helper",
    "description": "Automatically claim daily login rewards for Arknights: Endfield.",
    "language": "Python",
    "languageColor": "#3572A5",
    "stars": 15,
    "starsDisplay": "15",
    "forks": 2,
    "forksDisplay": "2",
    "link": "https://github.com/chiraitori/endfield-daily-helper",
    "image": "https://opengraph.githubassets.com/1/chiraitori/endfield-daily-helper",
    "isFork": false
  }
]
```

#### Response `404 Not Found`

```json
{
  "success": false,
  "message": "GitHub user or organization not found",
  "error": "github user or organization not found"
}
```

---

## Configuration

The service reads configuration exclusively from environment variables:

| Variable | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `PORT` | `int` | `8344` | Port the HTTP server binds to |
| `GIN_MODE` | `string` | `release` | Gin framework mode (`release` or `debug`) |
| `CACHE_TTL_MINUTES` | `int` | `10` | Time-to-live for cached profiles in minutes |

---

## Development & Build

### Requirements
- Go 1.20+
- Python 3.10+ (optional, for running Locust load tests)

### Local Run
```bash
go run cmd/api/main.go
```

### Build Binary
```bash
go build -o gh-pin-repo ./cmd/api
```

### Run Tests
```bash
go test -v ./internal/cache ./internal/service
```

### Regenerate Swagger Documentation
```bash
swag init -g cmd/api/main.go -o docs
```

---

## Production Deployment (systemd)

Service definition for `/etc/systemd/system/gh-pin-repo.service`:

```ini
[Unit]
Description=GitHub Pinned Repositories API
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/opt/gh-pin-repo-go
Environment="PORT=8344"
Environment="GIN_MODE=release"
Environment="CACHE_TTL_MINUTES=15"
ExecStart=/opt/gh-pin-repo-go/gh-pin-repo
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gh-pin-repo.service
```

---

## Load Testing (Locust)

The `test/` directory contains a Locust test harness simulating heavy read traffic across multiple profiles.

### Run Headless Benchmark
```bash
python -m locust -f test/locustfile.py --headless -u 200 -r 50 -t 30s --host http://localhost:8344 --only-summary
```

### Run with Web Dashboard
```bash
python -m locust -f test/locustfile.py --host http://localhost:8344
# Open http://localhost:8089 in browser
```

### Test Baseline Metrics (Single Node)
- **Concurrent Virtual Users**: 100
- **Throughput**: ~750 req/s
- **Failure Rate**: 0.00%
- **Median Latency (Cache Hit)**: < 1ms
