# ITAM — Hệ thống Quản lý Tài sản IT

> Giám sát phần cứng thời gian thực cho môi trường doanh nghiệp, bệnh viện.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/SQLite-WAL_Mode-003B57?logo=sqlite&logoColor=white" alt="SQLite">
  <img src="https://img.shields.io/badge/HTMX-Dashboard-3366CC" alt="HTMX">
  <img src="https://img.shields.io/badge/License-MIT-green" alt="MIT">
</p>

**[🇬🇧 English](README.md)**

---

## Tính năng

| Tính năng                       | Mô tả                                                    |
| ------------------------------- | -------------------------------------------------------- |
| 📊 **Dashboard thời gian thực** | Giám sát CPU, RAM, Disk với cập nhật trực tiếp qua HTMX  |
| 🔍 **Tìm kiếm & Lọc**           | Tìm theo hostname, IP, địa chỉ MAC                       |
| 📈 **Lịch sử sử dụng**          | Biểu đồ CPU/RAM theo thời gian (Chart.js)                |
| 🚨 **Cảnh báo**                 | Tự động cảnh báo khi CPU/RAM > 90%                       |
| 💾 **Backup tự động**           | Backup database khi server khởi động                     |
| 🛡️ **Circuit Breaker**          | Giới hạn 100 req/s + tăng thời gian chờ theo cấp số nhân |
| 📝 **Xoay Log**                 | Tự động xoay file log, 10MB/file, giữ 7 file             |
| 📥 **Xuất CSV**                 | Xuất danh sách tài sản dạng CSV                          |
| 🔑 **Token xác thực**           | Bearer token cho giao tiếp Agent-Server                  |
| ⏹️ **Kill Switch**              | Điều khiển agent từ xa (ngủ/tắt)                         |

---

## Kiến trúc

```
+-------------+     POST /api/report     +-------------+
|   Agent     | ------[Auth Token]-----> |   Server    |
|  (Windows)  |                          |  (Go HTTP)  |
+-------------+                          +------+------+
      ^                                         |
      |  {"command": "sleep_1h"}                |
      +----------------[Kill Switch]------------+
                                                |
+-------------+     HTMX polling         +------v------+
|   Trình     | <----------------------- |   SQLite    |
|   duyệt    |                          |   (WAL)     |
+-------------+                          +-------------+
```

**Cách hoạt động:**

1. **Agent** thu thập thông tin hệ thống (CPU, RAM, Disk, tiến trình) bằng `gopsutil`
2. Agent **gửi POST** dữ liệu lên server kèm token xác thực
3. **Server** lưu dữ liệu vào SQLite (chế độ WAL) và tạo cảnh báo
4. **Dashboard** cập nhật thời gian thực qua HTMX polling

---

## Bắt đầu nhanh

### Yêu cầu

- Go 1.21+
- Windows (agent) / Linux (server)
- Kết nối mạng giữa agent và server

### Biên dịch & Chạy

```bash
# Biên dịch cả agent và server
make build

# Chạy server
./build/server_bin

# Chạy agent trên máy client
cd agent && ./build/agent_bin
```

### Agent cho Production (File EXE duy nhất)

```bash
# Biên dịch agent với cài đặt nhúng sẵn (không cần config.json)
make build-agent-prod \
  SERVER_URL=http://192.168.1.100:8080/api/report \
  AUTH_TOKEN=token-bi-mat

# Kết quả: ./build/agent.exe — triển khai qua GPO
```

### Dashboard

Mở http://localhost:8080 sau khi khởi động server.

---

## Cấu hình

### Agent (`agent/config.json`)

```json
{
  "server_url": "http://itam-server:8080/api/report",
  "auth_token": "token-bi-mat",
  "top_process_limit": 5,
  "jitter_seconds": 30,
  "log_level": "error"
}
```

| Trường              | Mô tả                                             |
| ------------------- | ------------------------------------------------- |
| `server_url`        | Địa chỉ server                                    |
| `auth_token`        | Token bí mật (phải khớp với server)               |
| `top_process_limit` | Số tiến trình cần thu thập                        |
| `jitter_seconds`    | Trễ ngẫu nhiên (0–N giây) chống quá tải đồng thời |
| `log_level`         | `"debug"`, `"info"`, hoặc `"error"`               |

### Server (`server/config.json`)

```json
{
  "auth_token": "token-bi-mat",
  "port": 8080,
  "db_path": "data/itam.db"
}
```

> **Thứ tự ưu tiên cài đặt (Agent):** Flags lúc biên dịch → Biến môi trường (`ITAM_AUTH_TOKEN`) → `config.json` cạnh file exe → `./config.json`

---

## Hướng dẫn mở rộng

| Số máy   | Chu kỳ GPO | Jitter | Dashboard Poll | Database   |
| -------- | ---------- | ------ | -------------- | ---------- |
| 1–100    | 5 phút     | 15s    | 15s            | SQLite     |
| 100–500  | 5–15 phút  | 30s    | 30s            | SQLite     |
| 500–2000 | 15–30 phút | 60s    | 60s            | SQLite     |
| 2000+    | 30+ phút   | 60s    | 60s            | PostgreSQL |

```
Peak req/s = Số máy ÷ Jitter
Ví dụ: 400 máy ÷ 30s jitter ≈ 13 req/s (thấp hơn nhiều so với giới hạn 100)
```

Xem [docs/SCALING.md](docs/SCALING.md) để biết chi tiết.

