package handler

import (
	"log"
	"net/http"
	"user/config"
	"user/internal/models/dto"
	"user/internal/service"
	"user/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BlockHandler struct {
	sc service.BlockService
}

func NewBlockHandler(sc service.BlockService) *BlockHandler {
	return &BlockHandler{sc: sc}
}

// GetAllBlocks
// @Summary      Получить список заблокированных пользователей
// @Description  Возвращает всех пользователей, которых текущий пользователь заблокировал
// @Tags         block
// @Produce      json
// @Success      200  {array} models.Block "Список заблокированных"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/block/all [get]
func (h *BlockHandler) GetAllBlocks(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}
	blocks, err := h.sc.GetAllBlocks(userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, blocks)
}

// IsBlocked
// @Summary      Проверка, заблокирован ли текущий пользователь
// @Description  Проверяет, находится ли текущий пользователь в чёрном списке
// @Tags         block
// @Accept       json
// @Produce      json
// @Param        id path string true "ID пользователя для проверки" Format(uuid)
// @Success      200  {object} bool "true — заблокирован, false — нет"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/block/check [get]
func (h *BlockHandler) IsBlocked(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}
	target := c.Param("id")
	targetID := uuid.MustParse(target)

	isBlocked, err := h.sc.IsBlock(targetID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, isBlocked)
}

// BlockUser
// @Summary      Заблокировать пользователя
// @Description  Добавляет пользователя в чёрный список текущего пользователя
// @Tags         block
// @Produce      json
// @Param        id path string true "ID пользователя для блокировки" Format(uuid)
// @Success      200  {object} models.Block "Пользователь заблокирован"
// @Failure      400  {object} dto.ErrorResponse "Некорректный ID"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/block/{id} [post]
func (h *BlockHandler) BlockUser(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}
	blockID := c.Param("id")
	blockUUID := uuid.MustParse(blockID)

	blockedProfile, err := h.sc.BlockUser(userID, blockUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	// Публикация события (is_blocked: true)
	if err = utils.PublishBlockEvent(userID, blockUUID, true); err != nil {
		log.Printf("Failed to publish block event: %v", err)
	}

	c.JSON(http.StatusOK, blockedProfile)
}

// UnblockUser
// @Summary      Разблокировать пользователя
// @Description  Удаляет пользователя из чёрного списка
// @Tags         block
// @Produce      json
// @Param        id path string true "ID пользователя для разблокировки" Format(uuid)
// @Success      200  {object} dto.MessageResponse "Пользователь разблокирован"
// @Failure      400  {object} dto.ErrorResponse "Некорректный ID"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/block/{id} [delete]
func (h *BlockHandler) UnblockUser(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}
	blockedID := c.Param("id")
	blockedUUID := uuid.MustParse(blockedID)

	err = h.sc.UnblockUser(userID, blockedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	// Публикация события (is_blocked: false)
	if err = utils.PublishBlockEvent(userID, blockedUUID, false); err != nil {
		log.Printf("Failed to publish unblock event: %v", err)
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Success: true, Message: "Пользователь удалён из черного списка"})
}
