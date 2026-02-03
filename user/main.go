// @title 			User Service API
// @version         1.0
// @description 	API для управления профилем пользователя
// @contact.name   	@enemybye

// @host 			localhost:8002
// @BasePath 		/api
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
	"user/redis"
	"user/repository"
	"user/service"
	"user/ws"

	userdb "user/database"

	_ "user/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	config.Load()
	userdb.InitDB()

	r := gin.Default()
	api := r.Group("/api/user")

	initProfileRoutes(api)
	initBlockRoutes(api)
	initSettingsRoutes(api)
	initWebSocketRoutes(api)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ? Запуск процессов и сервера
	srv := &http.Server{
		Addr:    ":" + config.AppConfig.PortUser,
		Handler: r,
	}

	fmt.Println("[Swagger] User swagger was launched at http://localhost:8002/swagger/index.html#/")
	go func() {
		log.Printf("User service starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	// Получаем события в фоне
	go consumer.StartUserEventsConsumer(userdb.GetDB())
	go ws.PubSubBlock()
	go ws.PubSubStatus()

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

func initProfileRoutes(api *gin.RouterGroup) {
	repo := repository.NewProfileRepository(userdb.GetDB())
	sc := service.NewProfileService(repo)
	h := handler.NewProfileHandler(sc)

	userGroup := api.Group("/profile").Use(middleware.AuthMiddleware())
	{
		userGroup.GET("/all", h.FindAll)
		userGroup.GET("search", h.SearchProfiles)
		userGroup.GET("", h.GetProfile)
		userGroup.PUT("", h.UpdateProfile)
		userGroup.GET("/:id", middleware.ValidateUUID(), h.FindProfile)

		userGroup.GET("/username/:username/check", h.IsUsernameFree)

		userGroup.GET("/:id/status", middleware.ValidateUUID(), h.GetUserStatus)
	}
}

func initBlockRoutes(api *gin.RouterGroup) {
	repo := repository.NewBlockRepository(userdb.GetDB())
	sc := service.NewBlockService(repo)
	h := handler.NewBlockHandler(sc)

	blockGroup := api.Group("/block").Use(middleware.AuthMiddleware())
	{
		blockGroup.GET("/all", h.GetAllBlocks)                               // Список заблокированных пользователей
		blockGroup.POST("/:id", middleware.ValidateUUID(), h.BlockUser)      // Заблокировать пользователя
		blockGroup.DELETE("/:id", middleware.ValidateUUID(), h.UnblockUser)  // Разблокировать
		blockGroup.GET("/check/:id", middleware.ValidateUUID(), h.IsBlocked) // Является ли заблокированным
	}
}

func initSettingsRoutes(api *gin.RouterGroup) {
	repo := repository.NewSettingsRepository(userdb.GetDB())
	sc := service.NewSettingsService(repo)
	h := handler.NewSettingsHandler(sc)

	setGroup := api.Group("/settings").Use(middleware.AuthMiddleware())
	{
		setGroup.GET("", h.GetSettings)
		setGroup.PUT("", h.SaveSettings)
	}
}

func initWebSocketRoutes(api *gin.RouterGroup) {
	r := repository.NewProfileRepository(userdb.GetDB())
	{
		api.GET("/ws", middleware.AuthMiddleware(), func(c *gin.Context) {
			ws.Connect(c, &r)
		})
	}
}
