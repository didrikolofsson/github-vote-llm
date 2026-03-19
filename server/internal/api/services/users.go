package services

import "context"

type UserService interface {
	Signup(ctx context.Context, user *models.User) (*models.User, error)
	Login(ctx context.Context, user *models.User) (*models.User, error)
	Logout(ctx context.Context, user *models.User) (*models.User, error)
}

type userService struct {
}

func NewUserService() UserService {
	return &userService{}
}

func (s *userService) Signup(ctx context.Context, user *models.User) (*models.User, error) {
	return nil, nil
}

func (s *userService) Login(ctx context.Context, user *models.User) (*models.User, error) {
	return nil, nil
}

func (s *userService) Logout(ctx context.Context, user *models.User) (*models.User, error) {
	return nil, nil
}
