package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// AssetStore handles SQLite database operations
type AssetStore struct {
	db *sql.DB
}

// Global store instance
var store *AssetStore

// InitStore initializes the SQLite database
func InitStore(dbPath string) error {
	// Create directory if it doesn't exist
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// WAL mode + Long busy timeout + Transaction locking
	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=10000&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	// Critical for SQLite: Limit concurrency to avoid "database is locked" errors
	// Even with WAL, 1 writer is the limit.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Create assets table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS assets (
			mac_address TEXT PRIMARY KEY,
			hostname TEXT,
			os TEXT,
			ip_lan TEXT,
			cpu_model TEXT,
			cpu_usage REAL,
			cores INTEGER,
			ram_bytes INTEGER,
			ram_usage_bytes INTEGER,
			ram_percent REAL,
			disk_bytes INTEGER,
			disk_usage_bytes INTEGER,
			disk_percent REAL,
			uptime INTEGER,
			serial_number TEXT,
			bios_version TEXT,
			top_processes TEXT,
			last_seen DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create history table for tracking metrics over time
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS asset_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mac_address TEXT,
			cpu_usage REAL,
			ram_percent REAL,
			disk_percent REAL,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (mac_address) REFERENCES assets(mac_address)
		)
	`)
	if err != nil {
		return err
	}

	// Create alerts table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mac_address TEXT,
			hostname TEXT,
			alert_type TEXT,
			message TEXT,
			severity TEXT,
			is_resolved INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		)
	`)
	if err != nil {
		return err
	}

	// Create index for faster history queries
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_history_mac ON asset_history(mac_address)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_history_time ON asset_history(recorded_at)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(is_resolved)`)

	store = &AssetStore{db: db}
	log.Printf("Database initialized: %s", dbPath)
	return nil
}

// Close closes the database connection
func (s *AssetStore) Close() error {
	return s.db.Close()
}

// BackupDatabase creates a backup of the database with timestamp
func BackupDatabase(dbPath string) (string, error) {
	// Create backups directory if not exists
	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupName := fmt.Sprintf("itam_%s.db", timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	// Open source file
	src, err := os.Open(dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}
	defer dst.Close()

	// Copy file
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy database: %w", err)
	}

	log.Printf("Database backup created: %s", backupPath)
	return backupPath, nil
}

// CleanOldBackups removes backups older than specified days, keeping minimum count
func CleanOldBackups(backupDir string, keepDays int, minKeep int) {
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -keepDays)
	var backups []os.DirEntry

	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".db" {
			backups = append(backups, f)
		}
	}

	// Don't delete if we have fewer than minKeep backups
	if len(backups) <= minKeep {
		return
	}

	for _, f := range backups[:len(backups)-minKeep] {
		info, err := f.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(backupDir, f.Name())
			os.Remove(path)
			log.Printf("Removed old backup: %s", path)
		}
	}
}

// PruneOldData deletes history records older than specified days
func (s *AssetStore) PruneOldData(days int) (int64, error) {
	result, err := s.db.Exec(`
		DELETE FROM asset_history 
		WHERE recorded_at < datetime('now', 'localtime', ? || ' days')
	`, fmt.Sprintf("-%d", days))
	if err != nil {
		return 0, err
	}

	deleted, _ := result.RowsAffected()
	if deleted > 0 {
		log.Printf("Pruned %d old history records (> %d days)", deleted, days)
	}
	return deleted, nil
}

// Save stores an asset by MAC address (upsert)
func (s *AssetStore) Save(report AgentReport) error {
	// 1. Fetch old asset to check for hardware changes
	oldAsset, _ := s.GetByMAC(report.MACAddress)

	processesJSON, _ := json.Marshal(report.TopProcesses)

	_, err := s.db.Exec(`
		INSERT INTO assets (
			mac_address, hostname, os, ip_lan, cpu_model, cpu_usage, cores,
			ram_bytes, ram_usage_bytes, ram_percent, disk_bytes, disk_usage_bytes, disk_percent, uptime,
			serial_number, bios_version, top_processes, last_seen
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mac_address) DO UPDATE SET
			hostname = excluded.hostname,
			os = excluded.os,
			ip_lan = excluded.ip_lan,
			cpu_model = excluded.cpu_model,
			cpu_usage = excluded.cpu_usage,
			cores = excluded.cores,
			ram_bytes = excluded.ram_bytes,
			ram_usage_bytes = excluded.ram_usage_bytes,
			ram_percent = excluded.ram_percent,
			disk_bytes = excluded.disk_bytes,
			disk_usage_bytes = excluded.disk_usage_bytes,
			disk_percent = excluded.disk_percent,
			uptime = excluded.uptime,
			serial_number = excluded.serial_number,
			bios_version = excluded.bios_version,
			top_processes = excluded.top_processes,
			last_seen = excluded.last_seen
	`,
		report.MACAddress, report.Hostname, report.OS, report.IPLAN,
		report.CPU, report.CPUUsage, report.Cores,
		report.RAM, report.RAMUsage, report.RAMPercent, report.Disk, report.DiskUsage, report.DiskPercent, report.Uptime,
		report.SerialNumber, report.BiosVersion, string(processesJSON), time.Now(),
	)
	
	if err == nil {
		// 2. Check for alerts (thresholds and hardware changes)
		s.CheckAndCreateAlerts(oldAsset, report)
	}

	return err
}

// GetAll returns all assets sorted by hostname
func (s *AssetStore) GetAll() ([]Asset, error) {
	rows, err := s.db.Query(`
		SELECT mac_address, hostname, os, ip_lan, cpu_model, cpu_usage, cores,
			ram_bytes, ram_usage_bytes, ram_percent, disk_bytes, disk_usage_bytes, disk_percent, uptime,
			serial_number, bios_version, top_processes, last_seen
		FROM assets ORDER BY hostname
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var report AgentReport
		var lastSeen time.Time
		var processesJSON string

		err := rows.Scan(
			&report.MACAddress, &report.Hostname, &report.OS, &report.IPLAN,
			&report.CPU, &report.CPUUsage, &report.Cores,
			&report.RAM, &report.RAMUsage, &report.RAMPercent, &report.Disk, &report.DiskUsage, &report.DiskPercent, &report.Uptime,
			&report.SerialNumber, &report.BiosVersion, &processesJSON, &lastSeen,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(processesJSON), &report.TopProcesses)
		assets = append(assets, Asset{Report: report, LastSeen: lastSeen})
	}
	return assets, nil
}

