// @title 			User Service API
// @version         1.0
// @description 	API для управления профилем пользователя
// @contact.name   	@enemybye

// @host 			localhost:8002
// @BasePath 		/api/user

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your Bearer token
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user/config"
	"user/internal/handler"
	"user/internal/repository"
	"user/internal/router"
	userdb "user/pkg/database"
	"user/pkg/rabbitmq"
	"user/pkg/redis"
	"user/pkg/utils"

	_ "user/docs"
)

func main() {
	config.InitEnv()
	redis.InitUserRedis()
	userdb.InitDB()
	rabbitmq.InitRabbitMQ()

	// Инициализируем UpdateLastSeen через repository (без прямого DB-доступа из utils)
	pRepo := repository.NewProfileRepository(userdb.GetDB())
	utils.InitStatusUtils(pRepo.UpdateLastSeen)

	r := router.InitRouter()

	// ? Запуск процессов и сервера
	port := config.Env.AppPort
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	fmt.Println("[Swagger] User swagger was launched at http://localhost:" + port + "/swagger/index.html#/")
	go func() {
		log.Printf("User service starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	// Получаем события в фоне
	go handler.PubSubBlock()
	go handler.PubSubStatus()
	go handler.PubSubNewMessage()
	go handler.PubSubReadAck()

	// ? Завершение

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
	userdb.CloseDB()
	redis.Close()
}
