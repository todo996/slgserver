# Tam Quốc Việt Nam — Backend Go

Backend game chiến thuật SLG Tam Quốc viết bằng Go, được tổ chức thành năm dịch vụ độc lập và chuẩn bị để triển khai trên Railway, sử dụng Supabase PostgreSQL làm cơ sở dữ liệu.

## Kiến trúc dịch vụ

```text
Vercel Client
    ├── HTTPS → http-service
    └── WSS   → gate-service
                    ├── login-service
                    ├── chat-service
                    └── slg-service
                              │
                              ▼
                     Supabase PostgreSQL
```

| Dịch vụ | Vai trò | Cổng mặc định | Public |
| --- | --- | ---: | --- |
| `http-service` | API đăng ký và quản lý tài khoản | `8088` | Có |
| `gate-service` | Gateway WebSocket của client | `8004` | Có |
| `login-service` | Đăng nhập, session và danh sách server | `8003` | Không |
| `chat-service` | Trò chuyện trong game | `8002` | Không |
| `slg-service` | Logic bản đồ, thành trì, tướng, quân đội và liên minh | `8001` | Không |

Cả năm dịch vụ dùng chung một repository và một `Dockerfile.railway`. Biến `SERVICE_NAME` quyết định binary nào được chạy trong từng Railway Service.

## Công nghệ

- Go `1.22` khi build production.
- XORM.
- PostgreSQL/Supabase cho production.
- MySQL chỉ giữ lại để đối chiếu môi trường local cũ.
- WebSocket giữa client, Gateway và các dịch vụ nội bộ.
- Docker multi-stage build.
- GitHub Actions kiểm tra build và tích hợp PostgreSQL.

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
main/                         Điểm khởi động năm chương trình Go
server/                       Logic của từng dịch vụ
net/                          WebSocket, router và kết nối
supabase/migrations/          Migration PostgreSQL
Dockerfile.railway            Build đủ năm binary
railway.json                  Cấu hình Railway
Deploy/                       Mẫu biến môi trường
.github/workflows/            CI build và smoke-test
DEPLOYMENT_VI.md              Hướng dẫn triển khai đầy đủ
```

Thư mục biến môi trường trong repository là `deploy/railway.env.example`.

## Chạy kiểm tra

```bash
go mod tidy
go list ./... | grep -v '/main$' | xargs go test
```

Build riêng từng dịch vụ:

```bash
go build -o bin/gateserver ./main/gateserver.go
go build -o bin/httpserver ./main/httpserver.go
go build -o bin/loginserver ./main/loginserver.go
go build -o bin/chatserver ./main/chatserver.go
go build -o bin/slgserver ./main/slgserver.go
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

Backend Railway kết nối bằng `DATABASE_URL` lấy từ **Session Pooler cổng 5432** của Supabase.

Không chạy migration vào database của dự án khác. Nên tạo một Supabase Project riêng cho game.

## Railway

Tạo một Railway Project, sau đó tạo năm Service cùng trỏ tới repository này và nhánh `main`.

Ví dụ:

```dotenv
# http-service
SERVICE_NAME=http
PORT=8088

# gate-service
SERVICE_NAME=gate
PORT=8004

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

Tất cả dịch vụ đều có endpoint:

```text
/healthz
```

Chỉ tạo public domain cho `http-service` và `gate-service`.

## Kiểm tra tự động

Workflow backend thực hiện:

- unit test;
- build đủ năm binary;
- build Docker image;
- tạo PostgreSQL 16 sạch;
- chạy toàn bộ migration Supabase;
- kiểm tra đăng ký POST và bcrypt;
- khởi động đồng thời đủ năm dịch vụ;
- kiểm tra `/healthz` của từng dịch vụ.

## Triển khai

Xem [`DEPLOYMENT_VI.md`](./DEPLOYMENT_VI.md) để có danh sách biến môi trường và thứ tự cấu hình Supabase, Railway và Vercel.

## Giấy phép

Mã nguồn backend kế thừa giấy phép Apache License 2.0 của dự án gốc. Xem tệp [`LICENSE`](./LICENSE).
