# 🚀 GitHub Pinned Repositories API (Golang + Swagger)

API hiệu năng cao viết bằng **Golang** cho phép tra cứu các repository được ghim (pinned repositories) của bất kỳ GitHub user hoặc organization nào mà không cần cấu hình GitHub Token. Dự án tích hợp đầy đủ **Swagger UI**, bộ nhớ đệm **Sharded In-Memory Cache** và cơ chế **Singleflight** giúp chịu tải cực lớn (stress-tested mượt mà với Locust).

---

## 🌟 Tính Năng Nổi Bật

- 🔍 **Không cần API Token**: Sử dụng bộ cào HTML `goquery` tối ưu hóa, tự động nhận diện cả tài khoản cá nhân và tổ chức.
- ⚡ **Chịu tải cao (High Concurrency)**:
  - **Sharded Cache (32 shards, RWMutex)**: Phản hồi các truy vấn đã cache trong thời gian cực thấp (< 0.2ms).
  - **Singleflight Pattern**: Ngăn ngừa hoàn toàn hiện tượng *Cache Stampede / Thundering Herd*. Khi có hàng ngàn lượt gọi đồng thời cùng một username, chỉ có 1 request duy nhất được gửi tới GitHub.
  - **HTTP Connection Pooling**: Tái sử dụng TCP connections với `MaxIdleConns: 1000`.
- 📖 **Tài liệu Swagger UI tương tác trực tiếp**: Truy cập ngay trên trình duyệt tại `http://localhost:8080/swagger/index.html`.
- 🧪 **Kịch bản Stress Test bằng Locust**: Có sẵn trong thư mục `test/` để kiểm thử sức chịu tải.

---

## 📂 Cấu Trúc Dự Án

```
gh-pin-repo-go/
├── cmd/
│   └── api/
│       └── main.go           # Entry point khởi chạy HTTP server & Swagger
├── internal/
│   ├── api/
│   │   ├── handler.go        # Controller xử lý HTTP request và Swagger annotations
│   │   └── router.go         # Thiết lập Gin router, CORS, Swagger UI route
│   ├── cache/
│   │   ├── cache.go          # Sharded In-Memory Cache hiệu năng cao
│   │   └── cache_test.go     # Unit test concurrency & expiration
│   ├── model/
│   │   └── repo.go           # Struct PinnedRepo, ApiResponse, ErrorResponse
│   └── service/
│       ├── scraper.go        # Service cào HTML, Singleflight và Cache
│       └── scraper_test.go   # Unit test parser & number formatting
├── docs/                     # Swagger auto-generated (docs.go, swagger.json, swagger.yaml)
├── test/
│   ├── locustfile.py         # Kịch bản stress test bằng Locust
│   └── run_stress_test.ps1   # Script chạy nhanh stress test trên Windows
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 Hướng Dẫn Cài Đặt & Khởi Chạy

### Yêu Cầu Môi Trường
- **Go 1.20+**
- **Python 3.10+** (để chạy Locust stress test)

### 1. Khởi chạy Server

Chạy trực tiếp bằng Go:
```bash
go run cmd/api/main.go
```

Hoặc build file thực thi:
```bash
go build -o gh-pin-repo.exe ./cmd/api
.\gh-pin-repo.exe
```

Server sẽ lắng nghe tại cổng `http://localhost:8080`.

### 2. Mở Swagger UI
Mở trình duyệt truy cập:
👉 **[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)** (hoặc truy cập `http://localhost:8080/` để được tự động chuyển hướng).

---

## 📡 API Endpoints

### 1. Tra cứu Pinned Repositories

```http
GET /api/v1/pinned/:username
```

**Query Parameters:**
| Tham số | Kiểu | Mặc định | Mô tả |
| :--- | :--- | :--- | :--- |
| `refresh` | `boolean` | `false` | Ép buộc bỏ qua cache và tải dữ liệu mới nhất từ GitHub |

**Ví dụ Request:**
```bash
curl http://localhost:8080/api/v1/pinned/torvalds
```

**Ví dụ Response (HTTP 200):**
```json
{
  "success": true,
  "message": "Successfully retrieved pinned repositories",
  "total": 5,
  "cached": false,
  "data": [
    {
      "owner": "torvalds",
      "repo": "linux",
      "description": "Linux kernel source tree",
      "language": "C",
      "languageColor": "#555555",
      "stars": 247000,
      "starsDisplay": "247k",
      "forks": 64400,
      "forksDisplay": "64.4k",
      "link": "https://github.com/torvalds/linux",
      "isFork": false
    },
    {
      "owner": "torvalds",
      "repo": "AudioNoise",
      "description": "Random digital audio effects",
      "language": "C",
      "languageColor": "#555555",
      "stars": 4500,
      "starsDisplay": "4.5k",
      "forks": 217,
      "forksDisplay": "217",
      "link": "https://github.com/torvalds/AudioNoise",
      "isFork": false
    }
  ]
}
```

### 2. Kiểm tra Trạng thái (Health Check)

```http
GET /health
```

**Response (HTTP 200):**
```json
{
  "status": "ok",
  "message": "Service is healthy and ready to serve requests"
}
```

---

## 🧪 Stress Test Với Locust

Để kiểm thử khả năng chịu tải hàng triệu lượt request với độ trễ tối thiểu:

### 1. Cài đặt Locust
```bash
pip install locust
```

### 2. Chạy Stress Test tự động (PowerShell)
```powershell
.\test\run_stress_test.ps1 -Users 200 -SpawnRate 50 -RunTime 30s
```

### 3. Hoặc chạy lệnh trực tiếp bằng Locust CLI:
```bash
python -m locust -f test/locustfile.py --headless -u 200 -r 50 -t 30s --host http://localhost:8080 --only-summary
```

### 4. Giao diện trực quan (Locust Web UI):
```bash
python -m locust -f test/locustfile.py --host http://localhost:8080
```
Sau đó mở trình duyệt tại `http://localhost:8089` để theo dõi biểu đồ RPS, Response Time và Failure Rate theo thời gian thực.

---

## 🛡️ Cơ Chế Xử Lý Khi Có Lượng Người Dùng Lớn (10M Users)

1. **In-Memory Sharded Cache**: Được chia thành 32 phân vùng (`cacheShard`) độc lập, mỗi phân vùng sở hữu một `sync.RWMutex` riêng biệt. Khi có hàng trăm luồng đọc ghi đồng thời, hiện tượng nghẽn khóa (lock contention) bị triệt tiêu hoàn toàn.
2. **Singleflight Throttling**: Khi một người dùng nổi tiếng (ví dụ: `torvalds`) hết hạn cache và có 50.000 requests cùng ập đến trong cùng 1 giây, cơ chế `singleflight` chỉ gửi đúng **1 request** duy nhất đến GitHub. 49.999 requests còn lại sẽ chờ và dùng chung kết quả, tránh hoàn toàn lỗi `429 Too Many Requests` từ GitHub.
3. **Gin Release Mode + GOMAXPROCS**: Tối ưu hóa toàn bộ số nhân CPU có sẵn của máy chủ và loại bỏ các logging middleware gây chậm trễ I/O.