// Stats returns aggregate statistics (optimized to 1 query for 500+ nodes)
// AVG values are calculated from ONLINE machines only (last_seen < 5 minutes)
func (s *AssetStore) Stats() (total, online int, avgCPU, avgRAM, avgDisk float64) {
	s.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN last_seen > datetime('now', 'localtime', '-5 minutes') THEN 1 ELSE 0 END) as online,
			COALESCE((SELECT AVG(cpu_usage) FROM assets WHERE last_seen > datetime('now', 'localtime', '-5 minutes')), 0) as avg_cpu,
			COALESCE((SELECT AVG(ram_percent) FROM assets WHERE last_seen > datetime('now', 'localtime', '-5 minutes')), 0) as avg_ram,
			COALESCE((SELECT AVG(disk_percent) FROM assets WHERE last_seen > datetime('now', 'localtime', '-5 minutes')), 0) as avg_disk
		FROM assets
	`).Scan(&total, &online, &avgCPU, &avgRAM, &avgDisk)

	return
}

// Search returns paged assets matching the query with sorting
func (s *AssetStore) Search(query string, limit, offset int, sortBy, sortOrder string) ([]Asset, int, error) {
	searchPattern := "%" + query + "%"

	// Validate sort params (Whitelisting to prevent SQL Injection)
	validSortCols := map[string]string{
		"hostname":     "hostname",
		"os":           "os",
		"ip_lan":       "ip_lan",
		"cpu_usage":    "cpu_usage",
		"ram_percent":  "ram_percent",
		"disk_percent": "disk_percent",
		"uptime":       "uptime",
		"last_seen":    "last_seen",
	}

	sortCol, ok := validSortCols[sortBy]
	if !ok {
		sortCol = "last_seen" // Default sort
	}

	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC" // Default order
	}

	// Get total count for pagination
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM assets WHERE hostname LIKE ? OR ip_lan LIKE ? OR mac_address LIKE ?",
		searchPattern, searchPattern, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Dynamic ORDER BY
	sqlQuery := fmt.Sprintf(`
		SELECT mac_address, hostname, os, ip_lan, cpu_model, cpu_usage, cores,
			ram_bytes, ram_usage_bytes, ram_percent, disk_bytes, disk_usage_bytes, disk_percent, uptime,
			serial_number, bios_version, last_seen, top_processes
		FROM assets
		WHERE hostname LIKE ? OR ip_lan LIKE ? OR mac_address LIKE ?
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, sortCol, sortOrder)

	rows, err := s.db.Query(sqlQuery, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		var processesJSON string
		if err := rows.Scan(&a.Report.MACAddress, &a.Report.Hostname, &a.Report.OS, &a.Report.IPLAN, &a.Report.CPU,
			&a.Report.CPUUsage, &a.Report.Cores, &a.Report.RAM, &a.Report.RAMUsage, &a.Report.RAMPercent,
			&a.Report.Disk, &a.Report.DiskUsage, &a.Report.DiskPercent, &a.Report.Uptime,
			&a.Report.SerialNumber, &a.Report.BiosVersion, &a.LastSeen, &processesJSON); err != nil {
			return nil, 0, err
		}
		json.Unmarshal([]byte(processesJSON), &a.Report.TopProcesses)
		assets = append(assets, a)
	}
	return assets, total, nil
}

