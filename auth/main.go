// @title 			Auth Service API
// @version         1.0
// @description 	API для аутентификации пользователей в чат-платформе
// @contact.name   	@enemybye

// @host 			localhost:8001
// @BasePath 		/api
package main

import (
	"auth/database"
	"common/config"
	"common/middleware"
	"fmt"

	"github.com/gin-gonic/gin"

	_ "auth/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	rdb := config.AuthRedisClient()
	config.Load()
	database.InitDB()

	authRepo := NewAuthRepository(database.GetDB())
	authService := NewAuthService(authRepo)
	authHandler := NewAuthHandler(authService)

	r := gin.Default()
	r.ForwardedByClientIP = true
	api := r.Group("/api")

	sensitive := api.Group("")
	sensitive.Use(middleware.RateLimiterMiddleware(rdb, "10-M", "limiter:auth:"))
	{
		sensitive.POST("/auth/register", authHandler.Register)
		sensitive.POST("/auth/login", authHandler.Login)
		sensitive.POST("/auth/verify", authHandler.VerifyOTP)
		sensitive.POST("/auth/refresh", authHandler.Refresh)
	}

	resetGroup := api.Group("")
	resetGroup.Use(middleware.RateLimiterMiddleware(rdb, "3-H", "limiter:auth:"))
	{
		resetGroup.POST("/auth/forgot-password", authHandler.ForgotPassword)
		resetGroup.POST("/auth/reset-password", authHandler.ResetPassword)
	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/auth/me", authHandler.Me)
		protected.POST("/auth/logout", authHandler.LogoutCurrent)
		protected.POST("/auth/logout/all", authHandler.LogoutAll)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	fmt.Println("[Swagger] Auth swagger was launched at http://localhost:8001/swagger/index.html#/")
	err := r.Run(":" + config.AppConfig.PortAuth)
	if err != nil {
		panic(fmt.Sprintf("Не удалось запустить AuthService: %s", err))
	}
}
