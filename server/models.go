package main

import "time"

// TopProcess represents a running process
type TopProcess struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	CPU     float64 `json:"cpu_percent"`
	Memory  uint64  `json:"memory_bytes"`
	Command string  `json:"command"`
}

// AgentReport represents data sent by the agent
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
	BiosVersion  string       `json:"bios_version"`
}

// Asset represents a stored asset with metadata
type Asset struct {
	Report   AgentReport `json:"report"`
	LastSeen time.Time   `json:"last_seen"`
}
