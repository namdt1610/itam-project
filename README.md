# ITAM — IT Asset Management System

> Real-time hardware monitoring for enterprise environments.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/SQLite-WAL_Mode-003B57?logo=sqlite&logoColor=white" alt="SQLite">
  <img src="https://img.shields.io/badge/HTMX-Realtime_Dashboard-3366CC" alt="HTMX">
  <img src="https://img.shields.io/badge/License-MIT-green" alt="MIT">
</p>

**[🇻🇳 Tiếng Việt](README.vi.md)**

---

## Features

| Feature                    | Description                                         |
| -------------------------- | --------------------------------------------------- |
| 📊 **Real-time Dashboard** | Monitor CPU, RAM, Disk usage with live HTMX updates |
| 🔍 **Search & Filter**     | Search by hostname, IP, MAC address                 |
| 📈 **Usage History**       | CPU/RAM charts over time (Chart.js)                 |
| 🚨 **Alerts**              | Auto-alerts when CPU/RAM > 90%                      |
| 💾 **Auto Backup**         | Database backup on startup                          |
| 🛡️ **Circuit Breaker**     | Rate limiting (100 req/s) + exponential backoff     |
| 📝 **Log Rotation**        | Auto-rotate logs, 10MB/file, keep 7                 |
| 📥 **Export CSV**          | Export full asset list                              |
| 🔑 **Auth Token**          | Bearer token for agent-server communication         |
| ⏹️ **Kill Switch**         | Remote sleep/exit agents                            |

---

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
|  Dashboard  |                          |   (WAL)     |
+-------------+                          +-------------+
```

**How it works:**

1. **Agent** collects system info (CPU, RAM, Disk, processes) using `gopsutil`
2. Agent **POSTs** data to server with bearer token authentication
3. **Server** stores data in SQLite (WAL mode) and generates alerts
4. **Dashboard** updates in real-time via HTMX polling

---

## Quick Start

### Prerequisites

- Go 1.21+
- Windows (agent) / Linux (server)
- Network access between agents and server

### Build & Run

```bash
# Build both agent and server
make build

# Run server
./build/server_bin

# Run agent on a client machine
cd agent && ./build/agent_bin
```

### Production Agent (Single EXE)

```bash
# Build agent with embedded config (no config.json needed)
make build-agent-prod \
  SERVER_URL=http://192.168.1.100:8080/api/report \
  AUTH_TOKEN=your-secret-token

# Output: ./build/agent.exe — deploy via GPO
```

### Dashboard

Open http://localhost:8080 after starting the server.

---

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

| Field               | Description                                   |
| ------------------- | --------------------------------------------- |
| `server_url`        | Server endpoint                               |
| `auth_token`        | Secret token (must match server)              |
| `top_process_limit` | Number of top processes to collect            |
| `jitter_seconds`    | Random delay (0–N) to prevent thundering herd |
| `log_level`         | `"debug"`, `"info"`, or `"error"`             |

### Server (`server/config.json`)

```json
{
  "auth_token": "your-secret-token-here",
  "port": 8080,
  "db_path": "data/itam.db"
}
```

> **Config priority (Agent):** Build-time flags → Environment variable (`ITAM_AUTH_TOKEN`) → `config.json` next to exe → `./config.json`

---

## Scaling Guide

| Nodes    | GPO Interval | Agent Jitter | Dashboard Poll | Database   |
| -------- | ------------ | ------------ | -------------- | ---------- |
| 1–100    | 5 min        | 15s          | 15s            | SQLite     |
| 100–500  | 5–15 min     | 30s          | 30s            | SQLite     |
| 500–2000 | 15–30 min    | 60s          | 60s            | SQLite     |
| 2000+    | 30+ min      | 60s          | 60s            | PostgreSQL |

```
Peak req/s = Nodes ÷ Jitter
Example: 400 nodes ÷ 30s jitter ≈ 13 req/s (well under 100 limit)
```

See [docs/SCALING.md](docs/SCALING.md) for detailed tuning.

---

## API Reference

| Endpoint                           | Method | Auth  | Description           |
| ---------------------------------- | ------ | ----- | --------------------- |
| `/api/report`                      | POST   | Token | Agent data submission |
| `/api/assets`                      | GET    | —     | All assets (JSON)     |
| `/api/assets/table`                | GET    | —     | Assets table (HTMX)   |
| `/api/assets/detail?mac=XX`        | GET    | —     | Asset detail          |
| `/api/stats`                       | GET    | —     | Aggregate stats       |
| `/api/alerts`                      | GET    | —     | Alerts list           |
| `/api/alerts/resolve`              | POST   | —     | Resolve alert         |
| `/api/history?mac=XX`              | GET    | —     | Usage history (24h)   |
| `/api/export/csv`                  | GET    | —     | Export CSV            |
| `/api/backup`                      | GET    | —     | Manual backup         |
| `/health`                          | GET    | —     | Health check          |
| `/api/admin/command?mac=XX&cmd=YY` | POST   | Token | Kill switch           |
| `/dl/agent.exe`                    | GET    | —     | Download agent        |

### Kill Switch Commands

```bash
# Sleep agent for 1 hour
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  "http://server:8080/api/admin/command?mac=AA:BB:CC:DD:EE:FF&cmd=sleep_1h"

