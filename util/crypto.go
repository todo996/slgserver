package util

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/forgoer/openssl"
	"golang.org/x/crypto/bcrypt"
)

func AesCBCEncrypt(src, key, iv []byte, padding string) ([]byte, error) {
	data, err := openssl.AesCBCEncrypt(src, key, iv, padding)
	if err != nil {
		return nil, err
	}
	return []byte(hex.EncodeToString(data)), nil
}

func AesCBCDecrypt(src, key, iv []byte, padding string) ([]byte, error) {
	data, err := hex.DecodeString(string(src))
	if err != nil {
		return nil, err
	}
	return openssl.AesCBCDecrypt(data, key, iv, padding)
}

func Md5(text string) string {
	hashMd5 := md5.New()
	_, _ = io.WriteString(hashMd5, text)
	return fmt.Sprintf("%x", hashMd5.Sum(nil))
}

func Zip(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&b, 9)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Flush(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func UnZip(data []byte) ([]byte, error) {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, data)
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	unzipData, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return unzipData, nil
}

// Password giữ lại thuật toán MD5 cũ để xác thực và nâng cấp tài khoản đã tồn tại.
func Password(password, passwordCode string) string {
	return Md5(password + passwordCode)
}

// HashPassword dùng bcrypt trực tiếp trên mật khẩu người dùng. Production bắt
// buộc chạy qua HTTPS/WSS để mật khẩu được bảo vệ trên đường truyền.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword trả về (hợp_lệ, cần_nâng_cấp).
// Hàm hỗ trợ ba giai đoạn để cập nhật không làm gián đoạn người chơi:
//   1. bcrypt của mật khẩu gốc (chuẩn mới);
//   2. bcrypt của MD5 phía client (bản chuyển tiếp);
//   3. MD5 có passcode của hệ thống cũ.
func VerifyPassword(password, storedHash, legacyPasswordCode string) (bool, bool) {
	if isBcryptHash(storedHash) {
		if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil {
			return true, false
		}

		legacyClientValue := Md5(password)
		if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(legacyClientValue)) == nil {
			return true, true
		}
		return false, false
	}

	// Client mới gửi mật khẩu gốc; client cũ có thể vẫn gửi MD5 32 ký tự.
	legacyCandidates := []string{
		Password(Md5(password), legacyPasswordCode),
		Password(password, legacyPasswordCode),
	}

	valid := 0
	for _, candidate := range legacyCandidates {
		if len(candidate) == len(storedHash) {
			valid |= subtle.ConstantTimeCompare([]byte(candidate), []byte(storedHash))
		}
	}
	return valid == 1, valid == 1
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, "$2a$") ||
		strings.HasPrefix(value, "$2b$") ||
		strings.HasPrefix(value, "$2y$")
}
