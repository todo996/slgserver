BEGIN;

-- Đồng bộ với validation của client/backend và cho phép dùng địa chỉ email
-- làm tên tài khoản. ALTER TYPE giữ nguyên dữ liệu và unique constraint cũ.
ALTER TABLE IF EXISTS public.tb_user_info
    ALTER COLUMN username TYPE VARCHAR(50);

COMMIT;
