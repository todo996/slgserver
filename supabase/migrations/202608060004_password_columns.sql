BEGIN;

-- MySQL CHAR thường được code cũ xử lý như chuỗi không đệm, trong khi
-- PostgreSQL CHAR(n) trả về giá trị có khoảng trắng ở cuối. Điều đó làm
-- bcrypt 60 ký tự bị biến thành 64 ký tự và khiến xác thực thất bại.
ALTER TABLE public.tb_user_info
    ALTER COLUMN passcode DROP DEFAULT,
    ALTER COLUMN passwd DROP DEFAULT,
    ALTER COLUMN passcode TYPE VARCHAR(32) USING RTRIM(passcode),
    ALTER COLUMN passwd TYPE VARCHAR(128) USING RTRIM(passwd),
    ALTER COLUMN passcode SET DEFAULT '',
    ALTER COLUMN passwd SET DEFAULT '';

UPDATE public.tb_user_info
SET
    passcode = RTRIM(passcode),
    passwd = RTRIM(passwd)
WHERE
    passcode <> RTRIM(passcode)
    OR passwd <> RTRIM(passwd);

COMMIT;
