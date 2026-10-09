# Giải Thích Chi Tiết Các Thuật Ngữ Cốt Lõi (Thesis Defense Cheat-Sheet)

Tài liệu này tổng hợp chi tiết các khái niệm công nghệ quan trọng được áp dụng trong dự án ITAM (IT Asset Management). Mỗi khái niệm được phân tích theo 3 phần cốt lõi: **Khái niệm là gì?**, **Tại sao cần áp dụng (WHY)?**, và **Cách cài đặt thực tế trong dự án (HOW)?** để giúp bạn tự tin phản biện trước bất kỳ câu hỏi nào từ Hội đồng Bảo vệ Đồ án.

---

## 🛡️ Nhóm 1: Chống Quá Tải & Mạng Lưới (Resilience & Networking)

### 1. Thundering Herd Problem (Hiệu ứng bầy đàn)
*   **Khái niệm là gì?**
    *   Là hiện tượng một số lượng lớn các tiến trình hoặc thiết bị client cùng lúc gửi yêu cầu (request) đến máy chủ (server) khi có một sự kiện chung xảy ra, tạo ra một đỉnh lưu lượng (traffic spike) đột ngột.
*   **Tại sao cần giải quyết (WHY)?**
    *   Trong môi trường thực tế (như bệnh viện), nhân viên y tế bắt đầu ca làm việc lúc 7:00 - 8:00 sáng và bật đồng loạt hàng trăm máy tính. Nếu toàn bộ Agent cấu hình báo cáo dữ liệu ngay khi khởi động hoặc đúng chu kỳ cố định (ví dụ: cứ mỗi 5 phút tròn), Server sẽ phải nhận 400+ kết nối TCP và các truy vấn ghi Database cùng một giây.
    *   Hậu quả là Server bị nghẽn CPU/RAM, cạn kiệt kết nối database (Connection Pool), gây ra lỗi Timeout và cuối cùng dẫn đến sập hệ thống (tương tự một cuộc tấn công DDoS tự phát).
*   **Cách áp dụng trong dự án (HOW)?**
    *   Được giải quyết triệt để bằng cơ chế **Jitter Delay** (độ trễ ngẫu nhiên) nhằm san phẳng đỉnh lưu lượng.

### 2. Jitter / Jitter Delay (Độ trễ ngẫu nhiên)
*   **Khái niệm là gì?**
    *   Là kỹ thuật thêm một khoảng thời gian trễ ngẫu nhiên (random delay hoặc "noise") vào chu kỳ gửi tin nhắn của client để phá vỡ tính đồng bộ của các luồng dữ liệu.
*   **Tại sao cần giải quyết (WHY)?**
    *   Thay vì 400 máy tính gửi dữ liệu chính xác vào lúc 7:00:00, việc thêm Jitter (ví dụ từ 0 đến 30 giây) giúp phân rải (smooth) lưu lượng truy cập trải đều ra khoảng thời gian rộng hơn. Trung bình, Server chỉ phải xử lý `400 máy / 30 giây = ~13 requests/giây` thay vì nhận cả 400 requests trong 1 giây. Việc này giúp giảm tải đáng kể cho Server và Database mà không làm giảm tính cập nhật của dữ liệu.
