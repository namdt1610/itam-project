package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const (
	ServerURL  = "http://localhost:8080/api/report"
	AuthToken  = "8ac32ce17f8f6ffb8714b6916ed1a6bb5afc531d70c5b3ab7f6a7e5341ffb606"
	AgentCount = 500
)

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
}

type TopProcess struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	CPU     float64 `json:"cpu_percent"`
	Memory  uint64  `json:"memory_bytes"`
	Command string  `json:"command"`
}

func main() {
	var wg sync.WaitGroup
	fmt.Printf("Starting load test with %d simulated agents...\n", AgentCount)

	// Semaphore to limit concurrency (mimic real world spread, or just don't kill CLI)
	// Sending too fast might trigger server rate limits (429), which is expected
	concurrency := 50
	sem := make(chan struct{}, concurrency)

	for i := 0; i < AgentCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			sendReport(id)
		}(i)
		// Throttle: 500 requests / 10ms = 5s to finish, well under 100/s limit
		time.Sleep(15 * time.Millisecond)
	}

	wg.Wait()
	fmt.Println("Load test completed.")
}

func sendReport(id int) {
	mac := fmt.Sprintf("00:11:22:33:%02x:%02x", id/256, id%256)
	ip := fmt.Sprintf("192.168.%d.%d", id/256, id%256)

	report := AgentReport{
		OS:           "Windows 10 Pro",
		Hostname:     fmt.Sprintf("PC-Node-%03d", id),
		Uptime:       uint64(rand.Intn(100000)),
		CPU:          "Intel Core i5-10400 @ 2.90GHz",
		CPU_Usage:    rand.Float64() * 100,
		Cores:        6,
		RAM:          16 * 1024 * 1024 * 1024,
		RAM_Usage:    uint64(rand.Int63n(16 * 1024 * 1024 * 1024)),
		RAM_Percent:  rand.Float64() * 100,
		Disk:         512 * 1024 * 1024 * 1024,
		DiskUsage:    uint64(rand.Int63n(512 * 1024 * 1024 * 1024)),
		DiskPercent:  rand.Float64() * 100,
		IP_LAN:       ip,
		MAC_Address:  mac,
		SerialNumber: fmt.Sprintf("SN-%06d", id),
		TopProcesses: []TopProcess{
			{PID: 1234, Name: "chrome.exe", CPU: 15.5, Memory: 500 * 1024 * 1024},
			{PID: 5678, Name: "teams.exe", CPU: 2.1, Memory: 800 * 1024 * 1024},
		},
	}

	data, _ := json.Marshal(report)
	req, _ := http.NewRequest("POST", ServerURL, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+AuthToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("[Fail] Node %d: %v\n", id, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("[Fail] Node %d: Status %d\n", id, resp.StatusCode)
	} else {
		// Only print every 50th success to avoid spamming console
		if id%50 == 0 {
			fmt.Printf("[OK] Node %d reports sent.\n", id)
		}
	}
}
