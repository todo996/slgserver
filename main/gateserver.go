package main

import (
	"fmt"
	"os"

	"github.com/llr104/slgserver/config"
	"github.com/llr104/slgserver/net"
	"github.com/llr104/slgserver/server/gateserver"
	"github.com/llr104/slgserver/server/gateserver/controller"
)

func getGateServerAddr() string {
	return config.ListenAddress("gateserver", "8004")
}

func main() {
	fmt.Println("Khởi động Gate Server tại", getGateServerAddr(), "- thư mục:", mustWorkingDirectory())
	gateserver.Init()
	needSecret := config.File.MustBool("gateserver", "need_secret", false)
	s := net.NewServer(getGateServerAddr(), needSecret)
	s.Router(gateserver.MyRouter)
	s.SetOnBeforeClose(controller.GHandle.OnServerConnClose)
	s.Start()
}

func mustWorkingDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		return "không xác định"
	}
	return dir
}
