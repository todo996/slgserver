package logic

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/llr104/slgserver/constant"
	"github.com/llr104/slgserver/db"
	myhttp "github.com/llr104/slgserver/server/httpserver"
	"github.com/llr104/slgserver/server/loginserver/model"
	"github.com/llr104/slgserver/util"
)

type UserLogic struct{}

var DefaultUser = UserLogic{}

func (self UserLogic) CreateUser(ctx echo.Context) error {
	account := strings.TrimSpace(requestParam(ctx, "username"))
	password := requestParam(ctx, "password")
	hardware := strings.TrimSpace(requestParam(ctx, "hardware"))

	if !validUsername(account) || !validPasswordValue(password) {
		return myhttp.New("Tài khoản hoặc mật khẩu không hợp lệ.", constant.InvalidParam)
	}
	if utf8.RuneCountInString(hardware) > 64 {
		return myhttp.New("Mã thiết bị vượt quá độ dài cho phép.", constant.InvalidParam)
	}
	if self.UserExists("username", account) {
		return myhttp.New("Tài khoản đã tồn tại.", constant.UserExist)
	}

	passwordHash, err := util.HashPassword(password)
	if err != nil {
		return myhttp.New("Không thể bảo mật mật khẩu.", constant.DBError)
	}

	now := time.Now()
	user := &model.User{
		Username: account,
		Passcode: "",
		Passwd:   passwordHash,
		Hardware: hardware,
		Ctime:    now,
		Mtime:    now,
	}

	if _, err = db.MasterDB.Insert(user); err != nil {
		if isUniqueViolation(err) {
			return myhttp.New("Tài khoản đã tồn tại.", constant.UserExist)
		}
		return myhttp.New("Không thể tạo tài khoản.", constant.DBError)
	}
	return nil
}

func (self UserLogic) ChangePassword(ctx echo.Context) error {
	account := strings.TrimSpace(requestParam(ctx, "username"))
	password := requestParam(ctx, "password")
	newPassword := requestParam(ctx, "newpassword")

	if !validUsername(account) || !validPasswordValue(password) || !validPasswordValue(newPassword) {
		return myhttp.New("Thông tin đổi mật khẩu không hợp lệ.", constant.InvalidParam)
	}

	user := &model.User{}
	found, err := db.MasterDB.Where("username=?", account).Get(user)
	if err != nil {
		return myhttp.New("Không thể đọc dữ liệu tài khoản.", constant.DBError)
	}
	if !found {
		return myhttp.New("Tài khoản không tồn tại.", constant.UserNotExist)
	}

	valid, _ := util.VerifyPassword(password, user.Passwd, user.Passcode)
	if !valid {
		return myhttp.New("Mật khẩu hiện tại không chính xác.", constant.PwdIncorrect)
	}

	passwordHash, err := util.HashPassword(newPassword)
	if err != nil {
		return myhttp.New("Không thể bảo mật mật khẩu mới.", constant.DBError)
	}

	changeData := map[string]interface{}{
		"passwd":   passwordHash,
		"passcode": "",
		"mtime":    time.Now(),
	}
	if _, err = db.MasterDB.Table(user).Where("username=?", account).Update(changeData); err != nil {
		return myhttp.New("Không thể cập nhật mật khẩu.", constant.DBError)
	}
	return nil
}

func (UserLogic) UserExists(field, value string) bool {
	user := &model.User{}
	found, err := db.MasterDB.Where(field+"=?", value).Get(user)
	return err == nil && found && user.UId != 0
}

func requestParam(ctx echo.Context, name string) string {
	return ctx.FormValue(name)
}

func validUsername(value string) bool {
	length := utf8.RuneCountInString(value)
	if length < 3 || length > 20 {
		return false
	}

	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsNumber(char) || char == '_' || char == '-' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func validPasswordValue(value string) bool {
	// bcrypt chỉ xử lý tối đa 72 byte; giới hạn này được áp dụng trước khi băm.
	length := len([]byte(value))
	return length >= 8 && length <= 72
}

func isUniqueViolation(err error) bool {
	var postgresError *pq.Error
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return true
	}

	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}
