// @title 			Auth Service API
// @version         1.0
// @description 	API для аутентификации пользователей в чат-платформе
// @contact.name   	@enemybye

// @host 			localhost:8001
// @BasePath 		/api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your Bearer token
package main

import (
	"auth/config"
	"auth/internal/router"
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
)

func main() {
	config.InitEnv()
	redis.InitAuthRedis()
	authdb.InitDB()
	rabbitmq.InitRabbitMQ()

	r := router.InitRouter()

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
