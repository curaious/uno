package user

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *UserRepo
}

func NewUserService(repo *UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Authenticate(ctx context.Context, email, password string) (*User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if !user.PasswordAuthEnabled {
		return nil, fmt.Errorf("password authentication is disabled for this user")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id string) (*User, error) {
	return s.repo.GetByID(ctx, id)
}
