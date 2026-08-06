package main

import (
	"fmt"

	"github.com/llr104/slgserver/config"
	"github.com/llr104/slgserver/net"
	"github.com/llr104/slgserver/server/slgserver/run"
)

func getServerAddr() string {
	return config.ListenAddress("slgserver", "8001")
}

func main() {
	fmt.Println("Khởi động SLG Server tại", getServerAddr())
	run.Init()
	needSecret := config.File.MustBool("slgserver", "need_secret", false)
	s := net.NewServer(getServerAddr(), needSecret)
	s.Router(run.MyRouter)
	s.Start()
}
