package main

import (
	"common/config"
	"common/middleware"
	"common/rabbitmq"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user/consumer"
	"user/handler"
	"user/repository"
	"user/service"

	userdb "user/database"

	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()
	userdb.InitDB()

	r := gin.Default()
	r.ForwardedByClientIP = true
	api := r.Group("/api")
	initUserRoutes(api)

	srv := &http.Server{
		Addr:    ":" + config.AppConfig.PortUser,
		Handler: r,
	}

	go func() {
		log.Printf("User service starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	// Получаем события в фоне
	go consumer.StartUserEventsConsumer(userdb.GetDB())

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
	userdb.CloseDB()
}

func initUserRoutes(api *gin.RouterGroup) {
	repo := repository.NewUserRepository(userdb.GetDB())
	sc := service.NewUserService(repo)
	h := handler.NewUserHandler(sc)

	userGroup := api.Group("/user")
	{
		userGroup.GET("/:id", middleware.ValidateUUID(), h.FindUser)
	}
}
