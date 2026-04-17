package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed templates/*
var templatesFS embed.FS

// Rate limiter for /api/report endpoint
var (
	rateLimitMu       sync.Mutex
	rateLimitCount    int64
	rateLimitWindow         = time.Second
	maxRequestsPerSec int64 = 100 // Max 100 requests/second
	lastReset               = time.Now()
)

// Agent commands storage (kill switch)
var (
	agentCommandsMu sync.RWMutex
	agentCommands   = make(map[string]string) // MAC -> command
)

// getAgentCommand returns and clears command for an agent
func getAgentCommand(mac string) string {
	agentCommandsMu.Lock()
	defer agentCommandsMu.Unlock()
	cmd := agentCommands[mac]
	delete(agentCommands, mac) // One-time command
	return cmd
}

// setAgentCommand sets a command for an agent (called by admin API)
func setAgentCommand(mac, command string) {
	agentCommandsMu.Lock()
	defer agentCommandsMu.Unlock()
	agentCommands[mac] = command
}

// checkRateLimit returns true if request should be allowed
func checkRateLimit() bool {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	now := time.Now()
	if now.Sub(lastReset) > rateLimitWindow {
		rateLimitCount = 0
		lastReset = now
	}

	if rateLimitCount >= maxRequestsPerSec {
		return false // Rate limited
	}

	rateLimitCount++
	return true
}

// handleDashboard serves the main dashboard page
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFS(templatesFS, "templates/index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		fmt.Println("Template error:", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, nil)
}

// handleStatic serves static files (CSS, JS)
func handleStatic(w http.ResponseWriter, r *http.Request) {
	path := "web/static" + r.URL.Path[7:] // Remove "/static" prefix

	// Set correct MIME types
	if strings.HasSuffix(path, ".js") {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	} else if strings.HasSuffix(path, ".css") {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}

	http.ServeFile(w, r, path)
}

// checkAuthToken validates the bearer token from the request.
// Returns true if authorized, false if rejected (and writes HTTP error).
func checkAuthToken(w http.ResponseWriter, r *http.Request) bool {
	if config.AuthToken == "" {
		return true // No token configured, allow all
	}
	authHeader := r.Header.Get("Authorization")
	expectedToken := "Bearer " + config.AuthToken
	if authHeader != expectedToken {
		masked := "empty"
		if len(authHeader) > 10 {
			masked = authHeader[:10] + "..."
		}
		fmt.Printf("[AUTH FAIL] Received: '%s', Expected: '%s...'\n", masked, expectedToken[:10])
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// handleReport receives agent reports
func handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Circuit breaker: rate limit check
	if !checkRateLimit() {
		w.Header().Set("Retry-After", "5")
		http.Error(w, "Too many requests - server overloaded", http.StatusTooManyRequests)
		return
	}

	// Auth token validation
	if !checkAuthToken(w, r) {
		return
	}

	// Limit request body to 1MB to prevent OOM
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Request body too large or read error", http.StatusRequestEntityTooLarge)
		return
	}
	defer r.Body.Close()

	var report AgentReport
	if err := json.Unmarshal(body, &report); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Store the asset
	// Hardware change detection and performance threshold alerts are handled inside Save() now,
	// so we don't need to call CheckAndCreateAlerts here.
	if err := store.Save(report); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		LogError("Database save failed: %v", err)
		return
	}

	// We do save history though.
	store.SaveHistory(report.MACAddress, report.CPUUsage, report.RAMPercent)

	// Increment request counter (summary logged every minute)
	LogRequest()

	// Return JSON response with optional command
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"command": getAgentCommand(report.MACAddress),
	})
}

// handleAssets returns all assets as JSON
func handleAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := store.GetAll()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

// handleAssetsTable returns HTMX partial for assets table
func handleAssetsTable(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page := atoi(r.URL.Query().Get("page"), 1)
	limit := atoi(r.URL.Query().Get("limit"), 20)

	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "last_seen"
	}

	sortOrder := r.URL.Query().Get("order")
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	renderAssetsTableWithPagination(w, query, page, limit, sortBy, sortOrder)
}

func atoi(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n < 1 {
		return defaultVal
	}
	return n
}

// handleStats returns aggregate stats as JSON
func handleStats(w http.ResponseWriter, r *http.Request) {
	total, online, avgCPU, avgRAM, avgDisk := store.Stats()

	stats := map[string]interface{}{
		"total":   total,
		"online":  online,
		"avgCpu":  avgCPU,
		"avgRam":  avgRAM,
		"avgDisk": avgDisk,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleStatsCards returns HTMX partial for stats cards
func handleStatsCards(w http.ResponseWriter, r *http.Request) {
	total, online, avgCPU, avgRAM, avgDisk := store.Stats()
	renderStatsCards(w, total, online, avgCPU, avgRAM, avgDisk)
}

// handleProcessesTable returns HTMX partial for processes table
func handleProcessesTable(w http.ResponseWriter, r *http.Request) {
	renderProcessesTable(w)
}

// handleAssetDetail returns HTMX partial for asset detail modal
func handleAssetDetail(w http.ResponseWriter, r *http.Request) {
	mac := r.URL.Query().Get("mac")
	if mac == "" {
		http.Error(w, "Missing mac parameter", http.StatusBadRequest)
		return
	}

	asset, err := store.GetByMAC(mac)
	if err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}

	renderAssetDetail(w, asset)
}

// handleDownloadAgent serves the agent executable
func handleDownloadAgent(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat("agent.exe"); os.IsNotExist(err) {
		http.Error(w, "Agent executable not found on server", http.StatusNotFound)
		return
	}
	fmt.Println("[DOWNLOAD] Agent download requested...")
	http.ServeFile(w, r, "agent.exe")
}

// handleAlerts returns alerts as JSON
func handleAlerts(w http.ResponseWriter, r *http.Request) {
	includeResolved := r.URL.Query().Get("resolved") == "true"
	alerts, err := store.GetAlerts(50, includeResolved)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// handleAlertsTable returns HTMX partial for alerts
func handleAlertsTable(w http.ResponseWriter, r *http.Request) {
	renderAlertsTable(w)
}

// handleResolveAlert marks an alert as resolved
func handleResolveAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := store.ResolveAlert(req.ID); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleHistory returns asset history as JSON
func handleHistory(w http.ResponseWriter, r *http.Request) {
	macAddress := r.URL.Query().Get("mac")
	if macAddress == "" {
		http.Error(w, "Missing mac parameter", http.StatusBadRequest)
		return
	}

	history, err := store.GetHistory(macAddress)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleExportCSV exports all assets as CSV
func handleExportCSV(w http.ResponseWriter, r *http.Request) {
	csv, err := store.ExportAssetsCSV()
	if err != nil {
		http.Error(w, "Export error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=itam_assets.csv")
	w.Write([]byte(csv))
}

// handleBackup creates a manual database backup
func handleBackup(w http.ResponseWriter, r *http.Request) {
	backupPath, err := BackupDatabase("itam.db")
	if err != nil {
		http.Error(w, "Backup failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Backup created successfully",
		"path":    backupPath,
	})
}

// handleHealth returns server health status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	total, online, _, _, _ := store.Stats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "healthy",
		"total_assets": total,
		"online":       online,
		"uptime":       time.Since(serverStart).String(),
	})
}

// handleAdminCommand sets a command for a specific agent (kill switch)
func handleAdminCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Admin endpoints require auth token
	if !checkAuthToken(w, r) {
		return
	}

	mac := r.URL.Query().Get("mac")
	command := r.URL.Query().Get("cmd")

	if mac == "" || command == "" {
		http.Error(w, "Missing mac or cmd parameter", http.StatusBadRequest)
		return
	}

	setAgentCommand(mac, command)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("Command '%s' queued for agent %s", command, mac),
	})
}

// escapeHTML escapes a string for safe HTML rendering (XSS prevention)
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

var serverStart = time.Now()
