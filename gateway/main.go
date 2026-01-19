package main

import (
	"common/config"
	"common/middleware"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// TODO:
// 1. Изменение пароля
// 2. 2FA (TOTP || Google Authenticator)

// blacklist:access:[jti] - токен в черном списке
// limiter:reset:[ip] - rate-limit сброса пароля
// limiter:auth:[ip] - rate-limit авторизации и регистрации

func main() {
	config.Load()
	r := gin.Default()

	api := r.Group("/api")
	api.Use(middleware.CORSMiddleware())

	authGroup := api.Group("/auth")
	userGroup := api.Group("/user")
	proxyToBackend("http://localhost:"+config.AppConfig.PortAuth, authGroup)
	proxyToBackend("http://localhost:"+config.AppConfig.PortUser, userGroup)

	err := r.Run(":" + config.AppConfig.PortGateway)
	if err != nil {
		panic(fmt.Sprintf("Не удалось запустить GatewayService: %s", err))
	}
}

func proxyToBackend(backendURL string, group *gin.RouterGroup) {
	target, err := url.Parse(backendURL)
	if err != nil {
		log.Fatalf("Некорректный URL бэкенда %q: %v", backendURL, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprintf(w, "Backend service unavailable: %v", err)
	}

	group.Any("/*path", func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	})
}
