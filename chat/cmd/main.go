package main

import (
	"chat/config"
	"chat/internal/router"
	chatdb "chat/pkg/database"
	"chat/pkg/redis"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	config.InitEnv()
	redis.InitChatRedis()
	chatdb.InitDB()
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
	chatdb.CloseDB()
	redis.Close()
}
