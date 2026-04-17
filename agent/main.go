package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type Config struct {
	ServerURL       string `json:"server_url"`
	AuthToken       string `json:"auth_token"`
	TopProcessLimit int    `json:"top_process_limit"`
	JitterSeconds   int    `json:"jitter_seconds"`
	LogLevel        string `json:"log_level"`
}

type TopProcess struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	CPU     float64 `json:"cpu_percent"`
	Memory  uint64  `json:"memory_bytes"`
	Command string  `json:"command"`
}

type AgentReport struct {
	OS           string       `json:"os"`
	Hostname     string       `json:"hostname"`
	Uptime       uint64       `json:"uptime"`
	CPU          string       `json:"cpu_model"`
	CPU_Usage    float64      `json:"cpu_usage"`
	Cores        int          `json:"cores"`
	RAM          uint64       `json:"ram_bytes"`
	RAM_Usage    uint64       `json:"ram_usage_bytes"`
	RAM_Percent  float64      `json:"ram_percent"`
	Disk         uint64       `json:"disk_bytes"`
	DiskUsage    uint64       `json:"disk_usage_bytes"`
	DiskPercent  float64      `json:"disk_percent"`
	IP_LAN       string       `json:"ip_lan"`
	MAC_Address  string       `json:"mac_address"`
	TopProcesses []TopProcess `json:"top_processes"`
	SerialNumber string       `json:"serial_number"`
	BiosVersion  string       `json:"bios_version"`
}

// Build-time variables - override with: go build -ldflags "-X main.BuildServerURL=... -X main.BuildAuthToken=..."
var (
	BuildServerURL = ""  // Injected at build time
	BuildAuthToken = ""  // Injected at build time
)

var config Config

func loadConfig() {
	// Default configuration
	config = Config{
		ServerURL:       "http://localhost:8080/api/report",
		AuthToken:       "",
		TopProcessLimit: 5,
		JitterSeconds:   30,
		LogLevel:        "info",
	}

	// Priority 1: Build-time values (highest priority for single-exe deployment)
	if BuildServerURL != "" {
		config.ServerURL = BuildServerURL
	}
	if BuildAuthToken != "" {
		config.AuthToken = BuildAuthToken
	}

	// Priority 2: Environment variable for auth token
	if envToken := os.Getenv("ITAM_AUTH_TOKEN"); envToken != "" {
		config.AuthToken = envToken
	}

	// Priority 3: Config file next to executable (can override build-time values)
	exePath, err := os.Executable()
	if err == nil {
		configPath := filepath.Join(filepath.Dir(exePath), "config.json")
		if file, err := os.ReadFile(configPath); err == nil {
			if err := json.Unmarshal(file, &config); err == nil {
				fmt.Printf("Config overridden from: %s\n", configPath)
				return
			}
		}
	}

	// Priority 4: Config file in current directory
	if file, err := os.ReadFile("config.json"); err == nil {
		if err := json.Unmarshal(file, &config); err == nil {
			fmt.Println("Config overridden from ./config.json")
			return
		}
	}

	if config.AuthToken == "" {
		fmt.Println("WARNING: No auth token configured!")
	}
	fmt.Printf("Using config: Server=%s\n", config.ServerURL)
}

func main() {
	// Single Instance Check: Bind to a specific local port
	// If binding fails, another instance is already running
	lockPort := 61111
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", lockPort))
	if err != nil {
		fmt.Printf("Agent is already running (port %d busy). Exiting.\n", lockPort)
		return
	}
	defer listener.Close() // Keep port bound until program exits

	loadConfig()

	// Jitter: configurable random delay to prevent thundering herd
	jitter := config.JitterSeconds
	if jitter <= 0 {
		jitter = 30 // default 30s if not configured
	}
	jitterDelay := rand.Intn(jitter)
	if config.LogLevel != "error" {
		fmt.Printf("Jitter delay: %d seconds (max: %d)...\n", jitterDelay, jitter)
	}
	time.Sleep(time.Duration(jitterDelay) * time.Second)

	collectAndSend()
	fmt.Println("Agent completed. Exiting.")
}

