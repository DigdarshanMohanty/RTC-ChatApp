package services

import (
	"errors"
	"time"

	"chat-backend/config"
	"chat-backend/models"
	"chat-backend/repository"
	"chat-backend/utils"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	users  repository.UserRepository
	config *config.Config
}

func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{users: userRepo, config: cfg}
}

func (s *AuthService) Register(username, password string) (*models.User, error) {
	if len(username) < 3 || len(username) > 20 {
		return nil, errors.New("username must be between 3 and 20 characters")
	}
	if len(password) < 6 || len(password) > 100 {
		return nil, errors.New("password must be between 6 and 100 characters")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return s.users.Create(username, string(hashed))
}

func (s *AuthService) Login(username, password string) (string, *models.User, error) {
	if username == "" || password == "" {
		return "", nil, errors.New("username and password are required")
	}

	u, err := s.users.FindByUsername(username)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return "", nil, errors.New("invalid credentials")
	}
	token, err := s.CreateToken(u.ID, u.Username)
	return token, u, err
}

func (s *AuthService) CreateToken(userID int, username string) (string, error) {
	expiry := time.Duration(s.config.JWTExpiry) * time.Hour
	return utils.GenerateJWT(s.config.JWTSecret, userID, username, expiry)
}

func (s *AuthService) ParseToken(token string) (int, string, error) {
	return utils.ParseJWT(s.config.JWTSecret, token)
}
