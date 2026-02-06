package handler

import (
	"chat/internal/models/dto"
	"chat/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	sc    service.ChatService
	msgSc service.MsgService
}

func NewChatHandler(sc service.ChatService, msgSc service.MsgService) *ChatHandler {
	return &ChatHandler{sc: sc, msgSc: msgSc}
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
	userID := c.MustGet("userID").(uuid.UUID)

	chats, err := h.sc.GetAllChats(userID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, chats)
}

func (h *ChatHandler) CreatePrivateChat(c *gin.Context) {
	uID := c.Query("uid")

	user1ID := c.MustGet("userID").(uuid.UUID)
	user2ID, err := uuid.Parse(uID)
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

	_, _ = h.msgSc.CreateMessage(chat.ID, nil, &dto.MsgCreateRequest{
		Content:        "Группа создана",
		Type:           "system",
		ReplyToMessage: nil,
	})

	c.JSON(200, chat)
}
