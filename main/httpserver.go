package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	mw "github.com/labstack/echo/v4/middleware"
	"github.com/llr104/slgserver/config"
	"github.com/llr104/slgserver/db"
	"github.com/llr104/slgserver/server/httpserver/controller"
)

const vercelPreviewSuffix = "-yrhbmcgnrg-8940s-projects.vercel.app"

func main() {
	if err := db.TestDB(); err != nil {
		log.Fatal("Không thể kết nối cơ sở dữ liệu: ", err)
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(mw.Recover())
	e.Use(corsMiddleware())

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "httpserver",
		})
	})

	group := e.Group("")
	new(controller.AccountController).RegisterRoutes(group)

	e.Server.Addr = getHTTPAddr()
	log.Printf("HTTP Server đang lắng nghe tại %s", e.Server.Addr)
	log.Fatal(e.StartServer(e.Server))
}

func getHTTPAddr() string {
	return config.ListenAddress("httpserver", "8088")
}

func corsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			request := c.Request()
			response := c.Response()
			origin := strings.TrimSpace(request.Header.Get(echo.HeaderOrigin))
			allowed := origin == "" || isAllowedHTTPOrigin(origin)

			if origin != "" && allowed {
				response.Header().Set(echo.HeaderAccessControlAllowOrigin, origin)
				response.Header().Add(echo.HeaderVary, echo.HeaderOrigin)
				response.Header().Set(
					echo.HeaderAccessControlAllowMethods,
					"GET,POST,PUT,PATCH,DELETE,OPTIONS",
				)
				response.Header().Set(
					echo.HeaderAccessControlAllowHeaders,
					"Origin,Content-Type,Accept,Authorization",
				)
			}

			if request.Method == http.MethodOptions {
				if !allowed {
					return c.NoContent(http.StatusForbidden)
				}
				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	}
}

func isAllowedHTTPOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	if isSLGClientVercelOrigin(origin) {
		return true
	}

	for _, allowed := range allowedOrigins() {
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

func isSLGClientVercelOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	return host == "slgclient.vercel.app" || strings.HasSuffix(host, vercelPreviewSuffix)
}

func allowedOrigins() []string {
	value := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if value == "" {
		return []string{"*"}
	}

	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}

	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}
