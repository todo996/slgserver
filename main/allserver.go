package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	loginURL = "http://127.0.0.1:8003"
	chatURL  = "http://127.0.0.1:8002"
	slgURL   = "http://127.0.0.1:8001"
	gateURL  = "http://127.0.0.1:8004"
	httpURL  = "http://127.0.0.1:8088"
)

type childSpec struct {
	name   string
	binary string
	env    map[string]string
	health string
}

type childExit struct {
	name string
	err  error
}

type healthState struct {
	Status   string            `json:"status"`
	Service  string            `json:"service"`
	Children map[string]string `json:"children"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	binaryDir, err := executableDir()
	if err != nil {
		log.Fatal(err)
	}

	commonEnv := map[string]string{
		"LOGIN_HOST":     "127.0.0.1",
		"LOGIN_PORT":     "8003",
		"CHAT_HOST":      "127.0.0.1",
		"CHAT_PORT":      "8002",
		"SLG_HOST":       "127.0.0.1",
		"SLG_PORT":       "8001",
		"GATE_HOST":      "127.0.0.1",
		"GATE_PORT":      "8004",
		"HTTP_HOST":      "127.0.0.1",
		"HTTP_PORT":      "8088",
		"LOGIN_PROXY_URL": "ws://127.0.0.1:8003",
		"CHAT_PROXY_URL":  "ws://127.0.0.1:8002",
		"SLG_PROXY_URL":   "ws://127.0.0.1:8001",
	}
	if os.Getenv("GATE_PUBLIC_URL") == "" {
		if domain := strings.TrimSpace(os.Getenv("RAILWAY_PUBLIC_DOMAIN")); domain != "" {
			commonEnv["GATE_PUBLIC_URL"] = "wss://" + domain
		}
	}

	exits := make(chan childExit, 8)
	children := []childSpec{
		{name: "login", binary: "loginserver", health: loginURL + "/healthz"},
		{name: "chat", binary: "chatserver", health: chatURL + "/healthz"},
		{name: "slg", binary: "slgserver", health: slgURL + "/healthz"},
		{name: "gate", binary: "gateserver", health: gateURL + "/healthz"},
		{name: "http", binary: "httpserver", health: httpURL + "/healthz"},
	}

	for _, child := range children {
		child.env = commonEnv
		if err := startChild(ctx, binaryDir, child, exits); err != nil {
			stop()
			log.Fatalf("Không thể khởi động %s: %v", child.name, err)
		}
	}

	startupTimeout := durationFromEnv("STARTUP_TIMEOUT", 180*time.Second)
	for _, child := range children {
		if err := waitHealthy(ctx, child, exits, startupTimeout); err != nil {
			stop()
			log.Fatalf("%s không sẵn sàng: %v", child.name, err)
		}
		log.Printf("%s đã sẵn sàng", child.name)
	}

	gateProxy := newReverseProxy(gateURL)
	httpProxy := newReverseProxy(httpURL)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", aggregateHealth(children))
	mux.HandleFunc("/readyz", aggregateHealth(children))
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		if isWebSocket(request) {
			gateProxy.ServeHTTP(response, request)
			return
		}
		httpProxy.ServeHTTP(response, request)
	})

	publicPort := strings.TrimSpace(os.Getenv("PORT"))
	if publicPort == "" {
		publicPort = "8080"
	}
	publicServer := &http.Server{
		Addr:              ":" + publicPort,
		Handler:           mux,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	publicErrors := make(chan error, 1)
	go func() {
		log.Printf("Backend hợp nhất đang lắng nghe tại :%s", publicPort)
		if err := publicServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			publicErrors <- err
		}
	}()

	exitCode := 0
	select {
	case <-ctx.Done():
		log.Printf("Nhận tín hiệu dừng backend hợp nhất")
	case child := <-exits:
		exitCode = 1
		log.Printf("Thành phần %s đã dừng ngoài dự kiến: %v", child.name, child.err)
	case err := <-publicErrors:
		exitCode = 1
		log.Printf("Cổng public đã dừng ngoài dự kiến: %v", err)
	}

	stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = publicServer.Shutdown(shutdownCtx)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func executableDir() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("không xác định được tệp thực thi: %w", err)
	}
	return filepath.Dir(path), nil
}

func startChild(ctx context.Context, binaryDir string, spec childSpec, exits chan<- childExit) error {
	binaryPath := filepath.Join(binaryDir, spec.binary)
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = binaryDir
	cmd.Env = mergedEnv(os.Environ(), spec.env)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	log.Printf("Đã khởi động %s với PID %d", spec.name, cmd.Process.Pid)
	go func() {
		err := cmd.Wait()
		if ctx.Err() == nil {
			exits <- childExit{name: spec.name, err: err}
		}
	}()
	return nil
}

func waitHealthy(ctx context.Context, spec childSpec, exits <-chan childExit, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	client := &http.Client{Timeout: 3 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case child := <-exits:
			return fmt.Errorf("%s đã dừng khi khởi động: %w", child.name, child.err)
		case <-deadline.C:
			return fmt.Errorf("hết thời gian chờ %s", spec.health)
		case <-ticker.C:
			response, err := client.Get(spec.health)
			if err == nil {
				_ = response.Body.Close()
				if response.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

func aggregateHealth(children []childSpec) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		states := make(map[string]string, len(children))
		healthy := true
		client := &http.Client{Timeout: 2 * time.Second}
		for _, child := range children {
			result, err := client.Get(child.health)
			if err != nil {
				states[child.name] = "không phản hồi"
				healthy = false
				continue
			}
			_ = result.Body.Close()
			if result.StatusCode != http.StatusOK {
				states[child.name] = fmt.Sprintf("HTTP %d", result.StatusCode)
				healthy = false
				continue
			}
			states[child.name] = "ok"
		}

		statusCode := http.StatusOK
		status := "ok"
		if !healthy {
			statusCode = http.StatusServiceUnavailable
			status = "degraded"
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		response.WriteHeader(statusCode)
		_ = json.NewEncoder(response).Encode(healthState{
			Status:   status,
			Service:  "allserver",
			Children: states,
		})
	}
}

func newReverseProxy(rawTarget string) *httputil.ReverseProxy {
	target, err := url.Parse(rawTarget)
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(response http.ResponseWriter, request *http.Request, err error) {
		log.Printf("Lỗi reverse proxy %s: %v", request.URL.Path, err)
		http.Error(response, "Dịch vụ tạm thời chưa sẵn sàng.", http.StatusBadGateway)
	}
	return proxy
}

func isWebSocket(request *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(request.Header.Get("Upgrade")), "websocket") &&
		strings.Contains(strings.ToLower(request.Header.Get("Connection")), "upgrade")
}

func mergedEnv(base []string, overrides map[string]string) []string {
	values := make(map[string]string, len(base)+len(overrides))
	for _, item := range base {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			values[parts[0]] = parts[1]
		}
	}
	for key, value := range overrides {
		values[key] = value
	}
	result := make([]string, 0, len(values))
	for key, value := range values {
		result = append(result, key+"="+value)
	}
	return result
}

func durationFromEnv(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("%s không hợp lệ (%q), dùng %s", name, value, fallback)
		return fallback
	}
	return duration
}
