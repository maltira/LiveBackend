package router

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func ProxyToBackend(backendURL string) gin.HandlerFunc {
	return createProxy(backendURL)
}

func ProxyToBackendGroup(backendURL string, group *gin.RouterGroup) {
	proxy := createProxy(backendURL)
	group.Any("/*path", proxy)
}

func createProxy(backendURL string) gin.HandlerFunc {
	target, _ := url.Parse(backendURL)

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		if userID := req.Header.Get("X-User-ID"); userID != "" {
			req.Header.Set("X-User-ID", userID)
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprintf(w, "Backend service unavailable: %v", err)
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
