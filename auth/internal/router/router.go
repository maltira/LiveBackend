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
	login := public.Group("/login")
	register := public.Group("/register")

	otp := api.Group("/otp")
	otp.Use(middleware.RateLimiterMiddleware(rdb, "5-M", "auth:limiter:otp:"))

	{
		{
			register.POST("", aHandler.Register)
			register.POST("/verify", aHandler.VerifyEmail)
		}
		{
			login.POST("", aHandler.Login)
			login.POST("/verify", oHandler.VerifyLoginOTP)
		}
		{
			otp.POST("/send", oHandler.SendOTP)
		}

		public.POST("/refresh", tHandler.Refresh)
	}

	reset := api.Group("")
	reset.Use(middleware.RateLimiterMiddleware(rdb, "5-H", "auth:limiter:reset:"))
	{
		// Восстановление пароля
		reset.POST("/forgot-password", aHandler.ForgotPassword)
		reset.POST("/reset-password", aHandler.ResetPassword)
	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(tRepo))
	{
		protected.GET("/me", aHandler.Me)
		public.POST("/delete-account", oHandler.VerifyDeleteAccountOTP)

		protected.POST("/change/pass", aHandler.ChangePass)
		protected.POST("/change/pass/verify", oHandler.VerifyChangePasswordOTP)
		protected.POST("/change/email", aHandler.ChangeEmail)
		protected.POST("/change/email/verify", oHandler.VerifyChangeMailOTP)

		protected.GET("/sessions", tHandler.ListSessions)
		protected.POST("/logout", aHandler.LogoutCurrent)
		protected.POST("/logout/all", aHandler.LogoutAll)
		protected.DELETE("/logout/:token", tHandler.TerminateSession)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
