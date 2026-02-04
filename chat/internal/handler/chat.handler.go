package handler

import (
	"chat/internal/models/dto"
	"chat/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	sc service.ChatService
}

func NewChatHandler(sc service.ChatService) *ChatHandler {
	return &ChatHandler{sc: sc}
}

func (h *ChatHandler) IsChatExists(c *gin.Context) {
	id := c.Param("id")
	chatID := uuid.MustParse(id)

	exist, err := h.sc.IsChatExists(chatID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, exist)
}

func (h *ChatHandler) GetAllChats(c *gin.Context) {
	userID := c.MustGet("UserID").(uuid.UUID)

	chats, err := h.sc.GetAllChats(userID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, chats)
}

func (h *ChatHandler) CreatePrivateChat(c *gin.Context) {
	fID := c.Query("fid")
	sID := c.Query("sid")

	user1ID, err := uuid.Parse(fID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: "Неверный формат для fID (uuid)"})
		return
	}
	user2ID, err := uuid.Parse(sID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: "Неверный формат для sID (uuid)"})
		return
	}

	chat, err := h.sc.CreatePrivateChat(user1ID, user2ID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, chat)
}

func (h *ChatHandler) CreateGroupChat(c *gin.Context) {
	var req *dto.ChatCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: "Некорретные данные в теле запроса"})
		return
	}

	chat, err := h.sc.CreateGroupChat(req)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, chat)
}
