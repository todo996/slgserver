# Triển khai backend Tam Quốc

Backend dùng một repository GitHub nhưng chạy thành năm Railway Service độc lập. Sau khi thiết lập lần đầu, một lần push lên nhánh kết nối sẽ tự kích hoạt triển khai cho cả năm service.

## 1. Tạo Supabase PostgreSQL

1. Tạo một Supabase Project mới.
2. Chạy lần lượt các migration trong `supabase/migrations` theo thứ tự tên tệp.
3. Mở bảng **Connect** và sao chép chuỗi **Session pooler** cổng `5432`.
4. Không đưa `DATABASE_URL` vào mã nguồn hoặc tệp đã commit.

Các bảng game đã bật Row Level Security và thu hồi quyền của `anon` cùng `authenticated`. Client Cocos không kết nối Supabase trực tiếp; mọi truy cập dữ liệu phải đi qua backend Railway.

## 2. Tạo một Railway Project với năm Service

Tạo năm service từ cùng repository `slgserver`, cùng nhánh triển khai và cùng Dockerfile `Dockerfile.railway`.

| Railway Service | SERVICE_NAME | PORT | Public |
| --- | --- | ---: | --- |
| `gate-service` | `gate` | `8004` | Có, WebSocket |
| `http-service` | `http` | `8088` | Có, HTTPS |
| `login-service` | `login` | `8003` | Không |
| `chat-service` | `chat` | `8002` | Không |
| `slg-service` | `slg` | `8001` | Không |

Mỗi service dùng cấu hình build từ `railway.json`. Không cần năm repository và không cần năm Dockerfile.

## 3. Biến môi trường dùng chung

Đặt cho các service có truy cập cơ sở dữ liệu:

```dotenv
DATABASE_URL=<SESSION_POOLER_SUPABASE_PORT_5432>
DB_MAX_IDLE_CONNS=2
DB_MAX_OPEN_CONNS=10
XORM_SHOW_SQL=false
XORM_LOG_LEVEL=1
TZ=Asia/Bangkok
```

Đặt riêng cho `http-service`:

```dotenv
SERVICE_NAME=http
PORT=8088
CORS_ALLOWED_ORIGINS=https://ten-game.vercel.app
```

Đặt riêng cho `gate-service`:

```dotenv
SERVICE_NAME=gate
PORT=8004
GATE_NEED_SECRET=true
SLG_PROXY_URL=ws://slg-service.railway.internal:8001
CHAT_PROXY_URL=ws://chat-service.railway.internal:8002
LOGIN_PROXY_URL=ws://login-service.railway.internal:8003
```

Đặt cho ba service nội bộ:

```dotenv
# login-service
SERVICE_NAME=login
PORT=8003

# chat-service
SERVICE_NAME=chat
PORT=8002

# slg-service
SERVICE_NAME=slg
PORT=8001
```

## 4. Public domain và health check

- Tạo public domain cho `gate-service`; client dùng địa chỉ `wss://...`.
- Tạo public domain cho `http-service`; client dùng địa chỉ `https://...`.
- Health check của `http-service`: `/healthz`.
- Không tạo public domain cho `login`, `chat` và `slg`.

## 5. Cơ chế một lần push

Cả năm Railway Service đều kết nối tới cùng repository và nhánh. Khi có commit mới:

1. GitHub nhận một lần push.
2. Railway phát hiện commit mới ở cả năm service.
3. Mỗi service build cùng Dockerfile.
4. `SERVICE_NAME` quyết định binary cần chạy.
5. Năm service được thay phiên bản độc lập.

## 6. Kiểm tra trước khi phát hành

GitHub Actions trong `.github/workflows/server-ci.yml` thực hiện:

- tải và đồng bộ Go module;
- chạy test;
- build đủ năm binary;
- build Docker image dùng trên Railway.

Chỉ triển khai production khi workflow này hoàn tất thành công.