func collectAndSend() {
	hostInfo, err := host.Info()
	if err != nil {
		fmt.Printf("Error getting host info: %v\n", err)
		return
	}

	if config.LogLevel == "debug" {
		fmt.Printf("Hostname: %s\n", hostInfo.Hostname)
		fmt.Printf("OS: %s %s\n", hostInfo.Platform, hostInfo.PlatformVersion)
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		fmt.Printf("Error getting cpu info: %v\n", err)
	}
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		fmt.Printf("Error getting cpu percent: %v\n", err)
	}

	cpuModel := "Unknown"
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		fmt.Printf("Error getting memory info: %v\n", err)
	}

	diskPath := "/"
	if runtime.GOOS == "windows" {
		diskPath = "C:"
	}
	diskStat, err := disk.Usage(diskPath)
	if err != nil {
		fmt.Printf("Error getting disk info: %v\n", err)
	}

	ipLAN := getLocalIP()
	macAddress := getMACAddress()

	topProcs := getTopProcesses(config.TopProcessLimit)

	serialNumber := getSerialNumber()
	biosVersion := getBiosVersion()

	report := AgentReport{
		OS:           hostInfo.Platform + " " + hostInfo.PlatformVersion,
		Hostname:     hostInfo.Hostname,
		Uptime:       hostInfo.Uptime,
		CPU:          cpuModel,
		CPU_Usage:    percent[0],
		Cores:        runtime.NumCPU(),
		RAM:          vmStat.Total,
		RAM_Usage:    vmStat.Used,
		RAM_Percent:  vmStat.UsedPercent,
		Disk:         diskStat.Total,
		DiskUsage:    diskStat.Used,
		DiskPercent:  diskStat.UsedPercent,
		IP_LAN:       ipLAN,
		MAC_Address:  macAddress,
		TopProcesses: topProcs,
		SerialNumber: serialNumber,
		BiosVersion:  biosVersion,
	}

	if config.LogLevel == "debug" {
		jsonData, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println("Data collected:", string(jsonData))
	}

	sendData(report)
}

func sendData(data AgentReport) {
	jsonData, _ := json.Marshal(data)
	client := &http.Client{Timeout: 30 * time.Second}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest("POST", config.ServerURL, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Failed to create request: %v\n", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		if config.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+config.AuthToken)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Failed to send (Attempt %d/%d): %v\n", i+1, maxRetries, err)
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		// Check for circuit breaker response (429 Too Many Requests)
		if resp.StatusCode == 429 {
			retryAfter := resp.Header.Get("Retry-After")
			fmt.Printf("Server overloaded (429), backing off. Retry-After: %s\n", retryAfter)
			resp.Body.Close()

			// Exponential backoff: 5s, 10s, 20s
			backoffSeconds := 5 * (1 << i)
			if backoffSeconds > 60 {
				backoffSeconds = 60
			}
			fmt.Printf("Waiting %d seconds before retry...\n", backoffSeconds)
			time.Sleep(time.Duration(backoffSeconds) * time.Second)
			continue
		}

		// Check for unauthorized (401)
		if resp.StatusCode == 401 {
			fmt.Println("ERROR: Unauthorized - invalid auth_token")
			resp.Body.Close()
			return
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Parse server response for commands
			var serverResp struct {
				Status  string `json:"status"`
				Command string `json:"command"`
			}
			json.NewDecoder(resp.Body).Decode(&serverResp)
			resp.Body.Close()

			// Handle server commands (kill switch)
			if serverResp.Command != "" {
				handleServerCommand(serverResp.Command)
			}

			fmt.Printf("[%s] Success! Data sent to server.\n", time.Now().Format("15:04:05"))
			return
		}

		fmt.Printf("Server error (status %d), retrying...\n", resp.StatusCode)
		resp.Body.Close()
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	fmt.Println("Circuit breaker: gave up sending data after retries.")
}

// handleServerCommand processes remote commands from server
func handleServerCommand(cmd string) {
	switch cmd {
	case "sleep_1h":
		fmt.Println("Server command: sleep for 1 hour")
		time.Sleep(1 * time.Hour)
	case "sleep_24h":
		fmt.Println("Server command: sleep for 24 hours")
		time.Sleep(24 * time.Hour)
	case "exit":
		fmt.Println("Server command: exit immediately")
		os.Exit(0)
	default:
		if config.LogLevel == "debug" {
			fmt.Printf("Unknown server command: %s\n", cmd)
		}
	}
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "Unknown"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "Unknown"
}

func getMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "Unknown"
	}
	for _, interf := range interfaces {
		if len(interf.HardwareAddr) > 0 {
			if (interf.Flags&net.FlagLoopback) == 0 && (interf.Flags&net.FlagUp) != 0 {
				return interf.HardwareAddr.String()
			}
		}
	}
	return "Unknown"
}

func getTopProcesses(limit int) []TopProcess {
	var processes []TopProcess

	pids, err := process.Pids()
	if err != nil {
		return processes
	}

	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue
		}

		name, _ := proc.Name()
		cpuPercent, _ := proc.CPUPercent()
		memInfo, err := proc.MemoryInfo()
		if err != nil {
			continue
		}
		cmdline, _ := proc.Cmdline()

		processes = append(processes, TopProcess{
			PID:     pid,
			Name:    name,
			CPU:     cpuPercent,
			Memory:  memInfo.RSS,
			Command: cmdline,
		})
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Memory > processes[j].Memory
	})

	if len(processes) > limit {
		return processes[:limit]
	}
	return processes
}

func getSerialNumber() string {
	hostInfo, err := host.Info()
	if err != nil {
		return "Unknown"
	}
	if hostInfo.HostID != "" {
		return hostInfo.HostID
	}
	return "Unknown"
}

func getBiosVersion() string {
	hostInfo, err := host.Info()
	if err != nil {
		return "Unknown"
	}
	return hostInfo.KernelVersion
}
