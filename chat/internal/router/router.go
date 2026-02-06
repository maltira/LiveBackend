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

var (
	cRepo repository.ChatRepository        = repository.NewChatRepository(chatdb.GetDB())
	mRepo repository.MsgRepository         = repository.NewMsgRepository(chatdb.GetDB())
	pRepo repository.ParticipantRepository = repository.NewParticipantRepository(chatdb.GetDB())
)
var (
	cService service.ChatService        = service.NewChatService(cRepo)
	mService service.MsgService         = service.NewMsgService(mRepo, pRepo)
	pService service.ParticipantService = service.NewParticipantService(pRepo, cRepo)
)
var (
	cHandler *handler.ChatHandler        = handler.NewChatHandler(cService, mService)
	mHandler *handler.MsgHandler         = handler.NewMsgHandler(mService)
	pHandler *handler.ParticipantHandler = handler.NewParticipantHandler(pService)
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
	{
		api.GET("/:id/check", middleware.ValidateUUID(), cHandler.IsChatExists)
		api.GET("/all", cHandler.GetAllChats)

		api.POST("/create/private", cHandler.CreatePrivateChat)
		api.POST("/create/group", cHandler.CreateGroupChat)
	}
}

func initParticipantModule(api *gin.RouterGroup) {
	partApi := api.Group("").Use(middleware.ValidateUUID())
	{
		partApi.GET("/:id/user", pHandler.GetParticipant)
		partApi.GET("/:id/user/check", pHandler.IsParticipant)

		partApi.POST("/join/:id", pHandler.JoinToChat)
		partApi.DELETE("/leave/:id", pHandler.LeaveChat)

		partApi.DELETE("/:id/kick", pHandler.Kick)
		partApi.PUT("/:id/mute", pHandler.Mute)
		partApi.PUT("/:id/unmute", pHandler.Unmute)
	}

}

func initMessageModule(api *gin.RouterGroup) {
	msgGroup := api.Group("").Use(middleware.ValidateUUID())
	{
		msgGroup.GET("/:id/messages", mHandler.GetMessages)
		msgGroup.GET("/:id/last-message", mHandler.GetLastMessage)

		msgGroup.POST("/:id/send", mHandler.SendMessage)
		msgGroup.PUT("/message/:id", mHandler.UpdateMessage)
		msgGroup.DELETE("/message/:id", mHandler.DeleteMessage)
	}
}
