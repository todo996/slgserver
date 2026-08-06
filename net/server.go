package net

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/llr104/slgserver/log"
	"go.uber.org/zap"
)

var wsUpgrader = websocket.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	CheckOrigin:      websocketOriginAllowed,
}

type server struct {
	addr        string
	router      *Router
	needSecret  bool
	beforeClose func(WSConn)
}

func NewServer(addr string, needSecret bool) *server {
	return &server{
		addr:       addr,
		needSecret: needSecret,
	}
}

func (this *server) Router(router *Router) {
	this.router = router
}

func (this *server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", this.healthHandler)
	mux.HandleFunc("/", this.wsHandler)

	httpServer := &http.Server{
		Addr:              this.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.DefaultLog.Info("WebSocket service đang khởi động", zap.String("addr", this.addr))
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.DefaultLog.Error("WebSocket service dừng do lỗi", zap.Error(err))
		panic(err)
	}
}

func (this *server) SetOnBeforeClose(hookFunc func(WSConn)) {
	this.beforeClose = hookFunc
}

func (this *server) healthHandler(resp http.ResponseWriter, _ *http.Request) {
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(resp).Encode(map[string]string{
		"status": "ok",
		"type":   "websocket",
	})
}

func (this *server) wsHandler(resp http.ResponseWriter, req *http.Request) {
	wsSocket, err := wsUpgrader.Upgrade(resp, req, nil)
	if err != nil {
		log.DefaultLog.Info("Từ chối kết nối WebSocket", zap.Error(err))
		return
	}

	conn := ConnMgr.NewConn(wsSocket, this.needSecret)
	log.DefaultLog.Info("Client đã kết nối", zap.String("addr", wsSocket.RemoteAddr().String()))

	conn.SetRouter(this.router)
	conn.SetOnClose(ConnMgr.RemoveConn)
	conn.SetOnBeforeClose(this.beforeClose)
	conn.Start()
	conn.Handshake()
}

func websocketOriginAllowed(req *http.Request) bool {
	origin := strings.TrimSpace(req.Header.Get("Origin"))
	if origin == "" {
		// Kết nối giữa các service Railway không bắt buộc gửi Origin.
		return true
	}

	configured := strings.TrimSpace(os.Getenv("WS_ALLOWED_ORIGINS"))
	if configured == "" || configured == "*" {
		return true
	}

	for _, allowed := range strings.Split(configured, ",") {
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}
	return false
}
