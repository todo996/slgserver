package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/forgoer/openssl"
	"github.com/goinggo/mapstructure"
	"github.com/gorilla/websocket"
	gamenet "github.com/llr104/slgserver/net"
	loginproto "github.com/llr104/slgserver/server/loginserver/proto"
	"github.com/llr104/slgserver/util"
)

func main() {
	wsURL := envOr("AUTH_SMOKE_WS_URL", "ws://127.0.0.1:18080")
	origin := envOr(
		"AUTH_SMOKE_ORIGIN",
		"https://auth-preview-yrhbmcgnrg-8940s-projects.vercel.app",
	)
	username := envOr("AUTH_SMOKE_USERNAME", "auth-smoke@example.com")
	password := envOr("AUTH_SMOKE_PASSWORD", "MatKhau-AnToan-2026")

	header := http.Header{}
	header.Set("Origin", origin)
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, response, err := dialer.Dial(wsURL, header)
	if err != nil {
		status := "không có phản hồi HTTP"
		if response != nil {
			status = response.Status
		}
		fatalf("không thể mở WebSocket (%s): %v", status, err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))

	secretKey := readHandshake(conn)
	fmt.Printf("Handshake thành công, secret=%t\n", secretKey != "")

	request := &gamenet.ReqBody{
		Seq:  1,
		Name: "account.login",
		Msg: &loginproto.LoginReq{
			Username: username,
			Password: password,
			Hardware: "github-actions-auth-smoke",
		},
	}

	payload, err := util.Marshal(request)
	if err != nil {
		fatalf("không thể mã hóa JSON đăng nhập: %v", err)
	}
	if secretKey != "" {
		payload, err = util.AesCBCEncrypt(
			payload,
			[]byte(secretKey),
			[]byte(secretKey),
			openssl.ZEROS_PADDING,
		)
		if err != nil {
			fatalf("không thể mã hóa yêu cầu đăng nhập: %v", err)
		}
	}
	payload, err = util.Zip(payload)
	if err != nil {
		fatalf("không thể nén yêu cầu đăng nhập: %v", err)
	}
	if err = conn.WriteMessage(websocket.BinaryMessage, payload); err != nil {
		fatalf("không thể gửi yêu cầu đăng nhập: %v", err)
	}

	_, responsePayload, err := conn.ReadMessage()
	if err != nil {
		fatalf("không nhận được phản hồi đăng nhập: %v", err)
	}
	responsePayload, err = util.UnZip(responsePayload)
	if err != nil {
		fatalf("không thể giải nén phản hồi đăng nhập: %v", err)
	}
	if secretKey != "" {
		responsePayload, err = util.AesCBCDecrypt(
			responsePayload,
			[]byte(secretKey),
			[]byte(secretKey),
			openssl.ZEROS_PADDING,
		)
		if err != nil {
			fatalf("không thể giải mã phản hồi đăng nhập: %v", err)
		}
	}

	loginResponse := &gamenet.RspBody{}
	if err = util.Unmarshal(responsePayload, loginResponse); err != nil {
		fatalf(
			"phản hồi đăng nhập không phải JSON hợp lệ: %v; payload=%q",
			err,
			truncate(string(responsePayload), 300),
		)
	}
	fmt.Printf(
		"Phản hồi đăng nhập: name=%q seq=%d code=%d msg=%#v\n",
		loginResponse.Name,
		loginResponse.Seq,
		loginResponse.Code,
		loginResponse.Msg,
	)

	if loginResponse.Name != "account.login" || loginResponse.Seq != 1 {
		fatalf(
			"phản hồi đăng nhập không khớp: name=%q seq=%d code=%d",
			loginResponse.Name,
			loginResponse.Seq,
			loginResponse.Code,
		)
	}
	if loginResponse.Code != 0 {
		fatalf("đăng nhập bị từ chối, code=%d msg=%#v", loginResponse.Code, loginResponse.Msg)
	}

	loginData := &loginproto.LoginRsp{}
	if err = mapstructure.Decode(loginResponse.Msg, loginData); err != nil {
		fatalf("không thể đọc dữ liệu phiên đăng nhập: %v", err)
	}
	if loginData.Session == "" || loginData.UId <= 0 || loginData.Username != username {
		fatalf(
			"phiên đăng nhập không hợp lệ: uid=%d username=%q session_empty=%t msg=%#v",
			loginData.UId,
			loginData.Username,
			loginData.Session == "",
			loginResponse.Msg,
		)
	}

	fmt.Printf("Đăng nhập WebSocket thành công: uid=%d username=%s\n", loginData.UId, loginData.Username)
}

func readHandshake(conn *websocket.Conn) string {
	_, payload, err := conn.ReadMessage()
	if err != nil {
		fatalf("không nhận được handshake: %v", err)
	}
	payload, err = util.UnZip(payload)
	if err != nil {
		fatalf("không thể giải nén handshake: %v", err)
	}

	response := &gamenet.RspBody{}
	if err = util.Unmarshal(payload, response); err != nil {
		fatalf("handshake không phải JSON hợp lệ: %v payload=%q", err, truncate(string(payload), 300))
	}
	if response.Name != gamenet.HandshakeMsg {
		fatalf("mong đợi handshake nhưng nhận %q, code=%d", response.Name, response.Code)
	}

	handshake := &gamenet.Handshake{}
	if err = mapstructure.Decode(response.Msg, handshake); err != nil {
		fatalf("không thể đọc khóa handshake: %v msg=%#v", err, response.Msg)
	}
	return handshake.Key
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "…"
}

func fatalf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	annotation := strings.NewReplacer(
		"%", "%25",
		"\r", "%0D",
		"\n", "%0A",
	).Replace(message)
	fmt.Fprintf(os.Stderr, "::error title=Auth smoke thất bại::%s\n", annotation)
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
