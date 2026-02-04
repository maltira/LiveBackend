package router

import (
	"chat/internal/handler"
	"chat/internal/middleware"
	"chat/internal/repository"
	"chat/internal/service"
	chatdb "chat/pkg/database"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	api := r.Group("/api/chat")
	api.Use(middleware.AuthMiddleware())

	initChatModule(api)
	initParticipantModule(api)
	initMessageModule(api)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

func initChatModule(api *gin.RouterGroup) {
	repo := repository.NewChatRepository(chatdb.GetDB())
	sc := service.NewChatService(repo)
	h := handler.NewChatHandler(sc)

	{
		api.GET("/:id/check", middleware.ValidateUUID(), h.IsChatExists)
		api.GET("/all", h.GetAllChats)

		api.POST("/create/private", h.CreatePrivateChat)
		api.POST("/create/group", h.CreateGroupChat)
	}
}

func initParticipantModule(api *gin.RouterGroup) {
	db := chatdb.GetDB()
	pRepo := repository.NewParticipantRepository(db)
	cRepo := repository.NewChatRepository(db)
	sc := service.NewParticipantService(pRepo, cRepo)
	h := handler.NewParticipantHandler(sc)

	partApi := api.Group("").Use(middleware.ValidateUUID())
	{
		partApi.GET("/:id/user", h.GetParticipant)
		partApi.GET("/:id/user/check", h.IsParticipant)

		partApi.POST("/join/:id", h.JoinToChat)
		partApi.DELETE("/leave/:id", h.LeaveChat)

		partApi.DELETE("/:id/kick", h.Kick)
		partApi.PUT("/:id/mute", h.Mute)
		partApi.PUT("/:id/unmute", h.Unmute)
	}

}

func initMessageModule(api *gin.RouterGroup) {
	repo := repository.NewMsgRepository(chatdb.GetDB())
	pRepo := repository.NewParticipantRepository(chatdb.GetDB())
	sc := service.NewMsgService(repo, pRepo)
	h := handler.NewMsgHandler(sc)

	msgGroup := api.Group("").Use(middleware.ValidateUUID())
	{
		msgGroup.GET("/:id/messages", h.GetMessages)
		msgGroup.GET("/:id/last-message", h.GetLastMessage)

		msgGroup.POST("/:id/send", h.SendMessage)
		msgGroup.PUT("/message/:id", h.UpdateMessage)
		msgGroup.DELETE("/message/:id", h.DeleteMessage)
	}
}
