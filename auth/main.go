// @title 			Auth Service API
// @version         1.0
// @description 	API для аутентификации пользователей в чат-платформе
// @contact.name   	@enemybye

// @host 			localhost:8001
// @BasePath 		/api
package main

import (
	authdb "auth/database"
	"common/config"
	"common/middleware"
	"common/rabbitmq"
	"common/redis"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	_ "auth/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	config.Load()
	rdb := redis.AuthRedisClient()
	authdb.InitDB()

	authRepo := NewAuthRepository(authdb.GetDB())
	authService := NewAuthService(authRepo)
	authHandler := NewAuthHandler(authService)

	r := gin.Default()
	r.ForwardedByClientIP = true
	api := r.Group("/api")

	sensitive := api.Group("")
	sensitive.Use(middleware.RateLimiterMiddleware(rdb, "30-M", "limiter:auth:"))
	{
		sensitive.POST("/auth/register", authHandler.Register)
		sensitive.POST("/auth/login", authHandler.Login)
		sensitive.POST("/auth/verify", authHandler.VerifyOTP)
		sensitive.POST("/auth/refresh", authHandler.Refresh)
		sensitive.POST("/auth/resend", authHandler.ResendOTP)
	}

	resetGroup := api.Group("")
	resetGroup.Use(middleware.RateLimiterMiddleware(rdb, "3-H", "limiter:reset:"))
	{
		resetGroup.POST("/auth/forgot-password", authHandler.ForgotPassword)
		resetGroup.POST("/auth/reset-password", authHandler.ResetPassword)
		resetGroup.POST("/auth/delete/cancel", authHandler.DeleteCancel)
	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/auth/me", authHandler.Me)
		protected.GET("/auth/sessions", authHandler.ListSessions)

		protected.POST("/auth/delete", authHandler.Delete)
		protected.POST("/auth/delete/confirm", authHandler.DeleteConfirm)

		protected.POST("/auth/logout", authHandler.LogoutCurrent)
		protected.POST("/auth/logout/all", authHandler.LogoutAll)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{
		Addr:    ":" + config.AppConfig.PortAuth,
		Handler: r,
	}

	fmt.Println("[Swagger] Auth swagger was launched at http://localhost:8001/swagger/index.html#/")
	go func() {
		log.Printf("Auth service starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	// Блокируем main, ждём сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n[Shutting down]")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	rabbitmq.Close()
	authdb.CloseDB()
	redis.Close()
}
