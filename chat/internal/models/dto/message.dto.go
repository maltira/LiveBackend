package dto

import (
	"chat/internal/models"

	"github.com/google/uuid"
)

type MsgCreateRequest struct {
	Content        string     `json:"content" binding:"required,min=1,max=4096"`
	Type           string     `json:"type" binding:"required;oneof=text file system image video"`
	ReplyToMessage *uuid.UUID `json:"reply_to_message"`
}

type MsgUpdateRequest struct {
	Content        string     `json:"content"`
	ReplyToMessage *uuid.UUID `json:"reply_to_message"`
}

type GetMessagesResponse struct {
	Messages []models.Message `json:"messages"`
	Total    int64            `json:"total"`
}