// GetByMAC returns a single asset by MAC address
func (s *AssetStore) GetByMAC(macAddress string) (*Asset, error) {
	var report AgentReport
	var lastSeen time.Time
	var processesJSON string

	err := s.db.QueryRow(`
		SELECT mac_address, hostname, os, ip_lan, cpu_model, cpu_usage, cores,
			ram_bytes, ram_usage_bytes, ram_percent, disk_bytes, disk_usage_bytes, disk_percent, uptime,
			serial_number, bios_version, top_processes, last_seen
		FROM assets WHERE mac_address = ?
	`, macAddress).Scan(
		&report.MACAddress, &report.Hostname, &report.OS, &report.IPLAN,
		&report.CPU, &report.CPUUsage, &report.Cores,
		&report.RAM, &report.RAMUsage, &report.RAMPercent, &report.Disk, &report.DiskUsage, &report.DiskPercent, &report.Uptime,
		&report.SerialNumber, &report.BiosVersion, &processesJSON, &lastSeen,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(processesJSON), &report.TopProcesses)
	return &Asset{Report: report, LastSeen: lastSeen}, nil
}

// AllProcesses returns all processes from all assets
func (s *AssetStore) AllProcesses() []ProcessWithAsset {
	rows, err := s.db.Query(`SELECT hostname, top_processes FROM assets`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var all []ProcessWithAsset
	for rows.Next() {
		var hostname, processesJSON string
		if err := rows.Scan(&hostname, &processesJSON); err != nil {
			continue
		}

		var processes []TopProcess
		json.Unmarshal([]byte(processesJSON), &processes)

		for _, proc := range processes {
			all = append(all, ProcessWithAsset{
				Hostname: hostname,
				Process:  proc,
			})
		}
	}
	return all
}

// ProcessWithAsset links a process to its host
type ProcessWithAsset struct {
	Hostname string
	Process  TopProcess
}

// Alert represents a system alert
type Alert struct {
	ID         int       `json:"id"`
	MACAddress string    `json:"mac_address"`
	Hostname   string    `json:"hostname"`
	AlertType  string    `json:"alert_type"`
	Message    string    `json:"message"`
	Severity   string    `json:"severity"`
	IsResolved bool      `json:"is_resolved"`
	CreatedAt  time.Time `json:"created_at"`
}

// HistoryPoint represents a single history data point
type HistoryPoint struct {
	RecordedAt time.Time `json:"recorded_at"`
	CPUUsage   float64   `json:"cpu_usage"`
	RAMPercent float64   `json:"ram_percent"`
}

// SaveHistory records a history point for an asset
func (s *AssetStore) SaveHistory(macAddress string, cpuUsage, ramPercent float64) error {
	_, err := s.db.Exec(`
		INSERT INTO asset_history (mac_address, cpu_usage, ram_percent)
		VALUES (?, ?, ?)
	`, macAddress, cpuUsage, ramPercent)
	return err
}

// GetHistory returns history for an asset (last 24 hours)
func (s *AssetStore) GetHistory(macAddress string) ([]HistoryPoint, error) {
	rows, err := s.db.Query(`
		SELECT cpu_usage, ram_percent, recorded_at
		FROM asset_history
		WHERE mac_address = ? AND recorded_at > datetime('now', '-24 hours')
		ORDER BY recorded_at ASC
	`, macAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []HistoryPoint
	for rows.Next() {
		var h HistoryPoint
		if err := rows.Scan(&h.CPUUsage, &h.RAMPercent, &h.RecordedAt); err != nil {
			continue
		}
		history = append(history, h)
	}
	return history, nil
}

// CheckAndCreateAlerts checks thresholds and creates alerts
func (s *AssetStore) CheckAndCreateAlerts(oldAsset *Asset, report AgentReport) {
	// Hardware Change Detection
	if oldAsset != nil {
		// Detect RAM Downgrade (e.g. 16GB -> 8GB)
		// Usually memory varies a tiny bit in reporting, but drops > 1GB mean parts missing
		if oldAsset.Report.RAM > report.RAM && (oldAsset.Report.RAM-report.RAM) > 1024*1024*1024 {
			msg := fmt.Sprintf("Hardware Alert: RAM dropped from %.1fGB to %.1fGB!", 
				float64(oldAsset.Report.RAM)/1024/1024/1024, 
				float64(report.RAM)/1024/1024/1024)
			s.createAlert(report.MACAddress, report.Hostname, "HARDWARE_CHANGED", msg, "critical")
		}

		// Detect Disk Downgrade (e.g swap large drive for small drive)
		if oldAsset.Report.Disk > report.Disk && (oldAsset.Report.Disk-report.Disk) > 10*1024*1024*1024 {
			msg := fmt.Sprintf("Hardware Alert: Disk dropped from %.1fGB to %.1fGB!", 
				float64(oldAsset.Report.Disk)/1024/1024/1024, 
				float64(report.Disk)/1024/1024/1024)
			s.createAlert(report.MACAddress, report.Hostname, "HARDWARE_CHANGED", msg, "critical")
		}

		// Detect CPU Swap
		if oldAsset.Report.CPU != report.CPU && oldAsset.Report.CPU != "" {
			msg := fmt.Sprintf("Hardware Alert: CPU changed from '%s' to '%s'!", oldAsset.Report.CPU, report.CPU)
			s.createAlert(report.MACAddress, report.Hostname, "HARDWARE_CHANGED", msg, "critical")
		}
	}

	// High CPU alert (>90%)
	if report.CPUUsage > 90 {
		s.createAlert(report.MACAddress, report.Hostname, "HIGH_CPU",
			"CPU usage above 90%", "warning")
	}

	// Critical CPU alert (>95%)
	if report.CPUUsage > 95 {
		s.createAlert(report.MACAddress, report.Hostname, "CRITICAL_CPU",
			"CPU usage critically high (>95%)", "critical")
	}

	// High RAM alert (>90%)
	if report.RAMPercent > 90 {
		s.createAlert(report.MACAddress, report.Hostname, "HIGH_RAM",
			"RAM usage above 90%", "warning")
	}

	// Critical RAM alert (>95%)
	if report.RAMPercent > 95 {
		s.createAlert(report.MACAddress, report.Hostname, "CRITICAL_RAM",
			"RAM usage critically high (>95%)", "critical")
	}
}

func (s *AssetStore) createAlert(macAddress, hostname, alertType, message, severity string) {
	// Check if similar unresolved alert exists (within last hour)
	var count int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM alerts 
		WHERE mac_address = ? AND alert_type = ? AND is_resolved = 0
		AND created_at > datetime('now', '-1 hour')
	`, macAddress, alertType).Scan(&count)

	if count == 0 {
		s.db.Exec(`
			INSERT INTO alerts (mac_address, hostname, alert_type, message, severity)
			VALUES (?, ?, ?, ?, ?)
		`, macAddress, hostname, alertType, message, severity)
	}
}

// GetAlerts returns recent alerts
func (s *AssetStore) GetAlerts(limit int, includeResolved bool) ([]Alert, error) {
	query := `
		SELECT id, mac_address, hostname, alert_type, message, severity, is_resolved, created_at
		FROM alerts
	`
	if !includeResolved {
		query += ` WHERE is_resolved = 0`
	}
	query += ` ORDER BY created_at DESC LIMIT ?`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		var isResolved int
		if err := rows.Scan(&a.ID, &a.MACAddress, &a.Hostname, &a.AlertType,
			&a.Message, &a.Severity, &isResolved, &a.CreatedAt); err != nil {
			continue
		}
		a.IsResolved = isResolved == 1
		alerts = append(alerts, a)
	}
	return alerts, nil
}

// ResolveAlert marks an alert as resolved
func (s *AssetStore) ResolveAlert(id int) error {
	_, err := s.db.Exec(`
		UPDATE alerts SET is_resolved = 1, resolved_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// GetUnresolvedAlertCount returns count of unresolved alerts
func (s *AssetStore) GetUnresolvedAlertCount() int {
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM alerts WHERE is_resolved = 0`).Scan(&count)
	return count
}

// ExportAssetsCSV returns all assets as properly escaped CSV string
func (s *AssetStore) ExportAssetsCSV() (string, error) {
	assets, err := s.GetAll()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	writer.Write([]string{
		"MAC Address", "Hostname", "OS", "IP Address", "CPU Model",
		"CPU Usage %", "RAM Total GB", "RAM Usage %", "Disk GB", "Serial Number", "Last Seen",
	})

	// Write data rows
	for _, a := range assets {
		writer.Write([]string{
			a.Report.MACAddress,
			a.Report.Hostname,
			a.Report.OS,
			a.Report.IPLAN,
			a.Report.CPU,
			fmt.Sprintf("%.1f", a.Report.CPUUsage),
			fmt.Sprintf("%.2f", float64(a.Report.RAM)/1024/1024/1024),
			fmt.Sprintf("%.1f", a.Report.RAMPercent),
			fmt.Sprintf("%.2f", float64(a.Report.Disk)/1024/1024/1024),
			a.Report.SerialNumber,
			a.LastSeen.Format("2006-01-02 15:04:05"),
		})
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// CleanOldHistory removes history older than 7 days
func (s *AssetStore) CleanOldHistory() {
	s.db.Exec(`DELETE FROM asset_history WHERE recorded_at < datetime('now', '-7 days')`)
}
