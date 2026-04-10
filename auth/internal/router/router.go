package router

import (
	"auth/internal/handler"
	"auth/internal/middleware"
	"auth/internal/repository"
	"auth/internal/service"
	authdb "auth/pkg/database"
	"auth/pkg/redis"

	_ "auth/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter() *gin.Engine {
	rdb := redis.AuthRedis
	authRepo := repository.NewAuthRepository(authdb.GetDB())
	authService := service.NewAuthService(authRepo)
	authHandler := handler.NewAuthHandler(authService)
	otpHandler := handler.NewOtpHandler(authService)
	refreshHandler := handler.NewRefreshHandler(authService)

	r := gin.Default()
	r.ForwardedByClientIP = true
	api := r.Group("/api/auth")

	public := api.Group("")
	public.Use(middleware.RateLimiterMiddleware(rdb, "30-M", "auth:limiter:auth:"))
	{
		public.POST("/register", authHandler.Register)
		public.POST("/login", authHandler.Login)
		public.POST("/verify", otpHandler.VerifyOTP)
		public.POST("/resend", otpHandler.ResendOTP)
		public.POST("/refresh", refreshHandler.Refresh)

		public.POST("/logout", authHandler.LogoutCurrent)

		public.POST("/verify-email/:token", authHandler.VerifyEmail)
	}

	reset := api.Group("")
	reset.Use(middleware.RateLimiterMiddleware(rdb, "3-H", "auth:limiter:reset:"))
	{
		reset.POST("/forgot-password", authHandler.ForgotPassword)
		reset.POST("/reset-password", authHandler.ResetPassword)
		reset.PUT("/recovery/:id", middleware.ValidateUUID(), authHandler.RecoveryAccount)
	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", authHandler.Me)
		protected.GET("/sessions", refreshHandler.ListSessions)

		protected.POST("/change-mail", authHandler.ChangeMail)
		protected.POST("/change-pass", authHandler.ChangePass)

		protected.POST("/delete/:email", authHandler.Delete)

		protected.POST("/logout/all", authHandler.LogoutAll)
		protected.DELETE("/logout/:token", refreshHandler.TerminateSession)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
