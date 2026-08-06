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

// Password giữ lại thuật toán MD5 cũ để xác thực và nâng cấp các tài khoản
// đã tồn tại trước khi hệ thống chuyển sang bcrypt.
func Password(password, passwordCode string) string {
	return Md5(password + passwordCode)
}

// HashPassword tạo mật khẩu bcrypt dùng cho tài khoản mới và mật khẩu mới.
// Giá trị đầu vào hiện là chuỗi dẫn xuất từ client; kết nối production bắt buộc
// chạy qua HTTPS/WSS để giá trị này không bị lộ trên đường truyền.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword trả về (hợp lệ, cần_nâng_cấp).
// Tài khoản MD5 cũ được chấp nhận một lần rồi nâng cấp sang bcrypt.
func VerifyPassword(password, storedHash, legacyPasswordCode string) (bool, bool) {
	if isBcryptHash(storedHash) {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil, false
	}

	legacyHash := Password(password, legacyPasswordCode)
	if len(legacyHash) != len(storedHash) {
		return false, false
	}

	valid := subtle.ConstantTimeCompare([]byte(legacyHash), []byte(storedHash)) == 1
	return valid, valid
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, "$2a$") ||
		strings.HasPrefix(value, "$2b$") ||
		strings.HasPrefix(value, "$2y$")
}
