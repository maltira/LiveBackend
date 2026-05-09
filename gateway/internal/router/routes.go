package router

import (
	"gateway/config"
	"gateway/internal/middleware"

	"github.com/gin-gonic/gin"
)

func setupPublicAuthRoutes(auth *gin.RouterGroup) {
	// Регистрация
	register := auth.Group("/register")
	register.Use(middleware.RateLimiterMiddleware("30-M", "auth:limiter:register:"))
	{
		register.POST("", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		register.PUT("/verify", ProxyToBackend("http://auth:"+config.Env.PortAuth))
	}

	// Логин
	login := auth.Group("/login")
	login.Use(middleware.RateLimiterMiddleware("30-M", "auth:limiter:login:"))
	{
		login.POST("", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		login.POST("/verify", ProxyToBackend("http://auth:"+config.Env.PortAuth))
	}

	// OTP
	otp := auth.Group("/otp")
	otp.Use(middleware.RateLimiterMiddleware("5-M", "auth:limiter:otp:"))
	{
		otp.POST("/send", ProxyToBackend("http://auth:"+config.Env.PortAuth))
	}

	// Refresh
	auth.POST("/refresh",
		middleware.RateLimiterMiddleware("60-M", "auth:limiter:refresh:"),
		ProxyToBackend("http://auth:"+config.Env.PortAuth),
	)

	// Восстановление пароля
	reset := auth.Group("")
	reset.Use(middleware.RateLimiterMiddleware("5-H", "auth:limiter:reset:"))
	{
		reset.POST("/forgot-password", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		reset.POST("/reset-password", ProxyToBackend("http://auth:"+config.Env.PortAuth))
	}
}

func setupProtectedAuthRoutes(auth *gin.RouterGroup) {
	protected := auth.Group("").Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		protected.POST("/delete-account", ProxyToBackend("http://auth:"+config.Env.PortAuth))

		protected.POST("/change/pass", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		protected.PUT("/change/pass/verify", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		protected.POST("/change/email", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		protected.PUT("/change/email/verify", ProxyToBackend("http://auth:"+config.Env.PortAuth))

		protected.GET("/sessions", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		protected.POST("/logout", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		protected.POST("/logout/all", ProxyToBackend("http://auth:"+config.Env.PortAuth))
		protected.POST("/logout/:token_id", ProxyToBackend("http://auth:"+config.Env.PortAuth))
	}
}
