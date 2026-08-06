# Triển khai backend Tam Quốc bằng một Railway Service

Backend production chạy trong **một container duy nhất**. Container tự khởi động và giám sát năm thành phần nội bộ: HTTP, Gate, Login, Chat và SLG.

## 1. Tạo Supabase PostgreSQL

1. Tạo một Supabase Project mới dành riêng cho game.
2. Chạy lần lượt mọi tệp trong `supabase/migrations` theo đúng thứ tự tên.
3. Mở **Connect** và sao chép chuỗi **Session Pooler** cổng `5432`.
4. Không đưa `DATABASE_URL` thật vào mã nguồn.

Thứ tự migration:

```text
202608060001_initial_schema.sql
202608060002_lock_down_data_api.sql
202608060003_json_columns.sql
202608060004_password_columns.sql
```

Client không kết nối trực tiếp Supabase. Mọi dữ liệu đi qua backend Railway.

## 2. Tạo đúng một Railway Service

1. Tạo một Railway Project.
2. Chọn **New Service → GitHub Repo**.
3. Chọn repository `slgserver`.
4. Chọn nhánh `main`.
5. Không tạo thêm bốn service khác.
6. Không đặt Start Command thủ công.
7. Railway tự dùng `Dockerfile.railway` và `railway.json`.

Cấu trúc chạy bên trong container:

```text
Cổng public Railway $PORT
├── HTTP thường → httpserver :8088
└── WebSocket   → gateserver :8004
                    ├── loginserver :8003
                    ├── chatserver  :8002
                    └── slgserver   :8001
```

Các cổng `8001–8004` và `8088` chỉ mở trên localhost trong container, không cần public domain riêng.

## 3. Biến môi trường Railway

Thêm vào **Variables** của service duy nhất:

```dotenv
DATABASE_URL=<SESSION_POOLER_SUPABASE_PORT_5432>
DB_MAX_IDLE_CONNS=2
DB_MAX_OPEN_CONNS=10
XORM_SHOW_SQL=false
XORM_LOG_LEVEL=1
TZ=Asia/Bangkok

CORS_ALLOWED_ORIGINS=https://ten-game.vercel.app
WS_ALLOWED_ORIGINS=https://ten-game.vercel.app

GATE_NEED_SECRET=true
LOGIN_NEED_SECRET=false
CHAT_NEED_SECRET=false
SLG_NEED_SECRET=false
SLG_IS_DEV=false
GAME_SERVER_ID=1
STARTUP_TIMEOUT=180s
```

Không cần các biến sau của kiến trúc cũ:

```text
SERVICE_NAME
HTTP_PORT
GATE_PORT
LOGIN_PORT
CHAT_PORT
SLG_PORT
SLG_PROXY_URL
CHAT_PROXY_URL
LOGIN_PROXY_URL
```

Railway tự cấp biến `PORT`. Không đặt `PORT` thủ công trừ khi đang chạy local.

### `GATE_PUBLIC_URL`

Bộ khởi chạy tự tạo `GATE_PUBLIC_URL` từ `RAILWAY_PUBLIC_DOMAIN` nếu Railway cung cấp biến đó. Có thể đặt thủ công nếu cần:

```dotenv
GATE_PUBLIC_URL=wss://domain-backend.up.railway.app
```

## 4. Tạo một public domain

Trong service duy nhất:

1. Mở **Settings → Networking**.
2. Chọn **Generate Domain**.
3. Ghi lại domain HTTPS.

Ví dụ:

```text
https://tam-quoc-server-production.up.railway.app
```

Client dùng cùng domain với hai giao thức:

```dotenv
GAME_HTTP_URL=https://tam-quoc-server-production.up.railway.app
GAME_WS_URL=wss://tam-quoc-server-production.up.railway.app
```

Không thêm `/api` hoặc `/ws`. Reverse proxy tự nhận biết WebSocket Upgrade.

## 5. Healthcheck

`railway.json` đã cấu hình:

```text
/healthz
```

Phản hồi thành công có dạng:

```json
{
  "status": "ok",
  "service": "allserver",
  "children": {
    "http": "ok",
    "gate": "ok",
    "login": "ok",
    "chat": "ok",
    "slg": "ok"
  }
}
```

`/readyz` trả cùng trạng thái tổng hợp.

Nếu một tiến trình con chết ngoài dự kiến, bộ giám sát làm toàn container thoát với mã lỗi để Railway restart service. Điều này tránh trường hợp container còn xanh nhưng một phần game đã chết.

## 6. Đồng bộ với Vercel

Sau khi Vercel cấp domain client, ví dụ:

```text
https://tam-quoc-viet-nam.vercel.app
```

Cập nhật hai biến Railway:

```dotenv
CORS_ALLOWED_ORIGINS=https://tam-quoc-viet-nam.vercel.app
WS_ALLOWED_ORIGINS=https://tam-quoc-viet-nam.vercel.app
```

Nếu có nhiều domain, phân cách bằng dấu phẩy:

```dotenv
CORS_ALLOWED_ORIGINS=https://tam-quoc-viet-nam.vercel.app,https://game.example.com
WS_ALLOWED_ORIGINS=https://tam-quoc-viet-nam.vercel.app,https://game.example.com
```

Sau khi sửa Variables, Railway chỉ redeploy **một service**.

## 7. Cơ chế một lần deploy

Sau thiết lập ban đầu:

```text
Push một lần lên GitHub main
→ Railway build một Docker image
→ Railway deploy một container
→ container khởi động đủ năm thành phần
```

Không còn năm deployment độc lập.

## 8. Kiểm tra tự động

GitHub Actions thực hiện đúng mô hình production:

- chạy unit test;
- build `allserver` và năm binary;
- build Docker image;
- tạo PostgreSQL 16 sạch;
- chạy toàn bộ migration;
- chạy một container duy nhất;
- xác nhận `/healthz` có đủ năm thành phần;
- đăng ký tài khoản qua cổng public;
- xác nhận bcrypt trong PostgreSQL;
- kiểm tra WebSocket `101 Switching Protocols` qua cùng cổng public.

Chỉ merge vào `main` khi cả hai job CI đều thành công.

## 9. Lưu ý vận hành

Một service phù hợp cho giai đoạn demo và lượng người chơi ban đầu. Năm thành phần dùng chung CPU/RAM và cùng restart khi một thành phần lỗi. Khi lượng người chơi lớn, có thể tách lại thành nhiều service mà không thay đổi protocol game hoặc schema Supabase.
