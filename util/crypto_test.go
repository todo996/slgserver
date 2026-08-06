package util

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	password := "MatKhau-AnToan-2026"
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

func TestVerifyLegacyPasswordFromNewClientRequiresUpgrade(t *testing.T) {
	password := "MatKhau-Cu-2024"
	passcode := "abc123"
	legacyHash := Password(Md5(password), passcode)

	valid, needsUpgrade := VerifyPassword(password, legacyHash, passcode)
	if !valid {
		t.Fatal("tài khoản MD5 cũ phải đăng nhập được bằng client mới")
	}
	if !needsUpgrade {
		t.Fatal("tài khoản MD5 cũ phải được đánh dấu cần nâng cấp")
	}
}

func TestVerifyLegacyPasswordFromOldClientRequiresUpgrade(t *testing.T) {
	password := "MatKhau-Cu-2024"
	clientMD5 := Md5(password)
	passcode := "abc123"
	legacyHash := Password(clientMD5, passcode)

	valid, needsUpgrade := VerifyPassword(clientMD5, legacyHash, passcode)
	if !valid {
		t.Fatal("client MD5 cũ phải tiếp tục đăng nhập được trong thời gian chuyển đổi")
	}
	if !needsUpgrade {
		t.Fatal("client MD5 cũ phải được đánh dấu cần nâng cấp")
	}
}

func TestVerifyTransitionalBcryptRequiresUpgrade(t *testing.T) {
	password := "MatKhau-Chuyen-Tiep"
	clientMD5 := Md5(password)
	transitionalHash, err := HashPassword(clientMD5)
	if err != nil {
		t.Fatalf("không tạo được bcrypt chuyển tiếp: %v", err)
	}

	valid, needsUpgrade := VerifyPassword(password, transitionalHash, "")
	if !valid {
		t.Fatal("bcrypt chuyển tiếp của MD5 phải được chấp nhận")
	}
	if !needsUpgrade {
		t.Fatal("bcrypt chuyển tiếp phải được nâng cấp sang bcrypt mật khẩu gốc")
	}
}
