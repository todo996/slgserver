# Triển khai backend Tam Quốc

Backend dùng một repository GitHub nhưng chạy thành năm Railway Service độc lập. Sau khi thiết lập lần đầu, một lần push lên nhánh kết nối sẽ tự kích hoạt triển khai cho cả năm service.

## 1. Tạo Supabase PostgreSQL

1. Tạo một Supabase Project mới.
2. Chạy lần lượt mọi tệp trong `supabase/migrations` theo thứ tự tên tệp.
3. Mở **Connect** và sao chép chuỗi **Session pooler** cổng `5432`.
4. Không đưa `DATABASE_URL` thật vào mã nguồn hoặc tệp đã commit.

Các migration hiện thực hiện:

- tạo toàn bộ bảng game tương đương schema MySQL cũ;
- chuyển dữ liệu cấu trúc sang `JSONB`;
- sửa cột mật khẩu sang `VARCHAR` để PostgreSQL không đệm khoảng trắng;
- bật Row Level Security;
- thu hồi quyền trực tiếp của `anon` và `authenticated`.

Client Cocos không kết nối Supabase trực tiếp. Mọi truy cập dữ liệu phải đi qua backend Railway.

## 2. Tạo một Railway Project với năm Service

Tạo năm service từ cùng repository `slgserver`, cùng nhánh triển khai và cùng Dockerfile `Dockerfile.railway`.

| Railway Service | `SERVICE_NAME` | `PORT` | Truy cập public |
| --- | --- | ---: | --- |
| `gate-service` | `gate` | `8004` | Có, WebSocket/WSS |
| `http-service` | `http` | `8088` | Có, HTTPS |
| `login-service` | `login` | `8003` | Không |
| `chat-service` | `chat` | `8002` | Không |
| `slg-service` | `slg` | `8001` | Không |

Mỗi service dùng cấu hình build từ `railway.json`. Không cần năm repository và không cần năm Dockerfile.

## 3. Biến môi trường dùng chung

Đặt cho các service cần truy cập cơ sở dữ liệu:

```dotenv
DATABASE_URL=<SESSION_POOLER_SUPABASE_PORT_5432>
DB_MAX_IDLE_CONNS=2
DB_MAX_OPEN_CONNS=10
XORM_SHOW_SQL=false
XORM_LOG_LEVEL=1
TZ=Asia/Bangkok
```

Đặt domain public của Gateway cho `login-service` và có thể đặt thành Shared Variable trong Railway Project:

```dotenv
GATE_PUBLIC_URL=wss://domain-gate-railway
```

## 4. Biến riêng của từng service

### `http-service`

```dotenv
SERVICE_NAME=http
PORT=8088
CORS_ALLOWED_ORIGINS=https://ten-game.vercel.app
```

Có thể nhập nhiều origin, phân cách bằng dấu phẩy. API đăng ký và đổi mật khẩu chỉ nhận `POST`.

### `gate-service`

```dotenv
SERVICE_NAME=gate
PORT=8004
GATE_NEED_SECRET=true
SLG_PROXY_URL=ws://slg-service.railway.internal:8001
CHAT_PROXY_URL=ws://chat-service.railway.internal:8002
LOGIN_PROXY_URL=ws://login-service.railway.internal:8003
```

### `login-service`

```dotenv
SERVICE_NAME=login
PORT=8003
LOGIN_NEED_SECRET=false
GATE_PUBLIC_URL=wss://domain-gate-railway
```

### `chat-service`

```dotenv
SERVICE_NAME=chat
PORT=8002
CHAT_NEED_SECRET=false
```

### `slg-service`

```dotenv
SERVICE_NAME=slg
PORT=8001
SLG_NEED_SECRET=false
SLG_IS_DEV=false
```

## 5. Public domain và health check

- Tạo public domain cho `gate-service`; client dùng địa chỉ `wss://...`.
- Tạo public domain cho `http-service`; client dùng địa chỉ `https://...`.
- Health check của `http-service`: `/healthz`.
- Không tạo public domain cho `login`, `chat` và `slg`.
- Cố định các cổng nội bộ như bảng trên để địa chỉ `*.railway.internal` luôn đúng.

## 6. Bảo mật tài khoản

- Đăng ký tài khoản chỉ nhận HTTP `POST` qua HTTPS.
- Mật khẩu không được ghi vào query string hoặc log.
- Mật khẩu mới được lưu bằng bcrypt.
- Tài khoản MD5 cũ vẫn đăng nhập được một lần và tự nâng cấp sang bcrypt.
- Client chỉ lưu tên tài khoản, không lưu mật khẩu trong `localStorage`.
- Session và mật khẩu không được trả lại hoặc ghi vào log server.

## 7. Cơ chế một lần push

Cả năm Railway Service đều kết nối tới cùng repository và nhánh. Khi có commit mới:

1. GitHub nhận một lần push.
2. Railway phát hiện commit mới ở cả năm service.
3. Mỗi service build cùng `Dockerfile.railway`.
4. `SERVICE_NAME` quyết định binary cần chạy.
5. Năm service được cập nhật độc lập.

Nghĩa là người quản trị chỉ push một lần, còn Railway tạo năm deployment tự động.

## 8. Kiểm tra trước khi phát hành

GitHub Actions trong `.github/workflows/server-ci.yml` thực hiện:

- chạy unit test, bao gồm bcrypt và nâng cấp mật khẩu cũ;
- build đủ năm binary;
- build Docker image Railway;
- tạo PostgreSQL sạch;
- chạy toàn bộ migration Supabase;
- xác nhận schema và `JSONB`;
- mở HTTP Server và kiểm tra `/healthz`;
- xác nhận đăng ký bằng GET bị chặn;
- đăng ký bằng POST;
- xác nhận mật khẩu lưu bằng bcrypt;
- xác nhận tài khoản trùng trả đúng mã lỗi.

Chỉ triển khai production khi workflow này hoàn tất thành công.
