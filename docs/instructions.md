### Bước 1: Build file Agent "thông minh" (Tại máy bạn)
Bạn chạy lệnh build sau để nhúng sẵn IP của máy chủ bệnh viện và Mật khẩu vào trong Agent:

```bash
# Thay IP_SERVER và YOUR_TOKEN bằng thông số thật của bạn
go build -ldflags "-s -w" -o server_bin ./server

GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o server.exe ./server


#linux
go build -ldflags "-X main.ServerURL=http://192.168.80.135:8080/api/report -X main.AuthToken=YOUR_TOKEN -H=windowsgui" -o agent.exe ./agent

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.BuildServerURL=http://172.16.202.155:8080/api/report -X main.BuildAuthToken=AnBinh@2026 -H=windowsgui" -o agent.exe ./agent
```
*Kết quả:* Bạn có file `agent.exe` duy nhất, mang đi đâu cũng tự biết gửi dữ liệu về đúng chỗ.

### Bước 2: Chạy Server (Tại máy chủ Bệnh viện)
Bạn đưa 2 file cho Admin để chạy Dashboard:
1.  **File `server_bin`** (file thực thi của server).
2.  **File `config.json`** (để cấu hình port 8080 và token khớp với Agent).
*Kết quả:* Dashboard hoạt động tại `http://IP_SERVER:8080`.

### Bước 3: Phân phát Agent (Admin Bệnh viện làm)
Bạn chỉ cần đưa **duy nhất file `agent.exe`** đã build ở Bước 1 cho Admin và nhờ:
*   *"Anh dùng GPO tạo một **Scheduled Task** để chạy file này bằng quyền **SYSTEM** mỗi khi máy tính khởi động (Startup)."*

---

### Tổng kết những gì bạn có:
*   **Plug (Cắm):** Chạy `server_bin` trên máy chủ.
*   **Play (Chạy):** Quăng 1 file `agent.exe` cho Admin đẩy qua GPO.
*   **Xong:** Bạn mở trình duyệt lên và xem 400 máy hiện lên đầy đủ.
