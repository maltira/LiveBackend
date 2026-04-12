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
	aRepo := repository.NewAuthRepository(authdb.GetDB())
	oRepo := repository.NewOtpRepository(authdb.GetDB())
	tRepo := repository.NewTokenRepository(authdb.GetDB())
	aSc := service.NewAuthService(aRepo, tRepo)
	oSc := service.NewOtpService(oRepo)
	tSc := service.NewTokenService(tRepo)
	aHandler := handler.NewAuthHandler(aSc, oSc, tSc)
	oHandler := handler.NewOtpHandler(oSc, aSc, tSc)
	tHandler := handler.NewRefreshHandler(tSc)

	r := gin.Default()
	r.ForwardedByClientIP = true
	api := r.Group("/api/auth")

	public := api.Group("")
	public.Use(middleware.RateLimiterMiddleware(rdb, "30-M", "auth:limiter:auth:"))
	{
		public.POST("/register", aHandler.Register)
		public.POST("/login", aHandler.Login)
		public.POST("/resend", oHandler.ResendOTP)
		public.POST("/refresh", tHandler.Refresh)

		public.POST("/verify/login", oHandler.VerifyLoginOTP)
		// public.POST("/verify/email-change", oHandler.VerifyOTP)
		// public.POST("/verify/pass-change", oHandler.VerifyOTP)

		public.POST("/logout", aHandler.LogoutCurrent)

		public.POST("/verify-email", aHandler.VerifyEmail)
	}

	reset := api.Group("")
	reset.Use(middleware.RateLimiterMiddleware(rdb, "3-H", "auth:limiter:reset:"))
	{
		// Восстановление пароля
		reset.POST("/request/forgot-password", aHandler.ForgotPassword)
		reset.POST("/reset-password", aHandler.ResetPassword)
	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(tRepo))
	{
		protected.GET("/me", aHandler.Me)
		protected.GET("/sessions", tHandler.ListSessions)

		protected.POST("/change-mail", aHandler.ChangeMail)
		protected.POST("/change-pass", aHandler.ChangePass)

		protected.POST("/delete-account", aHandler.DeleteAccount)
		public.POST("/verify/del-account", oHandler.VerifyDeleteAccountOTP)

		protected.POST("/logout/all", aHandler.LogoutAll)
		protected.DELETE("/logout/:token", tHandler.TerminateSession)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