---

## API

| Endpoint                           | Method | Auth  | Mô tả                    |
| ---------------------------------- | ------ | ----- | ------------------------ |
| `/api/report`                      | POST   | Token | Gửi dữ liệu từ agent     |
| `/api/assets`                      | GET    | —     | Tất cả tài sản (JSON)    |
| `/api/assets/table`                | GET    | —     | Bảng tài sản (HTMX)      |
| `/api/assets/detail?mac=XX`        | GET    | —     | Chi tiết tài sản         |
| `/api/stats`                       | GET    | —     | Thống kê tổng hợp        |
| `/api/alerts`                      | GET    | —     | Danh sách cảnh báo       |
| `/api/alerts/resolve`              | POST   | —     | Xử lý cảnh báo           |
| `/api/history?mac=XX`              | GET    | —     | Lịch sử sử dụng (24h)    |
| `/api/export/csv`                  | GET    | —     | Xuất CSV                 |
| `/api/backup`                      | GET    | —     | Backup thủ công          |
| `/health`                          | GET    | —     | Kiểm tra sức khỏe server |
| `/api/admin/command?mac=XX&cmd=YY` | POST   | Token | Điều khiển agent         |
| `/dl/agent.exe`                    | GET    | —     | Tải agent                |

### Lệnh Kill Switch

```bash
# Cho agent ngủ 1 giờ
curl -X POST -H "Authorization: Bearer TOKEN" \
  "http://server:8080/api/admin/command?mac=AA:BB:CC:DD:EE:FF&cmd=sleep_1h"

# Cho agent ngủ 24 giờ
curl -X POST -H "Authorization: Bearer TOKEN" \
  "http://server:8080/api/admin/command?mac=AA:BB:CC:DD:EE:FF&cmd=sleep_24h"

# Tắt agent ngay lập tức
curl -X POST -H "Authorization: Bearer TOKEN" \
  "http://server:8080/api/admin/command?mac=AA:BB:CC:DD:EE:FF&cmd=exit"
```

---

## Triển khai qua GPO (Windows)

```powershell
# Tạo scheduled task chạy agent mỗi 15 phút
$action = New-ScheduledTaskAction -Execute "C:\ITAM\agent.exe"
$trigger = New-ScheduledTaskTrigger -Once -At (Get-Date) `
  -RepetitionInterval (New-TimeSpan -Minutes 15)
$settings = New-ScheduledTaskSettingsSet `
  -RandomDelay (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName "ITAM Agent" `
  -Action $action -Trigger $trigger -Settings $settings
```

**Các bước:**

1. Biên dịch agent: `make build-agent-prod SERVER_URL=... AUTH_TOKEN=...`
2. Sao chép `agent.exe` vào thư mục chia sẻ trên mạng
3. Tạo GPO để triển khai scheduled task
4. Agent chạy → thu thập dữ liệu → gửi lên server → thoát

---

## Tính năng Production

| Tính năng           | Chi tiết                                                                           |
| ------------------- | ---------------------------------------------------------------------------------- |
| **Circuit Breaker** | Server: giới hạn 100 req/s (trả 429). Agent: backoff 5s→10s→20s                    |
| **Xoay Log**        | `logs/itam.log`, 10MB/file, giữ 7 file, log tóm tắt mỗi phút                       |
| **Backup tự động**  | Khi khởi động → `backups/itam_YYYY-MM-DD_HH-MM-SS.db`. Giữ 7 ngày, tối thiểu 5 bản |
| **Dọn dữ liệu**     | Tự động xóa lịch sử > 30 ngày khi khởi động                                        |

---

## Cấu trúc dự án

```
itam-project/
├── agent/
│   ├── main.go           # Mã nguồn Agent (chạy trên Windows)
│   └── config.json       # Cấu hình Agent (gitignored)
├── server/
│   ├── main.go           # Khởi động server + tắt an toàn
│   ├── handlers.go       # Xử lý HTTP + xác thực + rate limiter
│   ├── store.go          # SQLite + backup + dọn dữ liệu
│   ├── models.go         # Cấu trúc dữ liệu
│   ├── config.go         # Cấu hình server
│   ├── logger.go         # Hệ thống xoay log
│   ├── templates.go      # Render giao diện HTMX
│   ├── utils.go          # Hàm tiện ích
│   └── templates/
│       └── index.html    # Dashboard (Tabler.io)
├── web/static/           # CSS, JS, thư viện vendor
├── cmd/loadtest/         # Kiểm tra tải (500 agent giả lập)
├── docs/SCALING.md       # Hướng dẫn mở rộng
└── Makefile
```

---

## Kiểm tra tải

```bash
make loadtest   # 500 agent giả lập
```

---

## Công nghệ sử dụng

| Tầng     | Công nghệ                        |
| -------- | -------------------------------- |
| Backend  | Go (net/http)                    |
| Database | SQLite (WAL, modernc.org/sqlite) |
| Agent    | gopsutil v3                      |
| Frontend | HTMX + Tabler.io + Chart.js      |

---

## Lộ trình phát triển

- [ ] Giám sát máy in (SNMP)
- [ ] Thiết bị mạng (switch, router)
- [ ] Kiểm kê phần mềm
- [ ] Điều khiển từ xa (tắt máy, khởi động lại)
- [ ] Xác thực LDAP/AD cho dashboard
- [ ] Hỗ trợ HTTPS/TLS

## Giấy phép

MIT
