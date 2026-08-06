# Tam Quốc Việt Nam — Backend Go

Backend game chiến thuật SLG Tam Quốc viết bằng Go. Bản production được đóng gói thành **một Railway Service duy nhất**, nhưng bên trong vẫn chạy đủ năm thành phần: HTTP, Gate, Login, Chat và SLG.

## Kiến trúc production

```text
Vercel Client/PWA
    │
    ├── HTTPS API ─┐
    └── WSS Game ──┤
                   ▼
        Một Railway Service
        ├── public proxy :$PORT
        ├── httpserver   :8088 nội bộ
        ├── gateserver   :8004 nội bộ
        ├── loginserver  :8003 nội bộ
        ├── chatserver   :8002 nội bộ
        └── slgserver    :8001 nội bộ
                   │
                   ▼
          Supabase PostgreSQL
```

Người vận hành chỉ tạo một Railway Service, một public domain và một deployment. Bộ khởi chạy `allserver` giám sát năm tiến trình con; nếu một thành phần dừng ngoài dự kiến, toàn container thoát để Railway tự khởi động lại.

## Một domain cho cả HTTPS và WSS

Reverse proxy public tự phân luồng:

- request WebSocket Upgrade → `gateserver`;
- request HTTP thông thường → `httpserver`;
- `/healthz` và `/readyz` → kiểm tra tổng hợp đủ năm thành phần.

Vì vậy client có thể dùng cùng một domain:

```dotenv
GAME_HTTP_URL=https://tam-quoc-server.up.railway.app
GAME_WS_URL=wss://tam-quoc-server.up.railway.app
```

## Công nghệ

- Go 1.22 khi build production.
- XORM.
- PostgreSQL/Supabase cho production.
- MySQL chỉ giữ lại để đối chiếu môi trường local cũ.
- WebSocket cho gameplay thời gian thực.
- Docker multi-stage build.
- GitHub Actions chạy unit test và smoke-test một container hoàn chỉnh.

## Bảo mật tài khoản

- Đăng ký và đổi mật khẩu chỉ nhận HTTP `POST`.
- Mật khẩu production truyền qua HTTPS/WSS.
- Mật khẩu mới được lưu bằng bcrypt.
- Tài khoản MD5 cũ được hỗ trợ đăng nhập và tự nâng cấp sang bcrypt.
- Mật khẩu và session không được ghi vào log.
- Client không truy cập Supabase trực tiếp.
- RLS được bật và quyền Data API của `anon`/`authenticated` bị thu hồi trên bảng game.

## Cấu trúc quan trọng

```text
main/allserver.go             Bộ giám sát và reverse proxy public
main/*server.go               Năm chương trình thành phần
server/                       Logic game và tài khoản
net/                          WebSocket, router và kết nối
supabase/migrations/          Migration PostgreSQL
Dockerfile.railway            Build allserver và đủ năm binary
railway.json                  Cấu hình một Railway Service
deploy/railway.env.example    Mẫu biến môi trường
.github/workflows/            CI và smoke-test container hợp nhất
DEPLOYMENT_VI.md              Hướng dẫn triển khai đầy đủ
```

## Chạy kiểm tra

```bash
go mod tidy
go list ./... | grep -v '/main$' | xargs go test
go build -o bin/allserver ./main/allserver.go
```

Build Docker production:

```bash
docker build -f Dockerfile.railway -t tam-quoc-server .
```

## Supabase PostgreSQL

Chạy migration theo đúng thứ tự:

```text
supabase/migrations/202608060001_initial_schema.sql
supabase/migrations/202608060002_lock_down_data_api.sql
supabase/migrations/202608060003_json_columns.sql
supabase/migrations/202608060004_password_columns.sql
```

Backend Railway kết nối bằng `DATABASE_URL` lấy từ **Session Pooler cổng 5432** của Supabase. Nên tạo một Supabase Project riêng cho game.

## Railway

Tạo đúng một Service từ repository này và nhánh `main`. Railway tự build bằng `Dockerfile.railway`; không cần `SERVICE_NAME`, không cần năm service và không cần cấu hình private networking.

Biến tối thiểu:

```dotenv
DATABASE_URL=<SESSION_POOLER_SUPABASE>
CORS_ALLOWED_ORIGINS=https://ten-game.vercel.app
WS_ALLOWED_ORIGINS=https://ten-game.vercel.app
GATE_NEED_SECRET=true
SLG_IS_DEV=false
TZ=Asia/Bangkok
```

Railway tự cấp `PORT`. Healthcheck dùng:

```text
/healthz
```

## Kiểm tra tự động

Workflow backend thực hiện:

- unit test;
- build bộ khởi chạy và đủ năm binary;
- build Docker image;
- tạo PostgreSQL 16 sạch;
- chạy toàn bộ migration Supabase;
- chạy đúng một container Docker;
- xác nhận health tổng và đủ năm thành phần;
- kiểm tra đăng ký POST và bcrypt qua cổng public;
- kiểm tra bắt tay WebSocket qua cùng cổng public.

## Triển khai

Xem [`DEPLOYMENT_VI.md`](./DEPLOYMENT_VI.md) để có danh sách biến môi trường và thứ tự cấu hình Supabase, Railway và Vercel.

## Giấy phép

Mã nguồn backend kế thừa Apache License 2.0 của dự án gốc. Xem [`LICENSE`](./LICENSE).
