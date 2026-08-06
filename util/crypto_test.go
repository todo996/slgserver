package util

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	password := "md5-client-value"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() lỗi: %v", err)
	}
	if hash == password {
		t.Fatal("HashPassword() không được trả lại mật khẩu gốc")
	}

	valid, needsUpgrade := VerifyPassword(password, hash, "")
	if !valid {
		t.Fatal("VerifyPassword() phải chấp nhận mật khẩu đúng")
	}
	if needsUpgrade {
		t.Fatal("bcrypt hiện tại không được đánh dấu cần nâng cấp")
	}

	valid, _ = VerifyPassword("sai-mat-khau", hash, "")
	if valid {
		t.Fatal("VerifyPassword() không được chấp nhận mật khẩu sai")
	}
}

func TestVerifyLegacyPasswordRequiresUpgrade(t *testing.T) {
	password := "legacy-client-md5"
	passcode := "abc123"
	legacyHash := Password(password, passcode)

	valid, needsUpgrade := VerifyPassword(password, legacyHash, passcode)
	if !valid {
		t.Fatal("tài khoản MD5 cũ phải đăng nhập được")
	}
	if !needsUpgrade {
		t.Fatal("tài khoản MD5 cũ phải được đánh dấu cần nâng cấp")
	}
}
