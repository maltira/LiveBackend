package router

import (
	"user/internal/handler"
	"user/internal/middleware"
	"user/internal/repository"
	"user/internal/service"
	userdb "user/pkg/database"

	_ "user/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter() *gin.Engine {
	pRepo := repository.NewProfileRepository(userdb.GetDB())
	bRepo := repository.NewBlockRepository(userdb.GetDB())
	sRepo := repository.NewSettingsRepository(userdb.GetDB())
	psc := service.NewProfileService(pRepo)
	bsc := service.NewBlockService(bRepo)
	ssc := service.NewSettingsService(sRepo)
	pHandler := handler.NewProfileHandler(psc)
	bHandler := handler.NewBlockHandler(bsc)
	sHandler := handler.NewSettingsHandler(ssc)

	r := gin.Default()
	api := r.Group("/api/user")

	initProfileRoutes(api, pHandler)
	initBlockRoutes(api, bHandler)
	initSettingsRoutes(api, sHandler)
	initWebSocketRoutes(api)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

func initProfileRoutes(api *gin.RouterGroup, h *handler.ProfileHandler) {
	api.POST("/profile", h.CreateProfile)
	userGroup := api.Group("/profile").Use(middleware.AuthMiddleware())
	{
		userGroup.GET("/all", h.FindAll)
		userGroup.GET("search", h.GetProfilesByQuery)
		userGroup.GET("", h.GetCurrentProfile)
		userGroup.PUT("", h.UpdateProfile)
		userGroup.GET("/:id", middleware.ValidateUUID(), h.GetProfileByID)

		userGroup.GET("/check-username", h.IsUsernameFree)
		userGroup.GET("/check-status/:id", middleware.ValidateUUID(), h.GetUserStatus)
	}
}

func initBlockRoutes(api *gin.RouterGroup, h *handler.BlockHandler) {
	blockGroup := api.Group("/block").Use(middleware.AuthMiddleware())
	{
		blockGroup.GET("/all", h.GetAllBlocks)                               // Список заблокированных пользователей
		blockGroup.POST("/:id", middleware.ValidateUUID(), h.BlockUser)      // Заблокировать пользователя
		blockGroup.DELETE("/:id", middleware.ValidateUUID(), h.UnblockUser)  // Разблокировать
		blockGroup.GET("/check/:id", middleware.ValidateUUID(), h.IsBlocked) // Является ли заблокированным
	}
}

func initSettingsRoutes(api *gin.RouterGroup, h *handler.SettingsHandler) {
	setGroup := api.Group("/settings").Use(middleware.AuthMiddleware())
	{
		setGroup.GET("", h.GetSettings)
		setGroup.PUT("/status", h.UpdateVisibleStatus)
		setGroup.PUT("/birth", h.UpdateVisibleBirthDate)
	}
}

func initWebSocketRoutes(api *gin.RouterGroup) {
	{
		api.GET("/ws", middleware.AuthMiddleware(), func(c *gin.Context) {
			handler.Connect(c)
		})
	}
}
