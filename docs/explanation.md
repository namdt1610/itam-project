Để chuẩn bị "vũ khí" cho buổi bảo vệ đồ án, tôi đã tổng hợp và giải thích toàn bộ các thuật ngữ "đao to búa lớn" mà chúng ta đã sử dụng trong báo cáo. Bạn chỉ cần hiểu bản chất của những từ này là có thể tự tin trả lời bất kỳ câu hỏi phản biện nào từ hội đồng!

---

### 🛡️ Nhóm 1: Chống quá tải & Mạng lưới (Cực kỳ quan trọng)

**1. Thundering Herd Problem (Hiệu ứng bầy đàn):**
*   **Giải thích:** Tưởng tượng 100 người cùng lúc lao qua 1 cái cửa hẹp. Trong IT, đó là hiện tượng hàng trăm máy tính cùng lúc thức dậy và gửi kết nối về Server (thường xảy ra vào 7h sáng ở bệnh viện). 
*   **Hậu quả:** Web Server bị quá tải, sập hoặc treo.

**2. Jitter / Jitter Delay (Độ trễ ngẫu nhiên):**
*   **Giải thích:** Là cách chúng ta giải quyết Thundering Herd. Thay vì 400 máy báo cáo đúng lúc 7:00:00, ta bắt mỗi máy bốc thăm một con số từ 0-30. Máy bốc số 5 sẽ gửi lúc 7:00:05, máy bốc số 20 gửi lúc 7:00:20.
*   **Tác dụng:** Dàn đều lưu lượng (Traffic Smoothing), giúp Server thở được.

**3. Exponential Backoff (Trì hoãn theo hàm mũ):**
*   **Giải thích:** Khi Server báo lỗi 429 (Quá tải), Agent không cố đấm ăn xôi gọi lại ngay lập tức. Nó sẽ đợi 5s. Nếu vẫn trượt, nó đợi 10s. Trượt tiếp, nó đợi 20s. Thời gian chờ tăng gấp đôi mỗi lần.
*   **Tác dụng:** Tránh việc làm Server đã "chết đuối" lại càng bị dìm xuống thêm bởi các lệnh thử lại (Retry) dồn dập.

**4. Circuit Breaker (Cầu dao tự ngắt / Ngắt mạch):**
*   **Giải thích:** Giống hệt cầu dao điện trong nhà. Khi Agent đã thử lại quá nhiều lần (hết 3 lần Backoff) mà Server vẫn lỗi, nó sẽ "cúp cầu dao" (từ bỏ việc gửi và đi ngủ tiếp), thay vì cứ chạy vòng lặp vô hạn gây treo máy tính của bác sĩ.

---

### ⚡ Nhóm 2: Tối ưu hóa hệ thống (Performance)

**5. Zero-Footprint (Hoạt động vô hình / Không để lại dấu vết):**
*   **Giải thích:** Ám chỉ một phần mềm chạy ngầm cực kỳ nhẹ, không tốn CPU, tốn rất ít RAM (chỉ 3-5MB). Bác sĩ dùng máy không hề có cảm giác là có phần mềm giám sát đang chạy.

**6. Single Binary / Zero Dependency (Chạy 1 file duy nhất / Không phụ thuộc):**
*   **Giải thích:** Nhờ Golang, toàn bộ code được nén thành đúng 1 file `agent.exe` duy nhất. Không cần cài .NET Framework, không cần Java, không cần file DLL đi kèm. Cứ vứt vào máy là chạy. Rất tiện để mang đi cài đặt hàng loạt (Deploy).

**7. Process Spawning (Khởi tạo tiến trình):**
*   **Giải thích:** Việc hệ điều hành phải tạo ra một chương trình mới từ đầu (rất tốn CPU). Trong báo cáo, ta lập luận rằng dùng Vòng lặp chạy ngầm (Persistent Daemon) sẽ tốt hơn là dùng Task Scheduler gọi nó mỗi 5 phút (vì gọi liên tục sẽ gây ra Process Spawning liên tục).

**8. Single Instance Lock (Khóa tiến trình đơn):**
*   **Giải thích:** Thuật toán chiếm dụng một cổng mạng ảo (Port 61111) để đảm bảo không bao giờ có 2 con Agent cùng chạy song song trên 1 máy tính. Nếu cố bật con thứ 2, nó sẽ tự tắt.

---

### 💾 Nhóm 3: Kiến trúc Dữ liệu & Giao diện

**9. Write-Ahead Logging - WAL (Nhật ký ghi trước):**
*   **Giải thích:** Chế độ đặc biệt của SQLite. Khi có nhiều người ghi dữ liệu, nó không ghi thẳng vào file `.db` làm file này bị "khóa" (Database Locked). Nó ghi tạm vào file `.wal` rồi rảnh rỗi mới chép sang. 
*   **Tác dụng:** Giúp người xem Dashboard và các Agent gửi dữ liệu diễn ra đồng thời cùng lúc mà không bị đụng độ nhau.

**10. HTMX & HATEOAS:**
*   **Giải thích:** Thay vì dùng React/Vue nặng nề tải cả đống Javascript về máy người dùng (SPA - Single Page App). HTMX cho phép Server "nấu" sẵn mã HTML rồi gửi về trình duyệt, trình duyệt chỉ việc dán đè (swap) vào màn hình. Rất nhẹ và siêu tốc độ.

**11. WMI (Windows Management Instrumentation):**
*   **Giải thích:** Đây là hệ thống lõi của Windows dùng để quản lý phần cứng. Code Golang của bạn gọi vào WMI để lấy được các thông tin "tuyệt mật" như Tên CPU, RAM, Serial ổ cứng, v.v.

---

**💡 Tip nhỏ khi bảo vệ đồ án:**
Nếu thầy cô hỏi: *"Tại sao em không dùng các phần mềm có sẵn mà phải tự viết?"*
Bạn hãy tự tin trả lời: *"Dạ, vì máy tính bệnh viện đời rất cũ, các phần mềm thương mại thường chiếm đến hàng trăm MB RAM gây chậm máy. Giải pháp của em dùng Golang (Single Binary) đáp ứng tiêu chí **Zero-Footprint**, kết hợp cơ chế **Jitter** nên cực kỳ tối ưu cho hạ tầng mạng y tế ạ!"*._