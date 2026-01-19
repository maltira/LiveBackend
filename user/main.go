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
	api := r.Group("/api/user")

	initProfileRoutes(api)
	initBlockRoutes(api)

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

func initProfileRoutes(api *gin.RouterGroup) {
	repo := repository.NewProfileRepository(userdb.GetDB())
	sc := service.NewProfileService(repo)
	h := handler.NewProfileHandler(sc)

	userGroup := api.Group("/profile").Use(middleware.AuthMiddleware())
	{
		userGroup.GET("/all", h.FindAll)
		userGroup.GET("", h.GetProfile)
		userGroup.PUT("", h.UpdateProfile)
		userGroup.GET("/:id", middleware.ValidateUUID(), h.FindProfile)

		userGroup.GET("/username/:username/check", h.IsUsernameFree)
	}
}

func initBlockRoutes(api *gin.RouterGroup) {
	repo := repository.NewBlockRepository(userdb.GetDB())
	sc := service.NewBlockService(repo)
	h := handler.NewBlockHandler(sc)

	blockGroup := api.Group("/block").Use(middleware.AuthMiddleware())
	{
		blockGroup.GET("/all", h.GetAllBlocks)                              // Список заблокированных пользователей
		blockGroup.POST("/:id", middleware.ValidateUUID(), h.BlockUser)     // Заблокировать пользователя
		blockGroup.DELETE("/:id", middleware.ValidateUUID(), h.UnblockUser) // Разблокировать
		blockGroup.GET("/check", h.IsUserBlocked)                           // Является ли заблокированным
	}
}
