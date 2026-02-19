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

	sensitive := api.Group("")
	sensitive.Use(middleware.RateLimiterMiddleware(rdb, "30-M", "auth:limiter:auth:"))
	{
		sensitive.POST("/register", authHandler.Register)
		sensitive.POST("/login", authHandler.Login)
		sensitive.POST("/verify", otpHandler.VerifyOTP)
		sensitive.POST("/resend", otpHandler.ResendOTP)
	}

	resetGroup := api.Group("")
	resetGroup.Use(middleware.RateLimiterMiddleware(rdb, "3-H", "auth:limiter:reset:"))
	{
		resetGroup.POST("/forgot-password", authHandler.ForgotPassword)
		resetGroup.POST("/reset-password", authHandler.ResetPassword)
		resetGroup.POST("/delete/cancel", authHandler.DeleteCancel)
	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/refresh", refreshHandler.Refresh)

		protected.GET("/me", authHandler.Me)
		protected.GET("/sessions", refreshHandler.ListSessions)

		protected.POST("/delete", authHandler.Delete)
		protected.POST("/delete/confirm", authHandler.DeleteConfirm)

		protected.POST("/change-mail", authHandler.ChangeMail)
		protected.POST("/change-pass", authHandler.ChangePass)

		protected.POST("/logout", authHandler.LogoutCurrent)
		protected.POST("/logout/all", authHandler.LogoutAll)
		protected.DELETE("/logout/:token", refreshHandler.TerminateSession)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
