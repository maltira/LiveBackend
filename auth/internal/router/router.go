package router

import (
	_ "auth/docs"
	"auth/internal/handler"
	"auth/internal/middleware"
	"auth/internal/repository"
	"auth/internal/service"
	authdb "auth/pkg/database"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter() *gin.Engine {
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
	login := public.Group("/login")
	register := public.Group("/register")
	otp := api.Group("/otp")

	{
		{
			register.POST("", aHandler.Register)
			register.PUT("/verify", aHandler.VerifyEmail)
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
	{
		// Восстановление пароля
		reset.POST("/forgot-password", aHandler.ForgotPassword)
		reset.POST("/reset-password", aHandler.ResetPassword)
	}

	protected := api.Group("")
	{
		protected.GET("/me", aHandler.Me)
		protected.POST("/delete-account", oHandler.VerifyDeleteAccountOTP)

		protected.POST("/change/pass", aHandler.ChangePass)
		protected.PUT("/change/pass/verify", oHandler.VerifyChangePasswordOTP)
		protected.POST("/change/email", aHandler.ChangeEmail)
		protected.PUT("/change/email/verify", oHandler.VerifyChangeMailOTP)

		protected.GET("/sessions", tHandler.ListSessions)
		protected.POST("/logout", aHandler.LogoutCurrent)
		protected.POST("/logout/all", aHandler.LogoutAll)
		protected.POST("/logout/:token_id", middleware.ValidateUUID("token_id"), tHandler.TerminateSession)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