# Sleep agent for 24 hours
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  "http://server:8080/api/admin/command?mac=AA:BB:CC:DD:EE:FF&cmd=sleep_24h"

# Exit agent immediately
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  "http://server:8080/api/admin/command?mac=AA:BB:CC:DD:EE:FF&cmd=exit"
```

---

## Deployment (Windows GPO)

```powershell
$action = New-ScheduledTaskAction -Execute "C:\ITAM\agent.exe"
$trigger = New-ScheduledTaskTrigger -Once -At (Get-Date) `
  -RepetitionInterval (New-TimeSpan -Minutes 15)
$settings = New-ScheduledTaskSettingsSet `
  -RandomDelay (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName "ITAM Agent" `
  -Action $action -Trigger $trigger -Settings $settings
```

1. Build production agent: `make build-agent-prod SERVER_URL=... AUTH_TOKEN=...`
2. Copy `agent.exe` to shared folder
3. Create GPO to deploy the scheduled task
4. Agent runs → collects data → sends to server → exits

---

## Production Features

| Feature             | Detail                                                                 |
| ------------------- | ---------------------------------------------------------------------- |
| **Circuit Breaker** | Server: 100 req/s limit (429). Agent: exponential backoff 5s→10s→20s   |
| **Log Rotation**    | `logs/itam.log`, 10MB/file, keep 7 rotated files, 1 log/min summary    |
| **Auto Backup**     | On startup → `backups/itam_YYYY-MM-DD_HH-MM-SS.db`. Keep 7 days, min 5 |
| **Data Pruning**    | Auto-delete history > 30 days on startup                               |

---

## Project Structure

```
itam-project/
├── agent/
│   ├── main.go           # Agent source (Windows client)
│   └── config.json       # Agent config (gitignored)
├── server/
│   ├── main.go           # Server entry + graceful shutdown
│   ├── handlers.go       # HTTP handlers + auth + rate limiter
│   ├── store.go          # SQLite + backup + pruning
│   ├── models.go         # Data models
│   ├── config.go         # Server configuration
│   ├── logger.go         # Rotating log system
│   ├── templates.go      # HTMX partial renderers
│   ├── utils.go          # Helper functions
│   └── templates/
│       └── index.html    # Dashboard (Tabler.io)
├── web/static/           # CSS, JS, vendor libs
├── cmd/loadtest/         # Load test (500 agents)
├── docs/SCALING.md       # Scaling guide
└── Makefile
```

---

## Load Testing

```bash
make loadtest   # 500 simulated agents
```

---

## Tech Stack

| Layer    | Technology                       |
| -------- | -------------------------------- |
| Backend  | Go (net/http)                    |
| Database | SQLite (WAL, modernc.org/sqlite) |
| Agent    | gopsutil v3                      |
| Frontend | HTMX + Tabler.io + Chart.js      |

---

## Roadmap

- [ ] Printer monitoring (SNMP)
- [ ] Network devices (switches, routers)
- [ ] Software inventory
- [ ] Remote actions (shutdown, restart)
- [ ] LDAP/AD authentication for dashboard
- [ ] HTTPS/TLS support

## License

MIT
