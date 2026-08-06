package main

import (
	"fmt"

	"github.com/llr104/slgserver/config"
	"github.com/llr104/slgserver/net"
	"github.com/llr104/slgserver/server/loginserver"
)

func getLoginServerAddr() string {
	return config.ListenAddress("loginserver", "8003")
}

func main() {
	fmt.Println("Khởi động Login Server tại", getLoginServerAddr())
	loginserver.Init()
	needSecret := config.File.MustBool("loginserver", "need_secret", false)
	s := net.NewServer(getLoginServerAddr(), needSecret)
	s.Router(loginserver.MyRouter)
	s.Start()
}
