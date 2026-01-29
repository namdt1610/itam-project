package main

import (
"encoding/json"
"fmt"
"log"
"runtime"

    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/shirou/gopsutil/v3/host"
    "github.com/shirou/gopsutil/v3/mem"

)

// Cấu trúc gói tin để gửi về Server
type AgentReport struct {
OS string `json:"os"`
Hostname string `json:"hostname"`
CPU string `json:"cpu_model"`
Cores int `json:"cores"`
RAM uint64 `json:"ram_gb"`
Disk uint64 `json:"disk_gb"`
}

func main() {
// 1. Lấy thông tin HOST (Tên máy, OS)
hostInfo, \_ := host.Info()

    // 2. Lấy thông tin CPU
    cpuInfo, _ := cpu.Info()
    cpuModel := "Unknown"
    if len(cpuInfo) > 0 {
    	cpuModel = cpuInfo[0].ModelName
    }

    // 3. Lấy thông tin RAM
    vmStat, _ := mem.VirtualMemory()

    // 4. Lấy thông tin Ổ CỨNG (Lấy tổng dung lượng ổ C:)
    // Lưu ý: Trên Windows thường là "C:", trên Linux là "/"
    diskPath := "/"
    if runtime.GOOS == "windows" {
    	diskPath = "C:"
    }
    diskStat, _ := disk.Usage(diskPath)

    // 5. Đóng gói dữ liệu
    report := AgentReport{
    	OS:       hostInfo.Platform + " " + hostInfo.PlatformVersion, // Ví dụ: Windows 10
    	Hostname: hostInfo.Hostname,
    	CPU:      cpuModel,
    	Cores:    runtime.NumCPU(),
    	RAM:      vmStat.Total / 1024 / 1024 / 1024, // Chia để ra GB
    	Disk:     diskStat.Total / 1024 / 1024 / 1024, // Chia để ra GB
    }

    // 6. In ra màn hình (Sau này thay dòng này bằng lệnh gửi HTTP Post)
    jsonData, _ := json.MarshalIndent(report, "", "  ")
    fmt.Println(string(jsonData))

}

// Đoạn thêm vào để bắn dữ liệu
func sendData(data AgentReport) {
jsonData, \_ := json.Marshal(data)
// IP này là IP con VM mày đang xin. Tạm thời để localhost test
url := "http://192.168.100.50:8080/api/report"

    resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        // Agent phải câm mồm nếu lỗi, không được hiện popup báo lỗi cho user
        return
    }
    defer resp.Body.Close()

}

# Lệnh thần thánh biến code Go thành file chạy Windows

# basic

GOOS=windows GOARCH=amd64 go build -o agent.exe main.go

# optimized

GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H=windowsgui" -o agent.exe ./agent/main.go

# check file's size

ls -lh

# make sure it's windows

## file agent.exe

[namdt@archlinux itam-project]$ ls -ls
total 6056
0 drwxr-xr-x 1 namdt namdt 34 Jan 26 11:21 agent
6048 -rwxr-xr-x 1 namdt namdt 6192640 Jan 26 14:51 agent.exe
4 -rw-r--r-- 1 namdt namdt 526 Jan 26 14:02 go.mod
4 -rw-r--r-- 1 namdt namdt 3242 Jan 26 14:02 go.sum
0 drwxr-xr-x 1 namdt namdt 34 Jan 26 14:36 server
[namdt@archlinux itam-project]$ file agent.exe
agent.exe: PE32+ executable for MS Windows 6.01 (console), x86-64, 8 sections

---

# Build server cho môi trường Linux

go build -ldflags "-s -w" -o server_bin ./server/main.go
