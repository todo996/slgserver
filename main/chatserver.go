package main

import (
	"fmt"

	"github.com/llr104/slgserver/config"
	"github.com/llr104/slgserver/net"
	"github.com/llr104/slgserver/server/chatserver"
)

func getChatServerAddr() string {
	return config.ListenAddress("chatserver", "8002")
}

func main() {
	fmt.Println("Khởi động Chat Server tại", getChatServerAddr())
	chatserver.Init()
	needSecret := config.File.MustBool("chatserver", "need_secret", false)
	s := net.NewServer(getChatServerAddr(), needSecret)
	s.Router(chatserver.MyRouter)
	s.Start()
}
