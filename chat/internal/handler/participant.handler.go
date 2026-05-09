package handler

import (
	"chat/internal/models/dto"
	"chat/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ParticipantHandler struct {
	sc service.ParticipantService
}

func NewParticipantHandler(sc service.ParticipantService) *ParticipantHandler {
	return &ParticipantHandler{sc: sc}
}

func (h *ParticipantHandler) GetParticipant(c *gin.Context) {
	id := c.Param("id")
	chatID := uuid.MustParse(id)

	uID := c.Query("uid")
	userID, err := uuid.Parse(uID)
	if err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: err.Error()})
		return
	}

	p, err := h.sc.GetParticipantByID(chatID, userID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, p)
}

func (h *ParticipantHandler) GetAllParticipants(c *gin.Context) {
	id := c.Param("id")
	chatID := uuid.MustParse(id)
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректный UUID активного пользователя"})
		return
	}

	isParticipant := h.sc.IsParticipant(chatID, userID)

	if isParticipant {
		p, err := h.sc.GetAllParticipants(chatID)
		if err != nil {
			c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
			return
		}
		c.JSON(200, p)
		return
	}
	c.JSON(403, dto.ErrorResponse{Code: 403, Error: "вы не являетесь участником чата"})
}

func (h *ParticipantHandler) IsParticipant(c *gin.Context) {
	id := c.Param("id")
	chatID := uuid.MustParse(id)
	uID := c.Query("uid")
	userID, err := uuid.Parse(uID)
	if err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: err.Error()})
		return
	}

	flag := h.sc.IsParticipant(chatID, userID)
	c.JSON(200, flag)
}

func (h *ParticipantHandler) JoinToChat(c *gin.Context) {
	id := c.Param("id")
	chatID := uuid.MustParse(id)
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректный UUID активного пользователя"})
		return
	}

	err = h.sc.JoinToChat(chatID, userID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, true)
}

func (h *ParticipantHandler) LeaveChat(c *gin.Context) {
	id := c.Param("id")
	chatID := uuid.MustParse(id)
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректный UUID активного пользователя"})
		return
	}

	err = h.sc.LeaveChat(chatID, userID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(200, true)
}

func (h *ParticipantHandler) Kick(c *gin.Context) {
	id := c.Param("id")
	chatID := uuid.MustParse(id)
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректный UUID активного пользователя"})
		return
	}
	kID := c.Query("uid")
	kickedID, err := uuid.Parse(kID)
	if err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: err.Error()})
		return
	}

	err = h.sc.KickParticipant(userID, chatID, kickedID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, true)
}

func (h *ParticipantHandler) Mute(c *gin.Context) {
	id := c.Param("id")
	chatID := uuid.MustParse(id)
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректный UUID активного пользователя"})
		return
	}
	var req dto.MuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: "Некорректные данные в теле запроса"})
		return
	}

	err = h.sc.MuteParticipant(userID, chatID, req.MutedID, req.UntilDate)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, true)
}

func (h *ParticipantHandler) Unmute(c *gin.Context) {
	id := c.Param("id")
	chatID := uuid.MustParse(id)
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректный UUID активного пользователя"})
		return
	}
	uID := c.Query("uid")
	mutedID, err := uuid.Parse(uID)
	if err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: err.Error()})
		return
	}

	err = h.sc.UnmuteParticipant(userID, chatID, mutedID)
	if err != nil {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(200, true)
}
