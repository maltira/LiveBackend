package service

import (
	"user/models"
	"user/repository"

	"github.com/google/uuid"
)

type UserService interface {
	FindByID(userID uuid.UUID) (*models.Profile, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (sc *userService) FindByID(userID uuid.UUID) (*models.Profile, error) {
	return sc.repo.FindByID(userID)
}
