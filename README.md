# ITAM - IT Asset Management System

He thong quan ly tai san IT cho moi truong doanh nghiep, hospital. Thu thap thong tin hardware, monitor CPU/RAM/Disk realtime.

## Features

- **Real-time Dashboard** - Monitor CPU, RAM, Disk usage (online machines only)
- **Search & Filter** - Tim kiem theo hostname, IP, MAC
- **Usage History** - Bieu do lich su CPU/RAM
- **Alerts** - Canh bao khi CPU/RAM > 90%
- **Auto Backup** - Tu dong backup database khi khoi dong
- **Circuit Breaker** - Rate limiting + exponential backoff
- **Log Rotation** - Tu dong xoay log file, giam 99% log size
- **Export CSV** - Xuat danh sach tai san

## Security & Maintenance

| Feature       | Status | Description                                  |
| ------------- | ------ | -------------------------------------------- |
| Auth Token    | OK     | Bearer token authentication for Agent-Server |
| Data Pruning  | OK     | Auto-delete history > 30 days on startup     |
| Rate Limiting | OK     | 100 req/s limit with 429 response            |
| Health Check  | OK     | `/health` endpoint for monitoring            |
| Kill Switch   | OK     | Remote command to sleep/exit agents          |

## Quick Start

```bash
# Build
make build

# Run server
./server_bin

# Run agent (on each client)
cd agent && ./agent_bin
```

Dashboard: http://localhost:8080

## Architecture

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
|   Browser   | <----------------------- |   SQLite    |
|  Dashboard  |                          |   Database  |
+-------------+                          +-------------+
```

## Configuration

### Agent (`agent/config.json`)

```json
{
  "server_url": "http://itam-server:8080/api/report",
  "auth_token": "your-secret-token-here",
  "top_process_limit": 5,
  "jitter_seconds": 30,
  "log_level": "error"
}
```

| Field               | Description                                      |
| ------------------- | ------------------------------------------------ |
| `server_url`        | ITAM server endpoint                             |
| `auth_token`        | Secret token for authentication                  |
| `top_process_limit` | Number of top processes to collect               |
| `jitter_seconds`    | Random delay (0 to N) to prevent thundering herd |
| `log_level`         | "debug", "info", "error"                         |

### Server Auth Token

Edit `server/handlers.go`:

```go
var serverConfig = struct {
    AuthToken string
}{
    AuthToken: "your-secret-token-here", // Must match agent!
}
```

### Scaling by Nodes

| Nodes    | GPO Interval | Jitter | Dashboard Poll |
| -------- | ------------ | ------ | -------------- |
| 1-100    | 5 min        | 15s    | 15s            |
| 100-500  | 5-15 min     | 30s    | 30s            |
| 500-2000 | 15-30 min    | 60s    | 60s            |

See [docs/SCALING.md](docs/SCALING.md) for details.

## API Endpoints

| Endpoint                           | Method | Auth  | Description           |
| ---------------------------------- | ------ | ----- | --------------------- |
| `/api/report`                      | POST   | Token | Agent data submission |
| `/api/assets`                      | GET    | -     | All assets (JSON)     |
| `/api/assets/detail?mac=XX`        | GET    | -     | Asset detail          |
| `/api/stats`                       | GET    | -     | Aggregate stats       |
| `/api/backup`                      | GET    | -     | Manual backup         |
| `/api/export/csv`                  | GET    | -     | Export CSV            |
| `/health`                          | GET    | -     | Server health status  |
| `/api/admin/command?mac=XX&cmd=YY` | POST   | -     | Kill switch           |

### Kill Switch Commands

```bash
# Sleep agent for 1 hour
curl -X POST "http://server:8080/api/admin/command?mac=AA:BB:CC:DD:EE:FF&cmd=sleep_1h"

# Sleep agent for 24 hours
curl -X POST "http://server:8080/api/admin/command?mac=AA:BB:CC:DD:EE:FF&cmd=sleep_24h"

# Exit agent immediately
curl -X POST "http://server:8080/api/admin/command?mac=AA:BB:CC:DD:EE:FF&cmd=exit"
```

## Production Features

### Circuit Breaker

- **Server**: 100 requests/second limit, returns `429 Too Many Requests`
- **Agent**: Exponential backoff on 429: 5s, 10s, 20s, then give up

### Log Rotation

- File: `logs/itam.log`
- Max size: 10MB per file
- Keep: 7 rotated files
- Summary logging: 1 log/minute instead of per-request

### Auto Backup

- Automatic backup on server startup
- Location: `backups/itam_YYYY-MM-DD_HH-MM-SS.db`
- Cleanup: Keep 7 days, minimum 5 backups

### Data Pruning

- Auto-delete history records > 30 days on startup
- Keeps database size manageable

## Project Structure

```
itam-project/
├── agent/
│   ├── main.go         # Agent source
│   └── config.json     # Agent config (with auth_token)
├── server/
│   ├── main.go         # Server entry
│   ├── handlers.go     # HTTP handlers + auth + rate limiter
│   ├── store.go        # SQLite + backup + pruning
│   ├── logger.go       # Log rotation
│   ├── templates.go    # HTMX partials
│   └── templates/      # HTML templates
├── docs/
│   └── SCALING.md      # Scaling guide
├── logs/               # Rotating log files
├── backups/            # Auto-generated backups
└── Makefile
```

## Roadmap

- [ ] Printer monitoring (SNMP)
- [ ] Network devices (switches, routers)
- [ ] Software inventory
- [ ] Remote actions (shutdown, restart)
- [ ] LDAP/AD authentication for dashboard

## Requirements

- Go 1.21+
- Windows (agent) / Linux (server)
- Network access between agents and server

## License

MIT