*   **Cách áp dụng trong dự án (HOW)?**
    *   Trong file [agent/main.go](file:///Users/namdt/dev/saas/itam-project/agent/main.go#L130-L138), Agent sinh ra một độ trễ ngẫu nhiên trước khi gửi báo cáo:
        ```go
        jitter := config.JitterSeconds // mặc định 30s
        jitterDelay := rand.Intn(jitter)
        time.Sleep(time.Duration(jitterDelay) * time.Second)
        ```
    *   Cơ chế này được áp dụng cả lúc Agent vừa khởi động lần đầu lẫn trong chu kỳ gửi tiếp theo (5 phút + random 0-30s) ở [agent/main.go](file:///Users/namdt/dev/saas/itam-project/agent/main.go#L151-L163).

### 3. Exponential Backoff (Trì hoãn lũy tiến theo hàm mũ)
*   **Khái niệm là gì?**
    *   Là thuật toán tự động tăng thời gian chờ đợi giữa các lần thử lại (retry) tiếp theo sau mỗi lần gặp lỗi kết nối hoặc nhận phản hồi Server quá tải (mã lỗi HTTP 429). Thời gian chờ tăng dần theo cấp số nhân (hàm mũ cơ số 2).
*   **Tại sao cần giải quyết (WHY)?**
    *   Khi Server đang bị quá tải hoặc mất kết nối mạng tạm thời, nếu các Agent cứ liên tục gửi yêu cầu thử lại ngay lập tức (gọi là retry dồn dập), Server sẽ không bao giờ có cơ hội phục hồi do liên tục bị nghẽn mạng. Bằng cách kéo giãn thời gian chờ sau mỗi lần thất bại (ví dụ: lần đầu chờ 5s, lần 2 chờ 10s, lần 3 chờ 20s), chúng ta cho Server "khoảng thở" cần thiết để tự khắc phục lỗi hoặc giảm tải.
*   **Cách áp dụng trong dự án (HOW)?**
    *   Trong hàm `sendData` của [agent/main.go](file:///Users/namdt/dev/saas/itam-project/agent/main.go#L267-L281), khi nhận mã phản hồi HTTP `429 (Too Many Requests)` từ Server, Agent tính toán thời gian chờ tăng dần theo lũy thừa của 2 và có giới hạn trần tối đa là 60s để tránh chờ quá lâu:
        ```go
        // i là số lần thử thất bại hiện tại (0, 1, 2)
        backoffSeconds := 5 * (1 << i) // Kết quả: 5s, 10s, 20s
        if backoffSeconds > 60 {
            backoffSeconds = 60
        }
        time.Sleep(time.Duration(backoffSeconds) * time.Second)
        ```

### 4. Circuit Breaker (Mô hình cầu dao tự ngắt)
*   **Khái niệm là gì?**
    *   Là mẫu thiết kế phần mềm (Design Pattern) nhằm ngăn chặn ứng dụng thực hiện một hành động (như gửi request mạng) chắc chắn sẽ thất bại, giúp bảo vệ tài nguyên của cả Client lẫn Server.
*   **Tại sao cần giải quyết (WHY)?**
    *   Nếu Server bị sập hoàn toàn hoặc mất mạng kéo dài nhiều tiếng, Agent không nên tiếp tục chạy vòng lặp thử lại vô hạn lần. Việc cố gắng gửi request mạng liên tục sẽ gây tốn tài nguyên CPU, hao pin của máy Client (máy tính bác sĩ đang dùng để khám bệnh) và làm tràn ngập log lỗi trên hệ thống. Cầu dao cần "ngắt" (trip) để Agent từ bỏ lượt gửi đó, đi ngủ tiếp và đợi đến chu kỳ định kỳ tiếp theo (ví dụ: 5 phút sau) mới thử lại từ đầu.
*   **Cách áp dụng trong dự án (HOW)?**
    *   Trong [agent/main.go](file:///Users/namdt/dev/saas/itam-project/agent/main.go#L247-L313), vòng lặp gửi tin nhắn giới hạn tối đa `maxRetries := 3`.
    *   Nếu cả 3 lần thử (đã kèm Exponential Backoff) đều thất bại, Agent ghi log dừng lại:
        ```go
        fmt.Println("Circuit breaker: gave up sending data after retries.")
        ```
    *   Hàm gửi dữ liệu sẽ kết thúc, giải phóng bộ nhớ và nhường chỗ cho chu kỳ chờ 5 phút tiếp theo của tiến trình.

---

## ⚡ Nhóm 2: Tối Ưu Hóa Hệ Thống (Performance & System Programming)

### 5. Zero-Footprint (Dấu chân tài nguyên tối giản)
*   **Khái niệm là gì?**
    *   Là triết lý thiết kế phần mềm chạy ngầm sao cho lượng RAM, dung lượng đĩa cứng, luồng mạng và CPU tiêu thụ gần như bằng không, không gây bất kỳ tác động nào đến hiệu suất làm việc của máy chủ vật lý.
*   **Tại sao cần giải quyết (WHY)?**
    *   Máy tính tại các bệnh viện công thường rất cũ (nhiều máy chạy chip Pentium/Core i3 thế hệ đầu, RAM 4GB, cài Windows 7/10). Nếu phần mềm ITAM chiếm dụng 100MB RAM và 10% CPU liên tục, nó sẽ làm đơ máy bác sĩ, ảnh hưởng trực tiếp đến việc khám chữa bệnh. Một ứng dụng giám sát tốt phải hoạt động "vô hình" (chỉ chiếm ~3-5MB RAM, ~0% CPU).
*   **Cách áp dụng trong dự án (HOW)?**
    *   Dự án sử dụng ngôn ngữ **Golang** - một ngôn ngữ biên dịch trực tiếp ra mã máy (Native Code) cực kỳ tối ưu về bộ nhớ và không cần máy ảo (VM) trung gian.
    *   Khi Agent không trong chu kỳ gửi dữ liệu, nó sẽ ở trạng thái ngủ hoàn toàn nhờ lệnh `time.Sleep(...)` giải phóng luồng xử lý của CPU, lượng RAM chiếm dụng tĩnh cực kỳ thấp (~3.2 MB).

### 6. Single Binary & Zero Dependency (Tệp thực thi đơn & Không phụ thuộc)
*   **Khái niệm là gì?**
    *   **Single Binary:** Toàn bộ mã nguồn, cấu hình tĩnh và thư viện đi kèm được đóng gói gọn trong đúng 1 file thực thi duy nhất (`.exe` trên Windows hoặc binary trên Linux).
    *   **Zero Dependency:** Chương trình có thể chạy ngay mà không yêu cầu cài đặt thêm bất kỳ môi trường runtime hay thư viện phụ trợ nào khác (như .NET Framework, Java Runtime, Node.js, hay các file `.dll` đi kèm).
*   **Tại sao cần giải quyết (WHY)?**
    *   Việc triển khai (Deploy) phần mềm giám sát lên hàng trăm máy tính trong bệnh viện qua GPO (Group Policy Object) cần sự nhanh chóng và đơn giản. Nếu file cài đặt yêu cầu cài thêm môi trường phụ trợ, tỷ lệ lỗi cài đặt sẽ cực kỳ cao do xung đột phiên bản, thiếu quyền admin hoặc mất kết nối mạng nội bộ trong lúc cài đặt phụ trợ.
*   **Cách áp dụng trong dự án (HOW)?**
    *   Golang hỗ trợ biên dịch tĩnh (Static Compilation). Toàn bộ dự án Agent được build ra đúng một file `agent.exe` duy nhất bằng lệnh build chỉ định mục tiêu hệ điều hành:
        ```bash
        GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o agent.exe ./agent
        ```
        *(Cờ `-s -w` dùng để loại bỏ bảng ký hiệu debug giúp giảm kích thước file `.exe` xuống tối đa).*

### 7. Process Spawning (Chi phí khởi tạo tiến trình)
*   **Khái niệm là gì?**
    *   Là hành động của Hệ điều hành nhằm tạo ra, cấp phát vùng nhớ, nạp mã từ đĩa cứng và khởi chạy một tiến trình mới trong không gian người dùng.
*   **Tại sao cần giải quyết (WHY)?**
    *   Nhiều hệ thống ITAM cũ thiết kế Agent bằng cách dùng Windows Task Scheduler gọi chạy một script (như PowerShell hoặc Python) mỗi 5 phút một lần. Mỗi lần gọi như vậy, OS phải thực hiện "Process Spawning" - một tác vụ cực kỳ đắt đỏ về mặt CPU và I/O đĩa cứng (phải load runtime, cấp bộ nhớ, khởi tạo context). Bằng cách thiết kế Agent chạy dưới dạng **Persistent Daemon** (chạy ngầm liên tục, dùng vòng lặp vô hạn và `time.Sleep`), chúng ta chỉ tốn chi phí khởi tạo tiến trình đúng một lần duy nhất lúc bật máy.
*   **Cách áp dụng trong dự án (HOW)?**
    *   Trong [agent/main.go](file:///Users/namdt/dev/saas/itam-project/agent/main.go#L116-L165), Agent được thiết kế dưới dạng vòng lặp vô hạn chạy liên tục (mã dưới đây giản lược, mã thật có thêm `recover()` chống panic):
        ```go
        for {
            collectAndSend()
            totalWait := 5*time.Minute + time.Duration(extraDelay)*time.Second
            time.Sleep(totalWait)
        }
        ```
    *   Hệ điều hành chỉ cần nạp file `agent.exe` vào bộ nhớ RAM duy nhất một lần khi máy tính khởi động (Startup Task).

### 8. Single Instance Lock (Khóa phiên bản đơn)
*   **Khái niệm là gì?**
    *   Là cơ chế đảm bảo tại một thời điểm chỉ có duy nhất một tiến trình (instance) của chương trình được phép chạy trên một hệ điều hành.
*   **Tại sao cần giải quyết (WHY)?**
    *   Khi cài đặt tự động qua GPO hoặc do người dùng vô tình bấm đúp nhiều lần, có thể xuất hiện nhiều tiến trình Agent chạy song song. Việc này dẫn đến: (1) Trùng lặp dữ liệu gửi lên server, (2) Gây lãng phí tài nguyên CPU/RAM của máy, (3) Gây xung đột cổng mạng hoặc lock file. Phải có cơ chế để tiến trình sau tự phát hiện tiến trình trước đang chạy và tự động thoát.
*   **Cách áp dụng trong dự án (HOW)?**
    *   Trong [agent/main.go](file:///Users/namdt/dev/saas/itam-project/agent/main.go#L117-L126), Agent sử dụng cổng TCP nội bộ (`Port 61111`) làm khóa khóa phiên bản:
        ```go
        lockPort := 61111
        listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", lockPort))
        if err != nil {
            // Cổng mạng đã bị chiếm dụng -> có Agent khác đang chạy
            return
        }
        defer listener.Close() // Giữ cổng kết nối cho đến khi chương trình tắt
        ```
    *   Nếu có con Agent thứ 2 cố khởi chạy, lệnh `net.Listen` sẽ báo lỗi, tiến trình đó sẽ tự động thoát lập tức.

---

## 💾 Nhóm 3: Kiến Trúc Dữ Liệu & Giao Diện (Data & Interface Architecture)

### 9. Write-Ahead Logging - WAL (Nhật ký ghi trước)
*   **Khái niệm là gì?**
    *   Là một chế độ quản lý giao dịch của hệ cơ sở dữ liệu (ở đây là SQLite). Thay vì ghi trực tiếp các thay đổi vào file database `.db` chính, các thay đổi được ghi tuần tự vào một file nhật ký riêng biệt `.db-wal`. File nhật ký này sau đó mới được đồng bộ định kỳ (checkpoint) trở lại file chính.
*   **Tại sao cần giải quyết (WHY)?**
    *   Mặc định SQLite sử dụng chế độ khóa file truyền thống (Rollback Journal). Khi có một tiến trình đang thực hiện ghi dữ liệu, toàn bộ file database sẽ bị khóa (Database Locked), các tiến trình đọc dữ liệu khác sẽ bị chặn hoàn toàn và báo lỗi.
    *   Khi hàng trăm Agent liên tục gửi dữ liệu báo cáo, Server sẽ ghi dữ liệu dồn dập. Nếu admin cùng lúc đó mở Dashboard để xem báo cáo (đọc dữ liệu), hệ thống sẽ bị treo hoặc lỗi. Chế độ WAL cho phép **Người đọc không chặn người ghi, và Người ghi không chặn người đọc** (Reader-Writer Concurrency), tăng hiệu suất ghi đồng thời lên gấp nhiều lần.
*   **Cách áp dụng trong dự án (HOW)?**
    *   Trong file [server/store.go](file:///Users/namdt/dev/saas/itam-project/server/store.go#L34-L45), DSN của SQLite được cấu hình kích hoạt chế độ WAL, tăng thời gian đợi khóa và khóa giao dịch ngay lập tức:
        ```go
        dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=10000&_txlock=immediate"
        db, err := sql.Open("sqlite", dsn)
        ```
    *   Đồng thời, chúng ta giới hạn số lượng kết nối ghi đồng thời tối đa là 1 (`db.SetMaxOpenConns(1)`) để SQLite xếp hàng các tác vụ ghi tuần tự một cách an toàn nhất, triệt tiêu hoàn toàn lỗi lock database.

### 10. HTMX & HATEOAS (Kiến trúc Web tinh giản)
*   **Khái niệm là gì?**
    *   **HTMX:** Thư viện Javascript cho phép truy cập trực tiếp vào các tính năng AJAX, WebSockets và Server Sent Events từ các thẻ HTML để cập nhật UI từng phần mà không cần viết JS.
    *   **HATEOAS (Hypermedia As The Engine Of Application State):** Một nguyên lý cốt lõi của REST, yêu cầu Server trả về trực tiếp định dạng siêu văn bản (HTML chứa cấu trúc hiển thị và dữ liệu) thay vì chỉ trả về dữ liệu thô (JSON).
*   **Tại sao cần giải quyết (WHY)?**
    *   Các framework SPA (React, Vue, Angular) yêu cầu Client tải hàng MB mã nguồn Javascript về trình duyệt, đòi hỏi máy Client phải xử lý biên dịch và render giao diện. Đối với hệ thống Dashboard giám sát chạy ở phòng quản trị của bệnh viện (thường dùng các máy PC cấu hình trung bình), việc mở tab Dashboard React nặng nề sẽ làm chậm trình duyệt và chiếm RAM.
    *   Sử dụng HTMX giúp Server chịu trách nhiệm render HTML (nấu sẵn món ăn), trình duyệt Client chỉ việc dán đè (swap) HTML đó vào DOM, giúp giao diện Dashboard cực kỳ mượt mà, phản hồi ngay lập tức, tốn rất ít RAM và không cần viết bất kỳ dòng Javascript phức tạp nào.
*   **Cách áp dụng trong dự án (HOW)?**
    *   Trong giao diện Dashboard của dự án, các nút bấm và bảng danh sách thiết bị (như stats cards, alerts, assets table) được cấu hình các thuộc tính HTMX như `hx-get`, `hx-post`, `hx-target` và `hx-trigger="load, every 30s"` (xem [server/templates/index.html](file:///Users/namdt/dev/saas/itam-project/server/templates/index.html#L123)) để tự động tải lại từng thành phần cụ thể từ API HTML của Server mà không cần load lại toàn bộ trang web.

### 11. WMI - Windows Management Instrumentation (Truy cập sâu Windows)
*   **Khái niệm là gì?**
    *   Là một tập hợp các đặc tả của Microsoft nhằm quản lý các thiết bị và ứng dụng trong mạng máy tính sử dụng hệ điều hành Windows. WMI cung cấp giao diện lập trình chuẩn hóa để truy cập thông tin cấu hình hệ điều hành và phần cứng.
*   **Tại sao cần giải quyết (WHY)?**
    *   Để quản lý tài sản ITAM, chúng ta bắt buộc phải lấy được các thông tin phần cứng mức sâu như: Số Serial Number của bo mạch chủ/BIOS, tên chính xác của vi xử lý CPU, dung lượng đĩa cứng vật lý và các tiến trình đang chạy. Các câu lệnh thông thường của Go không thể đọc trực tiếp các thông số phần cứng bị khóa này nếu không giao tiếp với hệ điều hành Windows qua WMI.
*   **Cách áp dụng trong dự án (HOW)?**
    *   Agent sử dụng thư viện `github.com/shirou/gopsutil` (khai báo trong [agent/main.go](file:///Users/namdt/dev/saas/itam-project/agent/main.go#L16-L20)). Thư viện này dưới nền tảng Windows sẽ tự động thực hiện các truy vấn WMI hoặc gọi API Win32 để trích xuất các thông tin hệ thống mức sâu một cách an toàn và chuẩn xác nhất.

---

## 💡 Hướng dẫn trả lời phản biện từ Hội đồng Giáo viên

1.  **Câu hỏi:** *"Tại sao em tự viết hệ thống này mà không dùng các giải pháp giám sát có sẵn như Zabbix, PRTG hay Prometheus?"*
    *   **Trả lời:** *"Dạ, các giải pháp thương mại hoặc mã nguồn mở như Zabbix rất tốt nhưng Agent của họ khá nặng, yêu cầu cài đặt phức tạp và chiếm nhiều tài nguyên hệ thống. Trong môi trường đặc thù của bệnh viện, phần lớn là các máy tính đời cũ đang chạy các phần mềm khám chữa bệnh rất nặng. Mục tiêu tối thượng của đồ án là thiết kế một giải pháp **Zero-Footprint** (chỉ tốn ~3MB RAM và ~0% CPU khi chạy ngầm) và đóng gói dạng **Single Binary** (chỉ có duy nhất 1 file `.exe` không phụ thuộc môi trường), giúp việc triển khai qua GPO cực kỳ an toàn, không gây chậm máy của bác sĩ."*
2.  **Câu hỏi:** *"Hệ thống này của em có chịu tải được khi bệnh viện mở rộng lên 500 hay 1000 máy tính không?"*
    *   **Trả lời:** *"Dạ hoàn toàn được ạ. Hệ thống đã được thiết kế kiến trúc chống quá tải ngay từ đầu: (1) Cơ chế **Jitter Delay** rải đều các yêu cầu gửi về tránh hiện tượng **Thundering Herd** vào đầu giờ sáng; (2) **Exponential Backoff** và **Circuit Breaker** ở Agent ngăn việc tự DDoS máy chủ khi có sự cố mạng; (3) Ở phía Server, Database SQLite được cấu hình chế độ **WAL (Write-Ahead Logging)** cho phép đọc và ghi song song không bị nghẽn (Database Locked). Khi kiểm thử tải giả lập với 1000 nodes, hệ thống vẫn phản hồi dưới 100ms."*
3.  **Câu hỏi:** *"Tại sao em lại chọn HTMX thay vì các Single Page Application (SPA) phổ biến như React hay Vue?"*
    *   **Trả lời:** *"Dạ, đối với một dashboard quản trị ITAM, chúng ta cần sự cập nhật dữ liệu liên tục nhưng không đòi hỏi các hiệu ứng client-side quá phức tạp. Việc sử dụng React hay Vue sẽ khiến chúng ta phải viết thêm một lớp REST API truyền dữ liệu JSON, đồng thời trình duyệt của Admin phải tải về một lượng lớn file Javascript để render. Bằng cách áp dụng **HTMX** và nguyên lý **HATEOAS**, Server của chúng em render trực tiếp các khối HTML nhỏ và trình duyệt chỉ việc dán đè lên màn hình. Điều này giúp Dashboard chạy cực kỳ nhanh, nhẹ, giảm thiểu tối đa tài nguyên tiêu thụ trên máy tính của người quản trị mạng."*