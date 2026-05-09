package router

import (
	"gateway/config"
	"gateway/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")

	// ====================== AUTH ======================
	setupAuthRoutes(api)

	// ====================== PROTECTED SERVICES ======================
	setupProtectedServices(api)
}

func setupAuthRoutes(api *gin.RouterGroup) {
	auth := api.Group("/auth")

	// Публичные роуты
	setupPublicAuthRoutes(auth)

	// Защищённые роуты auth
	setupProtectedAuthRoutes(auth)
}

func setupProtectedServices(api *gin.RouterGroup) {
	// User service
	userGroup := api.Group("/user")
	userGroup.Use(middleware.AuthMiddleware())
	ProxyToBackendGroup("http://user:"+config.Env.PortUser, userGroup)

	// Chat service
	chatGroup := api.Group("/chat")
	chatGroup.Use(middleware.AuthMiddleware())
	ProxyToBackendGroup("http://chat:"+config.Env.PortChat, chatGroup)
}
