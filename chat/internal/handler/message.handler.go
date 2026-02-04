package handler

import (
	"chat/internal/models/dto"
	"chat/internal/service"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MsgHandler struct {
	sc service.MsgService
}

func NewMsgHandler(sc service.MsgService) *MsgHandler {
	return &MsgHandler{sc: sc}
}

func (h *MsgHandler) GetMessages(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	id := c.Param("id")
	chatID := uuid.MustParse(id)

	limit := 50
	offset := 0
	if l := c.Query("offset"); l != "" {
		limit, _ = strconv.Atoi(l)
	}
	if o := c.Query("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	}

	messages, total, err := h.sc.GetMessages(userID, chatID, limit, offset)
	if err != nil {
		if err.Error() == "вы не являетесь участником чата" {
			c.JSON(403, dto.ErrorResponse{Code: 403, Error: err.Error()})
			return
		}
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, dto.GetMessagesResponse{Messages: messages, Total: total})
}

func (h *MsgHandler) GetLastMessage(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	id := c.Param("id")
	chatID := uuid.MustParse(id)

	msg, err := h.sc.GetLastMessage(chatID, userID)
	if err != nil {
		if err.Error() == "вы не являетесь участником чата" {
			c.JSON(403, dto.ErrorResponse{Code: 403, Error: err.Error()})
			return
		}
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, msg)
}

func (h *MsgHandler) SendMessage(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	id := c.Param("id")
	chatID := uuid.MustParse(id)

	var req *dto.MsgCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: "Некорректные данные в теле запроса"})
		return
	}

	msg, err := h.sc.CreateMessage(chatID, &userID, req)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, msg)
}

func (h *MsgHandler) UpdateMessage(c *gin.Context) {
	id := c.Param("id")
	msgID := uuid.MustParse(id)
	userID := c.MustGet("userID").(uuid.UUID)

	var req *dto.MsgUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: "Некорректные данные в теле запроса"})
		return
	}

	err := h.sc.UpdateMessage(msgID, userID, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(403, dto.ErrorResponse{Code: 403, Error: "Сообщение не найдено или вы не являетесь автором"})
			return
		}
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(200, true)
}

func (h *MsgHandler) DeleteMessage(c *gin.Context) {
	id := c.Param("id")
	msgID := uuid.MustParse(id)
	userID := c.MustGet("userID").(uuid.UUID)

	err := h.sc.DeleteMessage(msgID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(403, dto.ErrorResponse{Code: 403, Error: "Сообщение не найдено или вы не являетесь автором"})
			return
		}
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, true)
}
