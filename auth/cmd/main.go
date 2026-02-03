// @title 			Auth Service API
// @version         1.0
// @description 	API для аутентификации пользователей в чат-платформе
// @contact.name   	@enemybye

// @host 			localhost:8001
// @BasePath 		/api
package main

import (
	"auth/config"
	"auth/internal/handler"
	"auth/internal/middleware"
	"auth/internal/repository"
	"auth/internal/service"
	authdb "auth/pkg/database"
	"auth/pkg/rabbitmq"
	"auth/pkg/redis"
	"context"
	"errors"
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
	config.InitEnv()
	rdb := redis.InitAuthRedis()
	authdb.InitDB()
	rabbitmq.InitRabbitMQ()

	authRepo := repository.NewAuthRepository(authdb.GetDB())
	authService := service.NewAuthService(authRepo)
	authHandler := handler.NewAuthHandler(authService)

	r := gin.Default()
	r.ForwardedByClientIP = true
	api := r.Group("/api")

	sensitive := api.Group("")
	sensitive.Use(middleware.RateLimiterMiddleware(rdb, "30-M", "auth:limiter:auth:"))
	{
		sensitive.POST("/auth/register", authHandler.Register)
		sensitive.POST("/auth/login", authHandler.Login)
		sensitive.POST("/auth/verify", authHandler.VerifyOTP)
		sensitive.POST("/auth/refresh", authHandler.Refresh)
		sensitive.POST("/auth/resend", authHandler.ResendOTP)
	}

	resetGroup := api.Group("")
	resetGroup.Use(middleware.RateLimiterMiddleware(rdb, "3-H", "auth:limiter:reset:"))
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

	port := config.Env.AppPort
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	log.Println("[Swagger] Auth swagger was launched at http://localhost:" + port + "/swagger/index.html#/")
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

	log.Println("[Shutting down]")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	rabbitmq.Close()
	authdb.CloseDB()
	redis.Close()
}
