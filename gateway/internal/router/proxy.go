package router

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

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
	target, err := url.Parse(backendURL)
	if err != nil {
		log.Fatalf("Invalid backend URL %q: %v", backendURL, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	}

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprintf(w, "Backend service unavailable: %v", err)
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
