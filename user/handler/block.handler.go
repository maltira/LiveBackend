package handler

import (
	"net/http"
	"user/models/dto"
	"user/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BlockHandler struct {
	sc service.BlockService
}

func NewBlockHandler(sc service.BlockService) *BlockHandler {
	return &BlockHandler{sc: sc}
}

func (h *BlockHandler) GetAllBlocks(c *gin.Context) {
	id := c.MustGet("userID").(uuid.UUID)
	blocks, err := h.sc.GetAllBlocks(id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, blocks)
}

func (h *BlockHandler) IsUserBlocked(c *gin.Context) {
	var req dto.BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: http.StatusBadRequest, Error: "Некорректные данные в теле запроса"})
		return
	}

	isBlocked, err := h.sc.IsBlock(req.ProfileID, req.BlockedProfileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, isBlocked)
}

func (h *BlockHandler) BlockUser(c *gin.Context) {
	id := c.MustGet("userID").(uuid.UUID)
	blockID := c.Param("id")
	blockUUID := uuid.MustParse(blockID)

	err := h.sc.BlockUser(id, blockUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Пользователь добавлен в черный список"})
}

func (h *BlockHandler) UnblockUser(c *gin.Context) {
	id := c.MustGet("userID").(uuid.UUID)
	blockedID := c.Param("id")
	blockedUUID := uuid.MustParse(blockedID)

	err := h.sc.UnblockUser(id, blockedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Пользователь удалён из черного списка"})
}
