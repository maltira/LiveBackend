// @title 			Auth Service API
// @version         1.0
// @description 	API для аутентификации пользователей в чат-платформе
// @contact.name   	@enemybye

// @host localhost:8001
// @BasePath /api
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
	config.Load()
	database.InitDB()

	authRepo := NewAuthRepository(database.GetDB())
	authService := NewAuthService(authRepo)
	authHandler := NewAuthHandler(authService)

	r := gin.Default()
	api := r.Group("/api")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/refresh", authHandler.Refresh)
		api.POST("/auth/verify", authHandler.VerifyOTP)

		// protected
		api.GET("/auth/me", middleware.AuthMiddleware(), authHandler.Me)
		api.POST("/auth/logout", middleware.AuthMiddleware(), authHandler.Logout)
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	fmt.Println("[Swagger] Auth swagger was launched at http://localhost:8001/swagger/index.html#/")
	err := r.Run(":" + config.AppConfig.PortAuth)
	if err != nil {
		panic(fmt.Sprintf("Не удалось запустить AuthService: %s", err))
	}
}
