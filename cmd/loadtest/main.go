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
	AuthToken  = "itam-secret-token-default"
	AgentCount = 500
)

type AgentReport struct {
	OS           string       `json:"os"`
	Hostname     string       `json:"hostname"`
	Uptime       uint64       `json:"uptime"`
	CPU          string       `json:"cpu_model"`
	CPUUsage     float64      `json:"cpu_usage"`
	Cores        int          `json:"cores"`
	RAM          uint64       `json:"ram_bytes"`
	RAMUsage     uint64       `json:"ram_usage_bytes"`
	RAMPercent   float64      `json:"ram_percent"`
	Disk         uint64       `json:"disk_bytes"`
	DiskUsage    uint64       `json:"disk_usage_bytes"`
	DiskPercent  float64      `json:"disk_percent"`
	IPLAN        string       `json:"ip_lan"`
	MACAddress   string       `json:"mac_address"`
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
	fmt.Printf("Target: %s\n", ServerURL)

	// Semaphore to limit concurrency
	concurrency := 50
	sem := make(chan struct{}, concurrency)

	start := time.Now()
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for i := 0; i < AgentCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if sendReport(id) {
				mu.Lock()
				successCount++
				mu.Unlock()
			} else {
				mu.Lock()
				failCount++
				mu.Unlock()
			}
		}(i)
		time.Sleep(15 * time.Millisecond)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Total: %d requests\n", AgentCount)
	fmt.Printf("Success: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)
	fmt.Printf("Duration: %s\n", elapsed)
	fmt.Printf("Rate: %.1f req/s\n", float64(AgentCount)/elapsed.Seconds())
}

func sendReport(id int) bool {
	mac := fmt.Sprintf("00:11:22:33:%02x:%02x", id/256, id%256)
	ip := fmt.Sprintf("192.168.%d.%d", id/256, id%256)

	report := AgentReport{
		OS:           "Windows 10 Pro",
		Hostname:     fmt.Sprintf("PC-Node-%03d", id),
		Uptime:       uint64(rand.Intn(100000)),
		CPU:          "Intel Core i5-10400 @ 2.90GHz",
		CPUUsage:     rand.Float64() * 100,
		Cores:        6,
		RAM:          16 * 1024 * 1024 * 1024,
		RAMUsage:     uint64(rand.Int63n(16 * 1024 * 1024 * 1024)),
		RAMPercent:   rand.Float64() * 100,
		Disk:         512 * 1024 * 1024 * 1024,
		DiskUsage:    uint64(rand.Int63n(512 * 1024 * 1024 * 1024)),
		DiskPercent:  rand.Float64() * 100,
		IPLAN:        ip,
		MACAddress:   mac,
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
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if id%50 == 0 {
			fmt.Printf("[Fail] Node %d: Status %d\n", id, resp.StatusCode)
		}
		return false
	}

	if id%100 == 0 {
		fmt.Printf("[OK] Node %d sent\n", id)
	}
	return true
}
