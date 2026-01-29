package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	// Load config
	if err := LoadServerConfig(); err != nil {
		fmt.Printf("Critical: Failed to load config: %v\n", err)
		return
	}
	dbPath := config.DBPath
	// Auto-backup existing database on startup
	if _, err := os.Stat(dbPath); err == nil {
		backupPath, err := BackupDatabase(dbPath)
		if err != nil {
			fmt.Printf("Warning: Failed to backup database: %v\n", err)
		} else {
			fmt.Printf("Auto-backup created: %s\n", backupPath)
		}
		// Clean old backups - keep 7 days, minimum 5 backups
		CleanOldBackups("backups", 7, 5)
	}

	// Initialize SQLite database
	if err := InitStore(dbPath); err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		return
	}
	defer store.Close()

	// Initialize rotating logger (10MB max, keep 7 files)
	if err := InitLogger("logs/itam.log", 10, 7); err != nil {
		fmt.Printf("Warning: Failed to init logger: %v\n", err)
	} else {
		defer appLogger.Close()
	}

	// Start log summary goroutine
	go LogSummary()

	// Prune old data on startup (keep 30 days of history)
	store.PruneOldData(30)

	// API endpoints
	http.HandleFunc("/api/report", handleReport)
	http.HandleFunc("/api/assets", handleAssets)
	http.HandleFunc("/api/assets/table", handleAssetsTable)
	http.HandleFunc("/api/assets/detail", handleAssetDetail)
	http.HandleFunc("/api/stats", handleStats)
	http.HandleFunc("/api/stats/cards", handleStatsCards)
	http.HandleFunc("/api/processes/table", handleProcessesTable)
	http.HandleFunc("/api/alerts", handleAlerts)
	http.HandleFunc("/api/alerts/table", handleAlertsTable)
	http.HandleFunc("/api/alerts/resolve", handleResolveAlert)
	http.HandleFunc("/api/history", handleHistory)
	http.HandleFunc("/api/export/csv", handleExportCSV)
	http.HandleFunc("/api/backup", handleBackup)
	http.HandleFunc("/api/admin/command", handleAdminCommand)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/dl/agent.exe", handleDownloadAgent)

	// Static files
	http.HandleFunc("/static/", handleStatic)

	// Dashboard
	http.HandleFunc("/", handleDashboard)

	printBanner()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}

func printBanner() {
	fmt.Println("╔════════════════════════════════════════════╗")
	fmt.Println("║     ITAM Server - IT Asset Management      ║")
	fmt.Println("╠════════════════════════════════════════════╣")
	fmt.Println("║  Dashboard: http://localhost:8080          ║")
	fmt.Println("║  API:       http://localhost:8080/api      ║")
	fmt.Println("╚════════════════════════════════════════════╝")
}

func currentTime() string {
	return time.Now().Format("15:04:05")
}
