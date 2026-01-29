# ITAM Scaling Configuration Guide

Hướng dẫn tinh chỉnh cấu hình theo số lượng nodes.

## Quick Reference

| Nodes    | GPO Interval | Agent Jitter | Dashboard Poll | Database   |
| -------- | ------------ | ------------ | -------------- | ---------- |
| 1-100    | 5 phút       | 15s          | 15s            | SQLite     |
| 100-500  | 5-15 phút    | 30s          | 30s            | SQLite     |
| 500-2000 | 15-30 phút   | 60s          | 60s            | SQLite     |
| 2000+    | 30+ phút     | 60s          | 60s            | PostgreSQL |

---

## Agent Configuration

### File: `agent/config.json`

```json
{
  "server_url": "http://your-server:8080/api/report",
  "top_process_limit": 5,
  "log_level": "error"
}
```

### Jitter Setting (agent/main.go)

```go
// Line 76-77: Adjust based on GPO interval
jitterSeconds := rand.Intn(30)  // 500 nodes, GPO 5 phút
jitterSeconds := rand.Intn(60)  // 2000 nodes, GPO 30 phút
```

**Rule**: Jitter should be ~10-20% of GPO interval

---

## Dashboard Polling

### File: `server/templates/index.html`

| Component    | Line | Low traffic | High traffic |
| ------------ | ---- | ----------- | ------------ |
| Stats Cards  | 128  | `every 15s` | `every 60s`  |
| Alerts       | 318  | `every 30s` | `every 60s`  |
| Assets Table | 405  | `every 15s` | `every 60s`  |
| Processes    | 449  | `every 30s` | `every 60s`  |

---

## GPO Scheduling (Windows)

### Recommended Settings:

- **Trigger**: On a schedule, repeat every X minutes
- **Random delay**: Enable "Delay task for up to: 1 minute"
- **Run whether user is logged on or not**: ✅

### PowerShell deployment:

```powershell
$action = New-ScheduledTaskAction -Execute "C:\ITAM\agent.exe"
$trigger = New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Minutes 15)
$settings = New-ScheduledTaskSettingsSet -RandomDelay (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName "ITAM Agent" -Action $action -Trigger $trigger -Settings $settings
```

---

## Database Tuning

### SQLite (default, up to 2000 nodes)

No changes needed. Auto-backup on startup.

### PostgreSQL (2000+ nodes)

When migrating:

1. Change `store.go` to use `pq` driver
2. Update connection string in `main.go`
3. Add connection pooling

---

## Monitoring Thresholds

### Server Load Indicators:

| Metric        | Normal | Warning   | Critical |
| ------------- | ------ | --------- | -------- |
| Requests/sec  | <50    | 50-100    | >100     |
| Response time | <100ms | 100-500ms | >500ms   |
| DB size       | <500MB | 500MB-1GB | >1GB     |

### When to scale:

- Response time >500ms consistently → Increase poll intervals
- DB size >1GB → Enable history cleanup, consider PostgreSQL
- > 100 req/s sustained → Add caching layer

---

## Performance Formulas

```
Peak requests/sec = Nodes / Jitter_seconds
Example: 500 nodes / 30s jitter = ~17 req/s

Daily data points = Nodes × (1440 / GPO_interval_minutes)
Example: 500 nodes × (1440 / 15) = 48,000 records/day
```
