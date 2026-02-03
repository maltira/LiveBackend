package dto

import (
	"time"

	"github.com/google/uuid"
)

// * Информация

type MessageResponse struct {
	Message string `json:"message"`
}

type User struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	Password   string    `json:"-"`
	IsVerified bool      `json:"is_verified"`

	CreatedAt time.Time `json:"created_at"`
	DeletedAt time.Time `json:"deleted_at"`
}

// ! Ошибки

type ErrorResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}
