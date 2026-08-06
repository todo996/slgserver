package loginserver

import (
	"fmt"

	"github.com/llr104/slgserver/db"
	"github.com/llr104/slgserver/net"
	"github.com/llr104/slgserver/server/loginserver/controller"
)

var MyRouter = &net.Router{}

func Init() {
	if err := db.TestDB(); err != nil {
		panic(fmt.Errorf("login-service không thể kết nối cơ sở dữ liệu: %w", err))
	}

	// Chỉ đăng ký router sau khi mọi phụ thuộc đã sẵn sàng.
	initRouter()
}

func initRouter() {
	controller.DefaultAccount.InitRouter(MyRouter)
}
